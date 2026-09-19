variable "aws_region" {
  description = "AWS region to provision the lab in."
  type        = string
  default     = "us-east-1"
}

variable "name" {
  description = "Name prefix applied to every resource."
  type        = string
  default     = "ohtli-lab"
}

variable "instance_type" {
  description = "EC2 instance type for the lab host."
  type        = string
  default     = "c5.2xlarge"
}

variable "ssh_public_key" {
  description = "Public key material for SSH access to the lab host."
  type        = string
}

variable "ssh_allowed_cidrs" {
  description = "CIDR blocks allowed to reach the lab host over SSH."
  type        = list(string)
}
