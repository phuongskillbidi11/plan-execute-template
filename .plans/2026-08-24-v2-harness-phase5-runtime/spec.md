# Spec — V2 Harness Phase 5 (Runtime Integration)

> **Planner note:** Read `.plans/2026-08-24-v2-harness-foundation/`, `.plans/2026-08-24-v2-harness-phase2/`,
> `.plans/2026-08-24-v2-harness-phase3/`, and `.plans/2026-08-24-v2-harness-phase4-context/`
> (`spec.md` + `DECISION_LOG.md` of each) before this file. Phase 5 does not build a new
> orchestrating process — it closes the gap between "the primitives exist" and "a Claude Code
> session actually uses them by default."

---

## Goal

Phases 1–4 built triage, a lifecycle state machine, an agent adapter, and a context manager —
but nothing tells a Claude Code session to *use them automatically* for a natural-language
request, `eng context bundle` output never reaches `eng adapter prompt`, and the state machine
has no path shorter than "write a full spec/tasks/tests trio" or gentler than "one big approval
gate that means both 'the requirements are right' and 'this is safe to execute.'" Phase 5 closes
exactly these gaps — no new orchestration binary, no LLM-driven state transitions.

**Done looks like:** `eng adapter prompt <role> <plan-dir>` now prints the role's `METHOD.md`
**and** its curated context bundle in one call — the context manager's output becomes the
prepared context an agent session actually reads, not a separate manual step. A new
`harness/core/runtime/METHOD.md` documents the exact command sequence a Claude Code session
runs automatically when given a natural-language request, so a user can `cd project && eng
start` and just describe what they want. A genuinely small, low-risk request classified
`quick-fix` skips straight to execution with a minimal one-line plan and a compact structured
event — no `spec.md`/`tasks.md`/`tests.md` trio required — and can escalate back to the full
lifecycle the moment it turns out to be bigger than it looked. A normal feature request stops
after `spec.md` is drafted and waits for **explicit human approval of the requirements** —
a distinct, separately-tracked concept from the existing "approve this risky execution" gate —
before `tasks.md`/`tests.md` are even generated, restoring the V1 template's original
spec-then-confirm behavior. `.agent/logs/` gains bounded retention. The Capability Registry and
a new, deliberately unpopulated Tool Adapter interface give future MCP-style adapters a home
without building any of them. Every existing `eng` command, and the exact state-machine
behavior of every plan created before this phase, is unchanged.

---

## Background — what's already sufficient vs. what's missing

**Already sufficient, reused as-is:**
- `eng workflow start/status/advance` (Phase 3) already does "triage → create a plan → report
  status" in one command — the natural-language entry point already exists; it just isn't
  documented as *the* default entry point a session should reach for automatically.
- `eng context bundle` (Phase 4) already composes role-specific context; it just isn't wired
  into anything an agent session is told to call as part of preparing a role.
- `plan.yaml` + `events.jsonl` (Phase 3) already model exactly the "small persistent record"
  Quick Fix needs — no new persistence mechanism required, only a new compact event shape and
  a fast-path state transition.
- `internal/workflow.Decide` (Phase 3) is already a pure, deterministic function taking
  `Facts` — extending it with new states/facts keeps "mechanical state, not LLM interpretation"
  intact by construction; there is nothing to "keep deterministic" that isn't already so.
- `internal/capabilities` (Phase 3) already detects local tools — Phase 5 extends its output
  shape, not its detection mechanism.

**Missing, and why each is now in scope:**
- **No routing document.** Every role's `METHOD.md` says what *that role* does once invoked,
  but nothing says "when a human describes a requirement in plain language, run these `eng`
  commands in this order automatically." This is the actual, single biggest gap between "tools
  exist" and "the tools are the default experience."
- **`eng adapter prompt` and `eng context bundle` are still separate**, exactly as Phase 4's
  own Decision 7 predicted and deliberately deferred — "wiring them together is a natural next
  step once the bundle format has been used for a while." It has been; Phase 4's own tests
  exercised it end-to-end.
