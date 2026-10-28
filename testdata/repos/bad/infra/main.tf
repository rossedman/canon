terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "name" {}

resource "aws_s3_bucket" "this" {
  bucket = var.name
}
