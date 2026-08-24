package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModeLegacy(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# x"), 0o644)
	os.Mkdir(filepath.Join(dir, ".plans"), 0o755)
	if got := DetectMode(dir); got != "legacy" {
		t.Fatalf("expected legacy, got %q", got)
	}
}

func TestDetectModeNone(t *testing.T) {
	dir := t.TempDir()
	if got := DetectMode(dir); got != "none" {
		t.Fatalf("expected none, got %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ProjectName: "x", Mode: "modern", Stack: Stack{Type: "go"}}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "modern" || got.Stack.Type != "go" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
