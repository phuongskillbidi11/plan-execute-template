package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eng/internal/contextcfg"
	"eng/internal/docsearch"
	"eng/internal/planmeta"
	"eng/internal/project"
	"eng/internal/skillmatch"
	"eng/internal/skills"
	"eng/internal/taskscope"
)

func cmdContext(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: eng context <skills|project|task|bundle> ...")
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
	default:
		fmt.Println("Usage: eng context <skills|project|task|bundle> ...")
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

// selectSkills is the pure core behind `eng context skills`, reused by
// `eng context bundle` so the manifest can record exactly what was chosen.
func selectSkills(dir, request string, cfg contextcfg.Config) (selected []skills.Skill, total int, err error) {
	all, err := skills.Resolve(filepath.Join(harnessDir(), "skills"), filepath.Join(dir, "skills"))
	if err != nil {
		return nil, 0, err
	}
	var mustInclude []string
	if pcfg, err := project.Load(dir); err == nil {
		mustInclude = pcfg.EnabledSkills
	}
	maxSkills := cfg.MaxSkills
	if cfg.Strategy == "full" {
		maxSkills = 0
	}
	return skillmatch.Select(all, request, mustInclude, maxSkills), len(all), nil
}

func printSkillSelection(selected []skills.Skill, total int, cfg contextcfg.Config) {
	fmt.Printf("Selected %d/%d skills (strategy: %s, max_skills: %d)\n\n", len(selected), total, cfg.Strategy, cfg.MaxSkills)
	for _, s := range selected {
		fmt.Printf("- %-30s [%s] %s\n", s.Name, s.Domain, s.Description)
	}
	if len(selected) < total {
		fmt.Printf("\n%d skill(s) omitted as not relevant to this request.\n", total-len(selected))
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
	selected, total, err := selectSkills(dir, request, cfg)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	printSkillSelection(selected, total, cfg)
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

	meta, err := planmeta.Load(planDir)
	if err != nil {
		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
		os.Exit(1)
	}
	if request == "" {
		request, _ = taskscope.GoalSummary(filepath.Join(planDir, "spec.md"))
	}

	repoRoot, _ := os.Getwd()
	cfg := loadContextConfig(repoRoot)

	var manifest strings.Builder
	fmt.Fprintf(&manifest, "role: %s\nplan: %s\ngenerated_at: %s\nrequest: %q\n", role, meta.Plan, time.Now().UTC().Format(time.RFC3339), request)

	fmt.Printf("# Context bundle for role: %s\n\n", role)

	switch role {
	case "planner":
		byFile := selectProjectContext(repoRoot, request, cfg)
		fmt.Fprintf(&manifest, "project_sections:\n")
		for name, sections := range byFile {
			fmt.Printf("## From %s\n\n", name)
			for _, s := range sections {
				fmt.Printf("### %s\n%s\n", s.Title, s.Body)
				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
			}
		}
		selected, total, _ := selectSkills(repoRoot, request, cfg)
		fmt.Println("## Skills")
		printSkillSelection(selected, total, cfg)
		fmt.Fprintf(&manifest, "skills:\n")
		for _, s := range selected {
			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
		}

	case "plan-reviewer":
		fmt.Printf("## Plan\nrisk_level: %s\nrequires_approval: %v\n\n", meta.RiskLevel, meta.RequiresApproval)
		fmt.Fprintf(&manifest, "risk_level: %s\nrequires_approval: %v\nproject_sections:\n", meta.RiskLevel, meta.RequiresApproval)
		byFile := selectProjectContext(repoRoot, request, cfg)
		for name, sections := range byFile {
			fmt.Printf("## From %s\n\n", name)
			for _, s := range sections {
				fmt.Printf("### %s\n%s\n", s.Title, s.Body)
				fmt.Fprintf(&manifest, "  - %q: %q\n", name, s.Title)
			}
		}

	case "executor":
		fmt.Println("## Task scope")
		contextTask([]string{planDir})
		task, _ := taskscope.CurrentTask(filepath.Join(planDir, "tasks.md"))
		fmt.Fprintf(&manifest, "current_task_present: %v\n", task != "")
		selected, total, _ := selectSkills(repoRoot, request, cfg)
		fmt.Println("\n## Skills")
		printSkillSelection(selected, total, cfg)
		fmt.Fprintf(&manifest, "skills:\n")
		for _, s := range selected {
			fmt.Fprintf(&manifest, "  - %s\n", s.Name)
		}

	case "verifier":
		fmt.Printf("## Verification rules\nwrite_scope: %v\n", meta.WriteScope)
		fmt.Fprintf(&manifest, "write_scope: %v\n", meta.WriteScope)

	default:
		fmt.Println("Unknown role:", role)
		os.Exit(1)
	}

	manifestPath := filepath.Join(planDir, "context-manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest.String()), 0o644); err != nil {
		fmt.Println("warning: could not write context-manifest.yaml:", err)
		return
	}
	fmt.Printf("\n(context selection recorded in %s)\n", manifestPath)
}
