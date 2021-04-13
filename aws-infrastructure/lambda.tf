# Dummy content to provision temporary lambdas
data "archive_file" "dummy" {
    type = "zip"
    output_path = "lambda_function_payload.zip"

    source {
        content = "hello"
        filename = "dummy.txt"
    }
}

# Export lambda
resource "aws_lambda_function" "export" {
  function_name = "${local.name_prefix}-export"
  role = aws_iam_role.lambda.arn
  handler = "bin/main"
  runtime = "go1.x"
  filename = data.archive_file.dummy.output_path
  timeout     = 300
  memory_size = 1024

   tracing_config {
    mode = "Active"
  }

  vpc_config {
    subnet_ids = [data.aws_subnet.private-subnet.id, data.aws_subnet.private-subnet2.id]
    security_group_ids = [data.aws_security_group.vpc-private.id]
  }

  environment {
    variables = {
        GRAPHQL_ENDPOINT       : "http://${data.aws_lb.business-layer.dns_name}/contributor/dbExport",
        S3_BUCKET              : aws_s3_bucket.export.id
        DB_EXPORT_OUTPUT_QUEUE : "${local.name_prefix}-db-export-output"
        LOG_LEVEL              : "INFO"
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