# Usage

Detailed commands and workflows for the Global Engineering Harness (`eng`). For the big
picture (why the harness is shaped this way, the request-flow diagram, context engineering,
security model) see [`docs/ARCHITECTURE.md`](ARCHITECTURE.md). For skill authoring see
[`docs/skills.md`](skills.md); for tool/capability adapters see [`docs/tools.md`](tools.md).

Every command below was verified against the current `cli/` source (`cli/main.go`'s `usage()`
plus each subcommand's own flag parsing) while writing this document. Commands that don't
exist in the code — `eng migrate`, `eng benchmark` — are **not** listed anywhere here; they are
not implemented.

---

## Installation

The harness is a single Go binary, `eng`, plus a payload directory (`harness/`) that `eng
install` copies to `~/.engineering-harness/`.

### Build the CLI

```bash
cd cli
go build -o eng .          # produces ./eng (./eng.exe on Windows)
```

Requires Go (any version the repo's `cli/go.mod` supports — no separate Go toolchain document
exists yet; run `go version` and `cd cli && go build ./...` to confirm your toolchain works).

### Install the harness payload

```bash
./eng install --from .                 # run from the repo root (or wherever you built it)
```

This copies `harness/` into `~/.engineering-harness/` and also copies the `eng` binary itself
into `~/.engineering-harness/bin/`. It prints the PATH line for your platform:

```bash
# Linux / macOS
export PATH="$HOME/.engineering-harness/bin:$PATH"     # add to ~/.bashrc or ~/.zshrc to persist

# Windows (cmd.exe / PowerShell)
setx PATH "%PATH%;C:\Users\<you>\.engineering-harness\bin"   # open a new terminal afterward
```

Or apply it automatically:

```bash
./eng install --from . --add-to-path
```

On Linux/macOS this appends the export line to `~/.bashrc` and `~/.zshrc` if present (skips
ones that already have it). On Windows it runs `setx PATH` for you, refusing if the existing
`PATH` is already near `setx`'s 1024-character limit (add it manually in that case). This
mechanism is the same on all three platforms — there is no platform-specific installer beyond
this.

### Re-installing / upgrading

`eng install --from <path>` is safe to re-run — it overwrites `~/.engineering-harness/` with
whatever `harness/` tree you point it at. There is no separate `eng update` command; re-run
`install` after pulling a newer checkout.

---

## First project setup

```bash
cd my-project
eng init
eng doctor
```

`eng init` writes exactly one file: `.agent/project.yaml`. It auto-detects your stack (Go,
Node, Python, Rust, etc. — the same detection V1's `detect-project.sh` used), auto-detects
whether the directory looks like a legacy V1 project (`CLAUDE.md` and/or `.plans/` present) and
sets `mode: hybrid` in that case rather than `mode: modern`, and appends `.agent/logs/` to
`.gitignore` (creating it if needed). It never touches an existing `CLAUDE.md`, `.plans/`, or
`skills/` — if `.agent/project.yaml` already exists, `eng init` does nothing and says so.

```text
my-project/
├── src/
├── tests/
├── .agent/
│   └── project.yaml       ← the only file eng init creates
└── .plans/                ← created the first time you scaffold a plan (eng plan new / eng workflow start)
```

The full skill library, workflow templates, and hooks are **not** copied into your project —
they're resolved from `~/.engineering-harness/` (plus optional project-local `skills/` and a
private tier — see [`docs/skills.md`](skills.md#skill-sources-and-precedence)) at run time.

`eng doctor` reports: harness install status + version, project mode (`legacy`/`hybrid`/
`modern`/`none`/`broken`), detected stack, a skill summary (discovered/valid/warnings/broken
dependencies), and a per-adapter/per-capability availability table. Run it any time something
seems off before digging into individual commands.

---

## Daily usage

```bash
cd my-project
eng start
```

`eng start` runs `eng doctor`, prints a pointer to the runtime routing method
(`~/.engineering-harness/core/runtime/METHOD.md`), and launches `claude` if it's on PATH (or
tells you it isn't). If the project isn't initialized yet, it says so and suggests `eng init`
or `eng start --init` (initialize and continue in one step).

**If a spawned agent session reports `eng: command not found`** (fixed in Phase 9 — see
`docs/gotchas.md`): `eng start` prepends a known-working `eng` location to the launched
session's own `PATH`, so this should no longer happen. It also sets `ENG_HOME` (the harness
install directory), `ENG_PROJECT_ROOT`, and `ENG_VERSION` as environment variables in that
session — if `eng` still somehow isn't resolvable inside a nested shell the session spawns
(e.g. its own shell profile scripts reset `PATH`), fall back to `"$ENG_HOME/bin/eng"`
(`%ENG_HOME%\bin\eng.exe` on Windows) as an absolute path.

Inside that session, **speak naturally** — don't type `eng` commands yourself:

```text
Add CSV export to locker history.
```

The session follows `harness/core/runtime/METHOD.md`'s documented sequence: `eng workflow
start "<your exact text>"` → read the reported risk level → Quick Fix or Spec-First path →
`eng workflow advance` at every state → stop at every approval gate and ask you explicitly.

**You should not normally need to type** `eng plan new`, `eng context bundle`, `eng workflow
advance`, etc. yourself — those are the commands *the session* runs on your behalf. They stay
directly usable for debugging, CI, and advanced manual workflows — see
[Normal vs. advanced usage](#normal-vs-advanced-usage) below.

---

## Quick Fix workflow

**When it applies:** a small, localized, low-risk change — Triage's keyword heuristic looks for
words like `typo`, `rename`, `comment`, `formatting`, `"small change"`.

```text
small localized low-risk request
  → risk level "quick-fix" (TRIAGED)
  → edit the one task block in tasks.md
  → eng verify
  → PASS → COMPLETED, a compact "quick_fix" event recorded automatically
```

A quick-fix plan skips `PLANNED`/`REVIEWED`/`APPROVED` entirely — `TRIAGED → EXECUTING`
directly, once its minimal `spec.md`/`tasks.md`/`tests.md` are present (the `templates/
quickfix/` template, not the full `templates/plan/` one).

**A one-line change can still be high-risk.** Triage is a heuristic hint, not an authority — if
you know a change touches something sensitive (a migration, a flash/firmware write, production
config), scaffold it explicitly at the right risk level instead of trusting the keyword match:

```bash
eng plan new my-change --risk high-risk
```

**Escalation — Quick Fix turns out to be bigger than it looked:**

```bash
eng plan escalate <plan-dir> --to bug|feature|architecture|high-risk --reason "..."
```

This resets the plan to `TRIAGED` at the new risk level. It only records the escalation — you
still need to flesh out `spec.md`/`tasks.md`/`tests.md` into the full format before continuing.

**Known limitation:** Phase 8's benchmark found the triage keyword list currently
misclassifies plausible real-world phrasing (e.g. "increase the reconnect timeout from 1000 ms
to 1500 ms" triages to `feature`, not `quick-fix`) — see
[Known Limitations](../README.md#known-limitations). The underlying Quick Fix mechanism itself
works correctly once a plan is explicitly scaffolded at `--risk quick-fix`; the gap is
specifically in natural-language classification.

---

## Spec-First Feature workflow

**The default `planning_mode` for any project initialized under `eng init`** (an older
project without this field falls back to `auto_plan` — the single-step `TRIAGED → PLANNED`
path, unchanged from earlier phases; check `eng workflow status <plan-dir>`'s `Profile:` line
if you're unsure which one you're on).

```text
Requirement
  → eng workflow start "<text>"           (TRIAGED)
  → write ONLY spec.md — not tasks.md/tests.md yet
  → eng workflow advance                  (NEEDS_SPEC_APPROVAL)
  → STOP — show the human spec.md's Goal, ask for approval or revision
  → eng plan approve-spec <plan-dir>      (a REQUIREMENTS approval)
  → eng workflow advance                  (SPEC_APPROVED)
  → now write tasks.md and tests.md
  → eng workflow advance                  (PLANNED)
  → review → (approval, if the risk level requires it) → execute → verify → COMPLETED
```

**Spec approval is distinct from execution/high-risk approval.** `eng plan approve-spec`
answers "are these the right requirements?" — it does not grant permission to execute
anything risky. A separate gate, `eng plan approve` (triggered by `--risk high-risk` or
`--requires-approval`), answers "is it safe to let the Executor run this plan?" They use
different `plan.yaml` fields (`spec_approved_at` vs. `approved_at`) and different workflow
states (`NEEDS_SPEC_APPROVAL`/`SPEC_APPROVED` vs. `NEEDS_APPROVAL`/`APPROVED`).

**Verified core claim (Phase 8):** `tasks.md`/`tests.md` are genuinely absent/placeholder in
content until `eng plan approve-spec` has run — confirmed directly against a real fixture, not
merely documented. See `benchmarks/results/feature-csv-export-harness-v2.yaml`.

---

## Workflow states

```text
NEW*  TRIAGED  NEEDS_SPEC_APPROVAL  SPEC_APPROVED  PLANNED  REVIEWED
NEEDS_APPROVAL  APPROVED  EXECUTING  VERIFYING  COMPLETED
NEEDS_REPLAN  NEEDS_FIX  FAILED  BLOCKED  CANCELLED
```
<sub>* `NEW` is declared in the state enum but no plan is ever created in it — `eng plan new`
starts a plan directly at `TRIAGED`.</sub>

Transitions are computed by a **pure function**, `cli/internal/workflow.Decide(Facts) →
Decision` — deterministic and machine-readable, not a judgment call. `eng workflow advance`
applies **at most one transition per call** and never invokes an agent unattended; every stage
ends with a printed next command and a stop.

| State | What's happening | Typical next command |
|---|---|---|
| `TRIAGED` | Waiting on the Planner to write `spec.md` (Spec-First) or all three files (auto_plan), or the quick-fix minimal plan | `eng adapter prompt planner <dir>` |
| `NEEDS_SPEC_APPROVAL` | `spec.md` is written; waiting on the human | `eng plan approve-spec <dir>` |
| `SPEC_APPROVED` | Spec approved; waiting on `tasks.md`/`tests.md` | write those files, then `eng workflow advance` |
| `PLANNED` | Full plan present; waiting on review (if this risk level requires it) | `eng adapter prompt plan-reviewer <dir>`, then `eng plan review <dir> --verdict ...` |
| `REVIEWED` | Review passed or not required | `eng workflow advance` (auto-resolves approval need) |
| `NEEDS_APPROVAL` | Execution requires human sign-off (`high-risk` or `--requires-approval`) | `eng plan approve <dir>` |
| `APPROVED` | Drift-checked and clear; Executor may begin | `eng workflow advance` (starts `EXECUTING`) |
| `EXECUTING` | Executor working through `tasks.md` | `eng adapter prompt executor <dir>` |
| `VERIFYING` | All tasks checked off; `eng verify` runs automatically on the transition into this state | (automatic) |
| `COMPLETED` | Verified PASS | none |
| `NEEDS_REPLAN` | Review REJECTed, or drift detected before execution started | `eng adapter prompt planner <dir>` |
| `NEEDS_FIX` | `eng verify` FAILed, retry budget remains | `eng adapter prompt executor <dir>` |
| `FAILED` | `eng verify` FAILed and the retry budget is exhausted | human decision required |
| `BLOCKED` / `CANCELLED` | Explicitly set via `eng plan block`/`eng plan cancel` | human decision required |

**Drift detection** fires specifically at the `APPROVED → EXECUTING` transition — it compares
files changed since the plan's stamped `git_sha` (scoped to `write_scope` if set) and forces
`NEEDS_REPLAN` instead of silently continuing on a stale plan. It is not a continuous background
check; once a plan is past that gate, run `eng plan drift <dir>` directly if you want to check
again.

**What actually gates the `EXECUTING → VERIFYING` transition:** only `tasks.md`'s bottom
**Completion checklist** (the `- [ ]` items) — the per-task `**Status:**` marker next to each
individual task is for human/Executor tracking only and is never read by `eng`. Marking every
task's own `**Status:**` to `[x]` is not enough on its own; check off the Completion checklist
too. If `eng workflow advance` reports unchecked items, it now names the specific blocking
line(s) rather than a generic message (fixed in Phase 9 — see `docs/gotchas.md`).

---

## Command reference

Every command below is real — verified against `cli/main.go` and each subcommand's flag
parsing while writing this document.

### Setup

| Command | Purpose | Example |
|---|---|---|
| `eng install --from <path> [--add-to-path]` | Install the harness payload + binary | `eng install --from . --add-to-path` |
| `eng init` | Create `.agent/project.yaml` (only file it writes) | `eng init` |
| `eng doctor` | Report install/project/skill/tool status | `eng doctor` |
| `eng scan` | Print detected stack + file-extension counts | `eng scan` |
| `eng start [--init]` | Doctor, then launch the configured agent | `eng start` |

### Skills & context — *advanced/debug* (the running session calls these for you)

| Command | Purpose | Example |
|---|---|---|
| `eng skills list` | List every resolved skill (global+private+local) | `eng skills list` |
| `eng skills validate` | Check skill metadata/dependency issues (exit 1 on error) | `eng skills validate` |
| `eng context skills "<text>"` | Show which skills a request would select, and why | `eng context skills "add Modbus TCP monitoring"` |
| `eng context project "<text>"` | Show matching `docs/src-map.md`/`docs/gotchas.md` sections | `eng context project "auth token check"` |
| `eng context task <plan-dir>` | Show the current unchecked task + goal summary | `eng context task .plans/2026-08-26-my-feature` |
| `eng context bundle <role> <plan-dir> ["<text>"]` | Compose a role's full context + write the manifest | `eng context bundle executor .plans/2026-08-26-my-feature` |
| `eng context manifest <plan-dir>` | Pretty-print an existing `context-manifest.yaml` | `eng context manifest .plans/2026-08-26-my-feature` |

### Planning & lifecycle — *advanced/debug*

| Command | Purpose | Example |
|---|---|---|
| `eng triage "<text>"` | Heuristic risk-level hint (not authoritative) | `eng triage "fix the login bug"` |
| `eng plan new <name> [--risk <level>] [--requires-approval]` | Scaffold a plan, stamp current git SHA | `eng plan new my-feature --risk feature` |
| `eng plan drift [dir]` | Check whether relevant files changed since planning | `eng plan drift .plans/2026-08-26-my-feature` |
| `eng plan retry <dir> <build\|unit_test\|integration_test>` | Track a retry against the plan's budget | `eng plan retry .plans/2026-08-26-my-feature unit_test` |
| `eng plan review <dir> --verdict PASS\|REJECT [--blocking-issues N]` | Record a plan-reviewer verdict | `eng plan review <dir> --verdict PASS` |
| `eng plan approve <dir> [--by <name>]` | Grant execution-risk approval | `eng plan approve <dir> --by alice` |
| `eng plan approve-spec <dir> [--by <name>]` | Grant requirements (spec) approval | `eng plan approve-spec <dir> --by alice` |
| `eng plan escalate <dir> --to <level> [--reason "..."]` | Re-risk a quick-fix plan, reset to `TRIAGED` | `eng plan escalate <dir> --to bug --reason "wider than expected"` |
| `eng plan block <dir> --reason "..."` | Force state to `BLOCKED` | `eng plan block <dir> --reason "waiting on vendor"` |
| `eng plan cancel <dir> [--reason "..."]` | Force state to `CANCELLED` | `eng plan cancel <dir>` |
| `eng workflow start "<text>"` | Triage + `eng plan new` + report status | `eng workflow start "add CSV export"` |
| `eng workflow status [dir]` | Report a plan's lifecycle state + next action | `eng workflow status <dir>` |
| `eng workflow advance [dir]` | Apply the next safe state transition | `eng workflow advance <dir>` |
| `eng adapter prompt <role> <dir> ["<text>"]` | Print the assembled prompt + context bundle for a role | `eng adapter prompt executor <dir>` |
| `eng verify [dir]` | Run tests, check the git diff, write `verify-report.md` | `eng verify <dir>` |
| `eng hooks run <stage> [plan-dir]` | Run configured lifecycle hooks | `eng hooks run before_execute` |

### Tools & capabilities — *advanced/debug*

| Command | Purpose | Example |
|---|---|---|
| `eng capabilities list [--verbose] [--role <role>]` | Report which known tools are on PATH | `eng capabilities list --role executor` |
| `eng capabilities explain <role> <dir> ["<text>"]` | Explain tool routing for a request | `eng capabilities explain executor <dir> "inspect open PRs"` |
| `eng tools invoke <role> <capability> <dir> [args...]` | Invoke one capability — the only sanctioned path, always policy-checked and audited | `eng tools invoke executor git.status <dir>` |

### Maintenance

| Command | Purpose | Example |
|---|---|---|
| `eng logs prune [--dry-run]` | Apply `.agent/logs/` retention (max files/age/total size) | `eng logs prune --dry-run` |

---

## Normal vs. advanced usage

### Normal developer

```bash
eng init
eng doctor
eng start
```

...then plain natural language inside the session. That's the entire interface for day-to-day
work. You'll see `eng workflow status`/`eng plan approve*` mentioned in the session's own
output at approval gates — those are the only low-level commands a normal user is expected to
type by hand, and only when the session stops and asks.

### Advanced / debug / CI

Everything under [Command reference](#command-reference) marked *advanced/debug* — `eng plan`,
`eng workflow`, `eng context`, `eng tools`, `eng capabilities`, `eng hooks`. Reach for these
when:

- scripting a CI check (`eng verify`, `eng plan drift`)
- debugging why a skill/doc wasn't selected (`eng context skills`/`project`)
- inspecting exactly what a role would see before running an agent session
  (`eng adapter prompt`, `eng context bundle`)
- manually driving the state machine instead of letting a Claude Code session do it

---

## Project configuration (`.agent/project.yaml`)

Written once by `eng init`; every field below is real (`cli/internal/project/project.go`'s
`Config` struct) — nothing here is speculative.

```yaml
project_name: my-project
mode: modern                  # legacy | hybrid | modern — see Legacy compatibility below
harness_profile: software
config_version: 2
stack:
  type: go
  build_cmd: go build ./...
  test_cmd: go test ./...
  run_cmd: go run .
  lint_cmd: golangci-lint run
enabled_skills:
  - engineering/karpathy-guidelines   # always included regardless of the skill budget
domains: []                          # optional — e.g. [embedded, automation], fills the
                                      # skill router's domain-profile tier
private_skills_path: ""              # optional — a third skill-resolution tier between
                                      # global and project-local (any local dir or checkout)
workflow:
  triage: false                      # omit any of these three to default to true (enabled);
  plan_review: false                 # each field is independent — explicit false stays false,
  verifier: false                    # it does NOT fall back to "everything enabled" (fixed
                                      # in Phase 9 — see docs/gotchas.md)
  planning_mode: spec_first          # auto_plan | spec_first — eng init writes spec_first
  require_spec_approval: true        # defaults to true when unset
retry_budget:                        # optional — defaults to {build:2, unit_test:2, integration_test:1}
  build: 2
  unit_test: 2
  integration_test: 1
tools:                                # optional — Phase 7 tool policy, see docs/tools.md
  allow: []
  require_approval: []
  deny: []
```

`stack.*_cmd` fields are `executil.Command` — either a plain shell string or a structured argv;
`eng init` fills these from auto-detection, and you can hand-edit them at any time.

`eng doctor` reports each `workflow:` field's resolved value and whether it's explicit or
defaulted (`enabled (default)` / `disabled (explicit)`), plus the resolved `planning_mode` —
run it after hand-editing `workflow:` to confirm the config means what you intended.

---

## Context configuration (`.agent/context.yaml`)

Optional. Falls back to `~/.engineering-harness/context/default.yaml`, then to hard-coded
defaults — a project with none of these files works with zero extra configuration.

```yaml
strategy: selective              # selective | full — "full" disables the skill/doc budget entirely
max_skills: 5
max_docs: 8
max_log_lines: 300               # head+tail cap on verify/tool output shown inline
include_completed_tasks: false
summarize_tool_output: true
max_log_files: 100               # .agent/logs/ retention
max_log_age_days: 30
max_log_total_mb: 250
```

(Defaults shown above — `cli/internal/contextcfg.Default()`.) Every field is independently
optional in the override file; an unset field keeps its default rather than resetting to zero.

**Fallback behavior:** if a `planner` role's context bundle would select zero skills and zero
doc sections under `selective` strategy, `buildContextBundle` automatically retries once with
`strategy: full` for that call, so a Planner is never handed an empty bundle by an
overly-tight budget. This is a one-time fallback for that call, not a change to your config file.

---

## Legacy V1 workflow (reference)

Existing V1 projects are not forced to migrate — see
[Legacy compatibility](#legacy-compatibility) below. If you're still using the original
per-project template directly (no `eng` involved at all), here's the condensed command
reference; the full step-by-step tutorial that used to live in the README is preserved here.

```bash
# One-time setup (from a checkout of this repo)
bash /path/to/this/repo/scripts/init.sh          # detects project type, copies template files,
                                                  # writes .planner-executor/config.yaml

# Skills
./scripts/load_skill.sh list          # list all skills with descriptions
./scripts/load_skill.sh <name>        # print a skill's full content

# Plans
./scripts/plan-executor.sh new <feature-name>     # scaffold spec.md/tasks.md/tests.md
./scripts/plan-executor.sh list                   # list all plans + completion %
./scripts/plan-executor.sh status <plan-dir>      # show task checkboxes
./scripts/plan-executor.sh open <plan-dir>        # print all three files to stdout
./scripts/plan-executor.sh test                   # run build_cmd/test_cmd from config.yaml

# Re-detect build/test commands
./scripts/detect-project.sh
```

Claude acts as **Planner** (writes `spec.md`/`tasks.md`/`tests.md`, never touches source);
GitHub Copilot, OpenAI Codex, or Claude Code in executor mode acts as **Executor** (implements
one task at a time, runs the test after each, marks `[x]`, stops on failure). See
`CLAUDE.md`/`AGENTS.md`/`.github/copilot-instructions.md` at the repo root for the exact role
instructions each tool reads automatically.

**Task status markers** used in V1's `tasks.md`: `[ ]` not started, `[~]` in progress, `[x]`
complete, `[!]` failed (relay the error back to the Planner).

---

## Legacy compatibility

`eng doctor`/`eng init` classify every project into one of five modes
(`cli/internal/project.DetectModeResult`):

| Mode | Meaning |
|---|---|
| `none` | No `.agent/project.yaml`, no `CLAUDE.md`/`.plans/` — not yet touched by either V1 or V2 |
| `legacy` | `CLAUDE.md` and/or `.plans/` present, no `.agent/project.yaml` — a pure V1 project. Fully compatible, no action required. |
| `hybrid` | `.agent/project.yaml` present *and* it was created over an existing legacy shape (or `mode: hybrid` was set explicitly) — V1 files untouched, `eng` commands also work |
| `modern` | `.agent/project.yaml` present, no legacy shape detected — a project that started with `eng init` |
| `broken` | `.agent/project.yaml` exists but fails to parse |

Running `eng init` in a legacy project auto-detects the legacy shape and writes `mode: hybrid`
— it explicitly prints "Existing CLAUDE.md / .plans/ / skills/ were left untouched." There is
**no `eng migrate` command** — migration is never required, and nothing currently claims to
automate it. `eng doctor`/`eng workflow start` work identically against a hybrid project as
against a modern one, with zero forced migration steps — verified directly in Phase 8's
benchmark (`benchmarks/results/legacy-v1-compat.yaml`).

---

## Skill authoring

See [`docs/skills.md`](skills.md) for the full model (levels, dependencies, routing precedence,
sources). Quick reference:

```yaml
---
name: modbus
domain: automation
level: technology              # engineering | domain | technology
description: One sentence.
tags: [modbus, register]
triggers: [modbus, "40001"]
version: "1.0.0"
requires: []                   # hard dependency — always included, never dropped by budget
recommends: [networking/tcp-ip]  # soft — included only if the budget allows
capabilities: []               # reserved for capability-based routing
conflicts: []                  # surfaced by `eng skills validate`, not enforced
when_to_use: ...
when_not_to_use: ...
---

# Skill: modbus

## Purpose
...
## Method
...
```

A `SKILL.md` with no frontmatter at all still resolves via the legacy `# Skill: name` / `##
Purpose` heading convention — nothing requires migrating an old V1 skill.

```bash
mkdir -p skills/<domain>/<name>        # project-local skill
# write SKILL.md
eng skills validate                     # fix anything flagged as an error
```
