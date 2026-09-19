# `lab` environment

Provisions the single Ubuntu lab VM from README.md section 5 on AWS:
VPC + management subnet + internet gateway, a security group that only
allows SSH from `ssh_allowed_cidrs`, and a `c5.2xlarge` (8 vCPU / 16 GiB,
Nitro-based for nested virtualization) instance on Ubuntu 22.04, ready for
`ansible/roles/common` + `ansible/roles/lab_host_bootstrap` to configure.

AWS was chosen as the default target cloud since none was specified in
README.md; swap the provider block in `versions.tf` and the resources in
`terraform/modules/lab-host/main.tf` if you'd rather run this on GCP,
Azure, or a local libvirt/KVM host.

## ⚠️ This costs real money once applied

`terraform apply` here creates a billed AWS EC2 instance (a `c5.2xlarge`
is roughly $0.34/hr on-demand as of writing) plus a small amount of EBS
and networking cost. **Nothing in this PR has been applied** — no AWS
credentials were available in the environment this was authored in.
`terraform validate` and `terraform fmt -check` are clean; review the
plan yourself before your first `apply`.

## What you need to configure

1. **AWS credentials** — via `aws configure`, environment variables, or
   an assumed role. This module doesn't manage credentials itself.
2. **An SSH key pair for the lab** (don't reuse a personal key):
   ```bash
   ssh-keygen -t ed25519 -f ohtli-lab -C ohtli-lab
   ```
3. **`terraform.tfvars`** — copy `terraform.tfvars.example`, fill in
   `ssh_public_key` (contents of `ohtli-lab.pub`) and `ssh_allowed_cidrs`
   (your own IP(s), never `0.0.0.0/0`). This file is gitignored.
4. **A remote backend before running this from CI** — the default is a
   local backend (`terraform.tfstate` on disk, gitignored), fine for a
   single operator. For GitHub Actions (README.md section 10's
   `terraform-plan`/`terraform-apply` jobs), configure an S3 bucket +
   DynamoDB lock table (or Terraform Cloud) and add a `backend "s3" {}`
   block to `versions.tf`, then pass credentials via GitHub secrets.

## Usage

```bash
terraform init
terraform plan   # review before ever applying
terraform apply
terraform output # feed lab_host_ip / lab_host_ssh_user into ansible/inventories/lab/hosts.yml

# when done:
terraform destroy
```

## Outputs

| Output | Used by |
|---|---|
| `lab_host_ip` | `ansible/inventories/lab/hosts.yml` (`lab_host.ansible_host`) |
| `lab_host_ssh_user` | `ansible/inventories/lab/hosts.yml` (`lab_host.ansible_user`) |
| `topology_name` | Containerlab / CI inventory generation |
