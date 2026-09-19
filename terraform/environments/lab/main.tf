module "lab_host" {
  source = "../../modules/lab-host"

  name              = var.name
  instance_type     = var.instance_type
  ssh_public_key    = var.ssh_public_key
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
}
