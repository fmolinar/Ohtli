# Observability

Implements README.md section 8's MVP stack: Prometheus + Loki + Promtail
+ an OpenTelemetry Collector + Grafana, wired together with
`docker-compose.yml`.

- **Prometheus** scrapes itself, the OTel Collector's self-telemetry and
  forwarded-metrics ports, and the controller's `/metrics`
  (`../controller`) via `host.docker.internal` — the controller runs as a
  native process on the lab host (README.md section 2's architecture),
  not inside this compose stack.
- **Promtail** discovers every container via the Docker socket and ships
  their stdout/stderr to **Loki**.
- The **OTel Collector** accepts OTLP (grpc :4317, http :4318) and
  exports metrics to Prometheus (via its Prometheus exporter on :8889)
  and logs to Loki. Nothing emits OTLP yet — the controller currently
  logs JSON directly and exposes Prometheus metrics directly (see
  `controller/README.md`) — so today this is scaffolding for future
  instrumentation, not an active pipeline. Traces have no backend
  deployed (README.md doesn't call for one), so the traces pipeline is
  debug-only.
- **Grafana** is provisioned with both datasources and one starter
  dashboard (`grafana/dashboards/controller-overview.json`): gNMI
  session up/down, gNMI request p95 latency, and subscription update
  rate, all reading the controller's metrics.

## What you need to configure

1. **Grafana admin password** — this compose file reads it from a Docker
   secret, not an env var, so it never ends up in `docker inspect`
   output or shell history:
   ```bash
   cd observability
   openssl rand -base64 24 > grafana_admin_password.txt   # gitignored
   ```
2. **Point Prometheus at your controller** — `prometheus/prometheus.yml`
   assumes the controller listens on `:9400` on the same host running
   Docker (`host.docker.internal`, resolved via the `extra_hosts:
   host-gateway` entry in `docker-compose.yml` — Linux Docker Engine
   20.10+). Adjust the target if you run the controller elsewhere.
3. **Retention** — `loki/loki-config.yml` defaults to 7 days; extend
   `limits_config.retention_period` for anything longer-lived than a lab.

## Usage

```bash
cd observability
docker compose up -d
# Grafana:    http://localhost:3000  (admin / contents of grafana_admin_password.txt)
# Prometheus: http://localhost:9090
# Loki API:   http://localhost:3100
docker compose down
```

## Notes / limitations

- **Not started against a live Docker daemon in this environment** (no
  privileged Docker access here). Validated instead: `docker compose
  config` (clean, secret/volume/network wiring resolves correctly) and
  `yamllint` on every config file (clean). Run `docker compose up -d`
  for real and confirm all five containers report healthy before
  trusting this.
- Image tags are pinned (Prometheus v2.55.1, Loki/Promtail 3.2.1, Grafana
  11.3.0, otelcol-contrib 0.114.0) and were checked against the Docker
  Hub registry API to confirm they exist before being committed here.
- Grafana anonymous access is disabled and the admin password is a
  Docker secret, per README.md section 14 ("never expose ... Grafana
  directly to an untrusted network" -- this still assumes you don't
  publish port 3000 beyond your own management network).
