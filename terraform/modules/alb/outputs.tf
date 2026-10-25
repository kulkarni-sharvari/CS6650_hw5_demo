output "alb_dns_name" {
  description = "Public DNS name of the Application Load Balancer (the host to give Locust)"
  value       = aws_lb.this.dns_name
}

output "target_group_arn" {
  description = "ARN of the target group the ECS service registers its tasks with"
  value       = aws_lb_target_group.this.arn
}

output "alb_arn_suffix" {
  description = "Suffix of the ALB ARN, used as the LoadBalancer dimension of CloudWatch metrics"
  value       = aws_lb.this.arn_suffix
}

output "tg_arn_suffix" {
  description = "Suffix of the target group ARN, used as the TargetGroup dimension of CloudWatch metrics"
  value       = aws_lb_target_group.this.arn_suffix
}

output "alb_security_group_id" {
  description = "ID of the ALB security group; the tasks accept traffic from it"
  value       = aws_security_group.alb-sg.id
}
