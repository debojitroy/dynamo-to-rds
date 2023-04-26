variable "aws_region" {
  type = string
  default = "ap-south-1"
}

terraform {
  backend "s3" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  required_version = ">= 1.2.0"
}

provider "aws" {
  region  = var.aws_region
}

resource "aws_s3_bucket" "sample-bucket" {
  force_destroy = true

  tags = {
    Environment = "Dev"
  }
}