- **No workflow shorter than the full plan trio.** Every plan, regardless of risk level, must
  produce `spec.md`+`tasks.md`+`tests.md` before `EXECUTING` — there is no genuinely lightweight
  path for a one-line fix, and no escalation mechanism if a request initially looks small.
- **No distinction between "the requirements are approved" and "this execution is approved."**
  Phase 3's `requires_approval`/`eng plan approve` gate is about dangerous *execution*
  (high-risk, firmware, production). Nothing today lets a Planner stop after `spec.md` and ask
  "is this the right thing to build," which was the V1 template's own original, and explicitly
  requested-back, default behavior for feature work.
- **No context-selection fallback.** If `eng context bundle` selects nothing, today it just
  prints an empty section — there's no defined "this might mean the request needs different
  input" signal.
- **`.agent/logs/` has no retention.** Phase 4 introduced it; nothing bounds its growth.
- **Capability Registry is availability-only.** No version, no provider, no role-permission
  concept — needed as the foundation Requirement 12/17 ask for, still local-detection-only.
- **No Tool/MCP adapter boundary exists at all**, and the one Agent Adapter that does exist
  lives at a path (`harness/adapters/claude-code/`) that doesn't distinguish "launches an
  agent" from "exposes an external capability" — a distinction the brief is explicit about
  keeping separate.

---

## Design decisions

### Decision 1 — Natural-language routing is a new top-level `METHOD.md`, not new orchestration code
**Chosen:** `harness/core/runtime/METHOD.md` documents the exact command sequence a Claude Code
session follows when a human describes a requirement in plain language: run `eng workflow
start "<text>"`, read the reported risk level, follow either the Quick Fix or Spec-First path
(Decisions 3/5), and at every state use `eng workflow advance` for the mechanical transition
and `eng adapter prompt <role> <plan-dir>` for the role's prepared context — never skip a
printed gate.
**Why:** Every other role in this harness already works this way — a `METHOD.md` a session
reads and follows, with `eng` commands doing the mechanical bookkeeping. The "missing runtime"
the brief describes is not a missing *program*; it's a missing *instruction* at the top of the
hierarchy, exactly the same kind of gap Phase 3 closed for the Executor's stop conditions and
Phase 4 closed for context loading.
**Rejected:** A new Go process that drives a Claude Code session's turns programmatically —
this would require either scripting Claude Code's own conversation loop (no such API exists in
this harness's design) or a non-interactive `claude -p` auto-drive, which Phase 3's Decision 3
already rejected for the same unattended-code-change risk, unchanged here.

### Decision 2 — `eng adapter prompt` now always also builds the context bundle
**Chosen:** `contextBundle`'s composition logic (Phase 4) is refactored into a pure
`buildContextBundle(role, planDir, request) (string, error)` function. `eng context bundle`
(unchanged CLI surface) calls it and prints the result; `eng adapter prompt` (Phase 3) now
*also* calls it and appends its output after the role's `METHOD.md` content, in the same
single command.
**Why:** This is the literal ask of Requirement 1 — "the adapter should consume the Context
Manager output as the authoritative prepared context for that role" — and it's exactly the
"once the bundle format has been used for a while" condition Phase 4's Decision 7 set for
doing this.
**Rejected:** Keeping them permanently separate — Phase 4 explicitly framed that as temporary,
pending proof the bundle format works, which the intervening testing has now provided.

### Decision 3 — Quick Fix is a fast-path state transition, not a parallel state machine
**Chosen:** `TRIAGED` with `risk_level: quick-fix` transitions directly to `EXECUTING` once a
*minimal* plan (new `harness/templates/quickfix/` — one-line `spec.md`, one-task `tasks.md`,
one-command `tests.md`) is present, skipping `PLANNED`/`REVIEWED`/`APPROVED`/spec-approval
entirely. From `EXECUTING` onward it rejoins the existing `EXECUTING → VERIFYING → COMPLETED`
path unchanged. On `PASS`, `eng verify` appends one compact structured event
(`{"type":"quick_fix","summary":...,"files":[...],"verification":"PASS"}`, files taken from
the git diff it already computes) instead of a full plan document.
**Why:** Reuses every existing mechanism (`plan.yaml`, `events.jsonl`, `Decide`, `eng verify`)
— the only new things are a lighter template set and one new `Decide` branch. A parallel state
machine would duplicate the retry/verify/event logic Phase 2–3 already built correctly.
**Rejected:** Skipping `plan.yaml`/`events.jsonl` entirely for quick fixes and just editing
files — would leave zero structured evidence, violating the explicit "must leave enough
structured evidence for future agents/sessions" requirement.

