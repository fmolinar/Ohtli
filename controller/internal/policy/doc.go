// Package policy will validate a proposed remediation against ownership
// and safety rules (README.md sections 7 and 14: device/path allowlists,
// rate limiting, dry-run/approval gating) before internal/reconcile is
// allowed to execute it. Not yet implemented -- lands with Phase 4
// (controlled writes), once internal/telemetry and internal/topology give
// it real state to evaluate.
package policy
