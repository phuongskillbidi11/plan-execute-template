package tooladapter

import (
	"os/exec"
	"testing"

	"eng/internal/toolcap"
)

func TestCodexAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = CodexAdapter{}
}

func TestCodexAdapterCapabilitiesAllRead(t *testing.T) {
	c := NewCodexAdapter(true)
	for _, cap := range c.Capabilities() {
		if cap.Risk != toolcap.RiskRead {
			t.Fatalf("expected every CodexAdapter capability to be READ, got %+v", cap)
		}
	}
}

// TestCodexAdapterNoExecuteCapability guards Phase 10 spec.md's explicit
// scope decision: no codex.execute (write) capability exists yet.
func TestCodexAdapterNoExecuteCapability(t *testing.T) {
	c := NewCodexAdapter(true)
	for _, cap := range c.Capabilities() {
		if cap.Name == "codex.execute" {
			t.Fatal("codex.execute must not exist — write execution is explicitly out of scope")
		}
	}
}

func TestCodexAdapterUnavailableRefuses(t *testing.T) {
	c := NewCodexAdapter(false)
	if _, err := c.Doctor(); err == nil {
		t.Fatal("expected an error when unavailable")
	}
	if _, err := c.Invoke("codex.inspect", nil, "."); err == nil {
		t.Fatal("expected Invoke to refuse when unavailable")
	}
}

func TestCodexAdapterInvokeUnsupportedCapabilityErrors(t *testing.T) {
	c := NewCodexAdapter(true)
	if _, err := c.Invoke("codex.nonexistent", nil, "."); err == nil {
		t.Fatal("expected an error for an unsupported capability")
	}
}

// TestCodexAdapterLiveDoctorIfInstalled mirrors GitHubAdapter's own live
// test pattern exactly — skips gracefully if codex isn't on PATH in this
// environment. No automated test invokes a real codex exec/review AI
// call (kept out of go test for speed/determinism/no API cost) — see
// Task 8.6's separate, manual, bounded real verification.
func TestCodexAdapterLiveDoctorIfInstalled(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex not found on PATH in this environment")
	}
	c := NewCodexAdapter(true)
	msg, err := c.Doctor()
	if err != nil {
		t.Skip("codex installed but not logged in in this environment:", err)
	}
	if msg == "" {
		t.Fatal("expected a non-empty status message")
	}
}
