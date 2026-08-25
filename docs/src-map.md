# src-map — what already exists

> **Read this before writing any `spec.md`.** Its entire purpose is to stop the Planner
> (and, downstream, the Executor) from re-discovering or re-inventing code that already
> exists. An incomplete `src-map.md` is what lets the next plan duplicate a module, rename
> something that already has a name, or rebuild a helper that's one directory over.
>
> **Keep this file honest, not exhaustive.** One paragraph per module is enough: what it
> does, the one or two decisions that would surprise a reader, and which `.plans/` phase
> created it. Do not paste code here — link to it.

---

## How to use this file

**As Planner, before `spec.md`:** read every section whose area your feature touches. If a
function, class, or module already does roughly what you're about to design, use it — extend
or call it, don't parallel-build it. If you genuinely need to replace something documented
here, say so explicitly in `spec.md`'s Design decisions section and explain why the existing
version doesn't work, rather than silently duplicating it.

**As Planner, after a plan is confirmed and about to become `tasks.md`:** if the plan adds a
new file, module, or directory under the project's own source tree — or changes what an
existing entry below does — add a task whose only job is updating this file. Put it last in
`tasks.md`, after the feature's own tasks, so the map always describes what actually landed,
not what was planned. A one-line addition is enough; do not let this become its own multi-hour
task.

**As Executor:** treat a `docs/src-map.md`-update task exactly like any other task in
`tasks.md` — implement only what it says, run its verification command, mark it `[x]`.

---

## Format

Group by directory or module. For each one, cover:

- **What it does** — one or two sentences, plain language.
- **Key files** — the two or three files a reader actually needs, not a full listing (the
  filesystem already shows that).
- **Notable design decisions** — anything a reasonable reader would NOT guess by looking at
  the code for thirty seconds (a workaround, a deliberately-not-generic choice, a constraint
  from an external system). Skip this line entirely if there's nothing non-obvious.
- **Where it came from** — the `.plans/` folder that introduced or most recently changed it,
  so a reader who wants the full reasoning knows exactly where to look.

```markdown
### `path/to/module/` — one-line description

What it does: ...

Key files: `foo.ext` (...), `bar.ext` (...)

Notable: ... (omit this line if nothing is non-obvious)

From: `.plans/YYYY-MM-DD-feature-name/`
```

---

## Modules

### `cli/` — the `eng` CLI

What it does: Go CLI (`eng install`, `eng init`, `eng doctor`, `eng scan`,
`eng skills list`) that installs the harness payload globally and links a thin
`.agent/project.yaml` into any project.

Key files: `cli/main.go` (dispatch), `cli/internal/project/project.go` (mode detection),
`cli/internal/skills/skills.go` (global+local skill resolution)

From: `.plans/2026-08-24-v2-harness-foundation/`

### `harness/` — the installable harness payload

What it does: source tree copied by `eng install` into `~/.engineering-harness/` — core
Planner/Executor methodology, the first skill (`engineering/karpathy-guidelines`), the
`software` profile, and the plan templates.

Notable: skills here use YAML frontmatter for metadata; project-local skills without
frontmatter still resolve via the legacy `# Skill:` heading convention — see
`cli/internal/skills/skills.go`.

From: `.plans/2026-08-24-v2-harness-foundation/`

### `cli/internal/planmeta/`, `cli/internal/gitutil/`, `cli/internal/hooks/` — Phase 2 plan lifecycle state

What it does: `planmeta` reads/writes each plan's `plan.yaml` (git SHA, risk level, write
scope, retry counters/budget); `gitutil` wraps `git rev-parse`/`git diff` for drift checks;
`hooks` loads the lifecycle hook table (`harness/hooks/default.yaml`, project-overridable via
`.agent/hooks.yaml`).

Key files: `cli/plan_cmd.go` (`eng plan new/drift/retry`), `cli/verify_cmd.go` (`eng verify`),
`cli/hooks_cmd.go` (`eng hooks run`)

