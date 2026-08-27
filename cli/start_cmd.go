package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"eng/internal/capabilities"
	"eng/internal/project"
)

func cmdStart(args []string) {
	doInit := false
	for _, a := range args {
		if a == "--init" {
			doInit = true
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if project.DetectModeResult(dir).Mode == "none" {
		if doInit {
			cmdInit(nil)
		} else {
			fmt.Println("This project isn't initialized for the harness yet.")
			fmt.Println("Run `eng init` first, or `eng start --init` to initialize now and continue.")
			fmt.Println("(eng init only ever creates .agent/project.yaml — nothing else is touched.)")
			return
		}
	}

	fmt.Println("eng start")
	fmt.Println()
	cmdDoctor(nil)

	fmt.Println("\nFor natural-language requests, this session should consult:")
	fmt.Printf("  %s\n", filepath.Join(harnessDir(), "core", "runtime", "METHOD.md"))

	if capabilities.Detect("claude") {
		fmt.Println("\nLaunching Claude Code...")
		status := gatherBootstrapStatus(dir)
		prompt := renderBootstrapPrompt(status)
		c := exec.Command("claude", startClaudeArgs(prompt)...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = buildChildEnv(os.Environ(), engBinDirs(), harnessDir(), dir, harnessVersion())
		if err := c.Run(); err != nil {
			fmt.Println("\nCould not launch `claude` automatically:", err)
			fmt.Println("Run it yourself in this directory.")
		}
		return
	}

	fmt.Println("\n`claude` was not found on PATH. Configure an agent in .agent/project.yaml,")
	fmt.Println("or install one and re-run `eng start`.")
}

// startClaudeArgs returns the argv (excluding the program name) eng start
// passes to the launched `claude` process. A pure function, deliberately
// separated from the exec.Command call itself, so the bootstrap wiring is
// testable without spawning a real process — the same pattern buildChildEnv
// already established for the child environment. --append-system-prompt
// adds the harness's trusted runtime identity to Claude's default system
// prompt without replacing it (see DECISION_LOG.md Decision 1); os/exec
// never shells out this call on any platform, so prompt is passed as a
// single argv entry with no quoting/escaping hazard.
func startClaudeArgs(prompt string) []string {
	return []string{"--append-system-prompt", prompt}
}

// engBinDirs returns every directory known to contain a working `eng`
// binary right now — this process's own directory (covers a locally-built
// dev binary that was never `eng install`-ed) and the installed bin/
// directory — deduplicated. Phase 9 spec.md P1-3: a user can launch
// `eng start` via its full path in a shell whose PATH was never actually
// updated (e.g. Windows `setx` only affects new sessions) — everything
// `eng start` launches must not depend on that PATH being correct.
func engBinDirs() []string {
	var dirs []string
	if self, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(self))
	}
	dirs = append(dirs, binDir())
	return dedupStrings(dirs)
}

func dedupStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// buildChildEnv returns a copy of base with binDirs prepended to PATH
// (preserving whatever PATH already existed, never destroying it — PATH
// detection is case-insensitive, since Windows commonly stores it as
// "Path") and ENG_HOME/ENG_PROJECT_ROOT/ENG_VERSION set (replacing, not
// duplicating, any pre-existing entries). A pure function — no process
// spawn, no filesystem access — so it's testable without a real `claude`
// binary or a real child process. See Phase 9 spec.md P1-3 and
// DECISION_LOG.md Decision 10.
func buildChildEnv(base []string, binDirs []string, harnessHome, projectRoot, version string) []string {
	var existingPath string
	hasPath := false
	drop := map[string]bool{"ENG_HOME": true, "ENG_PROJECT_ROOT": true, "ENG_VERSION": true}

	out := make([]string, 0, len(base)+4)
	for _, kv := range base {
		key, val, ok := strings.Cut(kv, "=")
		if !ok {
			out = append(out, kv)
			continue
		}
		upper := strings.ToUpper(key)
		if upper == "PATH" {
			existingPath = val
			hasPath = true
			continue // rebuilt below, with binDirs prepended
		}
		if drop[upper] {
			continue // rebuilt below with the fresh value
		}
		out = append(out, kv)
	}

	newPath := strings.Join(dedupStrings(binDirs), string(os.PathListSeparator))
	if hasPath && existingPath != "" {
		newPath += string(os.PathListSeparator) + existingPath
	}
	out = append(out, "PATH="+newPath)

	if harnessHome != "" {
		out = append(out, "ENG_HOME="+harnessHome)
	}
	if projectRoot != "" {
		out = append(out, "ENG_PROJECT_ROOT="+projectRoot)
	}
	if version != "" {
		out = append(out, "ENG_VERSION="+version)
	}
	return out
}
