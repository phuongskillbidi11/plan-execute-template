package toolrouter

import (
	"testing"

	"eng/internal/tooladapter"
	"eng/internal/toolcap"
	"eng/internal/toolpolicy"
)

type fakeAdapter struct {
	name      string
	available bool
	caps      []toolcap.Capability
}

func (f fakeAdapter) Name() string                                   { return f.name }
func (f fakeAdapter) Provider() string                                { return "fake" }
func (f fakeAdapter) Version() string                                 { return "" }
func (f fakeAdapter) Available() bool                                 { return f.available }
func (f fakeAdapter) Capabilities() []toolcap.Capability              { return f.caps }
func (f fakeAdapter) Doctor() (string, error)                         { return "", nil }
func (f fakeAdapter) Invoke(string, []string, string) (string, error) { return "", nil }

func TestRouteAllowsReadByDefault(t *testing.T) {
	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
	if len(result.Allowed) != 1 {
		t.Fatalf("expected 1 allowed, got %+v", result)
	}
}

func TestRouteBlocksMissingAdapter(t *testing.T) {
	result := Route([]string{"git.status"}, nil, "executor", toolpolicy.Policy{}, false)
	if len(result.Blocked) != 1 {
		t.Fatalf("expected 1 blocked (no adapter), got %+v", result)
	}
}

func TestRouteBlocksUnavailableAdapter(t *testing.T) {
	a := fakeAdapter{name: "git", available: false, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
	if len(result.Blocked) != 1 {
		t.Fatalf("expected 1 blocked (unavailable), got %+v", result)
	}
}

func TestRouteNeedsApprovalForUnlistedWrite(t *testing.T) {
	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.push", Risk: toolcap.RiskWrite}}}
	result := Route([]string{"git.push"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
	if len(result.NeedsApproval) != 1 {
		t.Fatalf("expected 1 needs-approval, got %+v", result)
	}
}

// Uses "docker" and "git" (both real names in agent.RolePermissions'
// executor toolbox — see internal/agent/permissions.go) rather than
// synthetic names, since Route's role-toolbox check would otherwise deny
// both candidates before precedence has anything to decide between.
func TestRouteDeterministicProviderPrecedenceAlphabetical(t *testing.T) {
	g := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "shared.read", Risk: toolcap.RiskRead}}}
	d := fakeAdapter{name: "docker", available: true, caps: []toolcap.Capability{{Name: "shared.read", Risk: toolcap.RiskRead}}}
	result := Route([]string{"shared.read"}, []tooladapter.Adapter{g, d}, "executor", toolpolicy.Policy{}, false)
	if len(result.Allowed) != 1 || result.Allowed[0].Adapter != "docker" {
		t.Fatalf("expected docker to win alphabetical precedence (docker < git), got %+v", result)
	}
}

func TestRouteExplanationReasonsNonEmpty(t *testing.T) {
	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
	if len(result.Allowed) != 1 || result.Allowed[0].Reason == "" {
		t.Fatalf("expected a non-empty reason, got %+v", result)
	}
}
