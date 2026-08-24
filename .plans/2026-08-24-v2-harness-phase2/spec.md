# Spec — V2 Harness Phase 2 (Reliability and Execution Control)

> **Planner note:** Read `.plans/2026-08-24-v2-harness-foundation/spec.md` and
> `DECISION_LOG.md` in full before this file — Phase 2 builds directly on top of
> `.agent/project.yaml`, the `harness/` tree, and the `eng` CLI those established. This plan
> does not re-litigate Phase 1's decisions; it extends them additively, the same way Phase 1
> extended V1.

---

## Goal

Phase 1 gave the harness a way to exist once globally and be linked into any project. It did
not change how work actually gets planned or executed: a single Planner writes a plan, a
single Executor implements it and self-reports pass/fail, and nothing checks either of them.
Phase 2 breaks that single-authority loop into five distinct responsibilities — Triage,
Planner, Plan Reviewer, Executor, Verifier — connected by a small set of CLI primitives
(`eng plan`, `eng verify`, `eng hooks`, `eng triage`) that make each handoff inspectable
instead of implicit.

**Done looks like:** A plan folder can be stamped with the git commit it was planned against
and a declared write scope (`eng plan new <name> --risk <level>`); `eng plan drift` correctly
reports `PLAN_DRIFT_DETECTED` when a file in that scope changed after planning and `OK`
otherwise; `eng verify <plan-dir>` runs the project's test command, diffs the repo against the
plan's recorded git SHA, flags any changed file outside the declared write scope, and writes a
`verify-report.md` with a PASS/FAIL verdict — all without modifying any source file; `eng plan
retry <dir> <stage>` tracks a per-stage retry counter against a configurable budget and reports
`RETRY BUDGET EXHAUSTED` once it's spent; `eng hooks run <stage>` executes the shell-command
hooks configured for a lifecycle stage (`before_plan`, `after_plan`, `before_execute`,
`after_task`, `after_execute`, `on_failure`) and prints a reminder for stages that are
AI-performed rather than shell-runnable (e.g. `plan_review`); and every regression test from
Phase 1 (`eng init`/`doctor`/`scan`/`skills list`, plus the V1 script suite) still passes
unmodified.

---

## Background

The user's Phase 2 brief maps to nine numbered feature areas (Triage, Plan Reviewer, Verifier,
Hooks, Drift Detection, Stop Conditions, Retry Budget, Write Boundaries, Approval Gates). Read
together, eight of the nine are really one thing: **a plan needs a persistent record of the
state it was made against and the boundaries it's allowed to operate within, and that record
needs to be checkable by something other than the role that wrote it.** That's `plan.yaml`
(new) plus three small CLI verbs that read and update it (`eng plan`, `eng verify`, `eng
hooks`). The ninth (Task Triage) is a genuinely separate concern — classifying a request
*before* a plan exists — and gets its own minimal, explicitly-non-authoritative heuristic
(`eng triage`), because a real classifier needs the Skill Router this repo's own Phase 1 spec
already deferred until Task Triage exists (a dependency this plan satisfies without trying to
also build the router itself).

---

## Preliminary fix (before adding Phase 2 features)

`cli/internal/project/project.go`'s `DetectMode` silently returns `"hybrid"` when
`.agent/project.yaml` exists but fails to parse (corrupt YAML, or a field type mismatch from a
future/older `eng` version) — the parse error is discarded (`loadErr` is checked only for
`== nil`, never surfaced). This was a latent, low-consequence bug in Phase 1 (nothing else read
`project.yaml` besides `doctor`/`init`/`skills`), but Phase 2 adds four new fields to the same
struct (`workflow`, `retry_budget`, `require_approval`, `config_version`) written and read by
several more commands — a silently-swallowed parse error becomes a real correctness risk once
`eng verify`/`eng plan retry` start trusting fields from that file. Fixed as Task 1, before any
Phase 2 feature work, so the rest of this plan builds on a `project.yaml` reader that fails
loudly.

---

## Design decisions

