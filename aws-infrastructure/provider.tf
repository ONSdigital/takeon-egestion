# Set Provider as AWS and region
provider "aws" {
  region = var.region
  assume_role {
      role_arn = var.role_arn
  }
}

#setting to keep terraform state in s3
terraform {
  backend "s3" {}
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3"
    }    
  }
}