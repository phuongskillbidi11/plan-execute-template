# Phase 10 — Role Runtime Enforcement & Agent Delegation

## Goal

Turn `Planner → Plan Reviewer → Executor → Verifier` from a documented convention (four
METHOD.md prose files nobody's code reads) into a runtime-enforced lifecycle: role activation
is recorded and validated, execution cannot occur before `EXECUTING`, verification cannot be
satisfied by mechanical checks alone, and the workflow state machine can no longer "catch up"
after work has already happened. A minimal, read-oriented Codex adapter is added as a bounded
second-opinion delegation path, without weakening any of the above.

## Reproduced evidence (gap analysis)

### 1. Nothing technically enforces the CredoID-shaped bypass — reproduced directly

Built a deterministic local fixture (`benchmarks/fixtures/investigation-bypass/`) simulating the
real shape: a plan with a read-only "investigation" capability, advanced to `APPROVED`, then
`tasks.md`'s bottom Completion checklist hand-edited to fully `[x]` **while still `APPROVED`**
(simulating "the real work already happened, informally, before the state machine was told
anything"), then `eng workflow advance` run.

**Observed on current `main` (pre-Phase-10):**
```
APPROVED -> EXECUTING (no drift detected — Executor may begin)
EXECUTING -> VERIFYING (all tasks.md items checked off)
```
Both transitions fire in the *same* `eng workflow advance` invocation (`workflow.Decide`'s
`StateExecuting` case runs again immediately after the state write, since `tasksComplete()` was
already `true` the instant `EXECUTING` was entered). The state machine has no concept of "was
this task genuinely worked on during `EXECUTING`," only "does `tasks.md` currently say complete"
— a pure function of current file content, with no event-ordering awareness. This is the exact
mechanism behind the reported CredoID finding: real work happened while `APPROVED`, and the
state machine "caught up" the instant `eng workflow advance` was next run, because nothing
distinguishes "completed during EXECUTING" from "already complete before EXECUTING began."

`git diff`-based drift detection (`gitutil.ChangedFilesSince`) does not catch this class of
bypass at all: browser/UI investigation work typically produces no tracked-file diff, so
`checkDrift` — the *only* existing safeguard on the `APPROVED → EXECUTING` transition — is
structurally blind to it.

### 2. Role is a free-text claim, never a validated, persisted fact — confirmed by inspection

`cli/tools_cmd.go`'s `toolsInvoke` and `cli/internal/toolpolicy.Decide` both take `role` as a
bare string supplied by whoever runs `eng tools invoke <role> <capability> <plan-dir>` — there
is no check anywhere that this string corresponds to a role that has actually been activated for
*this* plan, nor that the *current workflow state* is compatible with that role. A session can
run `eng tools invoke executor <capability> <plan-dir>` while the plan is still `TRIAGED`, and
`toolpolicy.Decide` will allow it as long as the role-toolbox/risk-ceiling/project-policy checks
pass — it never looks at `plan.yaml`'s `state` field at all.

`cli/adapter_cmd.go`'s `cmdAdapter` (`eng adapter prompt <role> <plan-dir>`) is pure text
generation — it reads the role's `METHOD.md`, folds in a context bundle, and prints. It does not
validate role-vs-state, and it does not record that the call happened. This confirms the
reported "role prompt gap" exactly: nothing stops (or even notices) a single long-lived Claude
Code session self-narrating "I am now the Executor" without ever running this command, and
nothing distinguishes "this command was run for role X" from "the session just decided to call
itself role X out loud."

### 3. Mechanical verification and "did a Verifier actually review this" are the same code path

`workflow.Decide`'s `StateVerifying` case transitions straight to `StateCompleted` the instant
`eng verify`'s mechanical PASS is recorded (`workflowAdvance`'s `run_verify` action runs `eng
verify` synchronously as part of the *same* `EXECUTING → VERIFYING` transition, then immediately
calls `Decide` again). There is no second, independent verdict — `project.Workflow.Verifier`
(added in Phase 9, `*bool`, defaulting `true`) is accessor-complete (`VerifierEnabled()`) but has
**zero real consumers** — confirmed by grepping the codebase; it's reporting-only, exactly as
Phase 9's own `DECISION_LOG.md` already noted for `Triage`/`Verifier`. This is Phase 10's first
real consumer of that field.

### 4. `codex` capability semantics are conflated, confirmed against the real binary

`cli/internal/capabilities.Known = []string{"git", "claude", "codex", "docker", "gh"}`; `Detect`
is a bare `exec.LookPath`. `codex` **is** installed and authenticated in this environment
(verified directly: `codex doctor` reports auth configured via ChatGPT, reachable; `codex login
status` returns `Logged in using ChatGPT` in 0.3s, no network round-trip needed for the check).
`eng doctor`/`eng capabilities list` currently report only bare `available`/`unavailable` —
collapsing "the binary exists" and "the harness can actually invoke it for something useful"
into one signal, exactly the reported gap. No `tooladapter.Adapter` implementation for Codex
exists — `registeredAdapters` (`cli/tools_cmd.go`) only wires `git`, `github`, and the reference
MCP adapter.

## What can and cannot be technically enforced (read before designing the fix)

This is the honest boundary Phase 10 must document, not paper over (per the instruction's
repeated "do not pretend to enforce tools the harness cannot technically intercept"):

**Enforceable — the harness owns this invocation boundary:**
- `eng tools invoke <role> <capability> <plan-dir>` (the sole sanctioned path to an external
  capability, per Phase 7 design — already true, Phase 10 adds role/state validation to it).
- `eng adapter prompt <role> <plan-dir>` (the sole sanctioned path to a composed role
  prompt/context bundle — Phase 10 makes this the activation-recording boundary too).
- `eng workflow advance`'s own state transitions (Phase 10 adds the new invariants below).
- `eng plan <verb>` commands that write `plan.yaml` fields.

**Not enforceable — structurally outside a CLI harness's process boundary:**
- A Claude Code session's own native tools (Read/Edit/Write/Bash/WebFetch/browser-automation
  MCP tools). `eng` is a CLI the session *chooses* to invoke; there is no wrapper, sandbox, or
  daemon interposed on the session's own tool calls. A session that edits a file directly with
  its own Edit tool while the plan is `APPROVED` cannot be stopped by anything in this repo.

