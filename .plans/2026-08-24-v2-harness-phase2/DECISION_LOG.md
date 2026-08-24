# Decision Log — V2 Harness Phase 2

> Read alongside `spec.md`'s "Design decisions" section for full depth. This file exists so
> future sprints can find "what's decided" without re-reading the full spec.

---

## Decisions

### 2026-08-24 — `plan.yaml` sidecar carries all per-plan machine state
**Context:** Phase 2 needs to track git drift, risk level, write scope, and retry counters per
plan. `spec.md`/`tasks.md`/`tests.md` are Planner-authored prose; mixing tool-managed state
into them risks a Planner edit accidentally clobbering that state.
**Decision:** New file `.plans/<feature>/plan.yaml`, written by `eng plan new`, updated by
`eng plan drift`/`retry` and `eng verify`. `spec.md`/`tasks.md`/`tests.md` format is unchanged.
**Reasoning:** Mirrors the precedent Phase 1 already set with `.agent/project.yaml` — tool
state lives in its own YAML file, human-authored content stays in Markdown.
**Alternatives rejected:**
- YAML frontmatter inside `spec.md` — conflates Planner prose with tool state
- Three separate state files (drift/retry/writescope) — no benefit at this scale
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Plan-level write scope only; per-task write-sets deferred
**Context:** Requirement 19 asks for per-task `writes:` glob declarations.
**Decision:** `plan.yaml` declares one `write_scope` list for the whole plan, sourced from
`spec.md`'s existing "Affected files" table. No per-task write-set parsing in this phase.
**Reasoning:** Per-task write-sets need a structural parser for `tasks.md`'s markdown — real
new work, not a config addition. A plan-level scope answers the actual Phase 2 question
("did anything change outside what this plan said it would touch") without that parser.
**Alternatives rejected:** Building the per-task parser now — correctly identified as more
invasive than the rest of this phase; moved to "later improvements."
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Plan Reviewer and Verifier are methodology + report files, not enforcement gates
**Context:** Neither role has anything to physically stop an Executor from proceeding — this
harness has no orchestration engine; every role is a human-directed AI session.
**Decision:** `review.md` and `verify-report.md` are the enforcement surface — a distinct file
with a clear verdict and (for `eng verify`) a non-zero exit code. Nothing blocks a session from
ignoring them.
**Reasoning:** Real call-and-response enforcement requires the orchestration layer the user's
scope constraint explicitly defers ("complex remote orchestration" is out of scope). Making the
verdict impossible to miss is the correctly-sized foundation for this phase.
**Alternatives rejected:** A stateful workflow engine sequencing the roles — out of scope.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — `eng triage` is a keyword heuristic, explicitly labeled non-authoritative
**Context:** Requirement 1 (Task Triage) asks for risk classification and routing. Phase 1's
own spec deferred the real Skill Router until Task Triage exists.
**Decision:** `eng triage "<text>"` matches a small keyword table (quick-fix/bug/architecture/
high-risk, default feature) and always prints "(heuristic hint only — the Planner makes the
final call)". `harness/core/triage/METHOD.md` documents the Planner's actual responsibility.
**Reasoning:** A real classifier needs the Skill Router; building it now duplicates work.
An honest, zero-risk heuristic still gives `plan.yaml`'s `risk_level` a canonical vocabulary.
**Alternatives rejected:** No tool at all, methodology-only — rejected because a heuristic that
is honest about its limits is strictly more useful than nothing.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Hooks: YAML command table, full-replace project override (no merge)
**Context:** The user's brief specifies six lifecycle stages with named hooks per stage.
**Decision:** `harness/hooks/default.yaml` defines stages → hook names → shell commands (or ""
for AI-performed steps). A project's `.agent/hooks.yaml`, if present, fully replaces the
default — no partial merge.
**Reasoning:** Matches the requested shape while keeping hook definitions in config, not
prompts. Full-replace avoids ambiguous merge semantics (append vs. replace-per-stage vs.
replace-per-command) that a v1 of this mechanism doesn't need to solve yet.
**Alternatives rejected:** Deep-merge — more flexible, more ambiguous, no current use case
demands it.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — `require_approval` and task-level approval markers are declared, not enforced
**Context:** Requirement 9 (Human Approval Gates) explicitly says not to build MCP/device
control yet.
**Decision:** `.agent/project.yaml` gets an optional `require_approval:` category list;
`tasks.md` gets an optional `**Requires approval:**` field. `harness/core/executor/METHOD.md`
documents this as a hard stop in the Executor's behavioral contract — the same mechanism
already used for "stop on test failure."
**Reasoning:** Real enforcement needs a policy engine in front of actual tool/device access,
explicitly out of scope. A documented behavioral contract is the correctly-sized foundation,
consistent with how this harness already handles "stop and report" everywhere else.
**Alternatives rejected:** An `eng approve` gating command now — nothing yet to gate.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Preliminary fix: `DetectMode` must surface `.agent/project.yaml` parse errors
**Context:** Discovered while reviewing Phase 1 code before starting Phase 2: a corrupt or
type-mismatched `project.yaml` was silently treated as valid `hybrid` mode.
**Decision:** Fix `DetectMode`/`doctor` to report a distinct "project.yaml exists but failed to
parse: <err>" state instead of falling through to `hybrid` silently. Done as Task 1, before any
Phase 2 feature work.
**Reasoning:** Phase 2 adds four new fields to the same struct, read by more commands
(`eng verify`, `eng plan retry`) — a silently swallowed parse error becomes a real correctness
risk once those commands start trusting the file's contents.
**Alternatives rejected:** Leaving it for a later cleanup pass — rejected because Phase 2
directly increases the blast radius of this exact bug.
**Decided by:** Planner
**Status:** Active

---

## Superseded decisions

_None yet._
