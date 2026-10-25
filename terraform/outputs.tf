output "ecs_cluster_name" {
  description = "Name of the created ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the running ECS service"
  value       = module.ecs.service_name
}

output "alb_dns_name" {
  description = "DNS name of the load balancer: the host to give Locust (http://<this>/products/search?q=...)"
  value       = module.alb.alb_dns_name
}

output "cloudwatch_dashboard_name" {
  description = "Name of the CloudWatch dashboard with the scaling metrics"
  value       = module.monitoring.dashboard_name
}
