# Spec — V2 Harness Foundation (global install + `eng init`/`doctor`/`scan` + skill resolution)

> **Planner note:** This is a foundation/architecture plan, not a feature plan. It evolves
> this repository from a clone-per-project template (V1) toward a globally-installed harness
> (V2) without changing a single byte of V1's existing behavior. Read this file in full before
> `tasks.md` — the Design decisions section explains *why* each new file exists.

---

## Goal

Today, using this workflow in a new project means cloning/copying this entire repository into
that project (`scripts/init.sh`) — every project ends up with its own physical copy of scripts,
skill templates, and instruction files, and there is no way to update the workflow itself
without re-running that copy in every project. This plan adds a minimal, additive foundation
that lets the workflow instead be **installed once** (`eng install` → `~/.engineering-harness/`)
and **linked into** a project with a thin, few-line config file (`eng init` →
`.agent/project.yaml`) — while every existing V1 file, script, and behavior keeps working
completely unchanged for anyone who never touches `eng` at all.

**Done looks like:** `eng install` populates `~/.engineering-harness/` from this repo's new
`harness/` tree; `eng init`, run inside any project — brand new, or an existing V1
project with `CLAUDE.md`/`.plans/`/`skills/` already in place — writes a single new file
(`.agent/project.yaml`) and touches nothing else; `eng doctor` correctly reports a project as
`legacy` (V1, no `.agent/`), `hybrid` (V1 files + `.agent/`), or `modern` (fresh `eng init`),
and lists the skills resolved from global + project-local sources; `eng scan` prints detected
stack info respecting `.agentignore`. None of this requires deleting, renaming, or modifying
`scripts/*.sh`, `CLAUDE.md`, `AGENTS.md`, `.github/copilot-instructions.md`, `skills/`, or
`.plans/` in this repo or in any project that adopted V1.

---

## Background

The user requested a full architecture and gap analysis (delivered in-conversation, see also
the "Design" section below for the parts that need to live in this repo) before any
implementation. That analysis concluded the V1 template already contains working prototypes
of most of the V2 foundation — a plan format, a skill manifest, a project-type detector, a
copy-if-missing/copy-always upgrade model — but everything is scoped to "lives inside the one
project you copied it into." The highest-leverage, lowest-risk first slice is the one the user
also named explicitly as the MVP target: global install, `eng init`, minimal
`.agent/project.yaml`, legacy detection, a global skill directory, local+global skill
resolution, a skill manifest/router *foundation* (not the full classifier), `eng doctor`,
project scanning, and backward compatibility. Everything else the user listed (Plan Reviewer,
Verifier, Task Triage, Hooks, Policies, MCP adapters, worktrees, skill evals) explicitly
depends on this foundation existing first and is out of scope here.

---

## Design decisions

### Decision 1 — CLI runtime: Go, single static binary
**Chosen:** The `eng` CLI is written in Go and compiled to one binary per platform
(`eng` / `eng.exe`).
**Why:** User's explicit choice. Go gives native Windows support (this repo's actual dev
machine is Windows 10 + PowerShell — V1's bash scripts require WSL, a documented pain point in
this repo's own README troubleshooting section) with zero runtime dependency to install, and a
small dependency surface (only a YAML library is needed).
**Rejected:** Node.js (adds a Node runtime + npm publish story); Python (pipx/PATH friction on
Windows); staying Bash + adding a PowerShell port (permanently doubles script maintenance and
never produces one global install artifact).

### Decision 2 — This repository is the harness source; no new repo yet
**Chosen:** Add `harness/` (the installable payload) and `cli/` (Go source for `eng`) to this
same repository, alongside the untouched V1 root-level files.
**Why:** `eng install` for this MVP installs from a local filesystem path (`--from <path>`,
defaulting to `.`), pointed at a checkout of this repo. Splitting into a separate repo now would
sever the direct lineage from the template teams already adopted, for no benefit at this stage.
Requirement 38 ("don't pollute a *project's* git diff with harness content") is about consumer
projects, not this harness-source repo — this repo is expected to contain the payload it
distributes, the same way any package's source repo contains what it publishes.
**Rejected:** New repo now (premature — can be `git filter-repo`'d out later once the harness
stabilizes, without blocking this plan).

