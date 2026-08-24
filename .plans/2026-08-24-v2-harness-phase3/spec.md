# Spec — V2 Harness Phase 3 (Orchestration and Workflow Runtime)

> **Planner note:** Read `.plans/2026-08-24-v2-harness-foundation/` and
> `.plans/2026-08-24-v2-harness-phase2/` (`spec.md` + `DECISION_LOG.md` of both) in full before
> this file. Phase 3 does not reimplement Triage/Planner/Reviewer/Executor/Verifier — it wires
> the primitives those phases already built into a state machine that can be advanced
> mechanically instead of by human memory of "what comes next."

---

## Goal

Phases 1–2 gave every role (Triage, Planner, Reviewer, Executor, Verifier) its own tool
(`eng triage`, `eng plan new`, `eng plan drift`, `eng plan retry`, `eng verify`, `eng hooks
run`), but nothing tracks *which stage a given plan is actually in*, and nothing stops a human
from skipping a stage by accident (running the Executor before the Reviewer has approved, or
declaring a plan done without `eng verify` ever passing). Phase 3 adds one authoritative
lifecycle state per plan, a small set of transition rules that decide what happens next after
each stage's result, and two genuinely new capabilities behind that state machine: a real
approval gate the tooling itself enforces, and a first agent-adapter that turns "read the
Executor's METHOD.md and figure out what to paste into Claude Code" into a single command.

**Done looks like:** `eng workflow start "<requirement>"` triages a request, creates a plan
via the existing `eng plan new`, and records an initial lifecycle state; `eng workflow status
<dir>` reports the current state and the exact next action in one line, whether that's a code-
runnable check or a human/AI-driven step; `eng workflow advance <dir>` mechanically executes
every code-runnable transition (drift checks, hook stages, `eng verify`) and refuses to skip a
human-driven one — in particular, a plan whose risk level or declared category requires
approval cannot reach `EXECUTING` until `eng plan approve` has been run, enforced by code, not
by a documented convention someone might forget to read. Every transition is logged to an
append-only `events.jsonl` so a plan's history survives a replan cycle. Interrupting the
process and re-running `eng workflow status`/`advance` later resumes exactly where the
persisted state left off — resume is a property of the design, not a separate feature. All of
this ships alongside a first Claude Code adapter (`eng adapter prompt <role> <dir>`, `eng
start`), a capability registry (`eng capabilities list`), and a structured command-execution
mode that stops assuming `sh -c` is the only way to run a command — while every Phase 1/2
regression test still passes unmodified.

---

## Background — what's already sufficient vs. what's missing

**Already sufficient, reused as-is:**
- `eng triage` (heuristic hint), `eng plan new/drift/retry`, `eng verify`, `eng hooks run` —
  Phase 3 calls these, never reimplements their logic.
- `plan.yaml` as the single per-plan state file (Phase 2's Decision 1) — Phase 3 extends the
  schema rather than inventing a second state file, per this plan's own explicit instruction
  to reuse it.
- `harness/core/{triage,planner,plan-reviewer,executor,verifier}/METHOD.md` — these remain the
  *content* of what each role does; Phase 3 only adds a mechanical layer that knows when to
  point a human at the right one.
- The `[ ]`/`[x]` task-checkbox convention in `tasks.md`, already parsed by
  `scripts/plan-executor.sh`'s `list`/`status` commands — Phase 3's EXECUTING→VERIFYING
  transition reuses this exact convention (count remaining `- [ ]` lines) instead of inventing
  a new "is execution done" signal.

**Missing, and why each is now in scope:**
- **No persisted lifecycle state.** `plan.yaml`'s Phase 2 `status` field is write-once (set to
  `"planned"` at creation, never updated by any Phase 2 command) — it was scaffolding for
  Phase 3, not a real state machine. Nothing currently distinguishes "reviewed" from "review
  rejected" from "waiting on human approval."
