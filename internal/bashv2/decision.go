package bashv2

// PermissionPhase is the outcome of pre-spawn policy + validation (L1–L4).
type PermissionPhase int

const (
	// PhaseDeny blocks execution without prompting the user.
	PhaseDeny PermissionPhase = iota
	// PhaseAsk requires human confirmation (dangerous_tool flow).
	PhaseAsk
	// PhaseAllow proceeds without extra confirmation for this call.
	PhaseAllow
)

// GateResult is returned by [Session.Gate] before any process is spawned.
type GateResult struct {
	Phase   PermissionPhase
	Reason  string // machine-oriented tag for audit (e.g. policy_deny, validator_block)
	Message string // human/model-oriented message when Phase == PhaseDeny
}