### Decision 1 — A new `plan.yaml` sidecar carries all per-plan state; `spec.md`/`tasks.md`/`tests.md` are untouched
**Chosen:** Drift tracking, risk level, write scope, retry counters, and status all live in one
new file, `.plans/<feature>/plan.yaml`, written by `eng plan new` and updated by `eng plan
drift`/`eng plan retry`/`eng verify`. `spec.md`, `tasks.md`, `tests.md` keep exactly the format
Phase 1 (and V1) already use.
**Why:** Every one of Phase 2's stateful features (Requirement 15's `planned_at: git_sha`,
Requirement 18's write-sets, Requirement 22's retry counters) is *machine-read* state, not
human-authored plan content — mixing it into `spec.md`'s prose would make it fragile to parse
and awkward to hand-edit. A sidecar file is exactly the pattern `.agent/project.yaml` already
established in Phase 1.
**Rejected:** Embedding YAML frontmatter directly in `spec.md` (parseable, but conflates
Planner-authored content with tool-managed state — a Planner revising `spec.md`'s prose could
accidentally clobber a retry counter); a separate file per concern (`drift.yaml`,
`retry.yaml`, `writescope.yaml`) — three files to keep in sync for no benefit at this scale.

### Decision 2 — Plan-level write scope for the MVP; per-task write-sets are a later improvement
**Chosen:** `plan.yaml`'s `write_scope` is one list of glob patterns for the whole plan, filled
in by the Planner from `spec.md`'s existing "Affected files" table. Requirement 19's per-task
`writes:` field is **not** built in this phase.
**Why:** Per-task write-sets require structurally parsing `tasks.md`'s markdown into a task
list the tool can address individually — real parser work, not a config addition. A plan-level
scope answers the actual question Phase 2 needs answered ("did anything change outside what
this plan said it would touch") with a single flat list the Planner already has to write down
in `spec.md` anyway.
**Rejected:** Per-task write-sets now — correctly identified as more invasive than the rest of
this phase; deferred to "later improvements" below.

### Decision 3 — Plan Reviewer and Verifier are methodology + a report file, not enforcement code
**Chosen:** `harness/core/plan-reviewer/METHOD.md` and `harness/core/verifier/METHOD.md` define
what each role checks and writes (`review.md`, `verify-report.md`), but nothing in `eng` blocks
an Executor from proceeding if `review.md` says `CHANGES_REQUESTED`. `eng verify`'s only
enforcement is its own exit code (1 on FAIL) and the report it writes.
**Why:** This harness has no execution engine that runs Planner/Executor/Reviewer as
call-and-response — they're all the same kind of AI agent session, directed by a human relaying
between them (exactly like Phase 1's Copilot/Codex relay workflow). "Enforcement" at this stage
means making the right information impossible to miss (a distinct file, a non-zero exit code),
not a gate a script can physically close. Real enforcement (a Planner session literally cannot
proceed) needs the orchestration layer Requirement list explicitly defers to a later phase.
**Rejected:** Building a stateful workflow engine that sequences the roles — that's the "complex
remote orchestration" the user's scope constraint explicitly excludes from Phase 2.

### Decision 4 — Task Triage ships as a documented methodology plus a keyword heuristic, explicitly non-authoritative
**Chosen:** `eng triage "<request text>"` matches the request against a small hard-coded keyword
table (four buckets: quick-fix, bug, architecture, high-risk; unmatched falls through to
"feature") and prints a suggested level + workflow, always followed by "(heuristic hint only —
the Planner makes the final call)". `harness/core/triage/METHOD.md` documents the actual
classification the Planner is responsible for making.
**Why:** A real classifier needs semantic understanding of the request against the project's
own domain — exactly the Skill Router Phase 1 deferred pending Task Triage existing. Building
that now would mean building the router twice. A keyword heuristic is honest about being a
hint, costs almost nothing, and immediately unblocks the one thing Phase 2 actually needs from
triage: a documented, consistent level scheme (`quick-fix`/`bug`/`feature`/`architecture`/
`high-risk`) that `plan.yaml`'s `risk_level` field and `require_approval` policy can both refer
to.
**Rejected:** No triage tool at all (purely a methodology doc) — rejected because a zero-cost
heuristic that's honest about its limits is strictly more useful than nothing, and it gives
`risk_level` a canonical vocabulary instead of five different plans inventing their own labels.

### Decision 5 — Hooks are a YAML-configured command table; the harness ships a default, a project may fully override it
**Chosen:** `harness/hooks/default.yaml` defines the six lifecycle stages from the user's brief,
each stage a list of hook *names*, plus a `commands:` map from hook name to either a shell
command (`${test_cmd}` interpolated from `.agent/project.yaml`) or an empty string meaning "an
AI role performs this, not a shell command" (e.g. `plan_review`, `collect_logs`). A project may
drop a `.agent/hooks.yaml` that fully replaces the default — no partial merge.
**Why:** Matches the exact shape requested (`before_plan: [project_scan]`, etc.) while staying
inside "config, not prompts" — a project can add a stage's shell command (say, a real
`regression_test` script) without touching any `.md` instruction file. Full-replace instead of
merge avoids a whole class of "why did my override only partially apply" confusion for a v1 of
this mechanism.
**Rejected:** Deep-merging project hooks over global defaults — more flexible, but the merge
semantics (append? replace per-stage? replace per-command?) are exactly the kind of ambiguity
Karpathy's simplicity principle says to avoid until a real use case demands it.

