# Phase 10 Decision Log

## 1. `eng adapter prompt` becomes the activation boundary — no new `eng role activate` command

Every `core/*/METHOD.md` file and `core/runtime/METHOD.md` already instruct running `eng adapter
prompt <role> <plan-dir>` before acting in a role — the agent was always going to make this call.
Making that existing call validate role-vs-state and record the activation gets the enforcement
for free, with zero new command surface and zero change to the documented normal-usage flow
(`eng start` → natural language). A separate `eng role activate` command would duplicate this
call's job and give the agent two things to remember instead of one.

## 2. `role-state.yaml` is a new, separate per-plan file, not a `plan.yaml` field

`plan.yaml` (`planmeta.Meta`) is a small, stable, current-state snapshot already read/written by
a dozen call sites. Role activation is a distinct, higher-churn concern (it resets on every
replan cycle; `plan.yaml`'s other fields mostly don't). Keeping it separate means
`planmeta.Meta`'s shape and every existing caller stay untouched — purely additive at the
call-site level (new reads/writes in the specific places that need role-state, nowhere else).

## 3. No override flag for the task-completion temporal invariant

Considered an `eng plan override-invariant <plan-dir> --reason "..."` escape hatch for the rare
legitimate case (e.g. a plan with genuinely zero tasks). Rejected: an override flag is exactly
the kind of loophole that would let the next CredoID-shaped incident happen again, just with an
extra flag typed first. The correct remediation — uncheck the Completion checklist, let the
Executor genuinely complete it under `EXECUTING` — is already available with existing tools (no
new command needed) and doesn't weaken the invariant for anyone who doesn't bother finding the
override.

## 4. Only Executor and Verifier get hard, blocking transition gates

Planner and Plan Reviewer already have real gates (spec approval, review verdict) from Phases
3/5, and their `eng tools invoke` risk ceiling is already capped at `READ` (Phase 7). The
reported incident was specifically about Executor (work happened outside `EXECUTING`) and
Verifier (mechanical checks substituting for role review) — adding new hard transition
requirements to Planner/Reviewer beyond role-vs-state validation at the activation/tool-invoke
boundary would add friction to states this phase's own evidence never showed a problem in,
against the instruction's own "do not bundle unrelated refactors" and "keep Quick Fix
lightweight" guidance.

## 5. Verifier's own verdict command is `eng plan verify-review`, in the existing `eng plan` family

Considered a new `eng role verify` (mirroring a hypothetical `eng role activate`). Rejected once
Decision 1 established there's no `eng role` command group at all. `eng plan verify-review
<plan-dir> --verdict PASS|FAIL` sits naturally alongside the existing `eng plan review
--verdict PASS|REJECT` (Plan Reviewer's own verdict command) — same shape, same flag, same
`plan.yaml`-writing pattern, distinct name so it's never confused with mechanical `eng verify`.

## 6. Codex adapter: `codex.inspect`/`codex.review`/`codex.verify`, no `codex.execute`

Verified the real, installed `codex` CLI in this environment (`codex --help`, `codex exec
--help`, `codex review --help`) rather than guessing flags. `codex exec --sandbox read-only
"<prompt>"` and `codex review [...]` are real, non-interactive, read-only invocations. No write
capability is added — consistent with the instruction's explicit "start with read-only... do not
start with autonomous write execution," and with Phase 7's own pattern of never silently
defaulting a new capability above `READ` to `ALLOWED`. A future `codex.execute` would need its
own capability, its own risk tier, and its own explicit policy decision — not implied by this
one.

## 7. `CodexAdapter.Doctor()` uses `codex login status`, not the full `codex doctor`

Verified timing directly: `codex login status` returns in ~0.3s with no network round-trip
needed for a cached-credential check ("Logged in using ChatGPT"); the full `codex doctor` is
comprehensive but slower and far more verbose than a health-check callable on every `eng doctor`
run needs to be. `login status`'s exit code and output are sufficient to distinguish "installed"
(binary on PATH) from "wired" (adapter registered, `Doctor()` succeeds) from "invokable" (both,
surfaced as one combined field).

## 8. `agent.RolePermissions` gains `"codex"` for every role, since every `codex.*` capability is `READ`

The instruction's own suggested default (`codex.inspect` for Planner/Executor, `codex.review` for
Plan Reviewer, `codex.verify` for Verifier) is achievable without any per-role toolbox
restriction, because every Codex capability shipped in this phase is `READ`-risk and every role's
existing `RoleMaxRisk` already permits `READ`. Restricting *which* capability name each role
would realistically reach for is a documentation/METHOD.md concern, not a policy one — the
toolbox stays permissive (all four roles can use the `codex` adapter), and `toolpolicy.Decide`'s
role-toolbox check remains meaningful for adapters that do carry write capability (`git`,
`github`).

## 9. No project-level opt-out for the new role-enforcement gates

Considered a `workflow.enforce_roles: false` escape hatch alongside `triage`/`plan_review`/
`verifier`. Rejected: those three existing toggles gate *whether a stage runs at all*
(architectural choices a project can reasonably make); Phase 10's gates are a correctness
property of the state machine itself (a plan cannot claim to be `EXECUTING` before its Executor
was told so) — weakening that by config would silently reintroduce the exact defect this phase
exists to close, for any project that happened to set the flag. `workflow.verifier: false`
already provides the one dimension of real, legitimate flexibility (mechanical-only verification
for a project that doesn't want role-level review) without touching the Executor-activation gate
at all.

## 10. Verifier session independence is a documented recommendation, not a code mechanism

Section 19 asks whether some roles benefit from fresh session isolation, especially Verifier.
Phase 10 does not implement agent/session spawning (explicitly out of scope — "do not require
separate OS processes merely for appearance," and a real multi-process agent-spawning mechanism
is exactly the kind of new subsystem this phase is told not to build). The chosen design: same
Claude Code session may hold multiple roles across a plan's lifecycle, but each transition must
be explicit (Decision 1's activation call) and the prior role's elevated permissions do not
carry over (`toolpolicy.Decide`'s role-vs-active-role check uses `role-state.yaml`'s *current*
role, not whatever the session last claimed). `docs/USAGE.md` documents that a fresh Claude Code
session for Verifier specifically is recommended for stronger independence where practical, but
is not enforced — an honest "instructional, not technical" distinction, consistent with this
phase's own enforcement-boundary principle.

## 11. Two role events, not six

Section 17 suggests up to six event types (`role_activation_requested`, `role_activated`,
`role_prompt_generated`, `role_context_built`, `role_completed`, `role_blocked`) but also warns
"do not create noisy logs." Phase 10 emits exactly two: `role_activated` (on success, carrying
role, state, and context-manifest path as payload fields — folding "prompt generated"/"context
built" into this one event's fields rather than three separate events for one atomic action) and
`role_activation_denied` (on refusal, carrying the reason). `role_completed` doesn't need its own
event — the existing `state_changed` event already records every workflow transition, including
the ones a role's completion triggers.

## 12. Per-role context manifests are additive, not a breaking rename

`context-manifest-<role>.yaml` is new; the unqualified `context-manifest.yaml` keeps being
written on every `buildContextBundle` call (now as "whichever role activated most recently"),
so `eng context manifest <plan-dir>` with no role argument is byte-for-byte unchanged for any
existing caller or documentation that doesn't yet know roles exist.
