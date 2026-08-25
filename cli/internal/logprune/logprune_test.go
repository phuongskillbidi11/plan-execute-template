package logprune

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLog(t *testing.T, dir, name string, age time.Duration, size int) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := time.Now().Add(-age)
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestPruneRespectsMaxFilesButKeepsMostRecent(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "a.log", 3*time.Hour, 10)
	writeLog(t, dir, "b.log", 2*time.Hour, 10)
	writeLog(t, dir, "c.log", 1*time.Hour, 10)

	result, err := Prune(dir, 1, 0, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deleted) != 2 {
		t.Fatalf("expected 2 deleted, got %d: %v", len(result.Deleted), result.Deleted)
	}
	if _, err := os.Stat(filepath.Join(dir, "c.log")); err != nil {
		t.Fatal("the most recent file must never be deleted")
	}
}

func TestPruneDryRunDeletesNothing(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "a.log", 3*time.Hour, 10)
	writeLog(t, dir, "b.log", 1*time.Hour, 10)

	result, err := Prune(dir, 1, 0, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deleted) != 1 {
		t.Fatalf("expected 1 reported-would-delete, got %d", len(result.Deleted))
	}
	if _, err := os.Stat(filepath.Join(dir, "a.log")); err != nil {
		t.Fatal("dry-run must not actually delete anything")
	}
}

func TestPruneMissingDirIsNotAnError(t *testing.T) {
	result, err := Prune(filepath.Join(t.TempDir(), "nope"), 10, 30, 250, false)
	if err != nil || len(result.Deleted) != 0 {
		t.Fatalf("expected no error and no deletions, got %+v, %v", result, err)
	}
}

func TestPruneAgeLimit(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "old.log", 40*24*time.Hour, 10)
	writeLog(t, dir, "new.log", time.Hour, 10)

	result, err := Prune(dir, 0, 30, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Deleted) != 1 || result.Deleted[0] != filepath.Join(dir, "old.log") {
		t.Fatalf("expected only old.log deleted, got %+v", result.Deleted)
	}
}