### Decision 6 — `require_approval` and per-task `**Requires approval:**` are declared, not enforced
**Chosen:** `.agent/project.yaml` gains an optional `require_approval:` list (operation
categories: `production_deploy`, `database_migration`, `firmware_flash`, `plc_write`,
`destructive_operation`). `tasks.md` gains an optional `**Requires approval:**` field convention,
documented in `harness/core/executor/METHOD.md`'s constraints as a hard stop: an Executor that
reaches such a task must pause for explicit human confirmation before performing it, full stop,
no exception.
**Why:** Real enforcement means a policy engine sitting in front of actual tool/device access —
explicitly out of scope per the user's own constraint ("Do not implement MCP/device control yet
unless absolutely necessary"). Declaring the category and hard-wiring the stop into the
Executor's documented contract is the correct-sized foundation: it's the exact mechanism this
same repo already relies on for "stop on test failure" (a documented behavioral contract, not
code that can physically stop an LLM), just extended to a new trigger.
**Rejected:** Building an approval-gate CLI command now (`eng approve <task>`) — there's nothing
yet for it to gate; premature until a real tool-adapter layer exists to gate access to.

---

## Responsibilities (the five roles)

| Role | Reads | Writes | Notes |
|---|---|---|---|
| **Triage** | the raw request, `harness/core/triage/METHOD.md` | nothing (or, optionally, a `risk_level` the Planner copies into `plan.yaml`) | `eng triage` is a hint generator, not a gate |
| **Planner** | `docs/src-map.md`, `docs/gotchas.md`, resolved skills, prior `DECISION_LOG.md`s, triage output | `spec.md`, `tasks.md`, `tests.md`, `plan.yaml` (via `eng plan new`) | Unchanged from Phase 1 except it now stamps `plan.yaml` and declares `write_scope` |
| **Plan Reviewer** | `spec.md`, `tasks.md`, `tests.md`, same context the Planner used | `review.md` only | Read-only w.r.t. plan/source files; verdict is `APPROVED` or `CHANGES_REQUESTED` + reasons, keyed to the checklist in the user's brief |
| **Executor** | `spec.md` → `tasks.md` → `tests.md`, `plan.yaml` (`write_scope`, retry budget) | source files per `tasks.md`, `tasks.md` status markers, `errors.log` | Must run `eng plan drift` before starting and `eng plan retry` on each test failure; must hard-stop on any Stop Condition or an approval-gated task |
| **Verifier** | `plan.yaml`, `spec.md`/`tasks.md`/`tests.md`, the actual git diff, build/test output | `verify-report.md` only | Never modifies source; `eng verify` is this role's primary tool |

---

## Scope

### In scope (Phase 2 MVP)
- `cli/internal/planmeta/` — `plan.yaml` schema (`plan`, `risk_level`, `planned_at.git_sha`,
  `status`, `write_scope`, `retry`, `retry_budget`) + load/save
- `cli/internal/gitutil/` — `HeadSHA`, `ChangedFilesSince` (thin `git` subprocess wrappers)
- `cli/internal/hooks/` — hooks.yaml schema + loader (project override or global default)
- `eng plan new <name> [--risk <level>]` — scaffolds `.plans/YYYY-MM-DD-<name>/` from
  `harness/templates/plan/*` (now including `plan.yaml`, `review.md`, `verify-report.md`
  templates) and stamps `plan.yaml` with the current HEAD SHA and risk level
- `eng plan drift [dir]` — compares current changes against `plan.yaml`'s recorded SHA and
  `write_scope`; reports `OK` or `PLAN_DRIFT_DETECTED`
