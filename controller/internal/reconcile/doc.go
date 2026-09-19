// Package reconcile will drive the state machine in README.md section 7
// (Healthy -> Suspect -> ConfirmedFailure -> RemediationPending ->
// Remediating -> Verifying -> Recovered/Rollback), calling internal/policy
// to validate a change and internal/gnmi (or internal/gribi) to apply and
// verify it. Not yet implemented -- lands with Phase 4 (controlled
// writes); Phase 3 only observes and measures native convergence.
package reconcile
