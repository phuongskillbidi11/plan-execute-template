package toolcap

// Risk classifies how much trust invoking a capability requires — the
// safety model Phase 7 establishes before any DESTRUCTIVE/HIGH_RISK
// adapter (PLC write, Modbus write, ...) exists to use it (Requirement 4).
type Risk string

const (
	RiskRead        Risk = "READ"
	RiskWrite       Risk = "WRITE"
	RiskDestructive Risk = "DESTRUCTIVE"
	RiskHighRisk    Risk = "HIGH_RISK"
)

// RiskRank gives each tier a total order (READ < WRITE < DESTRUCTIVE <
// HIGH_RISK) so role ceilings and policy checks stay one integer
// comparison instead of a growing switch statement. An unknown Risk
// value ranks above every known tier — fail toward "more restrictive,"
// never less.
func RiskRank(r Risk) int {
	switch r {
	case RiskRead:
		return 0
	case RiskWrite:
		return 1
	case RiskDestructive:
		return 2
	case RiskHighRisk:
		return 3
	default:
		return 4
	}
}

// Capability is one named, risk-classified operation an Adapter exposes
// — e.g. {"git.status", RiskRead} or {"git.force_push", RiskDestructive}.
// Naming convention: "<adapter>.<operation>" (Phase 7 spec.md Decision 5).
type Capability struct {
	Name string
	Risk Risk
}
