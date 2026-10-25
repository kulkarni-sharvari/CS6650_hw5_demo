# One CloudWatch dashboard with the infrastructure view (CPU, memory, task count, target
# health) and the load balancer's request view (request count, response time, errors).
locals {
  ecs_dims = ["ClusterName", var.cluster_name, "ServiceName", var.service_name]
  alb_dim  = ["LoadBalancer", var.alb_arn_suffix]
  tg_dims  = ["LoadBalancer", var.alb_arn_suffix, "TargetGroup", var.target_group_arn_suffix]
}

resource "aws_cloudwatch_dashboard" "this" {
  dashboard_name = "${var.service_name}-scaling"

  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric", x = 0, y = 0, width = 12, height = 6
        properties = {
          title  = "ECS service CPU utilization (% of the task's reserved CPU)"
          region = var.region
          period = 60
          stat   = "Average"
          metrics = [
            concat(["AWS/ECS", "CPUUtilization"], local.ecs_dims, [{ label = "average" }]),
            concat(["AWS/ECS", "CPUUtilization"], local.ecs_dims, [{ label = "maximum", stat = "Maximum" }]),
          ]
          annotations = { horizontal = [{ label = "scaling target", value = var.target_cpu }] }
          yAxis       = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 12, y = 0, width = 12, height = 6
        properties = {
          title  = "ECS service memory utilization (%)"
          region = var.region
          period = 60
          stat   = "Average"
          metrics = [
            concat(["AWS/ECS", "MemoryUtilization"], local.ecs_dims, [{ label = "average" }]),
            concat(["AWS/ECS", "MemoryUtilization"], local.ecs_dims, [{ label = "maximum", stat = "Maximum" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 0, y = 6, width = 12, height = 6
        properties = {
          title  = "Tasks: desired vs running (Container Insights)"
          region = var.region
          period = 60
          stat   = "Average"
          metrics = [
            concat(["ECS/ContainerInsights", "DesiredTaskCount"], local.ecs_dims, [{ label = "desired" }]),
            concat(["ECS/ContainerInsights", "RunningTaskCount"], local.ecs_dims, [{ label = "running" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 12, y = 6, width = 12, height = 6
        properties = {
          title  = "Target group: healthy vs unhealthy targets"
          region = var.region
          period = 60
          stat   = "Average"
          metrics = [
            concat(["AWS/ApplicationELB", "HealthyHostCount"], local.tg_dims, [{ label = "healthy" }]),
            concat(["AWS/ApplicationELB", "UnHealthyHostCount"], local.tg_dims, [{ label = "unhealthy" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 0, y = 12, width = 12, height = 6
        properties = {
          title  = "ALB request count (per minute)"
          region = var.region
          period = 60
          stat   = "Sum"
          metrics = [
            concat(["AWS/ApplicationELB", "RequestCount"], local.alb_dim, [{ label = "requests" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 12, y = 12, width = 12, height = 6
        properties = {
          title  = "ALB target response time (seconds)"
          region = var.region
          period = 60
          stat   = "p95"
          metrics = [
            concat(["AWS/ApplicationELB", "TargetResponseTime"], local.alb_dim, [{ label = "p50", stat = "p50" }]),
            concat(["AWS/ApplicationELB", "TargetResponseTime"], local.alb_dim, [{ label = "p95", stat = "p95" }]),
            concat(["AWS/ApplicationELB", "TargetResponseTime"], local.alb_dim, [{ label = "p99", stat = "p99" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
      {
        type = "metric", x = 0, y = 18, width = 12, height = 6
        properties = {
          title  = "Errors: 5XX from the targets and from the ALB"
          region = var.region
          period = 60
          stat   = "Sum"
          metrics = [
            concat(["AWS/ApplicationELB", "HTTPCode_Target_5XX_Count"], local.alb_dim, [{ label = "5XX from tasks" }]),
            concat(["AWS/ApplicationELB", "HTTPCode_ELB_5XX_Count"], local.alb_dim, [{ label = "5XX from the ALB (502/503/504)" }]),
          ]
          yAxis = { left = { min = 0 } }
        }
      },
    ]
  })
}
