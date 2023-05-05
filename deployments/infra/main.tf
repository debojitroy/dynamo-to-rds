variable "aws_region" {
  type = string
  default = "ap-south-1"
}

variable "env" {
  type = string
  default = "dev"
}

variable "vpc_id" {
  type = string
}

variable "rds_sg_id" {
  type = string
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
  stream_view_type = "NEW_AND_OLD_IMAGES"

  tags = {
    Type = "DynamoDB Table"
    Environment = var.env
    Application = "pg-router"
  }
}

##
# Start - Preparing the Lambda Function
##

# Get the VPC details

data "aws_vpc" "launch_vpc" {
  id = var.vpc_id
}

data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }

  tags = {
    Tier = "Private"
  }
}

data "aws_subnet" "private_subnets" {
  for_each = toset(data.aws_subnets.private.ids)
  id       = each.value
}

resource "aws_security_group" "lambda_security_group" {
  vpc_id = var.vpc_id

  ingress {
    protocol  = -1
    self      = true
    from_port = 0
    to_port   = 0
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group_rule" "allow_lambda_to_rds_sg" {
  type              = "ingress"
  description       = "MySQL TLS from sg_lambda"
  from_port         = 3306
  to_port           = 3306
  protocol          = "tcp"
  source_security_group_id = aws_security_group.lambda_security_group.id
  security_group_id = var.rds_sg_id
}

# Base Policy for Lambda
data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

# Create Lambda Execution Role
resource "aws_iam_role" "dynamodb_consumer_lambda_role" {
  name               = "dynamodb_consumer_lambda_role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
}

# Create Cloudwatch Group for Lambda
resource "aws_cloudwatch_log_group" "pg_router_dynamodb_consumer_cw_log_group" {
  name              = "/aws/lambda/pg_router_dynamodb_consumer"
  retention_in_days = 3
}

## Create Additional Policy for logging
data "aws_iam_policy_document" "dynamodb_consumer_lambda_logging_statement" {
  statement {
    effect = "Allow"

    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["arn:aws:logs:*:*:*"]
  }
}

resource "aws_iam_policy" "dynamodb_consumer_lambda_logging_policy" {
  name        = "dynamodb_consumer_lambda_logging_policy"
  path        = "/"
  description = "IAM policy for logging from a PG Router DynamoDB consumer lambda"
  policy      = data.aws_iam_policy_document.dynamodb_consumer_lambda_logging_statement.json
}

# Attach to Lambda Policy - CW Logging
resource "aws_iam_role_policy_attachment" "dynamo_consumer_lambda_logs" {
  role       = aws_iam_role.dynamodb_consumer_lambda_role.name
  policy_arn = aws_iam_policy.dynamodb_consumer_lambda_logging_policy.arn
}

# Create additional policy to access Dynamodb Streams
data "aws_iam_policy_document" "dynamodb_consumer_lambda_access_dynamo_streams_statement" {
  statement {
    effect = "Allow"

    actions = [
      "dynamodb:GetRecords",
      "dynamodb:GetShardIterator",
      "dynamodb:DescribeStream",
      "dynamodb:ListStreams"
    ]

    resources = ["arn:aws:dynamodb:*:*:*"]
  }
}

resource "aws_iam_policy" "dynamodb_consumer_lambda_stream_policy" {
  name        = "dynamodb_consumer_lambda_stream_policy"
  path        = "/"
  description = "IAM policy for accessing dynamodb stream from PG Router DynamoDB consumer lambda"
  policy      = data.aws_iam_policy_document.dynamodb_consumer_lambda_access_dynamo_streams_statement.json
}

# Attach to Lambda Policy - Stream Access
resource "aws_iam_role_policy_attachment" "dynamo_consumer_lambda_stream" {
  role       = aws_iam_role.dynamodb_consumer_lambda_role.name
  policy_arn = aws_iam_policy.dynamodb_consumer_lambda_stream_policy.arn
}

# Attach VPC Access Role for Lambda
resource "aws_iam_role_policy_attachment" "iam_role_policy_attachment_lambda_vpc_access_execution" {
  role       = aws_iam_role.dynamodb_consumer_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

# Prepare the Archive
data "archive_file" "dynamodb_consumer_lambda_zip" {
  type        = "zip"
  source_file = "../${path.module}/dist/consumer/main"
  output_path = "../${path.module}/dist/consumer/dynamodb_consumer_lambda_payload.zip"
}

# Create the lambda function
resource "aws_lambda_function" "dynamodb_consumer_lambda" {
  filename      = "../${path.module}/dist/consumer/dynamodb_consumer_lambda_payload.zip"
  function_name = "pg_router_dynamodb_consumer"
  role          = aws_iam_role.dynamodb_consumer_lambda_role.arn
  handler       = "main"

  source_code_hash = data.archive_file.dynamodb_consumer_lambda_zip.output_base64sha256

  runtime = "go1.x"
  timeout = 120
  memory_size = 512

  environment {
    variables = {
      env = var.env
    }
  }

  vpc_config {
    security_group_ids =[aws_security_group.lambda_security_group.id]
    subnet_ids = [for s in data.aws_subnet.private_subnets : s.id]
  }

  depends_on = [
    aws_iam_role_policy_attachment.dynamo_consumer_lambda_logs,
    aws_iam_role_policy_attachment.dynamo_consumer_lambda_stream,
    aws_cloudwatch_log_group.pg_router_dynamodb_consumer_cw_log_group,
  ]
}

##
# End - Preparing the Lambda Function
##

# Create DynamoDB Stream Event Source mapping for Lambda
resource "aws_lambda_event_source_mapping" "pg_router_dynamodb_stream_event_mapping" {
  event_source_arn  = aws_dynamodb_table.pg_router_table.stream_arn
  function_name     = aws_lambda_function.dynamodb_consumer_lambda.arn
  starting_position = "LATEST"
}