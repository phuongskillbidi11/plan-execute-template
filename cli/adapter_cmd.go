package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eng/internal/agent"
	"eng/internal/planmeta"
	"eng/internal/rolestate"
)

func cmdAdapter(args []string) {
	if len(args) < 2 || args[0] != "prompt" {
		fmt.Println(`Usage: eng adapter prompt <planner|plan-reviewer|executor|verifier> <plan-dir> ["<request text>"]`)
		os.Exit(1)
	}
	role := agent.Role(args[1])
	if len(args) < 3 {
		fmt.Println("Usage: eng adapter prompt <role> <plan-dir>")
		os.Exit(1)
	}
	planDir, absErr := filepath.Abs(args[2])
	if absErr != nil {
		planDir = args[2]
	}
	request := ""
	if len(args) > 3 {
		request = strings.Join(args[3:], " ")
	}

	// Phase 10: this call is the activation boundary — the sole recorded,
	// validated step every core/*/METHOD.md file already instructs an
	// agent to run before acting in a role. A role/state mismatch refuses
	// here, before anything is printed (fail closed) — see spec.md's
	// "Role activation is the existing eng adapter prompt command."
	meta, err := planmeta.Load(planDir)
	if err != nil {
		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
		os.Exit(1)
	}
	ok, reason := rolestate.AllowedForState(string(role), meta.State, meta.RiskLevel == "quick-fix")
	if !ok {
		fmt.Printf("REFUSED: %s\n", reason)
		planmeta.AppendStructuredEvent(planDir, "role_activation_denied", map[string]interface{}{
			"role": string(role), "state": meta.State, "reason": reason,
		})
		os.Exit(1)
	}

	a := agent.ClaudeCodeAdapter{HarnessDir: harnessDir()}
	if !a.Available() {
		fmt.Println("note: `claude` was not found on PATH — printing the prompt for manual use anyway.")
	}

	prompt, err := a.RolePrompt(role, planDir)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println(prompt)

	// Phase 5: the context bundle is now always folded in here — the
	// Context Manager's output is the authoritative prepared context for
	// this role, not a separate manual step (Phase 5 spec.md Decision 2).
	bundle, bundleErr := buildContextBundle(string(role), planDir, request)
	if bundleErr != nil {
		fmt.Println("(no project-specific context bundle available:", bundleErr, ")")
	} else {
		fmt.Println(bundle)
	}

	// Record the activation itself — this, not the printed text above, is
	// what Task 5's `eng tools invoke` role check and Task 3's
	// ExecutorActivated/role-verification facts actually consult.
	now := time.Now().UTC().Format(time.RFC3339)
	state := &rolestate.RoleState{
		CurrentRole:       string(role),
		ActivatedAt:       now,
		ActivatedForState: meta.State,
		PromptGeneratedAt: now,
		ContextManifest:   roleManifestFileName(string(role)),
	}
	if err := rolestate.Save(planDir, state); err != nil {
		fmt.Println("warning: role activated but failed to record role-state.yaml:", err)
	}
	planmeta.AppendStructuredEvent(planDir, "role_activated", map[string]interface{}{
		"role": string(role), "state": meta.State, "context_manifest": state.ContextManifest,
	})
}
