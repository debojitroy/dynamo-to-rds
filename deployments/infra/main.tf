variable "aws_region" {
  type = string
  default = "ap-south-1"
}

variable "env" {
  type = string
  default = "dev"
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

# DynamoDB Table
resource "aws_dynamodb_table" "pg_router_table" {
  name           = "pg_router_table"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "p_key"
  range_key      = "s_key"

  attribute {
    name = "p_key"
    type = "S"
  }

  attribute {
    name = "s_key"
    type = "S"
  }

  ttl {
    attribute_name = "record_ttl"
    enabled        = true
  }

  stream_enabled = true
  stream_view_type = "NEW_IMAGE"

  tags = {
    Type = "DynamoDB Table"
    Environment = var.env
    Application = "pg-router"
  }
}



