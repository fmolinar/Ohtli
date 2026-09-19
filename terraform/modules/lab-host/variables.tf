variable "name" {
  description = "Name prefix applied to every resource this module creates."
  type        = string
  default     = "ohtli-lab"
}

variable "vpc_cidr" {
  description = "CIDR block for the lab VPC."
  type        = string
  default     = "172.20.0.0/16"
}

variable "management_subnet_cidr" {
  description = <<-EOT
    CIDR block for the management subnet the lab host's primary NIC sits
    in (SSH, gNMI, monitoring). Distinct from the 172.20.20.0/24
    Containerlab mgmt network the host runs internally -- see
    README.md section 2.
  EOT
  type        = string
  default     = "172.20.10.0/24"
}

variable "availability_zone" {
  description = "Availability zone for the management subnet and instance."
  type        = string
  default     = "us-east-1a"
}

variable "instance_type" {
  description = <<-EOT
    EC2 instance type for the lab host. Defaults to 8 vCPU / 16 GiB, the
    starting capacity README.md section 5 suggests for the six-to-eight
    node container lab. Must be a Nitro-based type for nested
    virtualization if you later add VM-based NOS images.
  EOT
  type        = string
  default     = "c5.2xlarge"
}

variable "root_volume_size_gb" {
  description = "Root EBS volume size in GiB (README.md suggests 80 GB to start)."
  type        = number
  default     = 80
}

variable "ssh_public_key" {
  description = <<-EOT
    Public key material for SSH access to the lab host. Generate a
    dedicated key pair for this lab; never reuse a personal key, and
    never commit the private half to this repository.
  EOT
  type        = string
}

variable "ssh_allowed_cidrs" {
  description = <<-EOT
    CIDR blocks allowed to reach the lab host over SSH (22/tcp). Keep
    this scoped to your own IP(s) or a bastion/VPN range -- README.md
    section 14 requires management services stay off untrusted networks.
  EOT
  type        = list(string)
}

variable "topology_name" {
  description = "Containerlab topology name, exposed as an output for Ansible inventory generation."
  type        = string
  default     = "isp-lab"
}

variable "tags" {
  description = "Additional tags applied to every resource."
  type        = map(string)
  default     = {}
}
