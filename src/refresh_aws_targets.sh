#!/usr/bin/env bash
# Keeps Prometheus pointed at every running ECS task.
#
# Prometheus cannot scrape the load balancer: each scrape would land on a random task and
# the per-task counters would be mixed. This script lists the public IP of every running
# task and writes them to aws-targets/targets.json, which Prometheus reads (file_sd).
# Tasks that scale out show up within one refresh, tasks that stop disappear.
#
# Usage: ./refresh_aws_targets.sh          # refresh every 15 seconds until Ctrl+C
#        ./refresh_aws_targets.sh --once   # refresh one time and print the targets
set -euo pipefail

CLUSTER="${CLUSTER:-CS6650L2-cluster}"
SERVICE="${SERVICE:-CS6650L2}"
REGION="${AWS_REGION:-us-east-1}"
PORT="${PORT:-8080}"
INTERVAL="${INTERVAL:-15}"
OUT="${OUT:-$(cd "$(dirname "$0")" && pwd)/aws-targets/targets.json}"

refresh() {
  local arns tasks nics tmp
  arns=$(aws ecs list-tasks --cluster "$CLUSTER" --service-name "$SERVICE" \
    --desired-status RUNNING --region "$REGION" --query 'taskArns[]' --output text)

  mkdir -p "$(dirname "$OUT")"
  tmp="$OUT.tmp"
  if [ -z "$arns" ]; then
    echo '[]' > "$tmp"; mv "$tmp" "$OUT"; return
  fi

  # shellcheck disable=SC2086
  tasks=$(aws ecs describe-tasks --cluster "$CLUSTER" --tasks $arns --region "$REGION" --output json)
  enis=$(jq -r '[.tasks[] | select(.lastStatus=="RUNNING") | .attachments[].details[]
                 | select(.name=="networkInterfaceId") | .value] | join(" ")' <<<"$tasks")
  if [ -z "$enis" ]; then
    echo '[]' > "$tmp"; mv "$tmp" "$OUT"; return
  fi
  # shellcheck disable=SC2086
  nics=$(aws ec2 describe-network-interfaces --network-interface-ids $enis --region "$REGION" --output json)

  # One Prometheus target per running task, labelled with the first 8 characters of the task id.
  jq -n --argjson tasks "$tasks" --argjson nics "$nics" --arg port "$PORT" '
    ($nics.NetworkInterfaces
      | map({key: .NetworkInterfaceId, value: .Association.PublicIp}) | from_entries) as $ip
    | [ $tasks.tasks[] | select(.lastStatus=="RUNNING")
        | (.attachments[].details[] | select(.name=="networkInterfaceId") | .value) as $eni
        | select($ip[$eni] != null)
        | {targets: [($ip[$eni] + ":" + $port)],
           labels: {task: (.taskArn | split("/") | last | .[0:8])}} ]' > "$tmp"
  mv "$tmp" "$OUT"   # atomic replace, so Prometheus never reads a half-written file
}

if [ "${1:-}" = "--once" ]; then
  refresh
  cat "$OUT"
  exit 0
fi

echo "Refreshing $OUT every ${INTERVAL}s for $CLUSTER/$SERVICE (Ctrl+C to stop)"
while true; do
  if refresh; then
    echo "$(date +%H:%M:%S) $(jq -r '[.[].targets[0]] | join(" ")' "$OUT")"
  else
    echo "$(date +%H:%M:%S) refresh failed (expired AWS credentials?)" >&2
  fi
  sleep "$INTERVAL"
done
