output "lab_host_ip" {
  description = "Public IP of the lab host. Feed into ansible/inventories/lab/hosts.yml."
  value       = module.lab_host.lab_host_ip
}

output "lab_host_ssh_user" {
  value = module.lab_host.lab_host_ssh_user
}

output "topology_name" {
  value = module.lab_host.topology_name
}
