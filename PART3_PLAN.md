# HW6 Part III: Horizontal Scaling with Auto Scaling, Task Plan

Tick each box as you finish it. Estimates are for you doing the work. **AWS wall-clock time** is called out separately, because it is what costs credits and Learner Lab session time.

## 0. Decision: do not destroy AWS yet

**Keep the current deployment until Part III is finished.** Part III extends the same Terraform project (it adds an ALB, a target group, auto scaling and a dashboard to the existing ECS service), so there is nothing to gain by destroying and rebuilding first. `terraform apply` will modify the service in place.

Destroy now **only if** you are stopping for more than about a day. Everything is rebuilt by Terraform in about 5 minutes, and all Part II evidence is already saved.

| Before any destroy | Done? |
|--|--|
| Part II screenshots saved in `images/hw6/` and copied to `NU-CS6650/hw6/images/` | [ ] |
| Locust CSV and HTML results kept (`results/` is gitignored, so back it up: `cp -R results ~/Desktop/hw6_part2_results`) | [ ] |
| Report and README committed and pushed, Part II tagged: `git tag hw6-part2-final` | [ ] |
| Part II numbers you will compare against are written down (below) | [ ] |

**Part II baseline to beat (the load that "broke" it):** 60 users, 1 task: about 1,500 req/s ceiling, CPU 97.7 %, median 27 ms, **p95 96 ms, p99 180 ms**, 0 failures.
**Note:** the 20-user test did **not** break the service (CPU 53 %, below the 70 % scaling target), so the Core Test must use the **60-user** run, not 20 users.

