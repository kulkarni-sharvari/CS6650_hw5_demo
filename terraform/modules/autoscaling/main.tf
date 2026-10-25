resource "aws_appautoscaling_target" "auto_appautoscaling" {
  min_capacity       = var.min_task
  max_capacity       = var.max_task
  resource_id        = "service/${var.cluster_name}/${var.service_name}"
  scalable_dimension = "ecs:service:DesiredCount" # "the thing that changes is the task count"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "cpu" {
  name               = "${var.service_name}-cpu-target"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.auto_appautoscaling.resource_id
  scalable_dimension = aws_appautoscaling_target.auto_appautoscaling.scalable_dimension
  service_namespace  = aws_appautoscaling_target.auto_appautoscaling.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = var.target_cpu
    scale_in_cooldown  = var.scale_in_cooldown
    scale_out_cooldown = var.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}