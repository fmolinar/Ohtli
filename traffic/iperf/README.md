# iperf3 traffic generation

Implements README.md section 9's MVP: an iperf3 server on `client-b`, a
client on `client-a`, run against the topology in `topology/isp.clab.yml`
once it's deployed.

## Scripts

- **`run-server.sh [container]`** — one-off iperf3 server (default
  `clab-isp-lab-client-b`, matching Containerlab's `clab-<lab-name>-<node>`
  naming for `topology/isp.clab.yml`).
- **`run-client.sh`** — iperf3 client with `-c`/`-s`/`-p`/`-t`/`-b`/`-o`
  flags (container, server IP, `udp`/`tcp`, duration, bitrate, output
  dir). Saves `--json` output under `results/` (gitignored) and, for UDP,
  prints the observed loss percentage and emits a GitHub Actions
  `::warning::` if it exceeds 1%.
- **`run-baseline.sh`** — orchestrates both: starts the server in the
  background, runs the client, waits for completion. This is what CI
  calls (README.md section 10's `traffic-baseline` job).

## Usage

```bash
# after `containerlab deploy --topo topology/isp.clab.yml`:
./traffic/iperf/run-baseline.sh                       # 60s UDP @ 100 Mbit/s, matches README section 9
./traffic/iperf/run-baseline.sh -p tcp -t 30           # 30s TCP throughput
```

Or drive the two ends manually, exactly as shown in README.md section 9:

```bash
./traffic/iperf/run-server.sh clab-isp-lab-client-b
./traffic/iperf/run-client.sh -c clab-isp-lab-client-a -s 198.51.100.2 -p udp -b 100M -t 60
```

## Notes / limitations

- Requires a running Containerlab deployment (`docker exec` targets the
  client containers by name) -- not runnable standalone in this
  environment; syntax-checked with `bash -n` and `shellcheck` (clean)
  instead of an end-to-end run. Exercise it for real once
  `feat/containerlab-topology` is deployed, ideally via the
  `traffic-baseline` CI job.
- This is the iperf3-only MVP stage of README.md section 9's traffic
  progression. TRex (multi-flow/scale) and a custom Go measurement agent
  are later stages -- add them under `traffic/trex/` and
  `traffic/custom-agent/` only when iperf3 stops being able to express
  the test you need.
