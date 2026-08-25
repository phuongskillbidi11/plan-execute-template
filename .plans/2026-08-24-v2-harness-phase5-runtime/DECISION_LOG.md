# Decision Log — V2 Harness Phase 5 (Runtime Integration)

> Read alongside `spec.md`'s "Design decisions" section for full depth.

---

## Decisions

### 2026-08-25 — Natural-language routing is a new `METHOD.md`, not new orchestration code
**Context:** The brief asks to move from "tools exist" to "the user just describes a
requirement" without building a process that drives Claude Code's conversation.
**Decision:** `harness/core/runtime/METHOD.md` documents the exact `eng` command sequence a
session follows for a plain-language request — no code executes it.
**Reasoning:** Every other role in this harness already works this way; the gap was a missing
top-level instruction, not a missing program. Scripting Claude Code's turns or auto-driving it
non-interactively would repeat the exact unattended-code-change risk Phase 3 already rejected.
**Alternatives rejected:** A new orchestrating Go process — no API exists for it to drive an
interactive Claude Code session's turns, and building one is out of scope.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — `eng adapter prompt` now folds in `eng context bundle`
**Context:** Phase 4's Decision 7 deliberately kept these separate "until the bundle format has
been used for a while."
**Decision:** `buildContextBundle` is factored out as a pure function; both `eng context
bundle` and `eng adapter prompt` call it.
**Reasoning:** The bundle format has now been exercised end-to-end across Phase 4's own tests;
this is exactly the condition Phase 4 set for merging them.
**Alternatives rejected:** Keeping them separate indefinitely — contradicts Phase 4's own
stated intent to merge once proven.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Quick Fix is a fast-path state transition, not a parallel state machine
**Context:** Every plan today must produce the full spec/tasks/tests trio before executing,
regardless of size.
**Decision:** `TRIAGED` + `risk_level: quick-fix` → `EXECUTING` directly, once a minimal
template-based plan exists; rejoins the normal `EXECUTING → VERIFYING → COMPLETED` path.
`eng verify` appends one compact structured event on `PASS`.
**Reasoning:** Reuses `plan.yaml`/`events.jsonl`/`Decide`/`eng verify` entirely — only new
pieces are a lighter template set and one `Decide` branch.
**Alternatives rejected:** Bypassing `plan.yaml`/`events.jsonl` for quick fixes — leaves no
structured evidence, violating the explicit persistence requirement.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Escalation is mechanical; fleshing out the plan stays a human/AI job
**Context:** A quick fix must be able to escalate once source inspection reveals broader
impact.
**Decision:** `eng plan escalate <dir> --to <level>` changes `risk_level`, resets `state` to
`TRIAGED`, records an event — it never regenerates `spec.md`/`tasks.md`/`tests.md` content.
**Reasoning:** Auto-regenerating plan content after work has already started would destroy it.
The mechanical fact-update belongs in code; deciding what the fuller plan should say does not.
**Alternatives rejected:** Auto-copying full templates over quick-fix files on escalation —
destructive.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Spec approval and execution approval are distinct, separately-tracked concepts
**Context:** Requirement 6 explicitly warns against conflating "the requirements are approved"
with "this risky execution is approved."
**Decision:** New states `NEEDS_SPEC_APPROVAL`/`SPEC_APPROVED`, new `plan.yaml` fields
`spec_approved_at`/`spec_approved_by`, new command `eng plan approve-spec` — entirely separate
from the existing `requires_approval`/`approved_at`/`approved_by`/`eng plan approve`.
**Reasoning:** These already answer different questions asked by different people at different
times; separate fields keep that visible in the data, not just in prose.
**Alternatives rejected:** Reusing the existing approval gate for spec confirmation too —
exactly the ambiguity the requirement warns against.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — `planning_mode` defaults to `auto_plan` when unset; only new `eng init` runs opt into `spec_first`
**Context:** The brief wants spec-first as the new default "unless there is a strong
compatibility reason not to" — and also demands Phase 3 state-machine behavior survive
unchanged for every existing plan.
**Decision:** `Workflow.PlanningModeOrDefault()` returns `"auto_plan"` when the field is empty
(every plan created under Phases 1–4). `eng init` writes `planning_mode: spec_first` only when
initializing a project that has never run `eng init` before; an existing `.agent/project.yaml`
is never touched (unchanged Phase 1 rule).
**Reasoning:** Satisfies both halves of the brief: zero behavior change for anything already in
flight; the new, more V1-faithful default applies only going forward.
**Alternatives rejected:** Defaulting empty `planning_mode` to `spec_first` — would silently
change the `TRIAGED` transition for every existing plan, including this repository's own
Phase 3/4 test fixtures.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Context-bundle fallback-to-full is one bounded retry, never source-tree expansion, Planner role only for MVP
**Context:** Requirement 10 forbids "recursively load the entire repository as the default
fallback" while wanting some expansion signal when nothing matches.
**Decision:** In `eng context bundle planner`, zero skills + zero doc sections matched → retry
once with `strategy: full` (uncapped counts, same two doc files, same skill set) within the
same call; flag `fallback_to_full: true` in the manifest. Not yet duplicated into the
`plan-reviewer`/`executor` branches.
**Reasoning:** "Full" was already bounded in Phase 4 (uncapped, not unscoped) — reusing it adds
no new surface and stays inside the existing filesystem/metadata boundary. Planner is where
"found nothing" is most consequential; triplicating the same check into other roles before
observing a real need there is premature.
**Alternatives rejected:** Falling back to a raw repository grep — genuinely unscoped.
Implementing the same check in all four role branches immediately — unobserved need elsewhere.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Log retention never deletes the single most recent log file
**Context:** "Do not delete logs needed by an active plan" would, done fully, require
cross-plan log-reference tracking.
**Decision:** `eng logs prune` deletes oldest-first beyond `max_files`/`max_age_days`/
`max_total_mb`, but never the most recently modified file.
**Reasoning:** Cheap, effective approximation — the log an active plan just referenced is, by
construction, the most recent one. Full reference tracking is real work for a benefit not
needed yet.
**Alternatives rejected:** Full cross-plan reference tracking — deferred explicitly.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Capability Registry: `Describe`/`DescribeAll` additive; `Detect`/`DetectAll` untouched
**Context:** Phase 3's `Detect`/`DetectAll` are already called by `doctor.go` and
`capabilities_cmd.go`.
**Decision:** New `Capability` struct + `Describe`/`DescribeAll` functions; existing functions
and their call sites are unmodified.
**Reasoning:** Zero risk to already-shipped call sites.
**Alternatives rejected:** Changing `DetectAll`'s return type — would require touching every
existing caller for a schema few of them need.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Agent Adapter and Tool Adapter are separate interfaces; Tool Router is a pure filter
**Context:** The brief explicitly warns against mixing "launches a coding agent" with "exposes
an external capability."
**Decision:** `internal/agent.Adapter` (Phase 3) unchanged. New `internal/tooladapter.Adapter`
with exactly one reference implementation (`GitAdapter`). New `internal/toolrouter.Filter`, a
pure function with no session to expose into.
**Reasoning:** Keeps the two concepts structurally separate, matches "foundation only, do not
overbuild."
**Alternatives rejected:** One unified adapter interface — the exact mixing the brief warns
against.
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — Role-based tool permissions: static table, reporting-only
**Context:** No real tool-invocation boundary exists yet for permissions to gate.
**Decision:** `internal/agent.RolePermissions`/`RoleMayUse`, consumed only by `eng capabilities
list --role <role>` for filtering/annotation.
**Reasoning:** Prepares the architecture without enforcing against something that doesn't
exist.
**Alternatives rejected:** Enforcing against `internal/tooladapter` calls now — nothing
meaningful to gate with only `GitAdapter` (git access is already unconditional everywhere).
**Decided by:** Planner
**Status:** Active

### 2026-08-25 — `TasksAndTestsReady` reuses `spec.md`'s `[Feature Name]` placeholder marker
**Context:** Caught during this plan's own design, before implementation: `eng plan new`
copies full non-empty template content for `tasks.md`/`tests.md` too, so an existence check
alone would mark them "ready" the instant a plan is scaffolded.
**Decision:** The new `SPEC_APPROVED → PLANNED` gate checks both files for the same
`[Feature Name]` marker `filesReady` already uses for `spec.md`.
**Reasoning:** `.plans/_template/tasks.md` and `tests.md` already use the identical placeholder
string — the direct fix, not a new mechanism. Same defect class as Phase 3's own
`filesReady` fix.
**Alternatives rejected:** A distinct marker per file — unnecessary divergence from an
already-shared convention.
**Decided by:** Planner
**Status:** Active

---

## Superseded decisions

_None yet._
