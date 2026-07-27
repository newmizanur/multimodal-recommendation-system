variable "project" {
  type    = string
  default = "multimodal-recsys"
}

variable "environment" {
  type    = string
  default = "dev"
}

# Only used by the aws provider block for the S3 bucket — the rest of
# this configuration doesn't care which cloud the underlying Kubernetes
# cluster runs on.
variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "allowed_upload_origins" {
  type    = list(string)
  default = ["*"]
}
