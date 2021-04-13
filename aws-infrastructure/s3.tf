# Defining s3 bucket for export
resource "aws_s3_bucket" "export" {
  bucket = "${local.name_prefix}-export"
  acl    = "private"
  
  versioning {
    enabled = true
  }

  force_destroy = true
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
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

resource "aws_s3_bucket_public_access_block" "export" {
  bucket = aws_s3_bucket.export.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
  




