# Set Provider as AWS and region
provider "aws" {
    region = var.region
    version = "~> 2"
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

resource "aws_iam_role_policy_attachment" "sqs" {
  role = aws_iam_role.iam_for_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSQSFullAccess"
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
        DB_EXPORT_OUTPUT_QUEUE: "${local.name_prefix}-db-export-output"
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


# input
resource "aws_sqs_queue" "db-export-input" {
  name = "${local.name_prefix}-db-export-input"
  redrive_policy = "{\"deadLetterTargetArn\":\"${aws_sqs_queue.dlq.arn}\",\"maxReceiveCount\":3}"

  tags = merge(
    var.common_tags,
    {
    Name = "${local.name_prefix}-db-export-input",
    "ons:name" = "${local.name_prefix}-db-export-input"
    }
  )
}

# output
resource "aws_sqs_queue" "db-export-output" {
  name = "${local.name_prefix}-db-export-output"
  redrive_policy = "{\"deadLetterTargetArn\":\"${aws_sqs_queue.dlq.arn}\",\"maxReceiveCount\":3}"

  tags = merge(
    var.common_tags,
    {
    Name = "${local.name_prefix}-db-export-output",
    "ons:name" = "${local.name_prefix}-db-export-output"
    }
  )
}


resource "aws_lambda_event_source_mapping" "db-export-trigger" {
  event_source_arn = aws_sqs_queue.db-export-input.arn
  function_name = aws_lambda_function.export.arn
  batch_size = 1
}


# Dead letter queue
resource "aws_sqs_queue" "dlq" {
  name = "${local.name_prefix}-export-dlq"

    tags = merge(
        var.common_tags,
        {
        Name = "${local.name_prefix}-export-dlq",
        "ons:name" = "${local.name_prefix}-export-dlq"
        }
    )
}