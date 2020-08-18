variable "environment_name" {}
variable "region" {}
variable "role_arn" {}
variable "account_environment_name" {}

# Define the standard AWS tags
variable "common_tags" {
  default = {
    "ons:owner:team"                = "spp-es-takeon"
    "ons:owner:business-unit"       = "DST"
    "ons:owner:contact"             = "David Morgan"
    "ons:application:name"          = "Takeon Core"
    "ons:application:environment"   = "Development"
    "ons:application:eol"           = "N/A"
    "ManagedBy"                     = "Terraform"
  }
}

variable "common_name_prefix" {
    default = "spp-es-takeon"
}
