
# input
resource "aws_sqs_queue" "db-export-input" {
  name                       = "${local.name_prefix}-db-export-input"
  redrive_policy             = "{\"deadLetterTargetArn\":\"${aws_sqs_queue.dlq.arn}\",\"maxReceiveCount\":3}"  
  visibility_timeout_seconds = 300
  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

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
  name           = "${local.name_prefix}-db-export-output"
  redrive_policy = "{\"deadLetterTargetArn\":\"${aws_sqs_queue.dlq.arn}\",\"maxReceiveCount\":3}"

  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

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

  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

    tags = merge(
        var.common_tags,
        {
        Name = "${local.name_prefix}-export-dlq",
        "ons:name" = "${local.name_prefix}-export-dlq"
        }
    )
}