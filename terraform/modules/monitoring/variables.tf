variable "service_name" {
  type        = string
  description = "Name of the ECS service (also used to name the dashboard)"
}

variable "cluster_name" {
  type        = string
  description = "Name of the ECS cluster"
}

variable "region" {
  type        = string
  description = "AWS region the metrics are read from"
}

variable "alb_arn_suffix" {
  type        = string
  description = "ARN suffix of the ALB (the LoadBalancer dimension of ALB metrics)"
}

variable "target_group_arn_suffix" {
  type        = string
  description = "ARN suffix of the target group (the TargetGroup dimension of ALB metrics)"
}

variable "target_cpu" {
  type        = number
  default     = 70
  description = "CPU target of the auto scaling policy, drawn as a line on the CPU chart"
}
