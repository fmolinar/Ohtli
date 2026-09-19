# ISP lab topology

Six-node Containerlab topology matching README.md sections 3 and 5: two PEs,
two P (core) routers, and two traffic-endpoint clients connected directly to
the PEs.

```
client-a -- pe1 -- p1 -- pe2 -- client-b
              \-------- p2 ------/
```

- `pe1`, `p1`, `p2`, `pe2` run [FRRouting](https://frrouting.org/) (`frrouting/frr:9.1.0`),
  configured via bind-mounted `daemons` and `frr.conf` files under `configs/<node>/`.
- `client-a`, `client-b` run `networkstatic/iperf3`, with IP addressing and a
  default route applied post-deploy via the `exec:` block in `isp.clab.yml`
  (the image has no network configuration tooling of its own).

## Address plan

Matches README.md section 3 exactly:

| Node | Loopback | eth1 | eth2 | eth3 |
|---|---|---|---|---|
| pe1 | 10.255.0.1/32 | 10.0.0.0/31 (→p1) | 10.0.0.2/31 (→p2) | 192.0.2.1/24 (→client-a) |
| p1  | 10.255.0.2/32 | 10.0.0.1/31 (→pe1) | 10.0.0.4/31 (→pe2) | — |
| p2  | 10.255.0.3/32 | 10.0.0.3/31 (→pe1) | 10.0.0.6/31 (→pe2) | — |
| pe2 | 10.255.0.4/32 | 10.0.0.5/31 (→p1) | 10.0.0.7/31 (→p2) | 198.51.100.1/24 (→client-b) |
| client-a | — | 192.0.2.2/24 | — | — |
| client-b | — | 198.51.100.2/24 | — | — |

Management addressing uses the `172.20.20.0/24` Containerlab mgmt network
declared in `isp.clab.yml`, isolated from the data-plane links above.

## Routing design

- **Underlay:** OSPF area 0 on loopbacks and the four provider p2p links only.
  Client-facing interfaces (`pe1:eth3`, `pe2:eth3`) are intentionally kept out
  of OSPF.
- **iBGP:** single AS `65000`, full mesh between `pe1`/`p1`/`p2`/`pe2` using
  loopbacks (`update-source lo`) and `next-hop-self`, matching README.md's
  "establish iBGP between provider nodes, using loopbacks."
- **Client reachability:** `pe1` and `pe2` each redistribute their directly
  connected client subnet (192.0.2.0/24 and 198.51.100.0/24 respectively)
  into BGP via a prefix-list + route-map, rather than a blanket
  `redistribute connected`, so no other connected interface leaks into BGP.
- No CE routers or eBGP yet — clients attach directly to the PEs per the
  README's reduced six-container first iteration. Add CE1/CE2 and eBGP when
  testing customer routing policy, VRFs, MPLS, or SRv6 (README section 3).
- No gNMI/OpenConfig target is configured on these FRR nodes yet. Per the
  README's recommended approach (section 4), FRR is the routing/CI
  foundation for phases 1-2; swap in or add SR Linux nodes when the gNMI
  control loop (phase 3+) becomes the priority.

## Usage

```bash
containerlab deploy --topo topology/isp.clab.yml
containerlab inspect --topo topology/isp.clab.yml
```

Verify OSPF/BGP convergence:

```bash
docker exec clab-isp-lab-pe1 vtysh -c "show ip ospf neighbor"
docker exec clab-isp-lab-pe1 vtysh -c "show ip bgp summary"
docker exec clab-isp-lab-pe1 vtysh -c "show ip route bgp"
```

Verify end-to-end reachability and run a traffic test (see `traffic/iperf/`):

```bash
docker exec clab-isp-lab-client-a ping -c 4 198.51.100.2
```

Tear down:

```bash
containerlab destroy --topo topology/isp.clab.yml --cleanup
```

## Notes / limitations

- Requires Docker and Containerlab with root/sudo privileges (network
  namespaces, veth creation). This was authored and YAML/config-syntax
  reviewed in a sandboxed environment without privileged Docker access, so a
  live `containerlab deploy` has **not** been run against it yet — validate
  in CI or on your lab host before relying on it (tracked as a CI job in
  `feat/ci-cd-pipeline`).
- FRR version is pinned (`9.1.0`); bump deliberately and re-verify `vtysh`
  config compatibility.
