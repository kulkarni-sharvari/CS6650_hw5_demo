variable "cluster_name" {
  type        = string
  description = "Name of the ECS cluster"
}

variable "service_name" {
  type        = string
  description = "Name of the ECS service"
}

variable "min_task" {
  type        = number
  default     = 2
  description = "Min number of tasks"
}

variable "max_task" {
  type        = number
  default     = 4
  description = "Miax number of tasks"
}

variable "target_cpu" {
  type        = number
  default     = 70
  description = " The average CPU % to hold"
}

variable "scale_out_cooldown" {
  type        = number
  default     = 300
  description = " Seconds to wait after adding tasks"
}

variable "scale_in_cooldown" {
  type        = number
  default     = 300
  description = " Seconds to wait after removing tasks"
}