- **No machine-readable Reviewer/Verifier verdict.** `review.md` is prose with checkboxes;
  `eng verify` already *computes* a boolean pass/fail internally (`verify_cmd.go`'s `pass`
  variable) but only prints it — it never persists it anywhere `eng workflow advance` could
  read back.
- **No approval enforcement.** Phase 2's `require_approval` and `**Requires approval:**` are
  read-the-docs conventions an Executor is trusted to follow. Nothing stops a plan from
  reaching execution without it.
- **No agent adapter.** Every role handoff today is "a human reads `core/<role>/METHOD.md`
  and pastes context into whichever AI tool they're using." That's fine as a floor, but this
  repository is itself developed with Claude Code — assembling the exact prompt (role rules +
  `spec.md`/`tasks.md`/`tests.md` paths + plan state) is mechanical and worth automating for
  the one adapter this repo actually uses daily.
- **No capability detection.** `eng doctor` reports harness/skills/mode but has no concept of
  "is `claude` even installed on this machine" — needed before an adapter can safely decide
  whether to offer auto-launch vs. print-and-wait.
- **`sh -c` is the only execution path.** Documented as a known gotcha in Phase 2
  (`docs/gotchas.md`) rather than fixed; Windows-native execution needs a path that doesn't
  assume a POSIX shell exists.

---

## Design decisions

### Decision 1 — `plan.yaml`'s `status` field is superseded by a richer `state` field, with a compatibility migration
**Chosen:** Add a `state` field to `planmeta.Meta` carrying the full lifecycle enum
(`NEW`/`TRIAGED`/`PLANNED`/`REVIEWED`/`APPROVED`/`EXECUTING`/`VERIFYING`/`COMPLETED`/`BLOCKED`/
`FAILED`/`NEEDS_REPLAN`/`NEEDS_APPROVAL`/`NEEDS_FIX`/`CANCELLED` — 14 states). The old `status`
field stays in the
struct (for YAML compatibility with files already on disk) but is no longer written by new
code. `planmeta.Load` infers `state` from a legacy `status` value when `state` is empty
(`"planned"→PLANNED`, `"reviewed"→REVIEWED`, `"executing"→EXECUTING`, `"verified"→COMPLETED`,
`"failed"→FAILED`; anything else, including a plan.yaml with neither field, defaults to `NEW`).
**Why:** `status` was never actually load-bearing — Phase 2 set it once at `eng plan new` and
no command ever read or updated it again, so there is no real behavior to preserve beyond "a
Phase-2-created plan.yaml must still parse and get a sensible starting state." This is the
same additive-migration pattern Phase 2 itself used for `config_version` on
`.agent/project.yaml` — continuing an established precedent rather than inventing a new one.
**Rejected:** A separate `workflow-state.yaml` sidecar — rejected for the same reason Phase 2
rejected splitting `plan.yaml` into multiple files: one authoritative per-plan state file,
per this plan's own explicit instruction ("reuse `plan.yaml` where appropriate").

### Decision 2 — Reviewer and Verifier verdicts live in `plan.yaml`, not in a parsed `review.md`
**Chosen:** `plan.yaml` gains `review: {verdict, blocking_issues, reviewed_at}` and
`verification: {verdict, verified_at}`. `eng verify` populates `verification` itself (it
already computes the verdict as code). A new `eng plan review <dir> --verdict PASS|REJECT
[--blocking-issues N]` command lets the Reviewer (human or AI, following
`core/plan-reviewer/METHOD.md`) record its verdict the same explicit way `eng plan retry`
already records a retry — `review.md` keeps being written by the Reviewer as the rich,
human-readable "why," but `eng workflow advance` only ever reads `plan.yaml`.
**Why:** The brief is explicit — "do not rely on natural-language parsing to determine whether
execution may continue." Parsing `[ ] APPROVED` vs. `[x] APPROVED` out of Markdown is exactly
the natural-language parsing this is meant to avoid; a one-line command the Reviewer runs as
the last step of their own methodology is no heavier than `eng plan retry` already is for the
Executor.
**Rejected:** YAML frontmatter on `review.md` (the pattern Phase 1 used for `SKILL.md`) —
considered, but `SKILL.md`'s frontmatter describes static metadata about a skill; a review
verdict is a state transition, which is exactly what `plan.yaml` already exists to hold.

