# ai-planner-executor

> **Claude plans. Copilot codes. You relay.**  
> A structured AI development workflow that separates thinking from implementation.

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Works with Claude Code](https://img.shields.io/badge/Claude-Code-orange)
![Works with GitHub Copilot](https://img.shields.io/badge/GitHub-Copilot-green)
![Language agnostic](https://img.shields.io/badge/Language-Agnostic-lightgrey)

---

## Overview

Most AI-assisted development fails not because the AI can't code, but because it starts coding before the problem is understood. This template enforces a clean separation: **Claude thinks, Copilot implements.**

**Claude (Planner)** reads the codebase, asks clarifying questions, and produces a three-file plan: `spec.md` (what and why), `tasks.md` (exact ordered checklist with file paths and line numbers), and `tests.md` (unambiguous pass/fail criteria). Claude never touches source code.

**GitHub Copilot (Executor)** reads the plan, implements one task at a time, runs the specified test after each task, marks it complete, and stops the moment a test fails — reporting the exact error back to you. You relay that error to Claude, Claude revises only the failing task, and the loop continues.

This workflow is built on **Andrej Karpathy's engineering principles**: think before coding, prefer simplicity, make surgical changes, and define done by user-visible outcomes — not by whether the build passes.

---

## Prerequisites

| Tool | Required | Notes |
|---|---|---|
| [Claude Code](https://claude.ai/code) | Yes | CLI or desktop app; free tier works |
| [GitHub Copilot](https://github.com/features/copilot) | Yes | Requires subscription; VS Code extension + Agent mode |
| Python 3 | Recommended | Used by helper scripts for JSON parsing; falls back to grep |
| Bash 4+ | Yes | All scripts are bash; macOS users: `brew install bash` |
| ESP-IDF (≥ v5.5) | ESP-IDF projects only | Other project types auto-detected |

---

## Quick start

### Option A — GitHub template (recommended)

Click **"Use this template"** → **"Create a new repository"** at the top of this page.

Then clone your new repo and run the init script from inside your target project:

```bash
# Clone your new repo (the template)
git clone https://github.com/your-org/your-repo /tmp/pe-template

# Go to the project you want to add the workflow to
cd ~/projects/my-project

# Run init — detects project type and copies template files
bash /tmp/pe-template/scripts/init.sh
```

### Option B — Manual copy

```bash
git clone https://github.com/your-org/ai-planner-executor /tmp/pe-template
cd ~/projects/my-project
cp -r /tmp/pe-template/. .
chmod +x scripts/*.sh
```

---

## Step-by-step setup

### Step 1 — Get the template

```bash
git clone https://github.com/your-org/ai-planner-executor /tmp/pe-template
```

### Step 2 — Run `init.sh` in your project root

```bash
cd ~/projects/my-project
bash /tmp/pe-template/scripts/init.sh
```

The script does four things automatically:

1. **Detects your project type** (ESP-IDF, Rust, Node.js, Python, Go, C++, Make)
2. **Copies template files** — skips any that already exist (safe to re-run)
3. **Writes `.planner-executor/config.yaml`** with detected build/test commands
4. **Fills placeholders** in `CLAUDE.md` with your project name and detected commands

Sample output:

```
Planner-Executor init
Target: /home/user/projects/my-project

→ Refreshing scripts...
  [OK]  Updated scripts/plan-executor.sh
  [OK]  Updated scripts/detect-project.sh

→ Copying template files (skip if already present)...
  [OK]  Created CLAUDE.md
  [OK]  Created .github/copilot-instructions.md

→ Detecting project type...
   Detected: nodejs
   Build:    npm run build
   Test:     npm test
   Lint:     npm run lint

  [OK]  Created .planner-executor/config.yaml

Done. Next steps:
  1. Review .planner-executor/config.yaml
  2. Fill in CLAUDE.md: architecture, key files, conventions
```

### Step 3 — Fill in `CLAUDE.md`

Open `CLAUDE.md` and complete the sections that the script could not auto-fill:

- **Architecture** — describe your folder structure and data flow
- **Key files** — list the 5–10 most important files with their roles
- **Coding conventions** — async/await preference, naming rules, etc.

This is the most important step. The more context you give Claude here, the more accurate its plans will be.

### Step 4 — Enable Copilot instruction files in VS Code

Add this to `.vscode/settings.json` (create the file if it doesn't exist):

```json
{
  "github.copilot.chat.useInstructionFiles": true
}
```

> **WSL users:** Always open VS Code from the WSL terminal (`code .`), not from Windows Explorer. This prevents UNC path errors that stop Copilot from reading workspace files.

### Step 5 — Verify the setup

**Verify Claude knows its role:**

Open Claude Code and type:
```
What is your role in this project?
```
Expected: Claude describes itself as the Planner and explains it will never write code directly.

**Verify Copilot knows its role:**

Open VS Code Copilot Chat and type:
```
What are my project instructions?
```
Expected: Copilot describes the Executor role — read the plan, implement tasks in order, stop on failure.

**Run the smoke tests:**

```bash
# Confirm skills load
./scripts/load_skill.sh list

# Scaffold a test plan
./scripts/plan-executor.sh new smoke-test

# Confirm the config test command runs
./scripts/plan-executor.sh test
```

### Step 6 — Review and adjust `config.yaml`

```bash
cat .planner-executor/config.yaml
```

Correct any commands that were wrongly detected:

```yaml
project_name: "my-project"
project_type: "nodejs"
build_cmd:    "npm run build"
test_cmd:     "npm test"
lint_cmd:     "npm run lint"
run_cmd:      "npm start"
env_setup:    ""          # e.g., ". ~/esp/esp-idf/export.sh" for ESP-IDF
```

---

## How to use the workflow

### 1 — Ask Claude to create a plan

In Claude Code (terminal or desktop app):

```
Claude, create a plan for adding JWT authentication to the /api/login endpoint.
Save it to .plans/2025-05-01-jwt-auth/
Include spec.md, tasks.md, and tests.md.
Follow .claude/planner-instructions.md.
```

Claude will:
- Check `skills/manifest.json` for a relevant skill
- Read the codebase (not write to it)
- Write `spec.md` first and ask you to confirm the design
- Then write `tasks.md` with exact file + line references
- Then write `tests.md` with pass/fail conditions

### 2 — Ask Copilot to implement the plan

Switch to VS Code → open Copilot Chat → select **Agent** mode in the dropdown:

```
Read the plan in .plans/2025-05-01-jwt-auth/
Start with spec.md, then tasks.md, then tests.md.
Implement each task in order.
After each task, run the test command from tests.md.
Mark [x] in tasks.md when the test passes.
Stop and report the exact error if a test fails.
```

Copilot will work through every `[ ]` checkbox, running the verification command after each one and marking it `[x]` before moving on.

### 3 — Relay failures back to Claude

If Copilot stops on a failing test:

```
Claude, Copilot failed on Task 2.1 of .plans/2025-05-01-jwt-auth/

Error:
  TypeError: Cannot read properties of undefined (reading 'userId')
  at authMiddleware (src/middleware/auth.js:14:22)

Context: Copilot added the middleware but the request object doesn't
have the decoded token attached yet.

Please update tasks.md with a corrected approach for Task 2.1 only.
```

Claude revises only that task. Hand the updated `tasks.md` back to Copilot:

```
Resume from Task 2.1 with the updated tasks.md.
```

### 4 — Repeat until done

The loop is: Plan → Implement → Test → (fail → Revise → Resume) → Done.

When all checkboxes in `tasks.md` are `[x]`, Copilot reports:

```
PLAN COMPLETE: jwt-auth
Tasks completed: 8
Files changed: src/middleware/auth.js, src/routes/api.js, tests/auth.test.js
All tests passed. Ready for review.
```

---

## Skill system

Skills are short knowledge files that give Claude targeted context about one area of your codebase — without loading the entire project into the prompt.

```
skills/
  manifest.json            ← auto-generated index
  example/
    SKILL.md               ← blank template
  planner-executor-setup/
    SKILL.md               ← documents this workflow itself
  [your-skill]/
    SKILL.md               ← your project-specific knowledge
```

**How it works:** Before writing a plan, Claude reads `skills/manifest.json`. If your request matches a skill (e.g., you mention "web UI" and there's a `web-ui` skill), Claude loads only that skill. This keeps prompts small and plans accurate.

**Progressive disclosure advantage:** Instead of pasting 500 lines of context into every conversation, you write it once in a skill and Claude fetches it on demand.

### Adding a custom skill

```bash
# 1. Copy the template
cp -r skills/example/ skills/api/

# 2. Edit skills/api/SKILL.md
#    Fill in: Purpose, When to use, Files involved,
#             Constraints, Patterns, Step-by-step, Tests

# 3. Register it (regenerates manifest.json)
./scripts/update-manifest.sh
```

**Good skill candidates:**

| Area | Suggested name |
|---|---|
| REST API / HTTP handlers | `api` |
| Web UI / frontend | `web-ui` |
| Database / migrations | `database` |
| Authentication | `auth` |
| Build and flash (embedded) | `build-flash` |
| Camera / sensor driver | `camera` |
| CI/CD pipeline | `ci` |

### CLI reference

```bash
# List all skills with descriptions
./scripts/load_skill.sh list

# Print a skill's full content (paste into any AI chat)
./scripts/load_skill.sh api

# Regenerate manifest after adding/editing skills
./scripts/update-manifest.sh
```

---

## Helper scripts reference

```bash
# Scaffold a new plan folder with empty spec/tasks/tests
./scripts/plan-executor.sh new <feature-name>

# List all plans and their task completion percentage
./scripts/plan-executor.sh list

# Show task checkboxes for a specific plan
./scripts/plan-executor.sh status .plans/2025-05-01-jwt-auth

# Print all three plan files to stdout (paste into Copilot)
./scripts/plan-executor.sh open .plans/2025-05-01-jwt-auth

# Run build + test commands from config.yaml
./scripts/plan-executor.sh test

# Re-run project detection and print suggested commands
./scripts/detect-project.sh

# Re-initialize or upgrade scripts (preserves CLAUDE.md and config)
bash scripts/init.sh --upgrade
```

---

## Customization

### Supported project types (auto-detected)

| Project type | Detected by | Default test command |
|---|---|---|
| ESP-IDF | `sdkconfig`, `idf_component.yml` | `idf.py build 2>&1 \| tail -5` |
| Rust | `Cargo.toml` | `cargo test` |
| Node.js / TypeScript | `package.json` | `npm test` (or pnpm/yarn) |
| Python | `pyproject.toml`, `requirements.txt` | `pytest -x` |
| Go | `go.mod` | `go test ./...` |
| C/C++ (CMake) | `CMakeLists.txt` | `ctest --test-dir build` |
| Generic | `Makefile` | `make test` |

### Manual override

Edit `.planner-executor/config.yaml` directly at any time:

```yaml
build_cmd: "make -j4 && make install"
test_cmd:  "make check"
env_setup: "export CC=clang && export CXX=clang++"
```

### Language-specific notes

**ESP-IDF:** Set `env_setup: ". ~/esp/esp-idf/export.sh"` in config so every build/test command sources the environment automatically.

**Python (monorepo):** Set `build_cmd: "pip install -e packages/my-lib"` and `test_cmd: "pytest packages/my-lib/tests"`.

**Docker-based tests:** Set `test_cmd: "docker compose run --rm test"`.

---

## Troubleshooting

### Copilot cannot read `.plans/` files (WSL)

**Symptom:** Copilot says it cannot find files or returns UNC path errors (`\\wsl.localhost\...`).

**Fix:** Open VS Code from inside the WSL terminal, not from Windows Explorer:
```bash
cd ~/projects/my-project && code .
```
The bottom-left status bar should show `WSL: Ubuntu-22.04` in green.

---

### Claude skips the plan and writes code directly

**Fix:** Remind Claude of its role:
```
You are the Planner. Do not write code. Read .claude/planner-instructions.md
and create a plan in .plans/ first.
```
If this keeps happening, ensure `CLAUDE.md` is in the project root (Claude Code loads it automatically).

---

### Copilot is not in Agent mode

**Symptom:** Copilot responds conversationally instead of reading files and running commands.

**Fix:** In the Copilot Chat panel, click the mode dropdown (shows "Ask" or "Chat") and switch to **Agent**.

---

### `./scripts/load_skill.sh list` shows nothing

**Symptom:** "No manifest found" or empty output.

**Fix:**
```bash
# Regenerate the manifest from skill files
./scripts/update-manifest.sh

# Confirm skills/ folder exists
ls skills/
```

---

### Copilot marks tasks `[x]` without running tests

**Symptom:** Tasks are marked complete but tests were not actually run.

**Fix:** Add this line to your Copilot prompt:
```
Do NOT mark any task [x] until you have run the exact test command
from tests.md and confirmed it passes. Show me the command output.
```

---

### Detected commands are wrong

**Fix:** Run detection and review the output:
```bash
./scripts/detect-project.sh
```
Then edit `.planner-executor/config.yaml` to correct any wrong values.

---

## Repository structure

```
ai-planner-executor/
├── CLAUDE.md                              ← edit this for your project
├── ROADMAP.md                             ← planned improvements
├── README.md                              ← this file
│
├── .claude/
│   └── planner-instructions.md           ← Claude's full role definition
│
├── .github/
│   └── copilot-instructions.md           ← Copilot's full role definition
│
├── .planner-executor/
│   └── config.yaml.template              ← copy → config.yaml, fill in values
│
├── scripts/
│   ├── init.sh                           ← one-command setup
│   ├── detect-project.sh                 ← auto-detect build/test commands
│   ├── plan-executor.sh                  ← manage plans (new/list/status/open/test)
│   ├── load_skill.sh                     ← list or print skills
│   └── update-manifest.sh               ← regenerate skills/manifest.json
│
├── skills/
│   ├── manifest.json                     ← auto-generated; do not edit by hand
│   ├── example/
│   │   └── SKILL.md                      ← blank template for new skills
│   └── planner-executor-setup/
│       └── SKILL.md                      ← documents this workflow itself
│
└── .plans/
    └── demo-feature/                     ← example plan (dark mode toggle)
        ├── spec.md
        ├── tasks.md
        └── tests.md
```

---

## Contributing

Contributions follow the same workflow this template enforces:

1. Open an issue describing the problem or improvement.
2. A plan will be created in `.plans/` following the Karpathy principles.
3. Implementation follows the plan — no unplanned changes.

**What makes a good contribution:**
- A new project type in `detect-project.sh` (add detection + sensible defaults)
- A new example skill for a common domain (`database`, `auth`, `ci`)
- A new example plan in `examples/` for a real project type
- A fix for a documented troubleshooting issue

**What to avoid:**
- Refactoring scripts without a clear problem statement
- Adding dependencies beyond `bash`, `python3`, and standard Unix tools
- Changing `CLAUDE.md` or `copilot-instructions.md` without a plan

---

## License

MIT — see [LICENSE](LICENSE).

You are free to use, copy, modify, and distribute this template in any project,
commercial or otherwise. No attribution required (but appreciated).

---

## Acknowledgements

Workflow philosophy based on [Andrej Karpathy's](https://karpathy.ai) engineering
principles: think before coding, prefer simplicity, make surgical changes, define
done by user-visible outcomes.

Built with [Claude Code](https://claude.ai/code) and [GitHub Copilot](https://github.com/features/copilot).