### Decision 4 — Escalation is a mechanical command; fleshing out the plan stays a human/AI responsibility
**Chosen:** `eng plan escalate <dir> --to bug|feature|architecture|high-risk [--reason "..."]`
changes `risk_level` and resets `state` to `TRIAGED`, records an `escalated` event, and prints
a reminder to flesh out `spec.md`/`tasks.md`/`tests.md` into the full format. It does **not**
auto-regenerate plan content.
**Why:** Regenerating `spec.md` after a human/AI has already started writing quick-fix notes in
it would silently destroy work. The mechanical, deterministic part (the state/risk-level fact)
belongs in code, per Requirement 22; deciding *what the fuller plan should say* is exactly the
kind of judgment call this harness has never automated.
**Rejected:** Auto-copying the full templates over the quick-fix files on escalation —
destructive, and contradicts "AI may propose facts... but mechanical gates must remain
machine-readable" by conflating a fact-update with content generation.

### Decision 5 — Spec approval and execution approval are two distinct, separately-tracked concepts
**Chosen:** Two new states, `NEEDS_SPEC_APPROVAL` and `SPEC_APPROVED`, sit between `TRIAGED`
and `PLANNED` when `planning_mode: spec_first` (Decision 6). A new `eng plan approve-spec <dir>
[--by name]` command sets new `plan.yaml` fields `spec_approved_at`/`spec_approved_by` —
entirely separate from Phase 3's existing `approved_at`/`approved_by`, which continue to mean
only "this risky execution may proceed."
**Why:** Requirement 6 is explicit that conflating these would create ambiguity. They already
answer different questions asked by different people at different times ("is this the right
thing to build" vs. "is it safe to run this now") — separate fields and separate commands keep
that distinction visible in `plan.yaml` itself, not just in prose.
**Rejected:** Reusing `requires_approval`/`eng plan approve` for spec confirmation too — exactly
the ambiguity Requirement 6 warns against; a `feature`-risk plan would then need `requires_
approval: true` just to get spec confirmation, incorrectly implying it's also execution-risky.

### Decision 6 — `planning_mode` defaults to `auto_plan` (today's behavior) when unset; only brand-new `eng init` runs opt into `spec_first`
**Chosen:** `.agent/project.yaml`'s existing `workflow` block gains `planning_mode` (`""` |
`auto_plan` | `spec_first`) and `require_spec_approval` (`*bool`, default `true`).
`Workflow.PlanningModeOrDefault()` returns `"auto_plan"` when the field is empty — the exact
behavior every plan created under Phases 1–4 already has. `eng init` (Phase 5 onward) writes
`planning_mode: spec_first` explicitly for every **new** project it initializes; it never
touches an existing `.agent/project.yaml` (unchanged from Phase 1's "already exists — not
overwritten" rule).
**Why:** This is the one place Phase 5's own compatibility requirement and its UX requirement
would conflict if resolved carelessly: Requirement 24 demands "Phase 3 state machine behavior"
survive unchanged, while Requirement 7 demands spec-first be the *default* "unless there is a
strong compatibility reason not to" — and there is one, precisely for plans that already exist.
Defaulting absence to `auto_plan` satisfies both: zero existing plan or project changes
behavior; only a project that has never run `eng init` before gets the new, arguably-more-
correct-to-V1 default.
**Rejected:** Defaulting empty `planning_mode` to `spec_first` — would silently change the
`TRIAGED` transition for every plan already in flight anywhere this harness has been installed,
including this very repository's own Phase 3/4 test fixtures.

### Decision 7 — Context-bundle fallback-to-full is one bounded retry, never source-tree expansion, wired into the Planner role for MVP
**Chosen:** If `eng context bundle planner`'s selective pass matches zero skills and zero doc
sections, it retries once, in the same call, with `strategy: full` (uncapped counts, same two
doc files and the same resolved-skill set — never raw source files), and records
`fallback_to_full: true` in `context-manifest.yaml`. This MVP wires the check into the
`planner` branch only — the role for which "found nothing" is most consequential (it's the
role deciding what to build). Extending the identical check to `plan-reviewer`'s project-context
lookup and `executor`'s skill lookup is mechanical (the same two-line emptiness check, reused)
and left as a later improvement rather than triplicated now for a case not yet observed in
practice.
**Why:** Requirement 10 explicitly forbids "recursively load the entire repository as the
default fallback" while still wanting *some* expansion signal. "Full" was already a bounded
concept in Phase 4 (uncapped, not unscoped) — reusing it costs nothing new and stays inside the
same filesystem/metadata surface. Scoping to one role first avoids speculative duplication.
**Rejected:** Falling back to a raw grep across the whole repository — genuinely unscoped, and
exactly what the requirement rules out. Also rejected: implementing the identical check in all
four role branches immediately — three of those call sites would be exercising unobserved need.

### Decision 8 — Log retention never deletes the single most recent log file
**Chosen:** `eng logs prune` (and an automatic prune after every `eng verify` run) deletes
`.agent/logs/*.log` files beyond `max_files`/`max_age_days`/`max_total_mb` (new fields on
`contextcfg.Config`), oldest-first, but the single most recently modified log is never deleted
regardless of any limit.
**Why:** "Do not delete logs needed by an active plan" would, done fully, require tracking
which `verify-report.md` across every non-terminal plan references which log path — real work
for a benefit this MVP doesn't need yet. Never deleting the most recent file is a cheap,
effective approximation: the log an active plan just referenced is, by construction, the most
recent one.
**Rejected:** Full cross-plan log-reference tracking — flagged explicitly as a later
improvement rather than built speculatively now.

### Decision 9 — Capability Registry gains `Describe`/`DescribeAll`, purely additive to `Detect`/`DetectAll`
**Chosen:** New `Capability{Name, Available, Provider, Version}` struct and `Describe(name)`/
`DescribeAll()` functions. `Detect(name) bool` and `DetectAll() map[string]bool` (Phase 3) are
unchanged — no existing caller (`doctor.go`, the Phase 3 `capabilities_cmd.go` body) is touched.
**Why:** Zero risk to already-shipped call sites; the richer schema is additive surface for the
new `--verbose`/`--role` flags on `eng capabilities list`.
**Rejected:** Changing `DetectAll`'s return type — would require touching every existing
caller for a schema few of them need.

### Decision 10 — Agent Adapter and Tool Adapter are separate interfaces in separate packages; the Tool Router is a pure filter with nothing to expose into yet
**Chosen:** `internal/agent.Adapter` (Phase 3, unchanged) stays "launches/talks to a coding
agent." A new `internal/tooladapter.Adapter` interface (`Name`, `Capability`, `Available`,
`PermissionLevel`, `Doctor`) is the foundation for future external-tool adapters, with exactly
one reference implementation (`GitAdapter`) to prove it compiles and is testable — not a real
capability gate. `internal/toolrouter.Filter` is a pure function narrowing a list of adapters
to those matching required capabilities and currently available; it exposes nothing to any
session because no session object exists in this architecture to expose into.
**Why:** The brief is explicit that these are different concepts and must not be mixed. Keeping
the Tool Router a plain filter function (not a service, not a registry with lifecycle) matches
"do not overbuild" and "foundation only" instructions exactly.
**Rejected:** A single unified `Adapter` interface covering both agents and tools — the exact
mixing the brief warns against; an agent adapter's `RolePrompt` and a tool adapter's
`PermissionLevel` answer unrelated questions.