### Decision 3 — The orchestrator advances mechanical stages automatically and stops at every human/AI-driven stage
**Chosen:** `eng workflow advance <dir>` performs, without asking: drift checks
(`before_execute`), the EXECUTING→VERIFYING transition once `tasks.md` has zero remaining
`- [ ]` lines, running `eng verify`, and applying the failure-routing table (Decision 6) to
whatever verdict comes back. It never attempts to write `spec.md`/`tasks.md`, never invokes an
AI agent unattended, and never auto-approves an approval-gated plan. Every human/AI-driven
stage ends with `eng workflow advance` printing the exact next command
(`eng adapter prompt <role> <dir>`) and stopping.
**Why:** This is the literal reading of "preserving human control at important gates" from the
Phase 3 goal statement, applied to *every* AI-driven stage, not only the explicitly-named
approval ones — matches the user's own explicit constraint ("Do NOT deeply integrate every
agent," "do not implement dangerous external tool execution yet"). It also means the
orchestrator has zero new failure modes around unattended code changes: everything it does
automatically is either read-only (status checks) or something Phase 2 already made safe
(`eng verify` never modifies source).
**Rejected:** Auto-invoking Claude Code non-interactively (`claude -p "<prompt>"`) for
Planner/Executor stages — real automation, but it means an unattended process can modify
source files with no human in the loop before the fact, which is a materially different (and
larger) risk than anything Phase 1–2 introduced. Flagged explicitly as a "later improvement,"
not built here.

### Decision 4 — Approval is enforced by a persisted flag + explicit command, not by inference
**Chosen:** `plan.yaml` gains `requires_approval` (bool, defaulted `true` when `risk_level` is
`high-risk`, settable explicitly via `eng plan new --requires-approval` otherwise) and
`approved_at`/`approved_by` (populated only by a new `eng plan approve <dir> [--by <name>]`
command). `eng workflow advance` refuses the REVIEWED→APPROVED transition — and therefore
every transition after it — while `requires_approval` is true and `approved_at` is empty,
reporting state `NEEDS_APPROVAL` instead.
**Why:** This is the one place Phase 3 is asked to move from "documented convention" (Phase 2)
to "actual workflow enforcement where possible" (Phase 3 goal #6). A boolean the orchestrator
checks before advancing state is real enforcement at the layer that exists today (the harness's
own bookkeeping) without requiring the tool-adapter/policy-engine layer that gating a real
`firmware flash`/`plc write` call would need — that part is correctly still out of scope.
**Rejected:** Trying to auto-detect "this plan needs approval" from `spec.md`'s prose (natural-
language inference, the exact thing Decision 2 already ruled out for review verdicts) —
`requires_approval` is a declared fact, not an inferred one, same as `risk_level` and
`write_scope` already are.