**Rough running cost** (approximate, us-east-1 list prices, check your lab's budget): one 0.25 vCPU task about $0.30/day; an ALB about $0.54/day plus load units; public IPv4 addresses about $0.12/day each. Part III with 2 to 4 tasks is roughly **$1 to $2 per day**. Cheap, but it keeps running until you destroy it. Ending a lab session does not reliably delete resources, so always destroy explicitly at the end (section 9).

---

## 1. Prepare

- [ ] Create a branch for this work: `git switch -c hw6-part3`.
- [ ] Decide the answers to the open questions (table below).
- [ ] Check AWS credentials work (`aws sts get-caller-identity`). Refresh the Learner Lab session if needed.

| Open question | Recommendation |
|--|--|
| Which test is "the load that broke Part II"? | The **60-user, 3-minute** run (`-u 60 -r 20`). Run it for **12 minutes** this time, because auto scaling needs several minutes to react (section 5) |
| How do I keep the HW5 dashboard "side by side" when tasks come and go? | Prometheus scrapes **each task's IP** through `file_sd` (a small script refreshes the list). Grafana panels then aggregate with `sum by (route)` and split with `instance`. Fallback: Grafana with the CloudWatch data source and the ALB metrics |
| Per-instance CPU? | Two views: CloudWatch Container Insights, and Prometheus `rate(process_cpu_seconds_total[30s]) / 0.25` per `instance` (the default Go collector already exports it) |

## 2. Change the service (about 30 min)

The search logic stays **unchanged**. Only what the load balancer needs is added.

- [ ] Add `GET /health` returning `200` with `{"status":"ok"}`.
- [ ] Exclude `/health` from the metrics and the request log (like `/metrics`), because the ALB calls it every 30 seconds **per target** and would pollute the dashboards.
- [ ] Test locally (`curl localhost:8080/health`, and confirm it does not show up in `/metrics`).
- [ ] Rebuild the image later with the Terraform `-replace` flags (README, "Changing the code and redeploying").

No other code change is needed. Request distribution across tasks is visible without code changes: each task writes to its **own CloudWatch log stream**, and the logs already contain `endpoint`, `status` and `duration_ms`.

## 3. Terraform (about 2 to 3 hours)

Add the pieces as modules in the style you already use. Make the scaling numbers **variables**, so the exploration phase is a `terraform apply -var ...` and not a rewrite.

| Piece | Settings from the assignment |
|--|--|
| **ALB** (internet-facing, default VPC subnets in at least 2 availability zones) and a listener on port 80 forwarding to the target group | Security group: allow HTTP 80 from the internet |
| **Target group** | Target type **ip**, HTTP, port **8080**, health check path **`/health`**, interval **30 s**, healthy threshold **2**. Also set: unhealthy threshold 2, timeout 5 s, matcher 200, **deregistration delay 30 s** (the default of 300 s makes scale-in and the resilience test very slow; say so in the report) |
| **ECS service** | `load_balancer { target_group_arn, container_name, container_port }`; set `desired_count` to the minimum (2) and add `lifecycle { ignore_changes = [desired_count] }` so Terraform does not fight the auto scaler; consider `health_check_grace_period_seconds` (60) |
| **Security groups** | ECS tasks accept 8080 **from the ALB's security group**. For the Prometheus per-task scraping, also allow 8080 from your own IP (temporary) |
| **Auto scaling** | `aws_appautoscaling_target` (ECS service, min **2**, max **4**) and `aws_appautoscaling_policy`: target tracking, **ECSServiceAverageCPUUtilization**, target **70**, scale-out cooldown **300**, scale-in cooldown **300** |
| **Container Insights** | `setting { name = "containerInsights", value = "enabled" }` on the cluster, for desired vs running task count and per-task data (small extra cost) |
| **CloudWatch dashboard** | ECS CPU and memory, **running vs desired task count**, ALB `RequestCount`, `TargetResponseTime`, `HealthyHostCount` and `UnHealthyHostCount`, `HTTPCode_Target_5XX_Count`, `HTTPCode_ELB_5XX_Count` |
| **Outputs** | ALB DNS name (so you do not have to look it up in the console) |

- [ ] Write the modules and variables.
- [ ] `terraform plan` and read it (no surprises such as a replaced cluster).
- [ ] **Risk check early:** Application Auto Scaling needs a service-linked role. In your account it did not show up in a quick `iam list-roles` check. If `terraform apply` fails on the scalable target with an IAM error, that is the Learner Lab restriction. Apply just the autoscaling resources first (`-target`) so you find out before building everything else.

## 4. Deploy and validate (about 45 min, AWS time starts)

- [ ] `terraform apply`, then wait for the tasks to start.
- [ ] Target group shows **2 healthy targets** (EC2 console, Target Groups).
- [ ] `curl "http://<ALB-DNS>/health"` and `curl "http://<ALB-DNS>/products/search?q=alpha"` return 200.
- [ ] Each task has its own log stream in `/ecs/CS6650L2`.
- [ ] The scalable target and the policy exist: `aws application-autoscaling describe-scalable-targets --service-namespace ecs --region us-east-1`.
- [ ] CloudWatch dashboard loads with data.
- [ ] Set up the Prometheus per-task scraping and recreate the Grafana panels (rate, errors, p95, per-instance CPU, task count via `count(up)`). Both dashboards open side by side.
- [ ] Take a screenshot of the idle state: 2 tasks, 2 healthy targets.

## 5. The Core Test (about 1 hour of AWS time)

Read the observation list first so the test runs **once**. Use the **ALB DNS name** as the host. Same class, same terms, same ramp as the Part II breaking run, but a longer duration:

```bash
locust -f test/locustfile.py --host http://<ALB-DNS> -u 60 -r 20 -t 12m --autostart --autoquit 600 --csv results/p3_core --csv-full-history --html results/p3_core.html ProductSearchUser
```

(No `:8080`: the ALB listens on port 80.) Write down the start time in UTC. Check Locust's own CPU with `top -o cpu -n 5` once; with 60 users from the laptop it should be fine, otherwise add `--processes 2`.

**Why 12 minutes:** the CPU target-tracking alarm needs about **3 consecutive 1-minute readings** above the target. After that the new task needs about 1 to 2 minutes to start and about 1 minute of health checks (2 successes at 30-second intervals) before the ALB sends it traffic. So the first scale-out is expected at roughly **4 to 6 minutes**, and a 3-minute test would end before anything happens.

**Expected behavior (a prediction, to be checked):** with 2 tasks, 60 users demand up to about 2,400 req/s from the laptop, so about 1,200 req/s per task, about **75 to 80 % CPU per task**. That is above 70 %, so a third task should be added. After that, about 800 req/s per task, about 50 % CPU. If the CPU stays below 70 % and nothing scales, rerun with more users (for example `-u 100`) and document that.

**Record during the run** (timestamps from your clock, and screenshots at each of these points):

| Where | What to record |
|--|--|
| **ECS service, Tasks** | Task count over time. When does it go from 2 to 3 (and 4)? How long from the start of load? |
| **Target group** | Healthy targets over time. What happens when a new task registers (initial, then healthy after 2 checks)? |
| **CloudWatch, CPU per instance** | Do the tasks have similar CPU (the ALB distributes evenly), or does a new task start low? What happens to the average after scale-out? |
| **ALB** | `RequestCount`, `TargetResponseTime`, 5XX counts |
| **Locust** | Median, p95, p99 vs the Part II run. Any failures during scaling? Is the service available the whole time? |
| **Prometheus / Grafana** | Rate, errors, p95 by route; per-instance CPU; instance count |
| **Auto scaling** | Desired vs running task count, and the scaling activity history: `aws application-autoscaling describe-scaling-activities --service-namespace ecs --region us-east-1` |
| **Log streams** | Request count per log stream (Logs Insights: `stats count(*) by @logStream`) to show how load is distributed |

After the test, **leave the idle service running for about 20 minutes** to watch **scale-in**. Target tracking removes capacity only after about 15 minutes below the target, plus the 300-second scale-in cooldown, so this is slow by design.

- [ ] Core Test run and screenshots taken.
- [ ] Scaling activity history saved (text).
- [ ] Scale-in observed.

## 6. Resilience test (about 30 min of AWS time)

- [ ] Start the same 60-user load (a shorter 8-minute run is enough).
- [ ] After the load is steady, ECS console, Tasks, select a running task, **Stop**.
- [ ] Observe and record: the target going `draining` and the target group's healthy count, ECS starting a replacement task, the time until the replacement is healthy, and **Locust failures during that time** (a few 502 or 504 responses from requests that were in flight on the stopped task are possible; record the count and the time).
- [ ] Does the load test continue successfully? Is the throughput and latency affected, and for how long?

## 7. Exploration (budget about 3 hours of AWS time; run each experiment once, in this order)

Change the setting with `terraform apply -var ...` (no image rebuild) unless noted. Write a one-line **prediction before each run**, then the observation.

| # | Experiment | Change | Question to answer |
|--|--|--|--|
| 1 | **Scaling target** | 50 %, 70 % (done in the Core Test), 90 % | Time to first scale-out, final task count, and the p95/p99 during the transition |
| 2 | **Min / max** | min 1 and max 2; min 2 and max 4 (done); min 4 and max 4 | Does a larger minimum avoid the latency spike? What does a lower maximum do under the same load? |
| 3 | **Cooldowns** | 60 s vs 300 s (done) vs 600 s, for scale-out and for scale-in | Does a shorter cooldown react sooner? Does it flap? |
| 4 | **Users** | 20, 60 (done), 100 or 200 | Where does the system run out of the maximum of 4 tasks? (4 tasks saturate at about 6,000 req/s, and the laptop cannot generate that: use `--processes` or Locust from CloudShell) |
| 5 | **Duration** | the Core Test with 25 minutes | Does it stabilize? Does it ever scale in while load is steady? |
| 6 | **Load shape: ramp** | Gradual increase (the existing `shape_step.py`: 10, 25, 50, 100 users in steps) | Does scaling keep up with a slow ramp? |
| 7 | **Load shape: spike** | Idle (5 users), then jump to 100 users | How long is the service overloaded before new tasks help? (needs a new `shape_spike.py`; the README mentions one that is not in the repo) |
| 8 | **Failure: stop several tasks** | Stop 2 of 3 tasks during load | Failure rate, recovery time |
| 9 | **Failure: all health checks fail** | `aws elbv2 modify-target-group --health-check-path /nope`, then restore it | What does the ALB do when no target is healthy? (an ALB with no healthy targets typically routes to all of them anyway, "fail open"; check and record) |

- [ ] Experiments 1 to 9 done, each with prediction, observation and one screenshot.

**Session-time note:** the Learner Lab session has a time limit and credentials expire. Group experiments by what needs a `terraform apply` (1 to 3) and run those back to back. Check the remaining session time before starting a long experiment.

## 8. Report and README (about 2 hours)

Add Part III to `NU-CS6650/hw6/report.md` and a short Part III section to the README. The deliverable asks for:

- [ ] **How the system solved the Part II bottleneck:** the saturated single task (97.7 % CPU, p95 96 ms, p99 180 ms) vs the scaled service (same load, per-task CPU, latency, tasks added).
- [ ] **Role of each component:** ALB (single entry point, distributes requests, terminates the client connection), target group (registers task IPs, runs health checks, only healthy targets get traffic, draining), auto scaling (changes the task count from a CloudWatch metric, with cooldowns).
- [ ] **Trade-offs vs vertical scaling:** vertical = bigger task (no code or topology change, hard ceiling, restart or downtime to resize, single point of failure); horizontal = more tasks (near-linear capacity, survives a task failure, scales in and out; needs a load balancer, stateless service, health checks, shared-nothing data, and costs the ALB). Your in-memory data is **identical in every task**, so this works only because it is read-only; a writable store would need an external database.
- [ ] **Predictions for different load patterns:** gradual ramp, spike, sustained, oscillating, backed by experiments 6 to 9.
- [ ] **Answer to the CPU-vs-request-metric question** (below), backed by what both dashboards showed.
- [ ] **Exploration findings** (section 7) and the resilience observations.
- [ ] Before/after comparison table with Part II.

**Answer outline for "would another trigger metric scale sooner or later than CPU?"** (verify against your dashboards; do not copy it without evidence):
* For this workload, CPU is **proportional to request rate** (0.16 to 0.19 ms per request, Part II). That makes CPU a good **leading** signal.
* **Request duration is a lagging, non-linear signal:** the median did not move from 5 to 60 users, and p95 and p99 only rose once the CPU was near its limit. A duration trigger would fire **later**, after users already feel it, and it is noisy because network time is included.
* **Requests per target** (`ALBRequestCountPerTarget`) is also proportional and leading, and does not depend on the per-request cost, so it would scale at about the **same time** as CPU, and it is easy to calibrate from Part II (about 900 to 1,000 req/s per task at 60 to 70 % CPU).
* **Requests in flight** would rise at saturation (queueing) and so would also lag CPU.
* A defensible pick: **CPU or requests per target**, with the number taken from the measured capacity of one task. Your scale-out timeline (time from load start to the first new task) is the evidence.

## 9. Clean up (do not skip)

- [ ] `cd terraform && terraform destroy -auto-approve`
- [ ] Confirm nothing is left (the ALB and the public IPs are the costly leftovers):
```bash
aws elbv2 describe-load-balancers --region us-east-1 --query 'LoadBalancers[].LoadBalancerName'
aws ecs list-clusters --region us-east-1
aws application-autoscaling describe-scalable-targets --service-namespace ecs --region us-east-1
```
- [ ] Revert `src/prometheus.yaml` if you changed it, and do not commit AWS addresses.

## Estimated time

| Block | Work | AWS time |
|--|--|--|
| 0 to 2 Prepare, decisions, service change | about 1.5 h | none |
| 3 Terraform | about 2.5 h | none |
| 4 Deploy, validate, dashboards | about 1.5 h | about 1 h |
| 5 Core Test (12 min + scale-in watch) | about 1 h | about 1 h |
| 6 Resilience | about 0.5 h | about 0.5 h |
| 7 Exploration | about 3.5 h | about 3 h |
| 8 Report and README | about 2.5 h | none |
| **Total** | **about 13 hours** | **about 5.5 hours** |

## Risks

| Risk | What to do |
|--|--|
| Learner Lab blocks creating the Application Auto Scaling service-linked role | Test the scalable target early (`-target`), before building the rest |
| Targets show unhealthy | ALB security group to ECS 8080, `/health` works, container logs |
| 3-minute tests end before anything scales | Use 12 minutes or longer |
| Load does not exceed 70 % CPU with 2 tasks | More users, or a 50 % target (experiment 1) |
| Terraform and auto scaling fight over the task count | `ignore_changes = [desired_count]` |
| Slow scale-in and slow stop of tasks | Deregistration delay 30 s (document it) |
| Laptop cannot generate the load for 4 tasks | `--processes`, or run Locust from CloudShell |
| Session expiry in the middle of an experiment | Check the remaining time, batch the `terraform apply` experiments |
