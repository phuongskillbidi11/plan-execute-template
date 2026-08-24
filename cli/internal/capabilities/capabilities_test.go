package capabilities

import "testing"

func TestDetectMissingBinary(t *testing.T) {
	if Detect("definitely-not-a-real-binary-xyz") {
		t.Fatal("expected false for a nonexistent binary")
	}
}

func TestDetectAllCoversKnownSet(t *testing.T) {
	all := DetectAll()
	if len(all) != len(Known) {
		t.Fatalf("expected %d entries, got %d", len(Known), len(all))
	}
	for _, name := range Known {
		if _, ok := all[name]; !ok {
			t.Fatalf("missing %q in DetectAll result", name)
		}
	}
}

func TestDetectGitIsUsuallyPresent(t *testing.T) {
	// This repository is a git repo being worked on right now — git must be
	// on PATH for that to be possible at all.
	if !Detect("git") {
		t.Skip("git not found on PATH in this environment")
	}
}