### Decision 5 — One Go interface for agent adapters; Claude Code is the only implementation, and it only assembles + optionally launches, never auto-drives
**Chosen:** `internal/agent.Adapter` has three methods: `Name() string`, `Available() bool`
(capability-detected), `RolePrompt(role Role, planDir string) (string, error)` (assembles a
plain-text prompt from the role's `METHOD.md` plus the plan's file paths and current state).
`ClaudeCodeAdapter` is the only implementation. `eng adapter prompt <role> <dir>` prints the
assembled prompt; `eng start` additionally execs into an attached, interactive `claude` process
in the current directory if the capability is detected (inheriting the terminal — the user
lands in a normal Claude Code session, nothing is scripted from there).
**Why:** "Prioritize Claude Code first... do not deeply integrate every agent" is explicit.
Assembling the right prompt (which role, which files, which state) is genuinely useful and
purely mechanical — no risk. Launching an *interactive* session the human then drives
normally is categorically different from Decision 3's rejected non-interactive auto-drive: the
human is still the one typing into Claude Code once it's open.
**Rejected:** A `collect result` responsibility in the adapter interface now (listed in the
Phase 3 brief's "possible responsibilities") — meaningless without non-interactive invocation,
which Decision 3 already deferred; an interface method with no real implementation until a
later phase is speculative surface area this plan's own Simplicity principle rules out.

### Decision 6 — Failure routing is a static lookup table, not a runtime decision
**Chosen:** `internal/workflow` encodes the transition rules from the Phase 3 brief as data:

| From | Trigger | To |
|---|---|---|
| `TRIAGED` | `spec.md`/`tasks.md`/`tests.md` all exist and are non-empty | `PLANNED` |
| `PLANNED` | `eng plan review --verdict PASS` (or review disabled and not required) | `REVIEWED` |
| `PLANNED` | `eng plan review --verdict REJECT` | `NEEDS_REPLAN` |
| `REVIEWED` | `requires_approval == false`, or `approved_at` set | `APPROVED` |
| `REVIEWED` | `requires_approval == true` and `approved_at` empty | `NEEDS_APPROVAL` |
| `APPROVED` | `eng plan drift` reports OK | `EXECUTING` |
| `APPROVED` | `eng plan drift` reports `PLAN_DRIFT_DETECTED` | `NEEDS_REPLAN` |
| `EXECUTING` | zero remaining `- [ ]` lines in `tasks.md` | `VERIFYING` (auto-runs `eng verify`) |
| `VERIFYING` | `eng verify` verdict `PASS` | `COMPLETED` |
| `VERIFYING` | `eng verify` verdict `FAIL`, retry budget remaining | `NEEDS_FIX` |
| `VERIFYING` | `eng verify` verdict `FAIL`, retry budget exhausted | `FAILED` |
| `NEEDS_FIX` | Executor retries, `tasks.md` reaches zero `- [ ]` again | `VERIFYING` |
| `NEEDS_REPLAN` | Planner revises `spec.md`/`tasks.md`, re-runs from `PLANNED` | `PLANNED` |
| any state | `eng plan block <dir> --reason "..."` | `BLOCKED` |
| any state | `eng plan cancel <dir> [--reason]` | `CANCELLED` |

`eng workflow advance` applies exactly one row per invocation and stops; it never chains
multiple automatic transitions silently past a state a human should see.
**Why:** "Do not let the agent improvise the next step" is explicit in the brief. A table an
Executor/Planner session cannot talk itself out of is the point.
**Rejected:** A generic pluggable state-machine library — this table fits in under 20 rows and
changes rarely; a dependency (and its abstraction cost) buys nothing a Go `switch` doesn't
already give for free.

### Decision 7 — Structured command execution is additive; `sh -c` remains the default for existing plain-string commands
**Chosen:** `internal/executil.Run` accepts either a plain string (`Shell string`, executed via
`sh -c` exactly as today) or a structured form (`Command string; Args []string`, executed via
`exec.Command` directly, no shell at all). `.agent/project.yaml`'s `stack.*_cmd` fields and
`hooks.yaml`'s `commands` map may use either form — YAML lets a scalar string and a
`{command, args}` mapping coexist under the same key, so this is 100% backward compatible with
every `project.yaml`/`hooks.yaml` written by Phase 1 or 2.
**Why:** "Support shell commands only as an explicit compatibility mode" — plain strings are
that compatibility mode, kept as the default because `detect.go` (Phase 1) still emits plain
strings and changing that is out of scope here. The structured form is what a future
`detect.go` or a hand-written `project.yaml` can opt into for real Windows-native execution.
**Rejected:** Migrating `detect.go` to emit structured commands now — a larger, unrelated
change to a file this plan doesn't otherwise touch; tracked as a later improvement.

### Decision 8 — `eng install` gains an opt-in PATH setup step; default behavior is print-only
**Chosen:** `eng install` copies the currently-running `eng` binary into
`~/.engineering-harness/bin/eng` (`.exe` on Windows) in addition to the existing payload copy,
and always prints the platform-appropriate PATH line. A new `--add-to-path` flag additionally
*applies* it: on Windows, calls `setx PATH "%PATH%;<bin-dir>"` via `os/exec`; on macOS/Linux,
appends an `export PATH="$HOME/.engineering-harness/bin:$PATH"` line to
`~/.bashrc`/`~/.zshrc` (whichever exists) if not already present.
**Why:** Resolves the exact gotcha Phase 2 documented, without silently mutating a user's shell
profile or Windows user environment on every install — an explicit flag matches how this
repo already treats every other consequential-but-reversible action (nothing here is
destructive; `setx` on a user-scope variable and appending one line to a profile file are both
trivially undoable, but still deserve an explicit opt-in rather than a silent default).
**Rejected:** Doing it unconditionally — `eng install` is already run more than once (Phase 2's
own workflow re-installs after every harness change); silently rewriting PATH/profile files on
every run is the kind of surprise this repo's own Karpathy principles single out.

---

## Responsibilities (extending Phase 2's table)

| Role | New in Phase 3 |
|---|---|
| **Triage** | `eng workflow start` calls `eng triage` automatically as the first step; still a hint, Planner still makes the final call on `risk_level` passed to `eng plan new` |
| **Planner** | Unchanged methodology; now also expected to set `plan.yaml`'s `requires_approval` from Triage's/Planner's own judgment when creating the plan |
| **Plan Reviewer** | Now also runs `eng plan review --verdict ...` as the explicit last step, in addition to writing `review.md` |
| **Executor** | Now also runs `eng workflow advance` before starting (drift-gates automatically) and after finishing all tasks (auto-triggers verification) |
| **Verifier** | Unchanged mechanism (`eng verify`); its verdict is now automatically persisted to `plan.yaml` |
| **Orchestrator (new)** | Owns `state` transitions per Decision 6's table; never authors plan content; never bypasses `NEEDS_APPROVAL` |
| **Agent Adapter (new)** | Detects capability, assembles role prompts; does not author plan content or run unattended |

---

## Scope

### In scope (Phase 3 MVP)
- `cli/internal/planmeta`: `state`, `review`, `verification`, `requires_approval`,
  `approved_at`/`approved_by` fields + legacy `status` migration; `AppendEvent` writing to
  `<plan-dir>/events.jsonl`
- `cli/internal/workflow`: state enum, the Decision 6 transition table, workflow-profile
  loading from `harness/workflows/*.yaml`
- `cli/internal/capabilities`: LookPath-based detection for `git`, `claude`, `codex`, `docker`
- `cli/internal/agent`: `Adapter` interface + `ClaudeCodeAdapter`
- `cli/internal/executil`: structured-or-shell command execution
- `eng workflow start "<text>"` / `eng workflow status [dir]` / `eng workflow advance [dir]`
- `eng plan review <dir> --verdict PASS|REJECT [--blocking-issues N]`
- `eng plan approve <dir> [--by <name>]`
- `eng plan block <dir> --reason "..."` / `eng plan cancel <dir> [--reason "..."]`
- `eng adapter prompt <role> <dir>`
- `eng start`
- `eng capabilities list`
- `eng install --add-to-path`; unconditional PATH line printed on every install
- `harness/workflows/{quick-fix,bug-fix,feature,architecture,high-risk}.yaml`
- `harness/adapters/claude-code/ADAPTER.md` (documents the adapter contract)
- `eng verify` and `eng hooks run` migrated to `internal/executil` (still `sh -c` by default —
  no behavior change for existing plain-string commands)
- `eng doctor` gains a "Capabilities" section

### Out of scope (explicitly excluded, with reasons)
- **Non-interactive, unattended agent invocation** (`claude -p` driving Planner/Executor stages
  automatically) — Decision 3; a materially larger risk category, deferred.
- **Codex/Copilot/Cursor/OpenCode adapters** — the `Adapter` interface is designed to allow
  them; only Claude Code is implemented, per the brief's explicit prioritization.
- **Full MCP ecosystem, Modbus/PLC/serial capability detection** — `capabilities` covers only
  what's mechanically checkable today (CLI tools via `LookPath`); device/protocol capabilities
  are a later phase's concern, same as Phase 1–2 already deferred them.
- **Rewriting `detect.go` to emit structured commands** — Decision 7; `executil` supports the
  structured form, nothing in this plan changes what Phase 1's detector produces.
- **A real distributed/parallel execution engine** — `eng workflow advance` operates on exactly
  one plan at a time; nothing here coordinates multiple plans or agents concurrently.
- **Telemetry backend, cloud-hosted harness, skill marketplace** — unrelated to orchestration;
  explicitly excluded by the user's own scope constraint.
- **Registry/profile mutation on every `eng install`** — Decision 8; opt-in only.

### Later improvements (explicitly deferred, not designed here)
- Non-interactive agent invocation with a real result-collection contract, once there's a
  policy layer to review what an unattended agent changed before it's trusted
- Additional adapters (Codex, Copilot) implementing the same `Adapter` interface
- `detect.go` emitting structured commands by default
- Per-task write-sets feeding a finer-grained drift check (still deferred from Phase 2)
- A real capability registry covering device protocols (serial, Modbus, OPC UA) once an MCP
  adapter layer exists to act on them

---

## Affected files

| File | Change type | Reason |
|---|---|---|
| `cli/internal/planmeta/planmeta.go` | Modify | Add `state`/`review`/`verification`/`requires_approval`/`approved_*` fields, legacy `status` migration, `AppendEvent` |
| `cli/internal/planmeta/planmeta_test.go` | Modify | Tests for migration + event appending |
| `cli/internal/workflow/workflow.go` | Create | State enum + transition table |
| `cli/internal/workflow/workflow_test.go` | Create | Transition-table tests |
| `cli/internal/workflow/profile.go` | Create | `harness/workflows/*.yaml` loader |
| `cli/internal/workflow/profile_test.go` | Create | Profile-loading tests |
| `cli/internal/capabilities/capabilities.go` | Create | LookPath-based detection |
| `cli/internal/capabilities/capabilities_test.go` | Create | Detection tests (using a fake PATH dir) |
| `cli/internal/agent/agent.go` | Create | `Adapter` interface + `ClaudeCodeAdapter` |
| `cli/internal/agent/agent_test.go` | Create | Prompt-assembly tests |
| `cli/internal/executil/executil.go` | Create | Structured-or-shell execution |
| `cli/internal/executil/executil_test.go` | Create | Both execution modes tested |
| `cli/workflow_cmd.go` | Create | `eng workflow start/status/advance` |
| `cli/start_cmd.go` | Create | `eng start` |
| `cli/capabilities_cmd.go` | Create | `eng capabilities list` |
| `cli/adapter_cmd.go` | Create | `eng adapter prompt` |
| `cli/plan_cmd.go` | Modify | Add `review`/`approve`/`block`/`cancel` subcommands; `new` gains `--requires-approval` |
| `cli/verify_cmd.go` | Modify | Persist `verification` to `plan.yaml`; use `executil` |
| `cli/hooks_cmd.go` | Modify | Use `executil` |
| `cli/install.go` | Modify | Copy binary to `bin/`; `--add-to-path` flag |
| `cli/doctor.go` | Modify | Add Capabilities section |
| `cli/main.go` | Modify | Dispatch new subcommands |
| `harness/workflows/quick-fix.yaml` | Create | Profile: triage → execute → verify |
| `harness/workflows/bug-fix.yaml` | Create | Profile: triage → plan → execute → verify |
| `harness/workflows/feature.yaml` | Create | Profile: triage → plan → review → execute → verify |
| `harness/workflows/architecture.yaml` | Create | Profile: triage → plan → review → approval → execute → verify |
| `harness/workflows/high-risk.yaml` | Create | Profile: triage → plan → review → approval → execute → verify (approval mandatory) |
| `harness/adapters/claude-code/ADAPTER.md` | Create | Adapter contract documentation |
| `harness/VERSION` | Modify | Bump `0.2.0-phase2` → `0.3.0-phase3` |
| `docs/src-map.md` | Modify | Add Phase 3 module entries (last task) |
| `docs/gotchas.md` | Modify | Mark the `eng` PATH gotcha resolved, cross-reference Decision 8 |
| `README.md` | Modify | Additive Phase 3 section |
| `ROADMAP.md` | Modify | Note Phase 3 plan link |

---

## Risks and unknowns

- **`exec.Command("claude").Run()` with inherited stdio (`eng start`)** — this is the first
  place `eng` launches another interactive program and blocks on it. If `claude` isn't
  actually interactive-terminal-friendly when spawned as a child process (some CLIs detect
  they're not attached to a real TTY and change behavior), `eng start` needs a graceful
  fallback to print-only — `tests.md` treats "prints instructions when launch isn't clean" as
  an acceptable pass condition, not just "successfully execs."
- **`setx` PATH length limits on Windows** — `setx` truncates the target variable at 1024
  characters in old Windows versions; `--add-to-path` should check the resulting length and
  warn rather than silently produce a broken PATH. Documented as a known limitation, not
  solved with extra registry-editing complexity in this MVP.
- **Legacy `status`→`state` migration is a one-way inference** — a Phase-2 plan.yaml with
  `status: executing` migrates to `state: EXECUTING`, but Phase 3's new fields
  (`requires_approval`, `review`, `verification`) will all be empty/zero for that plan. This is
  fine (it just means `eng workflow advance` treats it as "approval not required, no review
  recorded yet") but is worth stating explicitly rather than assuming.
- **`tasks.md` zero-`- [ ]`-lines as the EXECUTING→VERIFYING signal** reuses V1's own
  convention, but a `tasks.md` that was never fully following that convention (e.g., a very
  old plan, or one written by hand outside the template) could report "done" prematurely.
  Acceptable for MVP — this is exactly the signal `scripts/plan-executor.sh status` already
  trusts today, so it's not a new class of risk.

## Open questions

- [ ] Should `eng workflow advance` be safe to run repeatedly with no state change (i.e., is
  it idempotent when nothing new has happened)? **This plan requires yes** — every transition
  check is a pure read (file existence, `plan.yaml` fields, `git diff`), so calling `advance`
  twice in a row with nothing changed simply reports "no transition available yet, still in
  <state>" both times. This is what makes resume (goal #7) free instead of a separate feature.

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|---|---|---|---|
| Think Before Planning | spec.md written first; user-observable goal stated | 9/10 | Goal is CLI-observable (`eng workflow status` one-liner), not "roles are coordinated" |
| Simplicity First | ≥3 out-of-scope items with reasoning | 10/10 | 6 items excluded, each reasoned, plus 5 later improvements |
| Surgical Changes | Every file listed with change type and exact reason | 9/10 | New-package creation is coarser than single-symbol edits, same unavoidable caveat as Phases 1–2 |
| Goal-Driven Execution | Each scope item traces to an acceptance criterion | 9/10 | Every new `eng` subcommand and the full transition table gets a test in `tests.md` |

**Total: 37/40 → 9/10**

---

**User confirmation received:** [ ] Yes
**Confirmed on:** _pending_
