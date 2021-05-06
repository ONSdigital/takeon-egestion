# Importing existing resources for lambdas to use
data "aws_caller_identity" "current" {}
data "aws_security_group" "vpc-private" {
  filter {
    name   = "tag:Name"
    values = ["${local.vpc_prefix}-vpc-private"]
  }
}

data "aws_vpc" "main" {
  filter {
    name   = "tag:Name"
    values = ["${local.vpc_prefix}-vpc"]
  }
}


data "aws_subnet_ids" "private" {
  vpc_id = data.aws_vpc.main.id
  filter {
    name   = "tag:Name"
    values = ["${local.name_prefix}-private*"]
  }
}

data "aws_lb" "business-layer" {
  name = "${local.vpc_prefix}-ecs-app-lb"
}
