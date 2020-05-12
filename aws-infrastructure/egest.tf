# Defining s3 bucket for export

resource "aws_s3_bucket" "export" {
  bucket = "${var.environment_name}-${var.user}-export-bucket"
  acl    = "private"

  tags = {
    Name = "${var.environment_name}-${var.user}-Export-Bucket"
    App  = "takeon"
  }
}

# Defining role for egest

# Separate Lambda role for egestion

resource "aws_iam_role" "iam_for_lambda" {
  name = "${var.environment_name}-${var.user}-egestion-lambda-role"

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

# defining empty lambda for egest

data "archive_file" "dummy" {
    type = "zip"
    output_path = "lambda_function_payload.zip"
    
    source {
        content = "hello"
        filename = "dummy.txt"
    }
}

# Importing existing subnets

data "aws_security_group" "private-securitygroup" {
  filter {
    name = "tag:Name"
    values = ["${var.environment_name}-private-securitygroup"]
  }
}

data "aws_subnet" "private-subnet" {
  filter {
    name = "tag:Name"
    values = ["${var.environment_name}-private-subnet"]
  }
}

data "aws_subnet" "private-subnet2" {
  filter {
    name = "tag:Name"
    values = ["${var.environment_name}-private-subnet2"]
  }
}

data "aws_lb" "business-layer-lb" {
    name = "${var.environment_name}-${var.user}-bl"
}



# dbexport lambda

resource "aws_lambda_function" "db-export-lambda" {
  function_name = "takeon-db-export-lambda-${var.user}-main"
  role = aws_iam_role.iam_for_lambda.arn
  handler = "bin/main"
  runtime = "go1.x"
  filename = data.archive_file.dummy.output_path
  vpc_config {
    subnet_ids = [data.aws_subnet.private-subnet.id, data.aws_subnet.private-subnet2.id]
    security_group_ids = [data.aws_security_group.private-securitygroup.id]
  } 
  environment {
    variables = {
        GRAPHQL_ENDPOINT: "http://${data.aws_lb.business-layer-lb.dns_name}:8088/contributor/dbExport",
        S3_BUCKET       : aws_s3_bucket.export.id
    }
  }
}