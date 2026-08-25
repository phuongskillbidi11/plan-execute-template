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
