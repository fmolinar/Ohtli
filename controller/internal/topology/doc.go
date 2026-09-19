// Package topology will hold the controller's graph of nodes, links,
// metrics, and health (README.md section 7's "Topology graph" box), built
// from internal/telemetry state. Not yet implemented: the read-only
// controller (Phase 3) ships the state cache first; graph construction
// and the constrained-SPF path calculation land with Phase 4's
// controlled-writes reconciliation loop, once there's a decision that
// needs the graph.
package topology
