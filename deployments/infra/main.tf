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

# Kinesis Stream for DynamoDB Records
resource "aws_kinesis_stream" "pg_router_dynamodb_stream" {
  name             = "pg-router-dynamodb-stream"
  retention_period = 24

  shard_level_metrics = [
    "IncomingBytes",
    "IncomingRecords",
    "OutgoingBytes",
    "OutgoingRecords",
    "IteratorAgeMilliseconds",
    "ReadProvisionedThroughputExceeded",
    "WriteProvisionedThroughputExceeded"
  ]

  stream_mode_details {
    stream_mode = "ON_DEMAND"
  }

  tags = {
    Type = "DynamoDB Stream"
    Environment = var.env
    Application = "pg-router"
  }
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

###
# Commented due to current issue
#
# The mapping has to be done from console manually for now.
# Follow the known issue
# https://discuss.hashicorp.com/t/aws-dynamodb-kinesis-streaming-destination-iam-issue/49549
###

# DynamoDB Stream Destination
#resource "aws_dynamodb_kinesis_streaming_destination" "pg_router_dynamodb_stream_destination" {
#  stream_arn = aws_kinesis_stream.pg_router_dynamodb_stream.arn
#  table_name = aws_dynamodb_table.pg_router_table.arn
#}

###
# Uncomment when fixed
###