**The resolution Phase 10 actually implements, given that boundary:** the harness cannot prevent
a premature edit from happening, but it *can* — and after Phase 10, *does* — refuse to let the
workflow state machine retroactively legitimize it. `tasks.md` showing complete before
`EXECUTING` was ever entered is now a detected, blocking invariant violation, not a fact the
state machine silently trusts. This is the literal fix for the reproduced CredoID scenario, and
it is the correct scope for what a CLI-based harness can actually guarantee — not a claim that
Phase 10 prevents premature edits at the moment they happen.

`eng doctor` and this phase's docs state this distinction explicitly per capability/mechanism —
see the "Enforced vs. instructional" table in Task 9's documentation deliverable.

## Role runtime model

A new small package, `cli/internal/rolestate`, and one new small per-plan file,
`<plan-dir>/role-state.yaml` — deliberately not folded into `plan.yaml` (keeps `planmeta.Meta`'s
existing shape untouched, additive) and deliberately not a new distributed system (one file, one
struct, no daemon, no separate process).

```go
type RoleState struct {
    CurrentRole       string `yaml:"current_role"`        // "" | planner | plan-reviewer | executor | verifier
    ActivatedAt       string `yaml:"activated_at,omitempty"`
    ActivatedForState string `yaml:"activated_for_state,omitempty"` // workflow state at activation time
    PromptGeneratedAt string `yaml:"prompt_generated_at,omitempty"`
    ContextManifest   string `yaml:"context_manifest,omitempty"`    // path to this activation's context-manifest-<role>.yaml
}
```

Answers exactly the four questions section 5 of the instruction poses:
- **What role is active right now?** `RoleState.CurrentRole`.
- **Was its role prompt actually composed?** `RoleState.PromptGeneratedAt != ""`.
- **Is the current workflow state compatible with that role?** `rolestate.AllowedForState(role,
  state, isQuickFix)` — a pure, testable function (see State-to-role mapping below).
