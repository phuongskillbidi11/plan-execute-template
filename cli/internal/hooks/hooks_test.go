package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadGlobalDefault(t *testing.T) {
	global := filepath.Join(t.TempDir(), "default.yaml")
	writeYAML(t, global, "before_plan: [project_scan]\ncommands:\n  project_scan: eng scan\n")

	project := t.TempDir()
	cfg, err := Load(project, global)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Stage("before_plan"); len(got) != 1 || got[0] != "project_scan" {
		t.Fatalf("got %+v", got)
	}
}

func TestProjectOverrideReplacesGlobal(t *testing.T) {
	global := filepath.Join(t.TempDir(), "default.yaml")
	writeYAML(t, global, "before_plan: [project_scan]\n")

	project := t.TempDir()
	writeYAML(t, filepath.Join(project, ".agent", "hooks.yaml"), "before_plan: [custom_check]\n")

	cfg, err := Load(project, global)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Stage("before_plan")
	if len(got) != 1 || got[0] != "custom_check" {
		t.Fatalf("expected project override only, got %+v", got)
	}
}
