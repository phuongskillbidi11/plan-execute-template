package docsearch

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseSections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "src-map.md")
	writeFile(t, path, "# Title\n\n### cli/ — the eng CLI\n\nBody one.\n\n### harness/ — payload\n\nBody two.\n")

	sections, err := ParseSections(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 2 || sections[0].Title != "cli/ — the eng CLI" {
		t.Fatalf("got %+v", sections)
	}
}

func TestMatchRanksByScoreAndCaps(t *testing.T) {
	sections := []Section{
		{Title: "Modbus adapter", Body: "industrial protocol details"},
		{Title: "Login page", Body: "web authentication flow"},
		{Title: "Web API", Body: "http handlers and routes"},
	}
	matched := Match(sections, "add a web login", 1)
	if len(matched) != 1 {
		t.Fatalf("expected 1 result capped, got %d", len(matched))
	}
	if matched[0].Title != "Login page" {
		t.Fatalf("expected 'Login page' to rank highest, got %q", matched[0].Title)
	}
}

func TestMatchNoCapReturnsAll(t *testing.T) {
	sections := []Section{{Title: "web x", Body: "y"}, {Title: "web z", Body: "y"}}
	matched := Match(sections, "web", 0)
	if len(matched) != 2 {
		t.Fatalf("expected 2 with no cap, got %d", len(matched))
	}
}

// TestMatchIgnoresSubstringInsideUnrelatedWord mirrors the skillmatch fix
// for docsearch: a body word must not match merely because it's a substring
// of an unrelated request word.
func TestMatchIgnoresSubstringInsideUnrelatedWord(t *testing.T) {
	sections := []Section{{Title: "Unrelated", Body: "form a hypothesis about the cause"}}
	matched := Match(sections, "Maintain a C# WinForms configuration tool", 0)
	if len(matched) != 0 {
		t.Fatalf("expected 0 (no word-boundary match), got %d", len(matched))
	}
}

// TestMatchSingleBodyWordAloneDoesNotClearThreshold guards Phase 9 P1-2: a
// single generic body-word match must not clear MinMatchScore on its own,
// while a title match does.
func TestMatchSingleBodyWordAloneDoesNotClearThreshold(t *testing.T) {
	sections := []Section{{Title: "Something else entirely", Body: "covers protocol basics only"}}
	matched := Match(sections, "a totally unrelated firmware question about protocol handling", 0)
	if len(matched) != 0 {
		t.Fatalf("expected a lone body word not to clear MinMatchScore, got %d matched", len(matched))
	}
}

func TestMatchSingleTitleWordAloneClearsThreshold(t *testing.T) {
	sections := []Section{{Title: "auth/ — token validation", Body: "unrelated body text"}}
	matched := Match(sections, "add input validation to the auth check", 0)
	if len(matched) != 1 {
		t.Fatalf("expected a single title-word match to clear MinMatchScore, got %d", len(matched))
	}
}