### Decision 11 — Role-based tool permissions are a static, reporting-only table
**Chosen:** `internal/agent.RolePermissions map[Role][]string` and `RoleMayUse(role,
capability) bool`, consumed by `eng capabilities list --role <role>` to filter/annotate output.
Nothing currently blocks a role from actually invoking a tool with this table — there is no
tool-invocation boundary in this architecture yet for it to gate.
**Why:** "Prepare the architecture for role-based permissions," not enforce it against
something that doesn't exist yet. A visible, testable table is the correct-sized foundation;
real enforcement needs the Tool Adapter execution path this phase deliberately doesn't build.
**Rejected:** Building enforcement now against `internal/tooladapter` calls — there's exactly
one trivial reference adapter (`git`, already unconditionally accessible everywhere in this
harness) with nothing meaningful to gate.

### Decision 12 — `tasks.md`/`tests.md` share `spec.md`'s `[Feature Name]` placeholder marker; the new `TasksAndTestsReady` check reuses it
**Chosen:** The new fact needed for `SPEC_APPROVED → PLANNED` (are `tasks.md`/`tests.md`
actually written, not just template-scaffolded) checks for the literal `[Feature Name]` string
in both files, exactly the same marker Phase 3's `filesReady` already uses for `spec.md`.
**Why:** Caught during this plan's own design, before implementation: `eng plan new` copies
full non-empty template content for every file, including `tasks.md`/`tests.md`, so an
existence-and-non-empty check alone would consider them "ready" the instant the plan is
scaffolded — precisely the bug Phase 3's own Decision (spec.md's placeholder check) already
fixed once for `spec.md` alone. Extending the same check is the direct fix, not a new
mechanism.
**Rejected:** A different marker per file — `.plans/_template/tasks.md` and `tests.md` already
use the identical `[Feature Name]` string; inventing a second marker would be needless
divergence.

