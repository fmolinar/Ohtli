locals {
  tags = merge(
    {
      Project   = "ohtli"
      ManagedBy = "terraform"
    },
    var.tags,
  )
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_vpc" "lab" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge(local.tags, { Name = "${var.name}-vpc" })
}

resource "aws_internet_gateway" "lab" {
  vpc_id = aws_vpc.lab.id

  tags = merge(local.tags, { Name = "${var.name}-igw" })
}

resource "aws_subnet" "management" {
  vpc_id                  = aws_vpc.lab.id
  cidr_block              = var.management_subnet_cidr
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = true

  tags = merge(local.tags, { Name = "${var.name}-mgmt" })
}

resource "aws_route_table" "management" {
  vpc_id = aws_vpc.lab.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.lab.id
  }

  tags = merge(local.tags, { Name = "${var.name}-mgmt-rt" })
}

resource "aws_route_table_association" "management" {
  subnet_id      = aws_subnet.management.id
  route_table_id = aws_route_table.management.id
}

resource "aws_security_group" "lab_host" {
  name        = "${var.name}-host-sg"
  description = "Controlled ingress to the Ohtli lab host management interface"
  vpc_id      = aws_vpc.lab.id

  tags = merge(local.tags, { Name = "${var.name}-host-sg" })
}

resource "aws_vpc_security_group_ingress_rule" "ssh" {
  for_each = toset(var.ssh_allowed_cidrs)

  security_group_id = aws_security_group.lab_host.id
  description       = "SSH from an allowed management range"
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  cidr_ipv4         = each.value
}

resource "aws_vpc_security_group_egress_rule" "all" {
  security_group_id = aws_security_group.lab_host.id
  description       = "Unrestricted egress (container image pulls, apt, Containerlab, etc.)"
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_key_pair" "lab_host" {
  key_name   = "${var.name}-key"
  public_key = var.ssh_public_key

  tags = local.tags
}

resource "aws_instance" "lab_host" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  subnet_id              = aws_subnet.management.id
  vpc_security_group_ids = [aws_security_group.lab_host.id]
  key_name               = aws_key_pair.lab_host.key_name

  # Nested virtualization for VM-based NOS images is enabled by default on
  # Nitro-based instance families (README.md section 5); no extra CPU
  # options are required here.

  root_block_device {
    volume_size = var.root_volume_size_gb
    volume_type = "gp3"
    encrypted   = true
  }

  metadata_options {
    http_tokens = "required" # IMDSv2 only
  }

  tags = merge(local.tags, { Name = "${var.name}-host" })
}
