package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		cmdInstall(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "scan":
		cmdScan(os.Args[2:])
	case "skills":
		cmdSkills(os.Args[2:])
	case "plan":
		cmdPlan(os.Args[2:])
	case "verify":
		cmdVerify(os.Args[2:])
	case "hooks":
		cmdHooks(os.Args[2:])
	case "triage":
		cmdTriage(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`Usage: eng <command> [args]

Commands:
  install --from <path>   Install the harness payload into ~/.engineering-harness
  init                    Initialize the current directory as a harness-aware project
  doctor                  Report harness install status, project mode, and resolved skills
  scan                    Print detected stack and a file summary
  skills list             List resolved skills (global + project-local)
  plan new <name> [--risk <level>]   Scaffold a plan and stamp it with the current git SHA
  plan drift [dir]                   Check whether relevant files changed since planning
  plan retry <dir> <stage>           Track a retry against this plan's budget
  verify [dir]                       Run tests, check the git diff, write verify-report.md
  hooks run <stage>                  Run the configured hooks for a lifecycle stage
  triage "<text>"                    Heuristic risk-level hint (not authoritative)`)
}