### Decision 3 — Three-mode compatibility model, opt-in only
**Chosen:** `mode: legacy | hybrid | modern`, stored in `.agent/project.yaml`. A project with
**no** `.agent/` directory is always `legacy` by convention — no marker file, no imposed
opinion, ever required.
**Why:** Satisfies "a legacy project should continue to work" and "do not require immediate
migration" literally: the safest possible default for an untouched project is that `eng` writes
nothing to it until asked.
**Rejected:** A required marker file even for legacy projects (forces a write to every existing
project just to be recognized — contradicts "not require immediate migration").

### Decision 4 — Skill schema: additive YAML frontmatter, legacy heading style still valid
**Chosen:** New/global `SKILL.md` files carry a YAML frontmatter block (`name`, `domain`,
`description`, `tags`, `triggers`, `version`, `dependencies`, `conflicts`, `when_to_use`,
`when_not_to_use`). The resolver reads frontmatter when present; when absent it falls back to
today's extraction rule (`# Skill: <name>` heading + first sentence after `## Purpose`) — the
same rule `scripts/update-manifest.sh` already implements.
**Why:** Every existing project-local `skills/*/SKILL.md` (no frontmatter) keeps resolving
forever, unmodified. This is the same "don't touch what you don't need to" logic V1 already
uses for `CLAUDE.md` (user-owned, never overwritten).
**Rejected:** A hand-maintained global `manifest.yaml` — a second file to keep in sync with the
skills it describes, the exact failure mode `docs/src-map.md`'s own design note already warns
against ("this is why the map stays accurate... not a separate documentation sprint").

### Decision 5 — No manifest file required for resolution; live directory walk
**Chosen:** `eng skills list` (and the `doctor`/`init` calls that use the same resolver) walks
`<harness>/skills/**/SKILL.md` and `<project>/skills/**/SKILL.md` directly at query time and
merges by `name`, local overriding global.
**Why:** Skill counts are small (tens, not thousands); a live walk removes an entire class of
"manifest went stale" bugs for negligible cost. V1's `skills/manifest.json` +
`update-manifest.sh` are untouched and keep working for anyone still using `load_skill.sh`
directly — `eng` simply doesn't depend on that file.
**Rejected:** Porting `update-manifest.sh` to Go and requiring a sync step before every query.

---

## Scope

### In scope
- `cli/` — Go module for `eng`: `install`, `init`, `doctor`, `scan`, `skills list`
- `harness/` — installable source tree: `core/{planner,executor}/METHOD.md`,
  `core/principles/*` (copied from `.claude/principles/`, already domain-agnostic),
  `skills/engineering/karpathy-guidelines/SKILL.md` (promoted, frontmatter added),
  `profiles/software.yaml`, `templates/plan/*` (copied from `.plans/_template/`),
  `templates/.agentignore`, `VERSION`
- `.agent/project.yaml` schema, writer (`eng init`), reader (`eng doctor`/`eng scan`)
- Legacy-project detection (CLAUDE.md + `.plans/` present, no `.agent/` → reported as
  "Legacy Plan-Execute Project")
- `.agentignore` default template, honored by `eng scan`
- Global install path: `$HOME/.engineering-harness` via Go's `os.UserHomeDir()` (resolves
  correctly on Windows too)
- Zero behavior change to any existing root-level V1 file

### Out of scope (explicitly excluded)
- **Skill Router / free-text task classifier** — needs Task Triage (below) to exist first;
  this plan builds only the resolver a future router would route *over* (`eng skills list`
  returns everything resolved; it does not yet pick a subset for a given request).
- **Plan Reviewer, Verifier, Research Mode, Hooks, Policies, retry budgets, stop conditions** —
  explicitly deferred later phases per the user's own scope constraint; none of them can be
  meaningfully designed before this foundation exists to hang them on.
- **MCP/tool adapters beyond the three existing instruction-file adapters** — no new external
  service integrations in this phase.
- **`eng update` / version channels / multi-profile pinning beyond a single `VERSION` file** —
  a real versioning strategy needs more than one profile and a router to be worth designing;
  a single file today is enough to make `eng doctor` report something real.
- **A distribution/release pipeline for the `eng` binary** (cross-compiled releases, package
  manager formulas, self-bootstrapping `eng install eng`) — for this MVP a developer runs
  `go build` locally; packaging is its own later plan.
- **Git worktree isolation, task dependency graphs, write-sets, parallel execution** — nothing
  in the current or planned-next workflow executes multiple tasks concurrently, so there is
  nothing yet to isolate.
