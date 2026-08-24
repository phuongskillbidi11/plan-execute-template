package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"eng/internal/capabilities"
)

type Role string

const (
	RolePlanner  Role = "planner"
	RoleReviewer Role = "plan-reviewer"
	RoleExecutor Role = "executor"
	RoleVerifier Role = "verifier"
)

// Adapter lets eng talk to a specific AI coding agent. Implementations
// detect availability and assemble role-specific prompts; none of them
// drive an agent unattended — see Phase 3 DECISION_LOG for why.
type Adapter interface {
	Name() string
	Available() bool
	RolePrompt(role Role, planDir string) (string, error)
}

// ClaudeCodeAdapter is the reference implementation — the only one Phase 3
// implements, per the explicit "prioritize Claude Code, don't deeply
// integrate every agent" instruction.
type ClaudeCodeAdapter struct {
	HarnessDir string
}

func (a ClaudeCodeAdapter) Name() string { return "claude-code" }

func (a ClaudeCodeAdapter) Available() bool { return capabilities.Detect("claude") }

func (a ClaudeCodeAdapter) RolePrompt(role Role, planDir string) (string, error) {
	methodPath := filepath.Join(a.HarnessDir, "core", string(role), "METHOD.md")
	method, err := os.ReadFile(methodPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", methodPath, err)
	}

	abs, err := filepath.Abs(planDir)
	if err != nil {
		abs = planDir
	}

	return fmt.Sprintf(`%s

---

You are acting in the %s role for the plan at: %s

Read spec.md, tasks.md, tests.md, and plan.yaml in that folder before doing anything else.
`, string(method), role, abs), nil
}
