# Set Provider as AWS and region
provider "aws" {
  region = var.region
  version = "~> 3"
  assume_role {
      role_arn = var.role_arn
  }
}

terraform {
  backend "s3" {}
}

locals {
  name_prefix = "${var.common_name_prefix}-egestion-${var.environment_name}"
  vpc_prefix = "${var.common_name_prefix}-${var.environment_name}"
}

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


# Lambda role for egestion
resource "aws_iam_role" "iam_for_lambda" {
  name = "${local.name_prefix}-egestion-lambda-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": [
          "lambda.amazonaws.com",
          "apigateway.amazonaws.com"
        ]
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

# Attaching policy to role
resource "aws_iam_role_policy_attachment" "attach-ec2" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
}

resource "aws_iam_role_policy_attachment" "attach-basic_role" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_invoke" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaRole"
}

resource "aws_iam_role_policy_attachment" "s3" {
  role = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}


# Dummy content to provision temporary lambdas
data "archive_file" "dummy" {
    type = "zip"
    output_path = "lambda_function_payload.zip"

    source {
        content = "hello"
        filename = "dummy.txt"
    }
}


# Defining s3 bucket for export
resource "aws_s3_bucket" "export" {
  bucket = "${local.name_prefix}-export"
  acl    = "private"

  tags = merge(
      var.common_tags,
      {
      Name = "${local.name_prefix}-export",
      "ons:name" = "${local.name_prefix}-export"
      }
  )
}


# Export lambda
resource "aws_lambda_function" "export" {
  function_name = "${local.name_prefix}-export"
  role = aws_iam_role.iam_for_lambda.arn
  handler = "bin/main"
  runtime = "go1.x"
  filename = data.archive_file.dummy.output_path
  vpc_config {
    subnet_ids = [data.aws_subnet.private-subnet.id, data.aws_subnet.private-subnet2.id]
    security_group_ids = [data.aws_security_group.vpc-private.id]
  }
  environment {
    variables = {
        GRAPHQL_ENDPOINT: "http://${data.aws_lb.business-layer.dns_name}/contributor/dbExport",
        S3_BUCKET       : aws_s3_bucket.export.id
    }
  }

  tags = merge(
    var.common_tags,
    {
    Name = "${local.name_prefix}-export",
    "ons:name" = "${local.name_prefix}-export"
    }
  )
}
