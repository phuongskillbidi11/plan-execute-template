package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eng/internal/contextcfg"
	"eng/internal/docsearch"
	"eng/internal/planmeta"
	"eng/internal/project"
	"eng/internal/skillrouter"
	"eng/internal/skills"
	"eng/internal/taskscope"
)

func cmdContext(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng context <skills|project|task|bundle|manifest> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "skills":
		contextSkills(args[1:])
	case "project":
		contextProject(args[1:])
	case "task":
		contextTask(args[1:])
	case "bundle":
		contextBundle(args[1:])
	case "manifest":
		contextManifest(args[1:])
	default:
		fmt.Println("Usage: eng context <skills|project|task|bundle|manifest> ...")
		os.Exit(1)
	}
}

func loadContextConfig(dir string) contextcfg.Config {
	global := filepath.Join(harnessDir(), "context", "default.yaml")
	cfg, err := contextcfg.Load(dir, global)
	if err != nil {
		return contextcfg.Default()
	}
	return cfg
}

// privateSkillsRoot resolves .agent/project.yaml's optional
// private_skills_path relative to dir, or "" if unset/unreadable — "" means
// skip the private tier entirely (Phase 6 spec.md Decision 8).
func privateSkillsRoot(dir string) string {
	cfg, err := project.Load(dir)
	if err != nil || cfg.PrivateSkillsPath == "" {
		return ""
	}
	if filepath.IsAbs(cfg.PrivateSkillsPath) {
		return cfg.PrivateSkillsPath
	}
	return filepath.Join(dir, cfg.PrivateSkillsPath)
}

// selectSkills is the pure core behind `eng context skills`, reused by
// buildContextBundle so the manifest can record exactly what was chosen.
// It is the one authoritative skill-selection path (Phase 6 Requirement
// 19) — all routing (dependency expansion, domain-profile fills,
// recommends, budget) happens inside skillrouter.Route.
func selectSkills(dir, request string, cfg contextcfg.Config) (skillrouter.Selection, int, error) {
	all, err := skills.ResolveWithPrivate(filepath.Join(harnessDir(), "skills"), privateSkillsRoot(dir), filepath.Join(dir, "skills"))
	if err != nil {
		return skillrouter.Selection{}, 0, err
	}
	var mustInclude, domains []string
	if pcfg, err := project.Load(dir); err == nil {
		mustInclude = pcfg.EnabledSkills
		domains = pcfg.Domains
	}
	maxSkills := cfg.MaxSkills
	if cfg.Strategy == "full" {
		maxSkills = 0
	}
	sel, err := skillrouter.Route(all, request, mustInclude, domains, maxSkills)
	if err != nil {
		return skillrouter.Selection{}, 0, err
	}
	return sel, len(all), nil
}

func writeSkillSelection(w io.Writer, sel skillrouter.Selection, total int, cfg contextcfg.Config) {
	fmt.Fprintf(w, "Selected %d/%d skills (strategy: %s, max_skills: %d)\n\n", len(sel.Skills), total, cfg.Strategy, cfg.MaxSkills)
	for i, s := range sel.Skills {
		reason := ""
		if i < len(sel.Explanations) {
			reason = sel.Explanations[i].Reason
		}
		fmt.Fprintf(w, "- %-30s [%s] %s\n    selected because %s\n", s.Name, s.Domain, s.Description, reason)
	}
	if len(sel.Skills) < total {
		fmt.Fprintf(w, "\n%d skill(s) omitted as not relevant to this request.\n", total-len(sel.Skills))
	}
}

