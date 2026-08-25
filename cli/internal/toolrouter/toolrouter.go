package toolrouter

import (
	"sort"

	"eng/internal/tooladapter"
	"eng/internal/toolcap"
	"eng/internal/toolpolicy"
)

// Filter returns the subset of adapters whose Name() is in required and
// currently available. This is the Phase 5 filtering foundation, kept
// unchanged in shape — Phase 7's Adapter interface replaced the single
// Capability() string with plural, risk-tagged Capabilities(), so this
// now matches by adapter identity instead (Route, added by Phase 7, is
// the capability-aware, policy-aware successor).
func Filter(required []string, adapters []tooladapter.Adapter) []tooladapter.Adapter {
	want := map[string]bool{}
	for _, r := range required {
		want[r] = true
	}
	var out []tooladapter.Adapter
	for _, a := range adapters {
		if want[a.Name()] && a.Available() {
			out = append(out, a)
		}
	}
	return out
}

type Selection struct {
	Adapter    string
	Capability string
	Reason     string
}

type Blocked struct {
	Adapter    string
	Capability string
	Reason     string
}

type Result struct {
	Allowed       []Selection
	NeedsApproval []Blocked
	Blocked       []Blocked
}

// Route is the Phase 7 authoritative tool-selection path — deterministic
// provider precedence (alphabetical among available candidates; see
// Phase 7 spec.md Decision 5 for why no config field overrides this
// yet), then one toolpolicy.Decide call per required capability,
// bucketed into Allowed/NeedsApproval/Blocked with an explanation each.
func Route(required []string, adapters []tooladapter.Adapter, role string, policy toolpolicy.Policy, approved bool) Result {
	var result Result
	for _, capName := range required {
		var candidates []tooladapter.Adapter
		for _, a := range adapters {
			if !a.Available() {
				continue
			}
			for _, c := range a.Capabilities() {
				if c.Name == capName {
					candidates = append(candidates, a)
					break
				}
			}
		}
		if len(candidates) == 0 {
			result.Blocked = append(result.Blocked, Blocked{Capability: capName, Reason: "no available adapter provides this capability"})
			continue
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Name() < candidates[j].Name() })
		owner := candidates[0]

		var risk toolcap.Risk
		for _, c := range owner.Capabilities() {
			if c.Name == capName {
				risk = c.Risk
			}
		}

		decision := toolpolicy.Decide(capName, risk, owner.Name(), role, policy, approved)
		switch decision.Verdict {
		case toolpolicy.Allowed:
			result.Allowed = append(result.Allowed, Selection{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
		case toolpolicy.NeedsApproval:
			result.NeedsApproval = append(result.NeedsApproval, Blocked{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
		default:
			result.Blocked = append(result.Blocked, Blocked{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
		}
	}
	return result
}
