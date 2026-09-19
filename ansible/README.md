# Ansible baseline configuration

Implements README.md section 6 ("Ansible configuration model") for two
target groups:

- **`lab_host`** — the Terraform-provisioned Ubuntu VM (`terraform/environments/lab`).
  Real SSH, `become: true`. Roles: `common` (hostname/timezone, automation
  user, SSH hardening, NTP, DNS) and `lab_host_bootstrap` (pinned Docker +
  Containerlab install).
- **`routers`** — the Containerlab FRR nodes (`pe1`/`p1`/`p2`/`pe2`).
  `ansible_connection: local`; the `frr_baseline` role templates
  `daemons`/`frr.conf` straight into `topology/configs/<node>/`, which
  Containerlab bind-mounts into each container (see `topology/isp.clab.yml`).
  These containers don't expose SSH/NETCONF/gNMI management yet, so this is
  file-based push, not live device automation — that upgrade path is gNMI
  `Set` from the Go controller (README section 7) or a gNMI-capable NOS
  (README section 4), not Ansible.

## What you need to configure

1. **`ansible/inventories/lab/hosts.yml`** — replace the placeholder
   `ansible_host` / `ansible_user` under `lab_host` with the real values
   from `terraform output` once `terraform/environments/lab` is applied.
2. **An SSH key for the automation user** — set
   `common_automation_user_ssh_public_key` (e.g. in
   `inventories/lab/host_vars/lab-host.yml`, or `-e` on the CLI) to a real
   public key before running the `common` role. It's empty by default so a
   run without it simply skips key installation rather than locking you out.
3. **Collections** — `ansible-galaxy collection install -r requirements.yml`
   (needs `community.general` and `ansible.posix`).

## Usage

```bash
cd ansible
ansible-galaxy collection install -r requirements.yml

# Lab host baseline (needs real SSH access — not run in CI by default)
ansible-playbook playbooks/site.yml --limit lab_host

# FRR router config (local, safe to run anywhere with this repo checked out)
ansible-playbook playbooks/site.yml --limit routers
```

Run twice and confirm the second run reports no changes (README.md's
Phase 1 acceptance test) — the `frr_baseline` templates are pure functions
of `group_vars`/`host_vars`, so this should be true by construction.

## Notes / limitations

- Not run against a live Terraform-provisioned host in this environment
  (no cloud credentials / real VM available here). `ansible-lint
  --profile production` and `ansible-playbook --syntax-check` are clean;
  validate `--limit lab_host` against a real host before trusting it.
- The `routers` play was diffed against the static configs shipped in
  `topology/configs/` (from `feat/containerlab-topology`) and is
  functionally equivalent (same OSPF/BGP topology and addressing); expect
  a cosmetic diff (ACL naming, OSPF `network` statement notation) the
  first time you run it, since Ansible is now the source of truth for
  those files.
- gNMI service enablement (README section 6's checklist) is intentionally
  not implemented here — stock FRR doesn't expose an OpenConfig gNMI
  target. Revisit when an SR Linux node or a gNMI/OpenConfig translator is
  introduced (README section 4's phased NOS recommendation).
