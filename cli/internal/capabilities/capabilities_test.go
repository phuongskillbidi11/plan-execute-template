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

func TestDescribeAllCoversKnownSet(t *testing.T) {
	all := DescribeAll()
	if len(all) != len(Known) {
		t.Fatalf("expected %d entries, got %d", len(Known), len(all))
	}
}

func TestDescribeUnavailableHasNoVersion(t *testing.T) {
	c := Describe("definitely-not-a-real-binary-xyz")
	if c.Available || c.Version != "" {
		t.Fatalf("expected unavailable with no version, got %+v", c)
	}
}

func TestDescribeGitHasProviderSet(t *testing.T) {
	c := Describe("git")
	if c.Provider != "local-binary" {
		t.Fatalf("expected provider local-binary, got %q", c.Provider)
	}
}
