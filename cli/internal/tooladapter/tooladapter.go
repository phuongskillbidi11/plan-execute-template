package tooladapter

import (
	"fmt"
	"os/exec"
	"strings"

	"eng/internal/toolcap"
)

// Adapter exposes one external tool/capability to the harness. Distinct
// from internal/agent.Adapter (which launches/talks to a coding agent) —
// see Phase 5 spec.md Decision 10 for why these stay separate. This
// interface is a Phase 7 revision of the Phase 5 foundation (Phase 7
// spec.md Decision 1) — Phase 5's own DECISION_LOG called it
// "foundation only," not a frozen contract.
type Adapter interface {
	Name() string
	Provider() string // "local-binary" | "github-cli" | "mcp" | ...
	Version() string   // best-effort; "" if unknown or unavailable
	Capabilities() []toolcap.Capability
	Available() bool
	Doctor() (string, error)
	// Invoke runs capability with args in dir, returning its output. dir
	// is explicit (not the process's own cwd) so invocation is testable
	// without changing directories globally.
	Invoke(capability string, args []string, dir string) (string, error)
}

// GitAdapter is a local-binary reference implementation — git is already
// unconditionally required throughout this harness, so Available() is
// simply whether it's on PATH.
type GitAdapter struct {
	available bool
}

func NewGitAdapter(available bool) GitAdapter { return GitAdapter{available: available} }

func (g GitAdapter) Name() string     { return "git" }
func (g GitAdapter) Provider() string { return "local-binary" }

func (g GitAdapter) Version() string {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g GitAdapter) Available() bool { return g.available }

func (g GitAdapter) Capabilities() []toolcap.Capability {
	return []toolcap.Capability{
		{Name: "git.status", Risk: toolcap.RiskRead},
		{Name: "git.diff", Risk: toolcap.RiskRead},
		{Name: "git.log", Risk: toolcap.RiskRead},
		{Name: "git.commit", Risk: toolcap.RiskWrite},
		{Name: "git.push", Risk: toolcap.RiskWrite},
		{Name: "git.force_push", Risk: toolcap.RiskDestructive},
	}
}

func (g GitAdapter) Doctor() (string, error) {
	if g.available {
		return "git is on PATH", nil
	}
	return "", fmt.Errorf("git not found on PATH")
}

func (g GitAdapter) Invoke(capability string, args []string, dir string) (string, error) {
	if !g.available {
		return "", fmt.Errorf("git not found on PATH")
	}
	var gitArgs []string
	switch capability {
	case "git.status":
		gitArgs = append([]string{"status"}, args...)
	case "git.diff":
		gitArgs = append([]string{"diff"}, args...)
	case "git.log":
		gitArgs = append([]string{"log"}, args...)
	case "git.commit":
		gitArgs = append([]string{"commit"}, args...)
	case "git.push":
		gitArgs = append([]string{"push"}, args...)
	case "git.force_push":
		gitArgs = append([]string{"push", "--force"}, args...)
	default:
		return "", fmt.Errorf("git adapter does not support capability %q", capability)
	}
	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
