output "lab_host_ip" {
  description = "Public IP of the lab host, for Ansible inventory generation."
  value       = aws_instance.lab_host.public_ip
}

output "lab_host_ssh_user" {
  description = "SSH user for the lab host (Ubuntu default)."
  value       = "ubuntu"
}

output "topology_name" {
  description = "Containerlab topology name, passed through for downstream automation."
  value       = var.topology_name
}

output "security_group_id" {
  description = "Security group ID guarding the lab host, for reference or extension."
  value       = aws_security_group.lab_host.id
}

output "vpc_id" {
  description = "VPC ID the lab host runs in."
  value       = aws_vpc.lab.id
}
