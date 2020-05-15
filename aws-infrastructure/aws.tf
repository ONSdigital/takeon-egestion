# Setting the provider
provider "aws" {
  region = "eu-west-2"
  version = "~> 2.0"
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_access_key}"
}