- `eng plan retry <dir> <stage>` — increments and checks a retry counter against budget
- `eng verify [dir]` — runs the project's test command, checks the git diff against
  `write_scope`, writes `verify-report.md`, exits non-zero on FAIL
- `eng hooks run <stage>` — executes the configured hooks for a lifecycle stage
- `eng triage "<text>"` — keyword-heuristic risk-level hint (explicitly non-authoritative)
- `.agent/project.yaml` schema extension: `workflow` (triage/plan_review/verifier booleans,
  all default-true when the whole block is absent), `retry_budget` (defaults
  `{build: 2, unit_test: 2, integration_test: 1}` when absent), `require_approval` (defaults
  empty), `config_version` (new field, defaults to `1` when absent — Phase 1's format)
- `harness/core/triage/METHOD.md`, `harness/core/plan-reviewer/METHOD.md`,
  `harness/core/verifier/METHOD.md` — new methodology docs
- `harness/core/planner/METHOD.md`, `harness/core/executor/METHOD.md` — updated for the new
  lifecycle (stamp `plan.yaml`, run drift/retry, hard-stop on approval-gated tasks)
- `harness/hooks/default.yaml` — the six-stage default hook table
- `harness/templates/plan/plan.yaml`, `review.md`, `verify-report.md` — new plan templates
  (added to `harness/templates/plan/`, **not** to V1's `.plans/_template/` — see Decision-log
  entry and Out-of-scope below)
- The `DetectMode` parse-error fix (Preliminary fix, above)

### Out of scope (explicitly excluded, with reasons)
- **Per-task write-sets** (Requirement 19 in full) — needs structural `tasks.md` parsing;
  plan-level `write_scope` (Decision 2) covers the actual Phase 2 need.
- **Full MCP/tool-adapter ecosystem, policy engine that blocks real tool calls** — explicitly
  excluded by the user's scope constraint; `require_approval` is declared, not enforced (see
  Decision 6).
- **Automated orchestration that sequences Triage→Planner→Reviewer→Executor→Verifier without a
  human relaying between them** — this repo has no execution engine; every role is still a
  human-directed AI session, per Decision 3.
- **Semantic/NLP task classification** — `eng triage` is a keyword heuristic on purpose (Decision
  4); a real classifier depends on the Skill Router, itself out of scope until a phase after
  this one.
- **Modifying `.plans/_template/` or any V1 file's *behavior*** — Phase 2 templates live only
  under `harness/templates/plan/`, used by the new `eng plan new`; legacy
  `scripts/plan-executor.sh new` keeps scaffolding exactly the three files it always has.
- **Parallel execution, git worktrees, task dependency graphs** — nothing executes concurrently
  yet; still not needed.
- **A real approval-gate enforcement command (`eng approve`)** — nothing to gate without a tool
  adapter layer (Decision 6).

### Later improvements (explicitly deferred, not designed here)
- Per-task write-sets once `tasks.md` has a stable structural parser
- `eng doctor` surfacing retry/drift status across every plan in a project, not just one
- A `commands:` merge strategy for `.agent/hooks.yaml` richer than full-replace
- Wiring `eng hooks run` into an actual automated trigger (a file watcher, a CI step) instead of
  being invoked by hand/by the AI session
- A real Skill Router that uses `eng triage`'s risk-level vocabulary as one input signal
- Automated rollback on Verifier FAIL

---

## Affected files

