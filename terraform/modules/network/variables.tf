variable "service_name" {
  description = "Base name for SG"
  type        = string
}
variable "container_port" {
  description = "Port to expose on the SG"
  type        = number
}
variable "cidr_blocks" {
  description = "Which CIDRs can reach the service"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "alb_security_group_id" {
  description = "Allow inbound only by ALB SG"
  type        = string
}

variable "my_ip_cidr" {
  type        = string
  description = "My public IP. allowed to reach port 8080 on the tasks for Prometheus scraping"
}
