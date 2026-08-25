package toolrouter

import (
	"testing"

	"eng/internal/tooladapter"
)

func TestFilterMatchesRequiredAndAvailable(t *testing.T) {
	adapters := []tooladapter.Adapter{
		tooladapter.NewGitAdapter(true),
	}
	got := Filter([]string{"git"}, adapters)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
}

func TestFilterExcludesUnavailable(t *testing.T) {
	adapters := []tooladapter.Adapter{
		tooladapter.NewGitAdapter(false),
	}
	got := Filter([]string{"git"}, adapters)
	if len(got) != 0 {
		t.Fatalf("expected 0 matches for an unavailable adapter, got %d", len(got))
	}
}

func TestFilterExcludesUnrequested(t *testing.T) {
	adapters := []tooladapter.Adapter{
		tooladapter.NewGitAdapter(true),
	}
	got := Filter([]string{"docker"}, adapters)
	if len(got) != 0 {
		t.Fatalf("expected 0 matches for an unrequested capability, got %d", len(got))
	}
}
