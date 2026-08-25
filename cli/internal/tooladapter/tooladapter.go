package tooladapter

import "fmt"

// Adapter exposes one external tool/capability to the harness. Distinct
// from internal/agent.Adapter (which launches/talks to a coding agent) —
// see Phase 5 spec.md Decision 10 for why these stay separate.
type Adapter interface {
	Name() string
	Capability() string // matches a capabilities.Known entry
	Available() bool
	PermissionLevel() string // "read" | "read-write" | "execute" | "high-risk"
	Doctor() (string, error)
}

// GitAdapter is the only reference implementation in Phase 5 — it exists
// to prove the interface compiles and is testable, not as a real
// capability gate (git access is already unconditional throughout this
// harness).
type GitAdapter struct {
	available bool
}

func NewGitAdapter(available bool) GitAdapter { return GitAdapter{available: available} }

func (g GitAdapter) Name() string            { return "git" }
func (g GitAdapter) Capability() string      { return "git" }
func (g GitAdapter) Available() bool         { return g.available }
func (g GitAdapter) PermissionLevel() string { return "read-write" }

func (g GitAdapter) Doctor() (string, error) {
	if g.available {
		return "git is on PATH", nil
	}
	return "", fmt.Errorf("git not found on PATH")
}