---

## Responsibilities (extending Phase 3/4's table)

| Role | Phase 5 addition |
|---|---|
| **Runtime Router (new, not an AI role)** | The documented protocol (`harness/core/runtime/METHOD.md`) any session follows for a natural-language request — no code runs it; a Claude Code session reads and follows it, the same way every other role already reads its own `METHOD.md` |
| **Triage** | Unchanged mechanism; now the entry point documented as *the* default, via `eng workflow start` |
| **Planner** | In `spec_first` mode (the new default for new projects), stops after `spec.md` and waits for `eng plan approve-spec` before writing `tasks.md`/`tests.md` |
| **Plan Reviewer / Executor / Verifier** | Now receive their context via `eng adapter prompt`, which folds in `eng context bundle` automatically (Decision 2) |
| **Agent Adapter** | Unchanged interface; `ClaudeCodeAdapter` remains the only implementation |
| **Tool Adapter (new, foundation only)** | `GitAdapter` is the only implementation; nothing else is built |

---

## Scope

### In scope (Phase 5 MVP)
- `harness/core/runtime/METHOD.md` — the natural-language routing protocol
- `buildContextBundle` refactor; `eng adapter prompt` folds in the context bundle
- `harness/templates/quickfix/{spec.md,tasks.md,tests.md,plan.yaml}` — minimal templates
- `eng plan new --risk quick-fix` uses the minimal templates
- `internal/workflow`: `StateNeedsSpecApproval`, `StateSpecApproved`; `Facts.{IsQuickFix,
  PlanningMode, SpecReady, SpecApproved, RequireSpecApproval, TasksAndTestsReady}`; `Decide`'s
  `TRIAGED` case restructured per Decisions 3/5/6 (auto_plan path byte-for-byte unchanged)
- `project.Workflow`: `PlanningMode string`, `RequireSpecApproval *bool` +
  `PlanningModeOrDefault()`/`RequireSpecApprovalOrDefault()`; `eng init` writes
  `planning_mode: spec_first` for new projects only
- `planmeta.Meta`: `SpecApprovedAt`/`SpecApprovedBy`; `eng plan approve-spec`;
  `AppendStructuredEvent` for the compact quick-fix event
- `eng plan escalate <dir> --to <level> [--reason "..."]`
- `eng verify`'s quick-fix structured event + one bounded fallback-to-full retry in
  `buildContextBundle`
- `contextcfg.Config`: `MaxLogFiles`, `MaxLogAgeDays`, `MaxLogTotalMB`; `internal/logprune`;
  automatic prune after `eng verify`; standalone `eng logs prune [--dry-run]`