| File | Change type | Reason |
|---|---|---|
| `cli/internal/project/project.go` | Modify | Fix `DetectMode` swallowing parse errors (Preliminary fix); extend `Config` with `Workflow`, `RetryBudget`, `RequireApproval`, `ConfigVersion` |
| `cli/internal/project/project_test.go` | Modify | Add test for the parse-error fix and for new-field defaulting |
| `cli/internal/planmeta/planmeta.go` | Create | `plan.yaml` schema + load/save + `DefaultBudget()` |
| `cli/internal/planmeta/planmeta_test.go` | Create | Round-trip + default-budget tests |
| `cli/internal/gitutil/gitutil.go` | Create | `HeadSHA`, `ChangedFilesSince` |
| `cli/internal/gitutil/gitutil_test.go` | Create | Tests against a throwaway git repo in `t.TempDir()` |
| `cli/internal/hooks/hooks.go` | Create | hooks.yaml schema + loader + `Stage()` |
| `cli/internal/hooks/hooks_test.go` | Create | Default-vs-override loading tests |
| `cli/plan_cmd.go` | Create | `eng plan new/drift/retry` |
| `cli/verify_cmd.go` | Create | `eng verify` |
| `cli/hooks_cmd.go` | Create | `eng hooks run` |
| `cli/triage_cmd.go` | Create | `eng triage` |
| `cli/main.go` | Modify | Dispatch the four new subcommands |
| `harness/core/triage/METHOD.md` | Create | Triage classification methodology |
| `harness/core/plan-reviewer/METHOD.md` | Create | Plan Reviewer methodology |
| `harness/core/verifier/METHOD.md` | Create | Verifier methodology |
| `harness/core/planner/METHOD.md` | Modify | Add: run triage, stamp `plan.yaml`, declare `write_scope` |
| `harness/core/executor/METHOD.md` | Modify | Add: drift check before execute, retry-budget calls, stop conditions, approval-gate hard stop |
| `harness/hooks/default.yaml` | Create | Six-stage default hook table |
| `harness/templates/plan/plan.yaml` | Create | New plan template |
| `harness/templates/plan/review.md` | Create | New plan template |
| `harness/templates/plan/verify-report.md` | Create | New plan template |
| `harness/VERSION` | Modify | Bump `0.1.0-mvp` → `0.2.0-phase2` |
| `docs/src-map.md` | Modify | Add Phase 2 module entries (last task) |
| `README.md` | Modify | Additive section under the existing "V2 harness (preview)" |
| `ROADMAP.md` | Modify | Note Phase 2 plan link |

---

## Risks and unknowns

- **`sh -c` dependency for hook/test-command execution.** `eng hooks run` and `eng verify` both
  shell out via `sh -c` to support the `&&`-chained commands `detect.go` already produces (e.g.
  ESP-IDF's `env_setup && idf.py build`). On Windows this requires Git Bash's `sh.exe` on PATH —
  not a new dependency (V1's own scripts already required Bash 4+), but worth confirming on
  this machine before calling the plan done (see `tests.md` T-SH).
- **`git diff --name-only <sha>` semantics.** This diffs the working tree against a fixed commit,
  which includes uncommitted changes — the desired behavior for drift detection during active
  work, but worth stating explicitly since it differs from `git diff <sha>..HEAD` (commits only).
- **Glob matching for `write_scope`.** Go's `filepath.Match` does not support `**` natively;
  `verify_cmd.go`/`plan_cmd.go` special-case a `/**` suffix as prefix-match. This covers the
  patterns in the user's own examples (`src/api/**`) but not `**` in the middle of a pattern —
  acceptable for MVP, noted as a known limitation rather than silently wrong.
- **Retry budget defaults living in two places** (`planmeta.DefaultBudget()` in Go and
  `.agent/project.yaml`'s `retry_budget`). `plan.yaml`'s budget is copied from
  `.agent/project.yaml` at `eng plan new` time if present, else `planmeta.DefaultBudget()` —
  documented in `tasks.md` so the precedence is explicit, not just "whichever the code happens
  to check first."

---

## Open questions

- [ ] Should `eng plan new` refuse to run if `.agent/project.yaml`'s `mode` is `legacy` (i.e.,
  no `.agent/` exists yet)? **This plan defaults to: yes, print a clear message pointing at
  `eng init` first** — `plan.yaml`'s retry-budget defaulting needs *something* to read from,
  and forcing `eng init` first is one extra step, not a migration requirement (it still only
  ever adds `.agent/project.yaml`, per Phase 1's Decision 3).

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|---|---|---|---|
| Think Before Planning | spec.md written first; user-observable goal stated | 9/10 | Goal is CLI-observable (`eng verify` exit code + report), not "roles are defined" |
| Simplicity First | ≥3 out-of-scope items with reasoning | 10/10 | 6 items excluded, each with a one-line reason, plus 5 "later improvements" |
| Surgical Changes | Every file listed with change type and exact reason | 9/10 | New-package creation (`planmeta`, `gitutil`, `hooks`) is coarser than a single-symbol edit — same unavoidable-for-a-foundation-slice caveat as Phase 1 |
| Goal-Driven Execution | Each scope item traces to an acceptance criterion | 9/10 | Every new `eng` subcommand has a runnable pass/fail test in `tests.md` |

**Total: 37/40 → 9/10**

---

**User confirmation received:** [ ] Yes
**Confirmed on:** _pending_
