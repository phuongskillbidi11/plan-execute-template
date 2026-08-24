package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Detect(dir)
	if got.Type != "go" {
		t.Fatalf("expected type=go, got %q", got.Type)
	}
}

func TestDetectUnknown(t *testing.T) {
	dir := t.TempDir()
	got := Detect(dir)
	if got.Type != "unknown" {
		t.Fatalf("expected type=unknown, got %q", got.Type)
	}
}