- `internal/capabilities`: `Capability`, `Describe`, `DescribeAll`; `eng capabilities list
  --verbose --role <role>`
- `harness/adapters/agents/claude-code/ADAPTER.md` (moved from `harness/adapters/claude-code/`,
  nothing in code references the old path); `harness/adapters/tools/README.md` placeholder
- `internal/tooladapter` (`Adapter` interface + `GitAdapter`); `internal/toolrouter.Filter`
- `internal/agent.RolePermissions`/`RoleMayUse`
- `eng start`: safe first-run handling (`--init` opt-in flag; print-only default for
  uninitialized projects); a "consult the runtime method doc" banner before launching
- `eng context manifest <plan-dir>` — pretty-print an existing manifest

### Out of scope (explicitly excluded, with reasons)
- **Any code that drives a Claude Code session's conversation turns programmatically** —
  Decision 1; the routing document is followed by the session, not executed by `eng`.
- **Real Tool Adapter implementations beyond `GitAdapter`** — Requirement 15 explicitly says
  define, don't populate; no docker/ssh/github/database adapters this phase.
- **Any live PLC/Modbus/OPC UA/industrial control, firmware flashing, or production
  deployment adapter** — explicitly excluded by the user's own scope constraint.
- **Enforcement of role-based tool permissions against a real invocation** — Decision 11;
  nothing to enforce against yet.
- **Full cross-plan log-reference tracking** — Decision 8; the most-recent-file heuristic
  covers the practical case.
- **Semantic/symbol-level classification improvement** — Requirement 21 asks only to
  *inspect* whether combining more signals helps; this plan adds one small, mechanical signal
  (a `docs/gotchas.md` match never lowers, only holds-or-raises, the suggested level) and
  defers anything requiring source-symbol discovery.
- **A public plugin/MCP marketplace, vector search, distributed/parallel agent execution,
  cloud telemetry** — explicitly excluded by the user's own scope constraint, same as every
  prior phase.

### Later improvements (explicitly deferred, not designed here)
- Auto-injecting a runtime-method pointer into a project's own `CLAUDE.md`/`AGENTS.md` at
  `eng init` time — deliberately not done now, out of respect for those files' user-owned
  status (unchanged since Phase 1)
- Real Tool Adapters (docker, ssh, github) once a concrete need justifies one
- Enforcing role-based permissions against actual tool invocations, once a real invocation
  boundary exists
- Full log-reference-aware retention
- A richer, per-category `context-manifest.yaml` schema if the current flat one proves
  insufficient in practice
- Extending the fallback-to-full retry (Decision 7) from the `planner` role to
  `plan-reviewer`/`executor` if empty-match cases are observed there in practice

---

## Affected files

