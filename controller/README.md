# Controller

The read-only gNMI controller from README.md section 7 / Phase 3: for
every target in a device registry it runs `Capabilities`, `Get`, and a
streaming `Subscribe`, normalizes updates into an in-memory state cache,
and emits structured audit events (section 8's field list) and Prometheus
metrics. It does not write to devices -- Phase 4 adds gNMI `Set` /
approval-gated remediation on top of this.

```
cmd/controller/        entrypoint: wiring, HTTP server, reconnect loop
internal/devices/      device registry (YAML: address, TLS, subscribe paths)
internal/gnmi/         gNMI client: path parsing, Capabilities/Get/Subscribe
internal/telemetry/    device-independent state cache fed by Subscribe
internal/audit/        structured JSON audit event emitter
internal/metrics/      Prometheus metric definitions
internal/topology/     graph of nodes/links/health -- not yet implemented (Phase 4)
internal/policy/       change validation against ownership/safety rules -- not yet implemented (Phase 4)
internal/reconcile/    Healthy->...->Recovered state machine -- not yet implemented (Phase 4)
internal/gribi/        gRIBI forwarding-plane programming -- not yet implemented (Phase 5)
```

`ygnmi`/`ygot` (README's suggested libraries) were deliberately not used:
they need YANG-generated Go structs, which is more toolchain than a
read-only MVP against whatever paths a target's `Capabilities` actually
advertises needs. `internal/gnmi` talks the raw gNMI proto
(`github.com/openconfig/gnmi`) directly, with a small hand-rolled path
parser (`internal/gnmi/path.go`). Revisit if/when the controller needs
typed access to a specific YANG model.

## What you need to configure

- **`devices.yaml`** (gitignored) — copy `devices.example.yaml` and point
  it at real gNMI targets. **Nothing in `topology/isp.clab.yml` exposes
  gNMI today** — stock FRR has no OpenConfig gNMI target. This becomes
  real once an SR Linux node or a gNMI/OpenConfig translator is
  introduced (README.md section 4's phased NOS recommendation). Until
  then, this controller builds, unit-tests (via an in-memory fake gNMI
  server), and runs, but has nothing real to subscribe to in this repo's
  current topology.
- **mTLS material** — set `tls.ca_file` (and `cert_file`/`key_file` for
  mTLS) per target once a target issues certificates. `tls.insecure` /
  `insecure_skip_verify` exist for lab bring-up only (README.md section
  14: mTLS is required, least-privilege credentials, no untrusted
  exposure) -- never point them at anything reachable outside the lab.

## Usage

```bash
go build ./...
go test ./...

go run ./cmd/controller -registry devices.yaml -listen :9400
# GET :9400/healthz  -> liveness
# GET :9400/cache    -> JSON dump of the current state cache
# GET :9400/metrics  -> Prometheus exposition
```

## Test plan run in this environment

- `go build ./...`, `go vet ./...`, `gofmt -l .` (clean)
- `go test ./... -race` (clean) — `internal/gnmi` includes an end-to-end
  test against an in-memory fake gNMI server (`google.golang.org/grpc/test/bufconn`)
  exercising `Capabilities`, `Get`, and `Subscribe` through real gRPC
  plumbing, since there's no live gNMI target available in this
  environment to test against instead.
- **Not run**: against a real device. Point `devices.yaml` at one once
  gNMI is available (see above) and re-verify `controller-smoke`-style
  before trusting it operationally.
