# Wire together four focused modules: network, ecr, logging, ecs.

module "network" {
  source                = "./modules/network"
  service_name          = var.service_name
  container_port        = var.container_port
  alb_security_group_id = module.alb.alb_security_group_id
  my_ip_cidr            = var.my_ip_cidr
}

module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = var.log_retention_days
}

# Reuse an existing IAM role for ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

module "ecs" {
  source             = "./modules/ecs"
  service_name       = var.service_name
  image              = "${module.ecr.repository_url}:latest"
  container_port     = var.container_port
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [module.network.security_group_id]
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name     = module.logging.log_group_name
  ecs_count          = var.ecs_count
  region             = var.aws_region
  target_group_arn   = module.alb.target_group_arn

  # The service can only attach to the target group once the listener exists, and its
  # tasks can only start once the image has been pushed to ECR.
  depends_on = [module.alb, docker_registry_image.app]
}

module "alb" {
  source         = "./modules/alb"
  container_port = var.container_port
  service_name   = var.service_name
  vpc_id         = module.network.vpc_id
  subnet_id      = module.network.subnet_ids
}

module "autoscaling" {
  source       = "./modules/autoscaling"
  cluster_name = module.ecs.cluster_name
  service_name = module.ecs.service_name
  target_cpu   = var.target_cpu
}

// Build & push the Go app image into ECR
resource "docker_image" "app" {
  # Use the URL from the ecr module, and tag it "latest"
  name = "${module.ecr.repository_url}:latest"

  build {
    # relative path from terraform/ → src/
    context = "../src"
    # Dockerfile defaults to "Dockerfile" in that context
  }
}

resource "docker_registry_image" "app" {
  # this will push :latest → ECR
  name = docker_image.app.name
}

module "monitoring" {
  source                  = "./modules/monitoring"
  service_name            = module.ecs.service_name
  region                  = var.aws_region
  cluster_name            = module.ecs.cluster_name
  alb_arn_suffix          = module.alb.alb_arn_suffix
  target_group_arn_suffix = module.alb.tg_arn_suffix
  target_cpu              = var.target_cpu # the line drawn on the CPU chart
}
