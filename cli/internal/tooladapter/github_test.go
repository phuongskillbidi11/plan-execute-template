package tooladapter

import (
	"os/exec"
	"testing"

	"eng/internal/toolcap"
)

func TestGitHubAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = GitHubAdapter{}
}

func TestGitHubAdapterCapabilitiesAllRead(t *testing.T) {
	g := NewGitHubAdapter(true)
	for _, c := range g.Capabilities() {
		if c.Risk != toolcap.RiskRead {
			t.Fatalf("expected every GitHubAdapter capability to be READ, got %+v", c)
		}
	}
}

func TestGitHubAdapterUnavailableRefuses(t *testing.T) {
	g := NewGitHubAdapter(false)
	if _, err := g.Doctor(); err == nil {
		t.Fatal("expected an error when unavailable")
	}
	if _, err := g.Invoke("github.repo.read", nil, "."); err == nil {
		t.Fatal("expected Invoke to refuse when unavailable")
	}
}

func TestGitHubAdapterInvokeUnsupportedCapabilityErrors(t *testing.T) {
	g := NewGitHubAdapter(true)
	if _, err := g.Invoke("github.nonexistent", nil, "."); err == nil {
		t.Fatal("expected an error for an unsupported capability")
	}
}

func TestGitHubAdapterLiveDoctorIfInstalled(t *testing.T) {
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not found on PATH in this environment")
	}
	g := NewGitHubAdapter(true)
	msg, err := g.Doctor()
	if err != nil {
		t.Skip("gh installed but not authenticated in this environment:", err)
	}
	if msg == "" {
		t.Fatal("expected a non-empty status message")
	}
}
