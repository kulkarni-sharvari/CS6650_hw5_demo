variable "service_name" {
  type        = string
  description = "Base name used to name the ALB, target group and security group"
}

variable "container_port" {
  type        = number
  description = "Port the app listens on inside each task (8080). The target group sends traffic to this port"
}

variable "vpc_id" {
  type        = string
  description = "ID of the VPC that holds the ALB, the target group and the tasks"
}

variable "subnet_id" {
  type        = list(string)
  description = "Subnet IDs the ALB is placed in. It needs at least two, in different availability zones"
}
