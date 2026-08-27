package main

import (
	"os"
	"strings"
	"testing"
)

// TestBuildChildEnvPrependsPathNotReplace guards the real Phase 9 P1-3
// defect: a harness-launched agent's own PATH must gain the harness bin
// directory without losing whatever PATH the parent process already had.
func TestBuildChildEnvPrependsPathNotReplace(t *testing.T) {
	base := []string{"PATH=/usr/bin:/bin", "HOME=/home/x"}
	env := buildChildEnv(base, []string{"/opt/eng/bin"}, "", "", "")

	path := lookupEnv(t, env, "PATH")
	if !strings.HasPrefix(path, "/opt/eng/bin"+string(os.PathListSeparator)) {
		t.Fatalf("expected PATH to be prepended with the bin dir, got %q", path)
	}
	if !strings.Contains(path, "/usr/bin:/bin") {
		t.Fatalf("expected the original PATH to be preserved, got %q", path)
	}
	if !contains(env, "HOME=/home/x") {
		t.Fatalf("expected unrelated env vars to survive untouched, got %v", env)
	}
}

func TestBuildChildEnvWorksWithNoExistingPath(t *testing.T) {
	base := []string{"HOME=/home/x"}
	env := buildChildEnv(base, []string{"/opt/eng/bin"}, "", "", "")
	if lookupEnv(t, env, "PATH") != "/opt/eng/bin" {
		t.Fatalf("expected PATH to be created from scratch, got %v", env)
	}
}

// TestBuildChildEnvCaseInsensitivePathDetection guards Windows, where PATH
// commonly appears as "Path=" in os.Environ() — the existing value must
// still be found and preserved, not duplicated as a second PATH entry.
func TestBuildChildEnvCaseInsensitivePathDetection(t *testing.T) {
	base := []string{`Path=C:\Windows\System32`}
	env := buildChildEnv(base, []string{`C:\eng\bin`}, "", "", "")

	count := 0
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if strings.EqualFold(key, "PATH") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one PATH entry, got %d in %v", count, env)
	}
	path := lookupEnv(t, env, "PATH")
	if !strings.Contains(path, `C:\Windows\System32`) {
		t.Fatalf("expected the original Windows PATH value to be preserved, got %q", path)
	}
}

func TestBuildChildEnvDedupesBinDirs(t *testing.T) {
	env := buildChildEnv(nil, []string{"/opt/eng/bin", "/opt/eng/bin"}, "", "", "")
	path := lookupEnv(t, env, "PATH")
	if strings.Count(path, "/opt/eng/bin") != 1 {
		t.Fatalf("expected the duplicate bin dir to be deduplicated, got %q", path)
	}
}

func TestBuildChildEnvSetsEngVars(t *testing.T) {
	env := buildChildEnv(nil, []string{"/opt/eng/bin"}, "/home/x/.engineering-harness", "/home/x/project", "0.8.0-phase9")
	if lookupEnv(t, env, "ENG_HOME") != "/home/x/.engineering-harness" {
		t.Fatalf("expected ENG_HOME to be set, got %v", env)
	}
	if lookupEnv(t, env, "ENG_PROJECT_ROOT") != "/home/x/project" {
		t.Fatalf("expected ENG_PROJECT_ROOT to be set, got %v", env)
	}
	if lookupEnv(t, env, "ENG_VERSION") != "0.8.0-phase9" {
		t.Fatalf("expected ENG_VERSION to be set, got %v", env)
	}
}

// TestBuildChildEnvReplacesNotDuplicatesEngVars guards against a
// pre-existing ENG_HOME (e.g. a nested `eng start` inside another) leaking
// a stale value alongside the freshly-computed one.
func TestBuildChildEnvReplacesNotDuplicatesEngVars(t *testing.T) {
	base := []string{"ENG_HOME=/stale/old-path", "ENG_VERSION=0.1.0-stale"}
	env := buildChildEnv(base, nil, "/fresh/home", "", "0.9.0-fresh")

	if lookupEnv(t, env, "ENG_HOME") != "/fresh/home" {
		t.Fatalf("expected the fresh ENG_HOME to win, got %v", env)
	}
	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "ENG_HOME=") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one ENG_HOME entry, got %d in %v", count, env)
	}
}

func lookupEnv(t *testing.T, env []string, key string) string {
	t.Helper()
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && strings.EqualFold(k, key) {
			return v
		}
	}
	t.Fatalf("expected %s to be set, got %v", key, env)
	return ""
}

func contains(env []string, entry string) bool {
	for _, kv := range env {
		if kv == entry {
			return true
		}
	}
	return false
}
