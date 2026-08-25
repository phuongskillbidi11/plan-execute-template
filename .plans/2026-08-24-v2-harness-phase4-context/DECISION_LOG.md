# Decision Log — V2 Harness Phase 4 (Context Engineering)

> Read alongside `spec.md`'s "Design decisions" section for full depth.

---

## Decisions

### 2026-08-24 — New commands only; `eng verify`'s output format is the sole exception
**Context:** Phase 4 must not change behavior for projects unaware of it.
**Decision:** `eng context {skills,project,task,bundle}` are entirely new. Every other
Phase 1–3 command's output is untouched, except `eng verify`'s printed test-output block
(Decision 6), whose machine-readable contract (`Verdict:`, `plan.yaml`'s
`verification.verdict`, exit code) is unchanged.
**Reasoning:** Matches every prior phase's compatibility posture — a project with zero
awareness of this phase sees zero behavior change from commands it already uses.
**Alternatives rejected:** Making `eng skills list` selective by default — its existing "show
everything resolved" contract is still useful and other code doesn't expect it to change.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Skill selection: keyword score, `enabled_skills` always guaranteed
**Context:** Phase 1's skill frontmatter schema defined `tags`/`triggers` specifically for a
future router; no code has ever read them.
**Decision:** `skillmatch.Select` scores by substring match against `tags`/`triggers`/
description words, ranks, caps at `max_skills`. Skills named in `.agent/project.yaml`'s
`enabled_skills` are always included regardless of score or cap.
**Reasoning:** `enabled_skills` means "this project wants this skill" — new filtering must
never silently drop it. Keyword-over-metadata is the "simple retrieval" this phase is asked to
try before anything heavier.
**Alternatives rejected:** Embeddings/semantic similarity — out of scope; unjustified
complexity at this phase's target scale (tens to low hundreds of skills).
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Project-doc retrieval reuses the existing `### ` section convention
**Context:** `docs/src-map.md`/`docs/gotchas.md` already use one `### ` header per
module/gotcha, designed from the start to be independently scannable.
**Decision:** `docsearch.ParseSections` splits on `### `; `Match` keyword-scores each section,
capped at `max_docs`. No new file format.
**Reasoning:** Automates what a human Planner already does by eye; doesn't disturb the
existing authoring convention or break any current entry.
**Alternatives rejected:** A structured YAML/JSON rewrite of these files — breaks the
human-authoring workflow they were designed around for a marginal parsing-reliability gain.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Task-scoped context reuses `tasks.md`'s existing `## Task N` + `- [ ]` convention
**Context:** `scripts/plan-executor.sh` (V1) and `workflow_cmd.go`'s `tasksComplete` (Phase 3)
already trust unchecked-checkbox detection as the "is this task done" signal.
**Decision:** `taskscope.CurrentTask` splits on `## Task \d+` headers and returns the first
block still containing `- [ ]`.
**Reasoning:** Reuses an already-battle-tested signal instead of inventing new per-task
metadata (IDs, `depends_on`) that every future plan would need to adopt.
**Alternatives rejected:** Requiring structured task IDs now — that's the task-dependency-graph
requirement from the original harness brief, explicitly deferred past this phase.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Context budget: new optional `.agent/context.yaml`, mirroring `hooks.yaml`
**Context:** The brief specifies a `context:` config block with `max_skills`/`max_docs`/etc.
**Decision:** `contextcfg.Config` loads from `.agent/context.yaml` (full override) or
`harness/context/default.yaml`, defaulting to `Default()` if neither exists. The two bool
fields use pointer-typed override parsing so an unset field doesn't silently reset to `false`.
**Reasoning:** Matches Phase 2's `hooks.yaml` precedent (optional, project-overridable, global
default). The pointer-bool fix addresses a real ambiguity caught during design: Go's zero
value for `bool` (`false`) is indistinguishable from "not set" without pointers.
**Alternatives rejected:** Folding these fields into `.agent/project.yaml` — a cross-cutting
orchestration concern, not project identity, same reasoning that already split out
`hooks.yaml`.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Tool output: full log to `.agent/logs/`, bounded head+tail summary inline
**Context:** `runVerify` (Phase 3) embeds the entire test command's output unconditionally.
**Decision:** Full output goes to `.agent/logs/verify-<timestamp>.log`; the report/stdout gets
the first and last `max_log_lines/2` lines plus a pointer to the full log, when
`summarize_tool_output` is true (default).
**Reasoning:** Line-count bounding is model-agnostic per the brief's explicit instruction
against hard-coding token counts. Head+tail, not head-only, because failures are conventionally
reported at the end of test output.
**Alternatives rejected:** Head-only truncation — would hide the actual failure lines most test
runners print last.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — `eng context bundle` composes the other commands; `eng adapter prompt` is untouched
**Context:** Phase 3 already shipped `eng adapter prompt`, whose exact output
`workflow_cmd.go`'s "next action" messages point users at.
**Decision:** `eng context bundle` is a separate, new, composable command. It does not replace
or call into `eng adapter prompt`, and vice versa.
**Reasoning:** Changing an already-shipped command's output shape for every existing caller is
riskier than adding a new one. Merging them is a natural next step once the bundle format has
been used in practice.
**Alternatives rejected:** Making `eng adapter prompt` call `eng context bundle` internally now
— premature; the bundle format hasn't been validated yet.
**Decided by:** Planner
**Status:** Active

---

## Superseded decisions

_None yet._