Notable: `eng verify` and `eng hooks run` shell out via `sh -c` — requires a POSIX shell on
PATH (Git Bash's `sh.exe` on Windows), same dependency V1's own scripts always had. Separately,
`harness/hooks/default.yaml`'s built-in commands (`eng scan`, `eng plan drift .`, `eng verify
.`) assume the `eng` binary itself is on PATH — nothing in Phase 1 or Phase 2 installs `eng`
onto PATH (only `eng install` puts the harness *payload* in `~/.engineering-harness/`); a
project using `eng hooks run` today must put `eng` on PATH itself.

From: `.plans/2026-08-24-v2-harness-phase2/`

### `harness/core/triage/`, `harness/core/plan-reviewer/`, `harness/core/verifier/` — new roles

What it does: methodology docs for the three roles Phase 2 adds around the existing
Planner/Executor loop. As of Phase 2, none of them were enforced by code — `review.md` and
`verify-report.md` were the enforcement surface (a file with a clear verdict), not a gate a
script could close. **Phase 3 update:** the Reviewer's and Verifier's verdicts are now also
persisted as machine-readable fields on `plan.yaml` (`eng plan review`, and `eng verify`
automatically) and actually gate `eng workflow advance` — see the next entry.

From: `.plans/2026-08-24-v2-harness-phase2/`

### `cli/internal/workflow/`, `cli/internal/agent/`, `cli/internal/capabilities/`, `cli/internal/executil/` — Phase 3 orchestration

What it does: `workflow` holds the lifecycle state enum and the pure transition table
(`Decide`); `agent` defines the `Adapter` interface with `ClaudeCodeAdapter` as the only
implementation; `capabilities` detects which known CLI tools are on PATH; `executil` runs a
command either as a shell string (compatibility mode) or a structured argv (no shell).

Key files: `cli/workflow_cmd.go` (`eng workflow start/status/advance`), `cli/adapter_cmd.go`
(`eng adapter prompt`), `cli/internal/workflow/workflow.go` (the transition table)

Notable: `eng workflow advance` never writes plan content and never invokes an agent
unattended — every human/AI-driven stage ends with a printed next command and a stop. The
approval gate (`requires_approval`/`approved_at` on `plan.yaml`) is enforced here: a plan
cannot reach `EXECUTING` while it's set and unapproved (verified end-to-end in this plan's
`tests.md` T11b). A real defect was found and fixed during this plan's own execution: Go's
`flag` package silently drops flags placed after a positional argument (e.g.
`eng plan review <dir> --verdict PASS`) — see `docs/gotchas.md`.

From: `.plans/2026-08-24-v2-harness-phase3/`

### `cli/internal/contextcfg/`, `cli/internal/skillmatch/`, `cli/internal/docsearch/`, `cli/internal/taskscope/` — Phase 4 context selection

What it does: `contextcfg` loads the optional `.agent/context.yaml` budget (falling back to
`harness/context/default.yaml`, then hard-coded defaults); `skillmatch` finally makes
Phase 1's `tags`/`triggers` skill-frontmatter fields load-bearing by scoring them against a
request; `docsearch` parses `docs/src-map.md`/`docs/gotchas.md`'s existing `### ` sections
and keyword-matches them; `taskscope` extracts just the current unchecked task block and
`spec.md`'s Goal summary instead of whole files.

Key files: `cli/context_cmd.go` (`eng context skills/project/task/bundle`)

Notable: `enabled_skills` (Phase 1) is always included by `skillmatch.Select` regardless of
`max_skills` — the cap only limits additional discovered-but-not-required skills. A real
defect was found and fixed during this plan's own execution: `eng init` writes
`enabled_skills` entries as `domain/name` (e.g. `engineering/karpathy-guidelines`), but a
resolved skill's `Name` is its bare frontmatter name (`karpathy-guidelines`) — the "always
included" guarantee silently failed for exactly the entry `eng init` itself creates, until
`skillmatch.Select` was fixed to also register each entry's basename. `eng verify`'s test
output is now capped (`max_log_lines`, default 300, head+tail) with the full output written
to `.agent/logs/` — `eng workflow advance`'s gating logic is unaffected since it only ever
reads `plan.yaml`'s `verification.verdict`, not the report text.

From: `.plans/2026-08-24-v2-harness-phase4-context/`

From: `.plans/2026-08-24-v2-harness-phase3/`

### `harness/core/runtime/`, Phase 5 workflow/context/log extensions — runtime integration

What it does: `harness/core/runtime/METHOD.md` is the natural-language routing protocol a
Claude Code session follows for a plain-language request. `eng adapter prompt` now folds in
`eng context bundle`'s output automatically. `internal/workflow` gained `NEEDS_SPEC_APPROVAL`/
`SPEC_APPROVED` (a distinct concept from the existing execution-approval gate) and a
quick-fix fast path that skips straight from `TRIAGED` to `EXECUTING` with a minimal plan.
`internal/logprune` bounds `.agent/logs/` growth. `internal/tooladapter`/`internal/toolrouter`
are new, deliberately unpopulated foundations for future external-tool adapters, kept
structurally separate from `internal/agent`'s coding-agent adapters.

Key files: `harness/core/runtime/METHOD.md`, `cli/context_cmd.go` (`buildContextBundle`),
`cli/internal/workflow/workflow.go` (the extended transition table)

Notable: `planning_mode` defaults to `auto_plan` (Phase 1-4's exact behavior) whenever it is
unset — only a project initialized by Phase 5's `eng init` gets `spec_first` explicitly; no
existing project's state-machine behavior changes. Quick Fix's fast path deliberately skips
review/approval by design — `eng plan escalate` is the correction mechanism if a request
turns out to be broader than triage guessed.

From: `.plans/2026-08-24-v2-harness-phase5-runtime/`

### `cli/internal/skillgraph/`, `cli/internal/skillrouter/`, `cli/internal/skillvalidate/`, `cli/internal/skilleval/` — Phase 6 multi-domain skill ecosystem

What it does: `skillgraph` expands a skill's `requires:` transitively (deterministic order,
cycle detection, unknown-target errors). `skillrouter.Route` is the one authoritative
selection path `context_cmd.go`'s `selectSkills` calls — explicit skills, then their
required dependencies, then request matches, then a project's `domains:` profile fill,
then `recommends:`, all budget-aware, each with an explanation. `skillvalidate` checks
metadata/dependency issues (`eng skills validate`), warning-only for legacy skills.
`skilleval` loads `harness/evals/**/*.yaml` scenarios exercised by a `cli/` integration
test against the real `harness/skills` tree. `skills.Skill` gained `level`, `requires`
(renamed from the unused `dependencies`), `recommends`, `capabilities`, and a
`QualifiedName()` identity (`domain/name`) that `Resolve`/the new `ResolveWithPrivate`
merge by instead of bare name, preventing cross-domain collisions. `project.Config` gained
`domains` (plural, combinable domain profile) and `private_skills_path` (a third,
optional resolution tier between global and project-local).

Key files: `cli/internal/skillrouter/skillrouter.go`, `cli/internal/skills/skills.go`
(`QualifiedName`, `ResolveWithPrivate`), `docs/skills.md` (the authoring/routing guide)

Notable: `Resolve(global, local)` is now `ResolveWithPrivate(global, "", local)` — a pure
delegation, byte-identical behavior, covered by a regression test — so no pre-Phase-6
caller needed to change. `eng hooks run <stage> [plan-dir]` gained an optional third
argument fixing the Phase 5 gotcha where `drift_check`/`verify` only worked when invoked
from a directory that itself contained `plan.yaml`; omitting the argument defaults to `.`,
identical to every existing invocation.

From: `.plans/2026-08-25-v2-harness-phase6-skills/`

### `cli/internal/toolcap/`, `cli/internal/tooladapter/`, `cli/internal/toolpolicy/`, `cli/internal/toolrouter/`, `cli/internal/mcpregistry/` — Phase 7 tool adapter runtime

What it does: `toolcap` defines the capability/risk model (`READ < WRITE < DESTRUCTIVE <
HIGH_RISK`). `tooladapter`'s `Adapter` interface (revised from Phase 5's foundation-only
shape) is implemented by `GitAdapter` (upgraded), the new external reference
`GitHubAdapter` (read-only, via `gh`), and `ReferenceMCPAdapter` (deterministic mock MCP
server — no live transport). `toolpolicy.Decide` is the one policy function: built-in
hard deny, then project `tools.deny`/role toolbox/role risk ceiling/`tools.require_approval`
(gated on `plan.yaml`'s existing execution-approval field, not a new approval concept)/
`tools.allow`, then a safe risk-based default. `toolrouter.Route` (the new authoritative
path; `Filter` stays as Phase 5's simpler adapter-name filter) buckets each required
capability into Allowed/NeedsApproval/Blocked with a reason. `mcpregistry` loads
`harness/mcp/servers.yaml`, a static, credential-free discovery list. `eng tools invoke`
is the one invocation boundary — it always runs `toolpolicy.Decide` first, writes a
compact `tool_invocation` audit event via the existing `planmeta.AppendStructuredEvent`
(Phase 5) regardless of outcome, and reuses Phase 4/5's `writeFullLog`/`summarizeOutput`
for bounded output. `buildContextBundle`'s Planner/Executor sections gain a `## Tools`
section driven by the selected skills' `capabilities:` field (Phase 6) — the Requirement
28 connection from Skill Router to Tool Router, with no separate NL-capability-detection
logic.

Key files: `cli/internal/toolpolicy/toolpolicy.go` (`Decide`), `cli/internal/toolrouter/toolrouter.go`
(`Route`), `cli/internal/tooladapter/{tooladapter,github,reference_mcp}.go`

Notable: `Config.RequireApproval` (a pre-existing, never-read, undocumented field) was
deliberately *not* repurposed for the new `tools.require_approval` policy — a new, nested
`Config.Tools` field was added instead, since guessing at the old field's original intent
was a bigger risk than adding a clearly-scoped one. A project with no `tools:` block at
all (every project before this phase) gets exactly today's behavior: read capabilities
work, write-ish ones require the same plan approval Phase 3 already established. A real
defect was found and fixed during this plan's own execution: `ReferenceMCPAdapter` (Name()
`"mcp-docs"`) was implemented and wired into routing, but never added to
`agent.RolePermissions` — every role's toolbox check silently denied it regardless of
availability or policy, caught via a live `eng capabilities explain` walkthrough and now
covered by a regression test.

From: `.plans/2026-08-25-v2-harness-phase7-tools/`
