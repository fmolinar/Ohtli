terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Local backend by default so this environment works out of the box.
  # Switch to a remote backend (S3 + DynamoDB lock table, or Terraform
  # Cloud) before running this from CI -- see terraform/environments/lab/README.md.
}

provider "aws" {
  region = var.aws_region
}
