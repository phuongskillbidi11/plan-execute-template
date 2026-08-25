package contextcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingEverythingReturnsDefault(t *testing.T) {
	cfg, err := Load(t.TempDir(), filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg != Default() {
		t.Fatalf("expected Default(), got %+v", cfg)
	}
}

func TestProjectOverrideReplacesGlobal(t *testing.T) {
	global := filepath.Join(t.TempDir(), "default.yaml")
	os.WriteFile(global, []byte("max_skills: 3\n"), 0o644)

	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".agent"), 0o755)
	os.WriteFile(filepath.Join(project, ".agent", "context.yaml"), []byte("max_skills: 9\n"), 0o644)

	cfg, err := Load(project, global)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxSkills != 9 {
		t.Fatalf("expected project override (9), got %d", cfg.MaxSkills)
	}
}

func TestUnsetBoolDoesNotResetToFalse(t *testing.T) {
	global := filepath.Join(t.TempDir(), "default.yaml")
	// summarize_tool_output is NOT mentioned — must stay at Default()'s true.
	os.WriteFile(global, []byte("max_skills: 2\n"), 0o644)

	cfg, err := Load(t.TempDir(), global)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SummarizeToolOutput {
		t.Fatal("expected SummarizeToolOutput to remain true (Default), got false")
	}
}

func TestExplicitFalseIsRespected(t *testing.T) {
	global := filepath.Join(t.TempDir(), "default.yaml")
	os.WriteFile(global, []byte("summarize_tool_output: false\n"), 0o644)

	cfg, err := Load(t.TempDir(), global)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SummarizeToolOutput {
		t.Fatal("expected SummarizeToolOutput to be explicitly false")
	}
}

func TestLogRetentionDefaults(t *testing.T) {
	cfg := Default()
	if cfg.MaxLogFiles != 100 || cfg.MaxLogAgeDays != 30 || cfg.MaxLogTotalMB != 250 {
		t.Fatalf("unexpected log retention defaults: %+v", cfg)
	}
}
