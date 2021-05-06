variable "environment_name" {}
variable "region" {}
variable "role_arn" {}
variable "account_environment_name" {}
variable "common_name_prefix" { default = "spp-es-takeon" }

# Define the standard AWS tags
locals {
  name_prefix = "${var.common_name_prefix}-${var.environment_name}"
  vpc_prefix  = "${var.common_name_prefix}-${var.environment_name}"
  common_tags = {
    "ons:owner:team"              = "spp-dataclearing"
    "ons:owner:business-unit"     = "DST"
    "ons:owner:contact"           = "David Morgan"
    "ons:application:name"        = "Takeon Core"
    "ons:application:environment" = var.account_environment_name
    "ons:application:eol"         = "N/A"
    "ManagedBy"                   = "Terraform"
  }
}


