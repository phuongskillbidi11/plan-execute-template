# Planner-Executor Template — Roadmap

Phases are ordered by value-to-complexity ratio. Each phase is independent —
you can skip ahead or implement phases in a different order. Every enhancement
must itself follow the Karpathy principles: think before adding, keep it simple,
surgical changes only.

---

## Phase 1 — Auto-detection and config (implement now)

**Goal:** Eliminate manual placeholder replacement. A single `init.sh` command
sets up the template for any project type.

### Deliverables

| File | Action |
|---|---|
| `scripts/detect-project.sh` | Scan for sentinel files; output build/test/run commands |
| `.planner-executor/config.yaml.template` | Canonical config schema |
| `scripts/init.sh` | Orchestrator: copies template + runs detection + writes config |
| `scripts/plan-executor.sh` | Updated to read `build_cmd`/`test_cmd` from config |

### How detection works

Scan for well-known files in priority order:

```
idf_component.yml / sdkconfig  →  ESP-IDF
Cargo.toml                      →  Rust
package.json                    →  Node.js / TypeScript
requirements.txt / pyproject.toml / setup.py  →  Python
go.mod                          →  Go
CMakeLists.txt (non-ESP-IDF)    →  C/C++
Makefile                        →  Generic Make
```

### Upgrade path for existing projects

```bash
# In your existing project that already uses an older template:
curl -sSL https://raw.githubusercontent.com/.../init.sh | bash -s -- --upgrade
# --upgrade skips CLAUDE.md and copilot-instructions.md (preserves customizations)
# --upgrade only writes/updates scripts/ and .planner-executor/
```

---

## Phase 2 — Enhanced plan management

**Goal:** Replace raw file editing with a minimal interactive interface for
browsing, marking, and summarizing plans.

### Deliverables

| File | Action |
|---|---|
| `scripts/plan-executor.sh` | Add `tui` command (requires `fzf`) |
| `scripts/plan-executor.sh` | Add `done <folder> <task-id>` command to mark a task [x] manually |
| `scripts/plan-executor.sh` | Add `summary` command: aggregate all plan statuses into one table |

### TUI flow (fzf-based)

```
fzf list of plans → select plan → fzf list of tasks → select task → mark done / view test
```

No external dependencies beyond `fzf` (available via apt/brew/winget).

### Upgrade path

`plan-executor.sh` is self-contained. Drop the new version over the old one.
No config changes required.

---

## Phase 3 — Support for other AI executors

**Goal:** Make the Executor role pluggable — not just Copilot.

### Deliverables

| File | Action |
|---|---|
| `.cursor/rules.md` | Cursor-specific Executor instructions (same logic, Cursor format) |
| `scripts/executor.sh` | Feed a plan to any LLM API (OpenAI / Anthropic) via CLI |
| `.planner-executor/config.yaml` | Add `executor` key: `copilot` \| `cursor` \| `api` |
| `README.md` | Section: "Choosing your executor" |

### executor.sh sketch

```bash
executor.sh run .plans/2025-05-01-feature/
# Reads spec + tasks + tests, constructs system prompt, calls API,
# streams implementation instructions to stdout.
# Requires: ANTHROPIC_API_KEY or OPENAI_API_KEY in environment.
```

### Upgrade path

New files only — no existing files modified. Opt in by setting
`executor: api` in `.planner-executor/config.yaml`.

---

## Phase 4 — GitHub Actions integration

**Goal:** Automate plan creation and execution triggers from GitHub issues and PRs.

### Deliverables

| File | Action |
|---|---|
| `.github/workflows/generate-plan.yml` | On `issues` labeled `plan`: call Claude API, commit plan as PR |
| `.github/workflows/validate-plan.yml` | On PR touching `.plans/`: check Karpathy compliance (task granularity, test pass/fail conditions) |
| `.github/workflows/execute-plan.yml` | On issue comment `/execute`: run the plan via `executor.sh` in CI |
| `scripts/validate-plan.sh` | Standalone validator — checks tasks have file paths, tests have Pass/Fail |

### Karpathy compliance check (validate-plan.sh)

Checks applied to every `tasks.md`:
- Each `- [ ]` line contains a backtick-quoted file path
- No task is longer than 10 lines (surgical = small)
- Each `tests.md` block contains both `**Pass:**` and `**Fail:**`
- `spec.md` contains an `## Out of scope` section

### Upgrade path

New workflow files only. Requires `ANTHROPIC_API_KEY` secret in repo settings.

---

## Phase 5 — Automated failure recovery

**Goal:** Allow the executor to retry failed tasks by automatically invoking
Claude to amend the plan, up to a configurable retry limit.

### Deliverables

| File | Action |
|---|---|
| `scripts/executor.sh` | Add `--retry N` flag; on test failure, call Claude API with error + task, patch `tasks.md`, retry |
| `.plans/.../amendments/` | Auto-created subfolder; each retry writes `amendment-1.md`, `amendment-2.md` |
| `.planner-executor/config.yaml` | Add `max_retries: 3` key |

### Amendment log format

```
.plans/2025-05-01-feature/
  amendments/
    amendment-1.md    ← original task + error + Claude's revised task
    amendment-2.md    ← second attempt if first amendment also failed
```

### Upgrade path

Only `executor.sh` and config schema change. Opt in with `max_retries: 1`.

---

## Phase 6 — Documentation and real-world examples

**Goal:** Reduce time-to-first-plan for new users to under 10 minutes.

### Deliverables

| File | Action |
|---|---|
| `examples/esp-idf/` | Complete plan set: OTA update (3-file plan as-built) |
| `examples/python-fastapi/` | Complete plan set: add JWT auth endpoint |
| `examples/nextjs/` | Complete plan set: add server-side dark mode cookie |
| `examples/rust-cli/` | Complete plan set: add subcommand with clap |
| `README.md` | Add "Examples" section linking to each |
| `README.md` | Add animated GIF (record with `asciinema`, convert with `agg`) |

### Upgrade path

`examples/` is read-only reference material. No existing file changes.

---

## Upgrading existing projects

When a new template version is released, existing projects can upgrade without
losing their customizations using this rule:

**User-owned files** (never overwritten by upgrade):
- `CLAUDE.md`
- `.github/copilot-instructions.md`
- `.planner-executor/config.yaml`
- All `.plans/**` content

**Template-owned files** (safe to overwrite):
- `scripts/*.sh`
- `.claude/planner-instructions.md`
- `.cursor/rules.md`
- `.github/workflows/*.yml`

### Upgrade command (Phase 1+)

```bash
./scripts/init.sh --upgrade
```

This overwrites only template-owned files and re-runs detection to refresh
auto-detected commands in config.

### Manual upgrade (before init.sh --upgrade exists)

```bash
# Copy only scripts — preserve everything else
cp -r new-template/scripts/ ./scripts/
cp new-template/.claude/planner-instructions.md ./.claude/
```

---

## Backlog (unscheduled)

These ideas have merit but are below the current priority threshold:

- **VS Code extension** — sidebar panel showing plan status without opening
  terminal
- **Notion/Linear sync** — push plan tasks to a project management tool
- **Multi-executor parallel execution** — run independent task groups
  concurrently across Copilot + Claude API
- **Plan diffing** — `plan-executor.sh diff` compares two plan versions to show
  what changed between amendment rounds
