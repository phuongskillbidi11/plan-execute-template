package toolrouter

import "eng/internal/tooladapter"

// Filter returns the subset of adapters matching a required capability
// list and currently available. This is the entire Tool Router for
// Phase 5 — it exposes nothing to any agent session because no session
// object exists in this architecture to expose into yet (Requirement 16:
// foundation only).
func Filter(required []string, adapters []tooladapter.Adapter) []tooladapter.Adapter {
	want := map[string]bool{}
	for _, r := range required {
		want[r] = true
	}
	var out []tooladapter.Adapter
	for _, a := range adapters {
		if want[a.Capability()] && a.Available() {
			out = append(out, a)
		}
	}
	return out
}
