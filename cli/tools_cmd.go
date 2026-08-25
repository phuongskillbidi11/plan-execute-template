package main

import (
	"fmt"
	"os"
	"path/filepath"

	"eng/internal/capabilities"
	"eng/internal/mcpregistry"
	"eng/internal/planmeta"
	"eng/internal/project"
	"eng/internal/tooladapter"
	"eng/internal/toolcap"
	"eng/internal/toolpolicy"
)

func cmdTools(args []string) {
	if len(args) < 1 || args[0] != "invoke" {
		fmt.Println("Usage: eng tools invoke <role> <capability> <plan-dir> [args...]")
		os.Exit(1)
	}
	toolsInvoke(args[1:])
}

// registeredAdapters returns every adapter the harness knows about,
// evaluated fresh each call so Available()/Doctor() reflect live state.
// Shared by `eng tools invoke`, `eng capabilities explain`, `eng
// doctor`, and buildContextBundle's ## Tools section — the one place
// adapters are listed (Phase 7 Requirement 1: this is harness wiring,
// not a responsibility any adapter or command owns itself).
func registeredAdapters(repoRoot string) []tooladapter.Adapter {
	adapters := []tooladapter.Adapter{
		tooladapter.NewGitAdapter(capabilities.Detect("git")),
		tooladapter.NewGitHubAdapter(capabilities.Detect("gh")),
	}

	servers, _ := mcpregistry.Load(filepath.Join(harnessDir(), "mcp", "servers.yaml"))
	for _, s := range servers {
		if s.Name == "docs-search" && s.Transport == "mock" {
			adapters = append(adapters, tooladapter.NewReferenceMCPAdapter(filepath.Join(repoRoot, "docs")))
		}
	}
	return adapters
}

func toolsInvoke(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: eng tools invoke <role> <capability> <plan-dir> [args...]")
		os.Exit(1)
	}
	role := args[0]
	capability := args[1]
	planDir, err := filepath.Abs(args[2])
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	invokeArgs := args[3:]

	repoRoot, err := os.Getwd()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	meta, err := planmeta.Load(planDir)
	if err != nil {
		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
		os.Exit(1)
	}
	approved := meta.ApprovedAt != ""

	var policy toolpolicy.Policy
	if cfg, err := project.Load(repoRoot); err == nil {
		policy = cfg.Tools
	}

	adapters := registeredAdapters(repoRoot)
	var owner tooladapter.Adapter
	var risk toolcap.Risk
	found := false
	for _, a := range adapters {
		for _, c := range a.Capabilities() {
			if c.Name == capability {
				owner = a
				risk = c.Risk
				found = true
			}
		}
	}
	if !found {
		fmt.Printf("no adapter declares capability %q\n", capability)
		os.Exit(1)
	}
	if !owner.Available() {
		fmt.Printf("adapter %q is not available\n", owner.Name())
		os.Exit(1)
	}

	decision := toolpolicy.Decide(capability, risk, owner.Name(), role, policy, approved)
	audit := map[string]interface{}{
		"adapter":    owner.Name(),
		"capability": capability,
		"role":       role,
		"result":     string(decision.Verdict),
		"reason":     decision.Reason,
	}

	if decision.Verdict != toolpolicy.Allowed {
		planmeta.AppendStructuredEvent(planDir, "tool_invocation", audit)
		fmt.Printf("REFUSED (%s): %s\n", decision.Verdict, decision.Reason)
		os.Exit(1)
	}

	out, invokeErr := owner.Invoke(capability, invokeArgs, repoRoot)

	ctxCfg := loadContextConfig(repoRoot)
	logPath, logErr := writeFullLog(repoRoot, "tool-"+owner.Name(), out)
	display := out
	if ctxCfg.SummarizeToolOutput {
		display = summarizeOutput(out, ctxCfg.MaxLogLines)
	}
	if logErr == nil {
		audit["log_path"] = logPath
	}
	if invokeErr != nil {
		audit["result"] = "ERROR"
		audit["error"] = invokeErr.Error()
	}
	planmeta.AppendStructuredEvent(planDir, "tool_invocation", audit)

	fmt.Println(display)
	if invokeErr != nil {
		fmt.Println("error:", invokeErr)
		os.Exit(1)
	}
}
