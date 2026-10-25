# Region to deploy into
variable "aws_region" {
  type    = string
  default = "us-east-1" # "us-west-2"
}

# ECR & ECS settings
variable "ecr_repository_name" {
  type    = string
  default = "ecr_service"
}

variable "service_name" {
  type    = string
  default = "CS6650L2"
}

variable "container_port" {
  type    = number
  default = 8080
}

variable "ecs_count" {
  type    = number
  default = 2 # the auto scaling minimum
}

# How long to keep logs
variable "log_retention_days" {
  type    = number
  default = 7
}

variable "my_ip_cidr" {
  type        = string
  description = "My public IP. allowed to reach port 8080 on the tasks for Prometheus scraping"
}

variable "target_cpu" {
  type        = number
  default     = 70
  description = "Average CPU % the auto scaler tries to hold. Change it with: terraform apply -var target_cpu=50"
}
