# Importing existing resources for lambdas to use
data "aws_security_group" "vpc-private" {
  filter {
    name = "tag:Name"
    values = ["${local.vpc_prefix}-vpc-private"]
  }
}

data "aws_subnet" "private-subnet" {
  filter {
    name = "tag:Name"
    values = ["${local.vpc_prefix}-vpc-private-subnet"]
  }
}

data "aws_subnet" "private-subnet2" {
  filter {
    name = "tag:Name"
    values = ["${local.vpc_prefix}-vpc-private-subnet2"]
  }
}

data "aws_lb" "business-layer" {
    name = "${local.vpc_prefix}-bl"
}

data "aws_caller_identity" "current" {}