- **What operations may this role perform?** unchanged, existing Phase 7
  `agent.RolePermissions`/`agent.RoleMaxRisk`/`toolpolicy.Decide` — Phase 10 does not redesign
  the capability model, it connects role-state to it (see Tool-call enforcement below).

`role-state.yaml` is invalidated (reset to `CurrentRole: ""`) whenever the plan re-enters
`NEEDS_REPLAN` (drift or review rejection) — a stale activation from before a replan cycle must
not count afterward.

## Role activation is the existing `eng adapter prompt` command, not a new one

Considered adding a new `eng role activate <role> <plan-dir>` command. Rejected: `eng adapter
prompt <role> <plan-dir>` already is the documented, agent-run step every `core/*/METHOD.md`
file and `core/runtime/METHOD.md` instructs running before acting in a role, and it already
composes the prompt *and* folds in the context bundle (Phase 5 Decision 2). Making this existing
call **validate role-vs-state compatibility and record the activation** turns a call the agent
was already going to make into the enforcement boundary, with zero new command surface and zero
change to the documented normal-usage flow (`eng start` → natural language — section 29). See
DECISION_LOG.md Decision 1.

## State-to-role mapping

```go
// rolestate.AllowedForState — pure, testable.
TRIAGED, NEEDS_SPEC_APPROVAL, SPEC_APPROVED, NEEDS_REPLAN  -> planner
PLANNED, REVIEWED                                           -> plan-reviewer
APPROVED, EXECUTING, NEEDS_FIX                               -> executor
TRIAGED (when IsQuickFix)                                    -> executor (quick-fix's own fast path)
VERIFYING, COMPLETED                                         -> verifier
```
`REVIEWED`/`COMPLETED` are listed for their own role too (re-running `eng adapter prompt
plan-reviewer`/`verifier` after their gate has already passed is a legitimate "re-read my own
prior work" case, not a bypass) — any *other* role requesting a state it doesn't map to is
refused with a clear reason (fail closed, non-zero exit, no prompt/context printed).

## Two new hard transition gates (the actual CredoID fix)

1. **`APPROVED → EXECUTING` (and Quick Fix's `TRIAGED → EXECUTING`) requires the executor role
   to have been activated first.** `workflow.Facts` gains `ExecutorActivated bool`. `Decide`
   stays in `APPROVED`/`TRIAGED` with reason `"waiting on executor role activation (eng adapter
   prompt executor <plan-dir>)"` until true. This is what makes "role activation happens before
   work, not after" real: the harness will not agree the plan is `EXECUTING` until it has a
   recorded fact that the Executor was told so first.
2. **Task-completion temporal invariant.** At the exact moment `Decide` would transition into
   `EXECUTING` (both the formal and Quick Fix paths), it also checks whether `tasksComplete()` is
   *already* `true`. Under every normal flow this is false (the template's checklist starts
   unchecked) — the only way it's true here is if something checked every box before execution
   legitimately began, which is precisely the reproduced bypass shape. `Decide` refuses the
   transition with a distinct `Action: "invariant_violation"` and a reason naming the problem;
   remediation is to uncheck the checklist and let the Executor genuinely complete it under
   `EXECUTING` — no override flag is added (fail closed, per the instruction's explicit
   guidance; an override would just become a new loophole).

## Mechanical verification vs. role verification — kept genuinely separate

`planmeta.Meta` gains one additive field:
```go
type RoleVerification struct {
    Verdict    string `yaml:"verdict,omitempty"`     // PASS | FAIL
    VerifiedAt string `yaml:"verified_at,omitempty"`
    VerifiedBy string `yaml:"verified_by,omitempty"`
    Notes      string `yaml:"notes,omitempty"`
}
// Meta.RoleVerification RoleVerification `yaml:"role_verification,omitempty"`
```
A new command, `eng plan verify-review <plan-dir> --verdict PASS|FAIL [--by <name>] [--notes
"..."]` (naming/shape consistent with the existing `eng plan review`), records it and writes
`verifier-review.md` (mirroring `review.md`'s existing pattern). `workflow.Facts` gains
`RoleVerificationRequired` (= `cfg.Workflow.VerifierEnabled()` — Phase 9's accessor, now
actually consumed) and `RoleVerificationVerdict`. `Decide`'s `StateVerifying` case, on mechanical
`PASS`, now checks: if role verification is required and no verdict is recorded yet, stay in
`VERIFYING` ("mechanical verify PASSed — waiting on Verifier role verdict"); if role verification
FAILed, go to `NEEDS_FIX` (or `FAILED` if retry-exhausted) exactly like a mechanical FAIL; only
transition to `COMPLETED` once both are satisfied (or role verification isn't required, for a
project with `workflow.verifier: false` — fully backward compatible with Phase 9's own default).

Mechanical `eng verify` continues to run automatically on `EXECUTING → VERIFYING`, unchanged —
Phase 10 does not remove it, only stops it from being the *sole* gate to `COMPLETED` when a
Verifier role is configured to matter.

## Per-role context manifest evidence

`buildContextBundle` writes `<plan-dir>/context-manifest-<role>.yaml` (new, one per role,
never overwritten by a different role's activation) *and* continues writing the unqualified
`<plan-dir>/context-manifest.yaml` as a last-activation pointer (byte-identical to today's
single-file behavior for any caller that doesn't know about roles yet). `eng context manifest
<plan-dir> [role]` gains an optional second argument; omitted, it reads the unqualified file
exactly as today.

## Tool-call enforcement — connecting role-state to `toolpolicy.Decide`

`toolsInvoke` (`cli/tools_cmd.go`) now loads `role-state.yaml` before calling
`toolpolicy.Decide` and adds one more precedence check, ahead of everything else in `Decide`'s
existing chain: **the invoking role must match `RoleState.CurrentRole` for this plan, and the
current workflow state must be one `rolestate.AllowedForState` accepts for that role** —
otherwise `DENIED`, reason `"role not active for this plan/state"`, before any adapter is
touched, before any policy list is even consulted. This is what closes the "self-narrated role"
gap: claiming to be `executor` in the CLI invocation is no longer sufficient — the plan must
actually have recorded that activation first, via the same `eng adapter prompt executor
<plan-dir>` call every documented flow already tells the agent to make.

## Codex delegation — MVP scope decision

**Included**, read-only, additive, does not touch any enforcement code path above. A new
`tooladapter.CodexAdapter` (mirrors `GitHubAdapter`'s shape exactly) exposing three `READ`-risk
capabilities:
- `codex.inspect` → `codex exec --sandbox read-only "<prompt>"` (verified against the real,
  installed `codex` CLI in this environment — `exec` is documented as "Run Codex
  non-interactively"; `--sandbox read-only` is a real, documented flag, not guessed).
- `codex.review` → `codex review [--uncommitted|--base <branch>|--commit <sha>] "<prompt>"`.
- `codex.verify` → same invocation shape as `codex.inspect`, kept as a separate capability name
  so role-permission policy (below) can differ per name even though the underlying command is
  identical today.

No `codex.execute` capability exists — write execution is explicitly out of scope (section 22),
consistent with `toolcap.RiskWrite`-and-above requiring its own future, separately-reviewed
capability and policy decision, per Phase 7's own established pattern (`docs/tools.md`).

`Doctor()` runs `codex login status` (verified: ~0.3s, no network round-trip needed for a
cached-credential check) — distinguishing **installed** (`capabilities.Detect("codex")`) from
**wired** (the adapter is registered and its `Doctor()` succeeds) from **invokable** (both of
the above, exposed as one combined `eng doctor`/`eng capabilities` field). `agent.RolePermissions`
gains `"codex"` in every role's toolbox (Planner/Reviewer/Executor/Verifier can all *inspect* via
Codex — the instruction's own suggested default) — the `codex.*` capabilities' `READ` risk means
every role's existing risk ceiling already permits them without any role-ceiling change.

## Codex output — bounded and structured

`CodexAdapter.Invoke` reuses the existing `writeFullLog`/`summarizeOutput` compaction Phase
4/5/7 already established for verify/tool output — full output to `.agent/logs/codex-*.log`, a
bounded head+tail summary in the audit trail and printed output. No new compaction mechanism.

## Backward compatibility strategy

- **Legacy V1 projects** (no `.agent/`): completely untouched — none of this phase's code paths
  execute without a `.agent/project.yaml`, exactly like every prior phase's own workflow/context
  machinery.
- **A hybrid/modern project's pre-Phase-10 `plan.yaml`** with no `role-state.yaml`: `rolestate.Load`
  returns a zero-value `RoleState{}` (current_role `""`) when the file doesn't exist — behaves
  exactly like "no role has been activated yet," which is the correct, safe default (requires one
  `eng adapter prompt <role>` call to proceed past the new gates — not a silent brick, and not a
  silent bypass either).
- **`workflow.verifier: false`** (or any project whose `workflow:` block predates Phase 9):
  `RoleVerificationRequired` resolves `false` via the same `VerifierEnabled()` accessor Phase 9
  already built — `COMPLETED` is reachable via mechanical `PASS` alone, byte-identical to
  Phase 9's behavior.
- **Quick Fix** stays a single required command more than before (`eng adapter prompt executor
  <plan-dir>` before the fast-path transition fires) — not a new ceremony, since every documented
  flow already told the agent to run this command; it simply now has a recorded, checked effect.
- Every new `plan.yaml`/`project.yaml` field is `omitempty` — round-trips cleanly through an old
  file that never had it.

## Acceptance criteria (fixed before implementation)

1. The reproduced bypass fixture (`benchmarks/fixtures/investigation-bypass/`) now blocks: `eng
   workflow advance` on a plan with pre-checked `tasks.md` while `APPROVED` refuses to reach
   `EXECUTING`, reporting the invariant violation.
2. `eng tools invoke executor <capability> <plan-dir>` is `DENIED` when the executor role has not
   been activated for this plan, regardless of what the `<role>` argument claims.
3. `eng adapter prompt executor <plan-dir>` is refused (non-zero exit) when workflow state is
   `TRIAGED`/`PLANNED`/etc. (any state `rolestate.AllowedForState` doesn't map to `executor`,
   Quick Fix excepted).
4. `APPROVED → EXECUTING` only fires after `eng adapter prompt executor <plan-dir>` has
   succeeded for this plan.
5. `VERIFYING → COMPLETED` requires `eng plan verify-review --verdict PASS` in addition to
   mechanical `eng verify` PASS, whenever `workflow.verifier` resolves enabled; unaffected when
   disabled.
6. Each of Planner/Plan Reviewer/Executor/Verifier gets its own
   `context-manifest-<role>.yaml` once activated; the unqualified file still exists.
7. Quick Fix E2E (Phase 8/9's own proven sequence) still reaches `COMPLETED` with exactly one new
   required step (executor activation).
8. Legacy/hybrid E2E (Phase 8/9's own proven sequences) are byte-for-byte unchanged.
9. `eng doctor`/`eng capabilities list` distinguish `installed`/`wired`/`invokable` for `codex`,
   not a single `available` flag.
10. A real `codex.inspect` invocation via `eng tools invoke <role> codex.inspect <plan-dir>
    "<bounded prompt>"` succeeds against the actually-installed, actually-authenticated `codex`
    binary in this environment, is bounded in the printed/audited output, and is denied for a
    role/state combination it shouldn't be allowed in.

## Explicitly out of scope

Everything section 39 lists, restated for this plan's own record: full MCP marketplace,
plugin/skill marketplace, distributed agents, parallel autonomous swarm, production deployment
automation, live PLC write, arbitrary SSH execution, destructive database automation, cloud
orchestration, vector DB, agent memory platform, `codex.execute` (write capability), a wrapper/
sandbox that could technically intercept a Claude Code session's own native tool calls (not
buildable within this harness's architecture — see the enforcement-boundary section above).

## Fixture strategy

`benchmarks/fixtures/investigation-bypass/` — a minimal Go project (matching the established
fixture convention) with a `docs/src-map.md` naming a read-only "investigation" capability
concept, used purely to reproduce and regression-test the temporal invariant; no CredoID-specific
content, no private artifacts, nothing beyond what's needed to exercise the state machine.
