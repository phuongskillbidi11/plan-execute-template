package tooladapter

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"eng/internal/toolcap"
)

func TestGitAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = GitAdapter{}
}

func TestGitAdapterCapabilitiesIncludeExpectedRisks(t *testing.T) {
	g := NewGitAdapter(true)
	byName := map[string]toolcap.Risk{}
	for _, c := range g.Capabilities() {
		byName[c.Name] = c.Risk
	}
	if byName["git.status"] != toolcap.RiskRead {
		t.Fatalf("expected git.status to be READ, got %v", byName["git.status"])
	}
	if byName["git.push"] != toolcap.RiskWrite {
		t.Fatalf("expected git.push to be WRITE, got %v", byName["git.push"])
	}
	if byName["git.force_push"] != toolcap.RiskDestructive {
		t.Fatalf("expected git.force_push to be DESTRUCTIVE, got %v", byName["git.force_push"])
	}
}

func TestGitAdapterUnavailableRefuses(t *testing.T) {
	g := NewGitAdapter(false)
	if _, err := g.Doctor(); err == nil {
		t.Fatal("expected an error when unavailable")
	}
	if _, err := g.Invoke("git.status", nil, "."); err == nil {
		t.Fatal("expected Invoke to refuse when unavailable")
	}
}

func TestGitAdapterInvokeUnsupportedCapabilityErrors(t *testing.T) {
	g := NewGitAdapter(true)
	if _, err := g.Invoke("git.nonexistent", nil, "."); err == nil {
		t.Fatal("expected an error for an unsupported capability")
	}
}

func TestGitAdapterInvokeStatusInARealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found on PATH in this environment")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "t")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)

	g := NewGitAdapter(true)
	out, err := g.Invoke("git.status", nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected non-empty git status output")
	}
}
