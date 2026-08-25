package tooladapter

import (
	"fmt"
	"os/exec"
	"strings"

	"eng/internal/toolcap"
)

// GitHubAdapter is the Phase 7 external reference adapter — read-only,
// delegating entirely to the `gh` CLI's own authentication (a token
// stored in the OS keyring by `gh auth login`; the harness never reads,
// stores, or displays it — see Phase 7 spec.md Decision 10).
type GitHubAdapter struct {
	available bool
}

func NewGitHubAdapter(available bool) GitHubAdapter { return GitHubAdapter{available: available} }

func (g GitHubAdapter) Name() string     { return "github" }
func (g GitHubAdapter) Provider() string { return "github-cli" }

func (g GitHubAdapter) Version() string {
	out, err := exec.Command("gh", "--version").Output()
	if err != nil {
		return ""
	}
	lines := strings.SplitN(string(out), "\n", 2)
	return strings.TrimSpace(lines[0])
}

func (g GitHubAdapter) Available() bool { return g.available }

func (g GitHubAdapter) Capabilities() []toolcap.Capability {
	return []toolcap.Capability{
		{Name: "github.repo.read", Risk: toolcap.RiskRead},
		{Name: "github.pr.read", Risk: toolcap.RiskRead},
		{Name: "github.issue.read", Risk: toolcap.RiskRead},
	}
}

func (g GitHubAdapter) Doctor() (string, error) {
	if !g.available {
		return "", fmt.Errorf("gh not found on PATH")
	}
	if _, err := exec.Command("gh", "auth", "status").CombinedOutput(); err != nil {
		return "", fmt.Errorf("gh is installed but not authenticated: %w", err)
	}
	return "gh is on PATH and authenticated", nil
}

func (g GitHubAdapter) Invoke(capability string, args []string, dir string) (string, error) {
	if !g.available {
		return "", fmt.Errorf("gh not found on PATH")
	}
	var ghArgs []string
	switch capability {
	case "github.repo.read":
		ghArgs = append([]string{"repo", "view"}, args...)
	case "github.pr.read":
		ghArgs = append([]string{"pr", "list"}, args...)
	case "github.issue.read":
		ghArgs = append([]string{"issue", "list"}, args...)
	default:
		return "", fmt.Errorf("github adapter does not support capability %q", capability)
	}
	cmd := exec.Command("gh", ghArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
