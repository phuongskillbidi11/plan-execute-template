package tooladapter

import (
	"fmt"
	"os/exec"
	"strings"

	"eng/internal/toolcap"
)

// CodexAdapter is the Phase 10 read-only second-opinion delegation
// adapter — verified against the real, installed `codex` CLI rather than
// guessed (see .plans/2026-08-27-v2-harness-phase10-role-enforcement/
// DECISION_LOG.md Decision 6). Deliberately no write/execute capability —
// a future codex.execute would need its own capability, risk tier, and
// policy decision, not implied by this one.
type CodexAdapter struct {
	available bool
}

func NewCodexAdapter(available bool) CodexAdapter { return CodexAdapter{available: available} }

func (c CodexAdapter) Name() string     { return "codex" }
func (c CodexAdapter) Provider() string { return "cli-agent" }

func (c CodexAdapter) Version() string {
	out, err := exec.Command("codex", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c CodexAdapter) Available() bool { return c.available }

func (c CodexAdapter) Capabilities() []toolcap.Capability {
	return []toolcap.Capability{
		{Name: "codex.inspect", Risk: toolcap.RiskRead},
		{Name: "codex.review", Risk: toolcap.RiskRead},
		{Name: "codex.verify", Risk: toolcap.RiskRead},
	}
}

// Doctor runs `codex login status` — verified directly against the real
// binary to return in well under a second with no network round-trip
// needed for a cached-credential check (DECISION_LOG.md Decision 7).
// Distinguishes "wired" (this succeeds) from merely "installed"
// (Available() alone).
func (c CodexAdapter) Doctor() (string, error) {
	if !c.available {
		return "", fmt.Errorf("codex not found on PATH")
	}
	out, err := exec.Command("codex", "login", "status").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codex is installed but not logged in: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Invoke shells to the real, verified codex CLI — codex.inspect/
// codex.verify use `codex exec --sandbox read-only`, a real,
// non-interactive, read-only invocation (not guessed); codex.review uses
// the top-level `codex review` command with its own real optional flags.
func (c CodexAdapter) Invoke(capability string, args []string, dir string) (string, error) {
	if !c.available {
		return "", fmt.Errorf("codex not found on PATH")
	}
	prompt := strings.Join(args, " ")
	var cmdArgs []string
	switch capability {
	case "codex.inspect", "codex.verify":
		cmdArgs = []string{"exec", "--sandbox", "read-only", prompt}
	case "codex.review":
		cmdArgs = append([]string{"review"}, args...)
	default:
		return "", fmt.Errorf("codex adapter does not support capability %q", capability)
	}
	cmd := exec.Command("codex", cmdArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
