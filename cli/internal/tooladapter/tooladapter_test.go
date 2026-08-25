package tooladapter

import "testing"

func TestGitAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = GitAdapter{}
}

func TestGitAdapterAvailable(t *testing.T) {
	g := NewGitAdapter(true)
	if !g.Available() || g.PermissionLevel() != "read-write" {
		t.Fatalf("unexpected adapter state: %+v", g)
	}
	if _, err := g.Doctor(); err != nil {
		t.Fatalf("expected no error when available, got %v", err)
	}
}

func TestGitAdapterUnavailable(t *testing.T) {
	g := NewGitAdapter(false)
	if _, err := g.Doctor(); err == nil {
		t.Fatal("expected an error when unavailable")
	}
}
