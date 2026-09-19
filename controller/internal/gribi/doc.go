// Package gribi will wrap gRIBI (github.com/openconfig/gribigo) for
// direct forwarding-plane route programming, used instead of a gNMI Set
// when the goal is installing a route/forwarding entry rather than
// changing configuration (README.md section 7). Not yet implemented --
// this is Phase 5 (forwarding control and scale) work, gated on Phase 4's
// controlled-writes safety machinery existing first.
package gribi
