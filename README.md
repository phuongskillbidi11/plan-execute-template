# ai-planner-executor

> **Claude plans. Any AI codes. You relay.**  
> A structured AI development workflow that separates thinking from implementation.

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Works with Claude Code](https://img.shields.io/badge/Claude-Code-orange)
![Works with GitHub Copilot](https://img.shields.io/badge/GitHub-Copilot-green)
![Works with OpenAI Codex](https://img.shields.io/badge/OpenAI-Codex-412991)
![Language agnostic](https://img.shields.io/badge/Language-Agnostic-lightgrey)

---

## Overview

Most AI-assisted development fails not because the AI can't code, but because it starts coding before the problem is understood. This template enforces a clean separation: **Claude thinks, the Executor implements.**

**Claude (Planner)** reads the codebase, asks clarifying questions, and produces a three-file plan: `spec.md` (what and why), `tasks.md` (exact ordered checklist with file paths and line numbers), and `tests.md` (unambiguous pass/fail criteria). Claude never touches source code.

**The Executor** reads the plan, implements one task at a time, runs the specified test after each task, marks it complete, and stops the moment a test fails — reporting the exact error back to you. You relay that error to Claude, Claude revises only the failing task, and the loop continues.

**Supported executors — pick any one:**

| Executor | How it receives instructions |
|---|---|
| GitHub Copilot (VS Code Agent mode) | `.github/copilot-instructions.md` |
| OpenAI Codex CLI | `AGENTS.md` (repo root) |
| Claude Code (executor mode) | Direct prompt — paste plan contents |

This workflow is built on **Andrej Karpathy's engineering principles**: think before coding, prefer simplicity, make surgical changes, and define done by user-visible outcomes — not by whether the build passes.

---

## Prerequisites

| Tool | Required | Notes |
|---|---|---|
| [Claude Code](https://claude.ai/code) | Yes | CLI or desktop app; free tier works |
| [GitHub Copilot](https://github.com/features/copilot) | One of these | Requires subscription; VS Code extension + Agent mode |
| [OpenAI Codex CLI](https://github.com/openai/codex) | One of these | `npm install -g @openai/codex`; reads `AGENTS.md` automatically |
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

1. **Detects your project type** (ESP-IDF, Rust, Node.js, Python, Go, C++, C#/.NET, Make)
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
  [OK]  Created AGENTS.md

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

Open `CLAUDE.md` and complete the sections the script could not auto-fill:

- **Architecture** — describe your folder structure and data flow
- **Key files** — list the 5–10 most important files with their roles
- **Coding conventions** — async/await preference, naming rules, etc.
- **Current state** — update after each sprint; Claude reads this before planning

This is the most important step. The more context you give Claude here, the more accurate its plans will be.

### Step 4 — Enable your executor

**Using GitHub Copilot:**

Add this to `.vscode/settings.json` (create the file if it doesn't exist):

```json
{
  "github.copilot.chat.useInstructionFiles": true
}
```

> **WSL users:** Always open VS Code from the WSL terminal (`code .`), not from Windows Explorer. This prevents UNC path errors that stop Copilot from reading workspace files.

**Using OpenAI Codex CLI:**

No extra configuration needed — Codex reads `AGENTS.md` from the repo root automatically on every session.

```bash
# Verify Codex sees the instructions
codex "What is your role in this project?"
```

### Step 5 — Verify the setup

**Verify Claude knows its role:**

Open Claude Code and type:
```
What is your role in this project?
```
Expected: Claude describes itself as the Planner and explains it will never write code directly.

**Verify Copilot knows its role:**

Open VS Code Copilot Chat → Agent mode → type:
```
What are my project instructions?
```
Expected: Copilot describes the Executor role — read the plan, implement tasks in order, stop on failure.

**Verify Codex knows its role:**

```bash
codex "What are your instructions for this repo?"
```
Expected: Codex describes the role-switching table from `AGENTS.md` and the task loop.

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

### 2 — Ask the Executor to implement the plan

**GitHub Copilot (VS Code → Copilot Chat → Agent mode):**

```
Read the plan in .plans/2025-05-01-jwt-auth/
Start with spec.md, then tasks.md, then tests.md.
Implement each task in order.
After each task, run the test command from tests.md.
Mark [x] in tasks.md when the test passes.
Stop and report the exact error if a test fails.
```

**OpenAI Codex CLI:**

```bash
codex "Implement the plan in .plans/2025-05-01-jwt-auth/ — read spec.md, tasks.md, tests.md in that order, implement one task at a time, run each test, mark [x] on pass, stop and report on failure."
```

**Claude Code (executor mode):**

```
You are the Executor. Read .plans/2025-05-01-jwt-auth/spec.md,
tasks.md, and tests.md. Implement each task in order. Do not plan —
only implement what tasks.md says. Stop on the first failing test.
```

The Executor works through every `[ ]` checkbox, runs the verification command after each one, and marks it `[x]` before moving on.

### Task status markers

All executors use the same four markers in `tasks.md`:

| Marker | Meaning |
|---|---|
| `[ ]` | Not started |
| `[~]` | In progress (executor is currently working on this task) |
| `[x]` | Complete — verification command passed |
| `[!]` | Failed — executor stopped; relay the error to Claude |

### 3 — Relay failures back to Claude

When a test fails, the executor marks the task `[!]` and writes to `.plans/[folder]/errors.log`. Use the relay template:

```bash
cat .claude/relay-prompt-template.md
```

Fill in the template and paste it into Claude Code. Claude revises only the failing task. Hand the updated `tasks.md` back to the executor:

**Copilot:**
```
Resume from Task 2.1 with the updated tasks.md.
```

**Codex:**
```bash
codex "Resume implementing .plans/2025-05-01-jwt-auth/ from task 2.1 — tasks.md has been updated."
```

### 4 — Repeat until done

The loop is: Plan → Implement → Test → (fail → Revise → Resume) → Done.

When all checkboxes in `tasks.md` are `[x]`, the executor reports:

```
PLAN COMPLETE: jwt-auth
Tasks completed: 8
Files changed: src/middleware/auth.js, src/routes/api.js, tests/auth.test.js
All tests passed. Ready for review.
```

---

## Failure relay workflow

When a task is marked `[!]`, follow this structured relay:

1. **Find the error** — executor writes full output to `.plans/[folder]/errors.log`
2. **Open the relay template** — `cat .claude/relay-prompt-template.md`
3. **Fill in the blanks** — plan folder, failed task, exact error, what was attempted
4. **Paste into Claude Code** — Claude rewrites only the failing task
5. **Resume the executor** — give it the updated `tasks.md`

The relay template is executor-agnostic — it works for Copilot, Codex, and Claude Code executor mode. Replace "Copilot" in the template with whichever executor you're using.

---

## Skill system

Skills are short knowledge files that give Claude targeted context about one area of your codebase — without loading the entire project into the prompt.

```
skills/
  manifest.json            ← auto-generated index
  example/
    SKILL.md               ← blank template
  karpathy-guidelines/
    SKILL.md               ← universal: Karpathy principles applied to planning
  planner-executor-setup/
    SKILL.md               ← documents this workflow itself
  [your-skill]/
    SKILL.md               ← your project-specific knowledge
```

**How it works:** Before writing a plan, Claude reads `skills/manifest.json`. If your request matches a skill (e.g., you mention "web UI" and there's a `web-ui` skill), Claude loads only that skill. This keeps prompts small and plans accurate.

**Progressive disclosure advantage:** Instead of pasting 500 lines of context into every conversation, you write it once in a skill and Claude fetches it on demand.

**Skill metadata:** Each `SKILL.md` includes a header with `last_updated`, `confidence`, and `status` so Claude knows how fresh the knowledge is.

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

## Plan quality and self-evaluation

Every plan Claude writes is scored against a 10-point rubric before being handed to the Executor. The rubric lives in `.plans/_template/spec.md` and checks for:

- Clear goal statement (1 sentence, user-visible outcome)
- "Out of scope" section with at least 3 explicit exclusions
- All tasks name exact file + symbol + line range
- Every task group has a corresponding test
- No premature abstractions (helpers with only one caller)
- `tests.md` has binary pass/fail for every test (no judgment calls)

**Planning principles** (from `.claude/principles/`):

| File | Content |
|---|---|
| `karpathy.md` | Four core principles with applied checklists |
| `plan-quality.md` | 10-point self-evaluation rubric |
| `thinking-checklist.md` | Pre-planning checklist Claude runs before writing `spec.md` |

---

## Helper scripts reference

```bash
# Scaffold a new plan folder with empty spec/tasks/tests
./scripts/plan-executor.sh new <feature-name>

# List all plans and their task completion percentage
./scripts/plan-executor.sh list

# Show task checkboxes for a specific plan
./scripts/plan-executor.sh status .plans/2025-05-01-jwt-auth

# Print all three plan files to stdout (paste into any executor)
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
| C# / .NET | `*.csproj`, `*.sln` | `dotnet test` |
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

**C# / .NET:** Detection looks for `.csproj` or `.sln`. Default test command is `dotnet test`. Set `build_cmd: "dotnet build"`.

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

### Codex does not see AGENTS.md instructions

**Symptom:** Codex behaves generically and doesn't mention the Executor role or task markers.

**Fix:** Confirm `AGENTS.md` is at the repo root (not in a subdirectory) and that Codex is run from within the repo:
```bash
ls AGENTS.md           # must exist at root
codex "What is your role in this project?"
```

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

### Executor marks tasks `[x]` without running tests

**Symptom:** Tasks are marked complete but tests were not actually run.

**Fix:** Add this line to your executor prompt:
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
├── AGENTS.md                              ← OpenAI Codex CLI instructions
├── ROADMAP.md                             ← planned improvements
├── README.md                              ← this file
│
├── .claude/
│   ├── planner-instructions.md           ← Claude's full Planner role definition
│   ├── relay-prompt-template.md          ← structured template for relaying failures
│   └── principles/
│       ├── karpathy.md                   ← 4 Karpathy principles with checklists
│       ├── plan-quality.md               ← 10-point plan self-evaluation rubric
│       └── thinking-checklist.md         ← pre-planning checklist
│
├── .github/
│   └── copilot-instructions.md           ← GitHub Copilot Executor role definition
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
│   ├── karpathy-guidelines/
│   │   └── SKILL.md                      ← universal Karpathy principles skill
│   └── planner-executor-setup/
│       └── SKILL.md                      ← documents this workflow itself
│
└── .plans/
    ├── _template/                        ← copy this when starting a new plan
    │   ├── spec.md                       ← spec template with self-eval rubric
    │   ├── tasks.md                      ← tasks template with 4-status markers
    │   ├── tests.md                      ← tests template with pass/fail format
    │   ├── DECISION_LOG.md               ← record design decisions and rejections
    │   ├── sprint-summary.md             ← end-of-sprint retrospective template
    │   └── errors.log                    ← structured failure log for relay workflow
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
- Changing `CLAUDE.md` or executor instruction files without a plan

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

Built with [Claude Code](https://claude.ai/code), [GitHub Copilot](https://github.com/features/copilot), and [OpenAI Codex](https://github.com/openai/codex).