| File | Change type | Reason |
|---|---|---|
| `harness/core/runtime/METHOD.md` | Create | Natural-language routing protocol |
| `cli/context_cmd.go` | Modify | Refactor to `buildContextBundle`; add `eng context manifest`; bounded fallback-to-full |
| `cli/adapter_cmd.go` | Modify | Fold `buildContextBundle` output into `eng adapter prompt` |
| `harness/templates/quickfix/*` | Create | Minimal spec/tasks/tests/plan.yaml templates |
| `cli/plan_cmd.go` | Modify | `--risk quick-fix` template branch; `eng plan approve-spec`; `eng plan escalate` |
| `cli/internal/workflow/workflow.go` | Modify | New states, new `Facts` fields, restructured `TRIAGED` case |
| `cli/internal/workflow/workflow_test.go` | Modify | New tests; existing tests unchanged (zero-value Facts still hit the auto_plan path) |
| `cli/internal/project/project.go` | Modify | `Workflow.PlanningMode`/`RequireSpecApproval` + accessors |
| `cli/internal/planmeta/planmeta.go` | Modify | `SpecApprovedAt`/`SpecApprovedBy`; `AppendStructuredEvent` |
| `cli/init_cmd.go` | Modify | Write `planning_mode: spec_first` for new projects |
| `cli/workflow_cmd.go` | Modify | `gatherFacts` computes the new facts; `specReady`/`tasksAndTestsReady` helpers |
| `cli/verify_cmd.go` | Modify | Quick-fix structured event; call `logprune.Prune` after `writeFullLog` |
| `cli/internal/logprune/logprune.go` | Create | Retention logic |
| `cli/logs_cmd.go` | Create | `eng logs prune [--dry-run]` |
| `cli/internal/contextcfg/contextcfg.go` | Modify | `MaxLogFiles`/`MaxLogAgeDays`/`MaxLogTotalMB` |
| `cli/internal/capabilities/capabilities.go` | Modify | `Capability`, `Describe`, `DescribeAll` |
| `cli/capabilities_cmd.go` | Modify | `--verbose`/`--role` flags |
| `cli/internal/agent/permissions.go` | Create | `RolePermissions`/`RoleMayUse` |
| `cli/internal/tooladapter/tooladapter.go` | Create | `Adapter` interface + `GitAdapter` |
| `cli/internal/toolrouter/toolrouter.go` | Create | `Filter` |
| `harness/adapters/agents/claude-code/ADAPTER.md` | Create (moved) | Adapter-category separation |
| `harness/adapters/tools/README.md` | Create | Placeholder for future tool adapters |
| `cli/start_cmd.go` | Modify | Safe first-run handling; runtime-method banner |
| `cli/main.go` | Modify | Dispatch `eng logs`; wire new flags |
| `cli/triage_cmd.go` | Modify | Gotcha-match never-lowers-only-raises signal |
| `harness/VERSION` | Modify | Bump `0.4.0-phase4-context` → `0.5.0-phase5-runtime` |
| `docs/src-map.md`, `docs/gotchas.md`, `README.md`, `ROADMAP.md` | Modify | Docs integration (last task) |

---

## Risks and unknowns

- **`planning_mode` default resolution is the highest-compatibility-risk decision in this
  plan.** Mitigated by Decision 6's explicit empty-string-means-auto_plan rule and a dedicated
  test (`tests.md` T-LEGACY) that hand-writes a Phase-1-shaped `project.yaml` with no
  `planning_mode` field and confirms it still takes the one-step `TRIAGED → PLANNED` path.
- **Quick Fix's fast path bypasses review/approval by construction.** This is intentional
  (Decision 3) but means a misclassified high-risk change could reach `EXECUTING` without a
  gate — mitigated by Requirement 3's own "if uncertain, escalate" instruction being carried
  into `harness/core/triage/METHOD.md` as a hard rule, and by `eng plan escalate` existing as
  the correction mechanism; not mitigated by any code-level classifier (explicitly out of
  scope, Requirement 21).
- **`eng adapter prompt` now always shells out to `buildContextBundle`, which writes
  `context-manifest.yaml` on every call** — a Phase 4 behavior (unavoidable side effect of the
  bundle logic) now triggered from a new call site. Not a compatibility break (nothing reads
  that file expecting it to be absent), but worth stating explicitly.

## Open questions

- [ ] Should `harness/core/runtime/METHOD.md` also cover the `architecture`/`high-risk`
  workflows' research/ADR steps explicitly, or defer to `core/triage/METHOD.md`'s existing
  table? **This plan keeps the Runtime Router focused on the mechanical command sequence
  (which `eng` commands, in which order) and defers domain judgment (when research is needed)
  to `core/triage/METHOD.md`, which already owns that table** — avoids duplicating guidance
  across two documents that could drift apart.

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|---|---|---|---|
| Think Before Planning | spec.md written first; user-observable goal stated | 9/10 | Goal is CLI-observable (`eng adapter prompt`'s combined output, `eng start`'s banner), not "runtime exists" |
| Simplicity First | ≥3 out-of-scope items with reasoning | 10/10 | 7 items excluded, each reasoned, plus 5 later improvements |
| Surgical Changes | Every file listed with change type and exact reason | 9/10 | New-package creation is coarser than single-symbol edits, same unavoidable caveat as every prior phase |
| Goal-Driven Execution | Each scope item traces to an acceptance criterion | 9/10 | Every new command, state, and the compatibility-critical default all get a test in `tests.md` |

**Total: 37/40 → 9/10**

---

**User confirmation received:** [ ] Yes
**Confirmed on:** _pending_