- **Deleting or rewriting any existing V1 file** — this plan adds files only. A future
  `eng migrate` command (report-only cleanup suggestions) is intentionally not built here.

---

## Affected files

| File | Change type | Reason |
|---|---|---|
| `cli/go.mod` | Create | Go module definition |
| `cli/main.go` | Create | CLI entrypoint + subcommand dispatch |
| `cli/install.go` | Create | `eng install` |
| `cli/init_cmd.go` | Create | `eng init` |
| `cli/doctor.go` | Create | `eng doctor` |
| `cli/scan.go` | Create | `eng scan` |
| `cli/skills_cmd.go` | Create | `eng skills list` |
| `cli/internal/detect/detect.go` | Create | Go port of `scripts/detect-project.sh` |
| `cli/internal/project/project.go` | Create | `.agent/project.yaml` struct + mode detection |
| `cli/internal/skills/skills.go` | Create | SKILL.md frontmatter parser + resolver |
| `harness/VERSION` | Create | Harness payload version marker |
| `harness/core/planner/METHOD.md` | Create | Domain-agnostic Planner methodology |
| `harness/core/executor/METHOD.md` | Create | Domain-agnostic Executor methodology |
| `harness/core/principles/karpathy.md` | Create | Copy of `.claude/principles/karpathy.md` |
| `harness/core/principles/plan-quality.md` | Create | Copy of `.claude/principles/plan-quality.md` |
| `harness/core/principles/thinking-checklist.md` | Create | Copy of `.claude/principles/thinking-checklist.md` |
| `harness/skills/engineering/karpathy-guidelines/SKILL.md` | Create | Promoted from `skills/karpathy-guidelines/SKILL.md`, frontmatter added |
| `harness/profiles/software.yaml` | Create | First profile: bundles `engineering/karpathy-guidelines` |
| `harness/templates/plan/*.md` | Create | Copy of `.plans/_template/*.md` (5 files) |
| `harness/templates/.agentignore` | Create | Default ignore list |
| `docs/src-map.md` | Modify | Add `cli/` and `harness/` entries (last task, per this repo's own convention) |
| `README.md` | Modify | Additive "V2 harness (preview)" section; existing Quick Start untouched |
| `ROADMAP.md` | Modify | Note phases superseded by this plan; link to it |

---

## Risks and unknowns

- **Go is not currently installed on this development machine** (`go version` → command not
  found, confirmed 2026-08-24). This is a hard prerequisite for every task in `tasks.md` —
  `tests.md` T0 gates on it explicitly rather than assuming it silently.
- `eng install`'s source resolution is local-path-only for this MVP (no git remote fetch) —
  acceptable since this repo is both the source and, for now, the only consumer.
- Windows path handling (`~` expansion via `os.UserHomeDir()`, no `.exe` assumptions baked into
  scripts) needs verification on this actual machine — covered by `tests.md` T9.
- The `mode` transition in `eng init` (`legacy` → `hybrid` on first `.agent/` write) is a
  judgment call with no prior art in this repo to check against — flagged as an open question
  below rather than silently assumed.

---

## Open questions

- [ ] Should `eng init` on a detected-legacy project require an explicit confirmation before
  writing `.agent/project.yaml`, or is writing it unprompted (never touching anything else)
  safe by default? **This plan defaults to unprompted-but-additive** — mirrors `init.sh`'s own
  `copy_if_missing` philosophy, since `.agent/project.yaml` is a brand-new file with no prior
  meaning to overwrite. Revisit if real usage shows this surprises people.

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|---|---|---|---|
| Think Before Planning | spec.md written first; user-observable goal stated | 9/10 | Goal is CLI-observable (`eng doctor` output), not "code exists" |
| Simplicity First | ≥3 out-of-scope items with reasoning | 10/10 | 7 items excluded, each with a one-line reason |
| Surgical Changes | Every file listed with change type and exact reason | 9/10 | New-tree creation is inherently coarser than a single-symbol edit for a foundation plan — unavoidable, not sloppiness |
| Goal-Driven Execution | Each scope item traces to an acceptance criterion | 9/10 | Every `eng` subcommand has a runnable pass/fail test in `tests.md` |

**Total: 37/40 → 9/10**

---

**User confirmation received:** [ ] Yes
**Confirmed on:** _pending_