func contextSkills(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: eng context skills "<request text>"`)
		os.Exit(1)
	}
	request := strings.Join(args, " ")
	dir, _ := os.Getwd()
	cfg := loadContextConfig(dir)
	sel, total, err := selectSkills(dir, request, cfg)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	writeSkillSelection(os.Stdout, sel, total, cfg)
}

// selectProjectContext is the pure core behind `eng context project`.
func selectProjectContext(dir, request string, cfg contextcfg.Config) map[string][]docsearch.Section {
	maxDocs := cfg.MaxDocs
	if cfg.Strategy == "full" {
		maxDocs = 0
	}
	result := map[string][]docsearch.Section{}
	for _, name := range []string{"docs/src-map.md", "docs/gotchas.md"} {
		sections, err := docsearch.ParseSections(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		result[name] = docsearch.Match(sections, request, maxDocs)
	}
	return result
}

func allSectionsEmpty(byFile map[string][]docsearch.Section) bool {
	for _, sections := range byFile {
		if len(sections) > 0 {
			return false
		}
	}
	return true
}

func contextProject(args []string) {
	if len(args) == 0 {
		fmt.Println(`Usage: eng context project "<request text>"`)
		os.Exit(1)
	}
	request := strings.Join(args, " ")
	dir, _ := os.Getwd()
	cfg := loadContextConfig(dir)

	byFile := selectProjectContext(dir, request, cfg)
	for _, name := range []string{"docs/src-map.md", "docs/gotchas.md"} {
		matched, ok := byFile[name]
		if !ok {
			fmt.Printf("(%s not found or unreadable — skipping)\n", name)
			continue
		}
		fmt.Printf("## From %s (%d matched)\n\n", name, len(matched))
		for _, s := range matched {
			fmt.Printf("### %s\n%s\n", s.Title, s.Body)
		}
	}
}

func contextTask(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng context task <plan-dir>")
		os.Exit(1)
	}
	planDir, _ := filepath.Abs(args[0])

	goal, _ := taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
	task, err := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("## Goal summary")
	fmt.Println(goal)
	fmt.Println("\n## Current task")
	if task == "" {
		fmt.Println("(no unchecked task found — all tasks may be complete)")
	} else {
		fmt.Println(task)
	}
}

// buildContextBundle is the pure core behind both `eng context bundle` and
// `eng adapter prompt` (Phase 5 Decision 2) — it returns the composed
// context text and writes context-manifest.yaml as a side effect.
func buildContextBundle(role, planDir, request string) (string, error) {
	meta, err := planmeta.Load(planDir)
	if err != nil {
		return "", fmt.Errorf("no %s found in %s", planmeta.FileName, planDir)
	}
	if request == "" {
		request, _ = taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
	}

	repoRoot, _ := os.Getwd()
	cfg := loadContextConfig(repoRoot)

	var manifest strings.Builder
	var out strings.Builder
	fmt.Fprintf(&manifest, "role: %s\nplan: %s\ngenerated_at: %s\nrequest: %q\n", role, meta.Plan, time.Now().UTC().Format(time.RFC3339), request)
	fmt.Fprintf(&out, "# Context bundle for role: %s\n\n", role)

	switch role {
	case "planner":
		byFile := selectProjectContext(repoRoot, request, cfg)
		sel, total, _ := selectSkills(repoRoot, request, cfg)
		if allSectionsEmpty(byFile) && len(sel.Skills) == 0 && cfg.Strategy != "full" {
			fbCfg := cfg
			fbCfg.Strategy = "full"
			byFile = selectProjectContext(repoRoot, request, fbCfg)
			sel, total, _ = selectSkills(repoRoot, request, fbCfg)
			fmt.Fprintf(&manifest, "fallback_to_full: true\n")
			fmt.Fprintf(&out, "(no matches under 'selective' strategy — fell back to 'full' for this call)\n\n")
		}
		fmt.Fprintf(&manifest, "project_sections:\n")
		for name, sections := range byFile {
			fmt.Fprintf(&out, "## From %s\n\n", name)
			for _, s := range sections {
				fmt.Fprintf(&out, "### %s\n%s\n", s.Title, s.Body)
				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
			}
		}
		fmt.Fprintf(&out, "## Skills\n")
		writeSkillSelection(&out, sel, total, cfg)
		fmt.Fprintf(&manifest, "skills:\n")
		for i, s := range sel.Skills {
			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
		}

	case "plan-reviewer":
		fmt.Fprintf(&out, "## Plan\nrisk_level: %s\nrequires_approval: %v\n\n", meta.RiskLevel, meta.RequiresApproval)
		fmt.Fprintf(&manifest, "risk_level: %s\nrequires_approval: %v\nproject_sections:\n", meta.RiskLevel, meta.RequiresApproval)
		byFile := selectProjectContext(repoRoot, request, cfg)
		for name, sections := range byFile {
			fmt.Fprintf(&out, "## From %s\n\n", name)
			for _, s := range sections {
				fmt.Fprintf(&out, "### %s\n%s\n", s.Title, s.Body)
				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
			}
		}

	case "executor":
		goal, _ := taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
		task, _ := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
		fmt.Fprintf(&out, "## Goal summary\n%s\n\n## Current task\n", goal)
		if task == "" {
			fmt.Fprintf(&out, "(no unchecked task found — all tasks may be complete)\n")
		} else {
			fmt.Fprintf(&out, "%s\n", task)
		}
		fmt.Fprintf(&manifest, "current_task_present: %v\n", task != "")
		sel, total, _ := selectSkills(repoRoot, request, cfg)
		fmt.Fprintf(&out, "\n## Skills\n")
		writeSkillSelection(&out, sel, total, cfg)
		fmt.Fprintf(&manifest, "skills:\n")
		for i, s := range sel.Skills {
			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
		}

	case "verifier":
		fmt.Fprintf(&out, "## Verification rules\nwrite_scope: %v\n", meta.WriteScope)
		fmt.Fprintf(&manifest, "write_scope: %v\n", meta.WriteScope)

	default:
		return "", fmt.Errorf("unknown role: %s", role)
	}

	manifestPath := filepath.Join(planDir, "context-manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o644); err != nil {
		return out.String(), fmt.Errorf("context bundle built but failed to write manifest: %w", err)
	}
	fmt.Fprintf(&out, "\n(context selection recorded in %s)\n", manifestPath)
	return out.String(), nil
}

func contextBundle(args []string) {
	if len(args) < 2 {
		fmt.Println(`Usage: eng context bundle <planner|plan-reviewer|executor|verifier> <plan-dir> ["<request text>"]`)
		os.Exit(1)
	}
	role := args[0]
	planDir, _ := filepath.Abs(args[1])
	request := ""
	if len(args) > 2 {
		request = strings.Join(args[2:], " ")
	}
	out, err := buildContextBundle(role, planDir, request)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func contextManifest(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng context manifest <plan-dir>")
		os.Exit(1)
	}
	planDir, _ := filepath.Abs(args[0])
	data, err := os.ReadFile(filepath.Join(planDir, "context-manifest.yaml"))
	if err != nil {
		fmt.Println("no context-manifest.yaml found — run `eng context bundle <role> <plan-dir>` first")
		os.Exit(1)
	}
	fmt.Print(string(data))
}
