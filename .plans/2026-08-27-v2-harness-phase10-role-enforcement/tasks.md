# Phase 10 Tasks — Role Runtime Enforcement & Agent Delegation

Execute in order. Every task: reproduce (if not already done in spec.md) → regression test →
minimal fix → focused verification → note any downstream re-check needed. Do not bundle
unrelated refactors.

---

## Task 1 — Reproduce the bypass with a deterministic fixture

- [x] **1.1** Created `benchmarks/fixtures/investigation-bypass/` — a minimal Go project
  (`go.mod` + `main.go`) plus a `docs/src-map.md` naming a generic read-only "investigation"
  capability concept. No CredoID-specific content.
- [x] **1.2** From a scratch copy, scaffolded a `--risk feature` plan, advanced it cleanly to
  `APPROVED`, then hand-checked every line of `tasks.md`'s bottom Completion checklist to `[x]`
  **while still `APPROVED`**. `eng workflow advance` (2 calls): `APPROVED -> EXECUTING` (no
  drift — the checklist edit produces no tracked-file diff), then immediately
  `EXECUTING -> VERIFYING -> COMPLETED` in the very next call. Mechanical `eng verify`'s own
  report shows an **empty git diff since the plan's git_sha** — proving zero actual execution
  ever happened, exactly the reproduced CredoID signature.
- [x] **1.3** Recorded as the failing baseline in this task's own notes (above); properly
  recorded as a benchmark result once fixed, in Task 9.3.

**Verify:** the bypass is reproduced against the real binary, not asserted from spec.md's prose
alone.

---

## Task 2 — `rolestate` package

- [x] **2.1** Create `cli/internal/rolestate/rolestate.go`: `RoleState` struct (per spec.md),
  `FileName = "role-state.yaml"`, `Load(planDir) (*RoleState, error)` (returns a zero-value
  `&RoleState{}` — not an error — when the file doesn't exist, matching Decision on backward
  compatibility), `Save(planDir, *RoleState) error`, `Reset(planDir) error` (writes a zero-value
  state — used on `NEEDS_REPLAN`).
- [x] **2.2** Add `AllowedForState(role, state string, isQuickFix bool) (bool, string)` — pure,
  per spec.md's state-to-role mapping table; returns `(false, "<reason>")` for any unmapped
  combination.
- [x] **2.3** Add `NextRole(state string, isQuickFix bool) string` — the single deterministic
  "who should act next" function, reused by `eng doctor`/`eng workflow status` (Task 7) so there
  is exactly one place this mapping is defined (`AllowedForState` and `NextRole` share the same
  underlying table).
- [x] **2.4** Unit tests: round-trip `Load`/`Save`; `Load` on a missing file returns a usable
  zero value, not an error; every state-to-role mapping row from spec.md, both positive
  (`AllowedForState` true) and at least one negative per role (a role requesting an incompatible
  state); `NextRole` for every non-terminal state.

**Verify:** `go test ./internal/rolestate/... -v` — all green; package has zero dependencies on
`cli/*.go` (pure, testable in isolation, matching `workflow`'s own existing design principle).

---

## Task 3 — Workflow state machine: executor-activation gate + temporal invariant

- [x] **3.1** Add `workflow.Facts.ExecutorActivated bool`. In `Decide`'s `StateApproved` case,
  after the existing drift check, add: if `!f.ExecutorActivated`, stay in `APPROVED` with reason
  `"waiting on executor role activation (eng adapter prompt executor <plan-dir>)"`. Same check
  added to the `StateTriaged` quick-fix branch before its existing `EXECUTING` transition.
- [x] **3.2** Add the temporal invariant: in both of the above transition points, if
  `f.TasksComplete` is *already* `true` at the moment the transition into `EXECUTING` would
  otherwise fire, refuse with `Decision{NextState: <unchanged>, Reason: "tasks.md already shows
  complete before execution began — this looks like retroactive completion; uncheck the
  Completion checklist and let the Executor genuinely complete it under EXECUTING", Action:
  "invariant_violation"}`. This check runs *before* the executor-activation check (a plan that's
  both un-activated and suspiciously pre-complete should report the more serious problem first).
- [x] **3.3** Add `workflow.Facts.RoleVerificationRequired bool` and
  `RoleVerificationVerdict string`. In `Decide`'s `StateVerifying` case, on mechanical `PASS`:
  if `RoleVerificationRequired && RoleVerificationVerdict == ""`, stay in `VERIFYING` ("mechanical
  verify PASSed — waiting on Verifier role verdict (eng plan verify-review)"); if
  `RoleVerificationRequired && RoleVerificationVerdict == "FAIL"`, go to `NeedsFix` (or `Failed`
  if retry-exhausted, reusing the existing mechanical-FAIL branch's logic); otherwise (not
  required, or required and `PASS`) transition to `Completed` as today.
- [x] **3.4** Unit tests in `cli/internal/workflow/workflow_test.go`: `APPROVED` stays `APPROVED`
  when `ExecutorActivated` is false; transitions to `EXECUTING` once true (and `TasksComplete` is
  false, the normal case); the temporal-invariant case (`TasksComplete` true, `ExecutorActivated`
  false or true) refuses with `Action: "invariant_violation"`; Quick Fix's `TRIAGED` branch mirrors
  all three; `VERIFYING` with `RoleVerificationRequired=true` and no verdict stays `VERIFYING`;
  with verdict `FAIL` goes to `NeedsFix`; with verdict `PASS` (or `RoleVerificationRequired=false`)
  goes to `Completed` — confirm this last case is byte-identical to Phase 9's existing
  `TestVerifyingRoutesOnVerdict` behavior when `RoleVerificationRequired` is false (backward
  compatibility regression guard).

**Verify:** `go test ./internal/workflow/... -v` — all green, including every pre-existing test
unmodified and passing (Phase 3/5/9's own transition tests must not regress).

---

## Task 4 — `eng adapter prompt` becomes the activation boundary

- [x] **4.1** In `cli/adapter_cmd.go`'s `cmdAdapter`: before composing the prompt, loads
  `plan.yaml` and calls `rolestate.AllowedForState(role, meta.State, meta.RiskLevel ==
  "quick-fix")`. On false: prints the reason, appends `role_activation_denied`
  (`planmeta.AppendStructuredEvent`, fields `role`/`state`/`reason`), exits non-zero — no prompt
  or context bundle printed (fail closed).
- [x] **4.2** On success: composes the prompt and context bundle as before (per-role manifest
  writing implemented as part of Task 6.5, done alongside this task since they're tightly
  coupled); writes `role-state.yaml` (`CurrentRole`, `ActivatedAt`, `ActivatedForState`,
  `PromptGeneratedAt`, `ContextManifest`); appends a `role_activated` event with the same fields.
- [x] **4.3** `cli/workflow_cmd.go`'s `workflowAdvance` now calls `rolestate.Reset(planDir)`
  whenever a transition's resulting state is `NEEDS_REPLAN` (covers both the drift-detected and
  review-REJECT paths, since both flow through the same `StatePlanned`/`StateApproved` `Decide`
  branches into the same state-write code path).
- [x] **4.4** No new `cli/adapter_cmd_test.go` unit test file: this codebase's own established
  convention (`plan_cmd_test.go`, `skilleval_integration_test.go`) only unit-tests pure helper
  functions in `package main`, never `cmd*` functions that call `os.Exit` — `cmdAdapter`'s new
  logic is thin glue over already-unit-tested pure functions (`rolestate.AllowedForState`,
  `Load`/`Save`, both covered in Task 2). Verified instead via real end-to-end runs against the
  actual binary (below) — matching how every prior phase's own CLI-glue changes were verified.

**Verify:** confirmed for real — `eng adapter prompt executor <plan-dir>` against a plan still in
`TRIAGED` printed `REFUSED: role executor is not compatible with state TRIAGED`, exit 1, no
prompt; against the same plan advanced to `APPROVED`, it succeeded (exit 0), wrote
`role-state.yaml` (`current_role: executor`, `activated_for_state: APPROVED`), wrote
`context-manifest-executor.yaml` alongside the existing `context-manifest.yaml`, and recorded
both a `role_activation_denied` event (from the earlier refusal) and a `role_activated` event in
`events.jsonl`. The subsequent `eng workflow advance` then correctly fired
`APPROVED -> EXECUTING`, which had been refused before activation.

---

## Task 5 — `eng tools invoke` checks the active role

- [x] **5.1** In `cli/tools_cmd.go`'s `toolsInvoke`: after loading `plan.yaml`, also loads
  `role-state.yaml`. Before calling `toolpolicy.Decide`: denies (with a specific reason
  distinguishing "no activation on record" from "role incompatible with current state") when
  `rs.CurrentRole != role`, or when `rolestate.AllowedForState(role, meta.State, ...)` is false
  — writes the `tool_invocation` audit event and exits non-zero, without calling
  `toolpolicy.Decide` or touching the adapter.
- [x] **5.2** No new `cli/tools_cmd_test.go` unit test file — same rationale as Task 4.4
  (`toolsInvoke` is a `cmd*`-style function calling `os.Exit`; the codebase's own convention
  tests such glue via real end-to-end runs, not synthetic unit tests around exit paths). Verified
  for real, below.

**Verify:** confirmed for real against `benchmarks/fixtures/investigation-bypass/` — a fresh
quick-fix plan's `eng tools invoke executor git.status <plan-dir>` (no activation yet) printed
`REFUSED (DENIED): role not active for this plan — no activation on record for executor (run
\`eng adapter prompt executor <plan-dir>\` first)`, exit 1. After `eng adapter prompt executor
<plan-dir>` succeeded, the identical `eng tools invoke` call succeeded (real `git status` output
returned, exit 0) — proving the gate closes exactly the reported "self-narrated role" bypass:
claiming `role=executor` in the CLI call is no longer sufficient on its own.

---

## Task 6 — Mechanical vs. role verification: `eng plan verify-review` + per-role manifests

- [x] **6.1** Added `planmeta.RoleVerification` struct and `Meta.RoleVerification` field (both
  `omitempty`).
- [x] **6.2** Added `planVerifyReview` in `cli/plan_cmd.go` (`eng plan verify-review <plan-dir>
  --verdict PASS|FAIL [--by <name>] [--notes "..."]`), mirroring `planReview`'s exact shape.
  Corrected the plan's own assumption during implementation: `planReview` does **not** write
  `review.md` (that's a static template file `eng plan new` scaffolds, filled in by hand) — so
  `planVerifyReview` doesn't write `verifier-review.md` either; instead added
  `harness/templates/plan/verifier-review.md` as a new scaffolded template file, matching
  `review.md`'s real, verified pattern exactly rather than an assumed one.
- [x] **6.3** Wired `plan verify-review` into `cmdPlan`'s dispatch switch and `main.go`'s usage.
- [x] **6.4** `gatherFacts` populates `RoleVerificationRequired: cfg.Workflow.VerifierEnabled()`
  and `RoleVerificationVerdict: meta.RoleVerification.Verdict`.
- [x] **6.5** `buildContextBundle` writes both `context-manifest-<role>.yaml` and the unqualified
  `context-manifest.yaml`. `contextManifest` (`eng context manifest <plan-dir> [role]`) reads the
  qualified file when a role argument is given.
- [x] **6.6** Added `TestRoleVerificationRoundTrip`/`TestRoleVerificationDefaultsToEmptyOnOldPlanYAML`
  to `cli/internal/planmeta/planmeta_test.go`. No new `plan_cmd_test.go`/`context_cmd_test.go`
  unit tests for the CLI glue itself — same established-convention rationale as Tasks 4.4/5.2;
  verified for real instead (below).

**Verify:** confirmed for real, end-to-end, against `benchmarks/fixtures/investigation-bypass/`:
after tasks complete, `EXECUTING -> VERIFYING` fired and mechanical `eng verify` PASSed, but the
plan correctly *stayed* `VERIFYING` ("mechanical verify PASSed — waiting on Verifier role
verdict") instead of auto-completing. `eng plan verify-review <dir> --verdict PASS --by t
--notes "..."` recorded `plan.yaml`'s `role_verification` block; the next `eng workflow advance`
then correctly reached `COMPLETED`. `context-manifest-executor.yaml`,
`context-manifest-verifier.yaml`, and the unqualified `context-manifest.yaml` all present and
correct in the plan directory.

---

## Task 7 — `eng doctor` / `eng workflow status` observability

- [x] **7.1** Decided during implementation (per the task's own instruction to check first):
  `eng doctor` has no plan-dir concept at all (`cmdDoctor` takes no plan argument, scopes to the
  project) — `active_role`/`next_role` genuinely belong to `eng workflow status` (which does take
  a plan-dir), not `eng doctor`. `eng doctor`'s own Phase 9 `Workflow:` block (`triage`/
  `plan_review`/`verifier`/`planning`) is project-level config, unaffected. No change needed
  here beyond Task 7.3's `Tools:` section extension.
- [x] **7.2** `cli/workflow_cmd.go`'s `workflowStatus` now prints `Active role:`/`Next role:`
  (via `rolestate.NextRole`) after the existing `Requires approval:` line, and (only when
  `State == VERIFYING`) `Mechanical verification:`/`Role verification:` lines after the existing
  `Next:` line — every pre-existing line unchanged, only new ones added.
- [x] **7.3** `cli/doctor.go`'s `Tools:` section now shows `installed=`/`wired=yes`/
  `invokable=` for **every** adapter (not codex-specific — a small, more general fix than
  special-casing one adapter, and `wired` is trivially `yes` for anything reaching this loop
  since only registered adapters appear in it at all). `invokable` calls the adapter's own
  `Doctor()`.
- [x] **7.4** Verified for real: `eng workflow status <dir>` on a real scratch plan shows
  `Active role:   executor` / `Next role:     executor` correctly; `eng doctor` on this repo
  shows `codex      installed=true  wired=yes invokable=true  [3 capabilities]` — matching this
  environment's real, verified state exactly.

**Verify:** confirmed — both `eng doctor` and `eng workflow status`'s new output matches real
runs, not just documentation. The instruction's illustrative `eng doctor` shape used an
"Agents:" section label for Codex; implemented instead as part of the existing `Tools:` section
(Codex is a `tooladapter.Adapter`, exposing `codex.*` capabilities in the same dot-naming
convention as `git.*`/`github.*` — architecturally the right fit for Phase 7's existing
capability/policy/audit machinery, not a second coding-agent subsystem — see DECISION_LOG.md
Decision 6).

---

## Task 8 — Codex adapter MVP

- [x] **8.1** Create `cli/internal/tooladapter/codex.go`: `CodexAdapter` struct, `Name()`
  `"codex"`, `Provider()` `"cli-agent"`, `Capabilities()` → `codex.inspect`/`codex.review`/
  `codex.verify`, all `toolcap.RiskRead`. `Doctor()` runs `codex login status` (verified real
  command). `Invoke` shells to `codex exec --sandbox read-only "<prompt>"` for
  `inspect`/`verify`, `codex review [...] "<prompt>"` for `review` (verified real flags — no
  guessed syntax).
- [x] **8.2** Added `NewCodexAdapter(available bool) CodexAdapter` and registered it in
  `cli/tools_cmd.go`'s `registeredAdapters`.
- [x] **8.3** Added `"codex"` to every role's entry in `agent.RolePermissions`.
- [x] **8.4** Implemented differently than originally planned, more simply: rather than
  extending `cli/internal/capabilities`, `eng doctor`'s own `Tools:` loop (Task 7.3) computes
  `installed`/`wired`/`invokable` directly from each real `tooladapter.Adapter`'s
  `Available()`/`Doctor()` — generic for every adapter, not a codex-specific addition to a
  second package. Smaller diff, same observable result.
- [x] **8.5** Added `cli/internal/tooladapter/codex_test.go` mirroring `github_test.go`'s exact
  pattern, plus `TestCodexAdapterNoExecuteCapability` (explicit scope guard). No automated test
  invokes a real `codex exec`/`codex review` AI call — `TestCodexAdapterLiveDoctorIfInstalled`
  only checks `Doctor()` (fast, no API cost), matching `github_test.go`'s own convention exactly.
- [x] **8.6** Ran a real, bounded, one-time `codex.inspect` invocation: `eng tools invoke
  executor codex.inspect <plan-dir> "In one sentence, what does this repository's go.mod module
  name say? Do not modify anything."` against the actually-installed, actually-authenticated
  `codex` binary. Confirmed: real output (Codex read `go.mod` via a read-only sandboxed
  PowerShell call and answered correctly, 4,501 tokens), `.agent/logs/tool-codex-*.log` written,
  a `tool_invocation` audit event recorded, and a separate denial case (`role=planner`, not
  activated) correctly refused. Recorded in
  `benchmarks/results/codex-adapter-live-verification.yaml`.

**Verify:** `eng doctor` shows `codex: installed=yes wired=yes invokable=yes` in this
environment; `eng capabilities list --role planner` includes `codex`; a real, bounded
`codex.inspect` call succeeds end-to-end.

---

## Task 9 — Investigation-bypass regression scenarios (the full required list)

- [x] **9.1** Covered every scenario in instruction section 34, each traceable to a specific
  test or a real end-to-end run recorded above/below: Planner cannot execute
  (`TestDecideRoleRiskCeilingDenied`, pre-existing Phase 7 test, confirmed unweakened); Reviewer
  cannot execute (new `TestDecideRoleRiskCeilingDeniedForPlanReviewer`); Executor blocked before
  activation / allowed after (`TestApprovedStaysApprovedUntilExecutorActivated`,
  `TestApprovedTransitionsOnceActivatedAndNotPreComplete`, plus real E2E in Task 4); Verifier
  cannot mutate (new `TestDecideRoleRiskCeilingDeniedForVerifier`); task completion before
  `EXECUTING` rejected / workflow cannot retroactively legitimize completed work (the temporal-
  invariant tests + `benchmarks/results/investigation-bypass-blocked.yaml`, Task 9.3); role
  prompt generated per role / role context evidence generated per role (real E2E for **all
  four** roles — planner and plan-reviewer activation verified fresh in this task, executor and
  verifier already verified in Tasks 4/6 — plus `TestBuildContextBundleWritesPerRoleManifest`);
  mechanical verify != role verify (Task 6's `VERIFYING` gate tests + real E2E); Quick Fix
  remains lightweight (real E2E in Task 5 — one extra required activation call, otherwise
  unchanged); legacy/hybrid unaffected (deferred to Task 11's full regression re-run, which is
  the right place to prove "unaffected" against the actual Phase 8/9 E2E sequences).
- [x] **9.2** Codex-specific scenarios: wired-and-invokable and audited are covered by the real
  E2E in Task 8.6; role permissions by new `TestRoleMayUseCodexInEveryRolesToolbox`
  (`cli/internal/agent/permissions_test.go`); output-bounded structurally (Codex's `Invoke` path
  reuses the same `writeFullLog`/`summarizeOutput` every other adapter's output goes through —
  confirmed via the real E2E's own log file). "Detected but unwired" is covered by
  `TestCodexAdapterUnavailableRefuses` (an unavailable adapter's `Doctor()` errors — the exact
  condition that shows as `installed=false invokable=false` in `eng doctor`); no separate
  synthetic doctor-output test was added, since `cmdDoctor` isn't structured for output-capture
  testing (consistent with this codebase's established `cmd*` testing convention).
- [x] **9.3** Re-ran the exact bypass shape against the fixed binary, on a fresh fixture copy
  (not the same scratch dir as the original reproduction): confirmed it now stays `APPROVED`
  with the invariant-violation reason, persists across repeated `eng workflow advance` calls,
  and the correct remediation (uncheck, activate, advance) proceeds cleanly. Recorded in
  `benchmarks/results/investigation-bypass-blocked.yaml`.

**Verify:** every named scenario in instruction sections 34 and 44 has a passing test or a real,
recorded end-to-end run, traceable back to this checklist.

---

## Task 10 — Documentation

- [x] **10.1** `docs/ARCHITECTURE.md`: added "Role runtime enforcement (Phase 10)" section after
  "The lifecycle state machine" — state-to-role mapping, the two hard gates, and the
  enforced-vs-instructional boundary table; also updated "Tool / capability model" and
  "Security / safety model" sections and "Current maturity".
- [x] **10.2** `docs/USAGE.md`: updated the Workflow states table (APPROVED/VERIFYING rows),
  added a "Task-completion temporal invariant (Phase 10)" paragraph, a new "## Role activation"
  section, a new "## Codex delegation" section, and `codex.*` rows in the command reference
  table. Verifier-session-independence stated explicitly as a recommendation, not an enforced
  mechanism (Decision 10).
- [x] **10.3** `docs/tools.md`: added "## The Codex adapter — read-only second-opinion
  delegation (Phase 10)" section, and updated "Inspecting routing"/"Invoking a capability" to
  cover the installed/wired/invokable distinction generically, not Codex-specific.
- [x] **10.4** Added an "Activation (Phase 10)" note to `harness/core/runtime/METHOD.md` and
  each of the four role `METHOD.md` files (`planner`, `plan-reviewer`, `executor`, `verifier`) —
  each states that `eng adapter prompt <role> <plan-dir>` now validates and records activation,
  and that a refusal means the workflow isn't ready for that role yet.
- [x] **10.5** `docs/gotchas.md`: added "### The workflow state machine could 'catch up' after
  real work already happened" (trap/symptom/fix format, linked to this plan and to
  `benchmarks/results/investigation-bypass-blocked.yaml`).
- [x] **10.6** `README.md`: updated "Current phase / maturity", "Tool / MCP runtime", "Known
  limitations" (Phase 10 item resolved-pointer added, plus the three genuinely remaining open
  items — no fresh-session Verifier isolation, no interception of an agent's own native tool
  calls, browser/MCP tool use outside `eng tools invoke` is instructional only), and the Roadmap
  pointer text. `ROADMAP.md`: updated the intro paragraph and "Where things stand" to mention
  Phase 10's closed bypass (linking to the same benchmark result file) and "Now" to mention the
  enforced role lifecycle and Codex delegation path, plus the Verifier-isolation latent risk —
  no duplication, each fact stated once and cross-referenced to `docs/ARCHITECTURE.md`.

**Verify:** no documentation duplication (each fact stated once, cross-referenced); every
command/field shown was checked against the actual Phase 10 code before being documented.

---

## Task 11 — Full regression re-verification

- [x] **11.1** `cd cli && go build ./... && go vet ./... && go test ./...` — zero failures
  (re-run twice: cached and fresh `-count=1`, both clean across all 25 packages).
- [x] **11.2** Re-ran the V1 regression check (confirmed `scripts/` has zero diff this phase —
  the raw V1 path is unaffected by construction; re-ran the hybrid/harness-v2 path instead,
  since that's the one that actually runs Phase 10 code), Quick Fix E2E, and Spec-First feature
  E2E, each on a fresh fixture copy against the built Phase 10 binary — all reached `COMPLETED`
  with exactly the expected new activation steps and no other behavior change. Full detail in
  `benchmarks/results/phase8-9-scenarios-phase10-reverify.yaml`.
- [x] **11.3** Re-ran `go test ./... -run TestRouterEvalScenarios -v -count=1` — all 5 scenarios
  pass unchanged, confirming skill routing is unaffected by this phase.
- [x] **11.4** Recorded `benchmarks/results/phase8-9-scenarios-phase10-reverify.yaml` per Phase
  9's own established convention (new file, no prior result files overwritten).
- [x] **11.5** Every spec.md acceptance criterion (1-10) verified: 1-6 and 9-10 via Tasks 1-9's
  own recorded real runs (`investigation-bypass-blocked.yaml`,
  `codex-adapter-live-verification.yaml`, `TestApproved*`/`TestQuickFix*`/`TestVerifying*` in
  `workflow_test.go`, `TestBuildContextBundleWritesPerRoleManifest`); 7-8 via this task's own
  fresh Quick Fix and Spec-First E2E re-runs (criterion 8's "byte-for-byte unchanged" holds for
  every pre-existing line/behavior — the only diff anywhere is the designed additive one: new
  `Active role:`/`Next role:` status lines and the required activation calls).

**Verify:** every command in this task was actually run in this session, with real output.

---

## Task 12 — Git

- [ ] **12.1** Inspect the complete diff. Exclude stray binaries/logs/scratch output. Confirm no
  CredoID-specific or private dogfooding content was copied in — the fixture is synthetic, per
  Task 1.
- [ ] **12.2** Stage only Phase 10's durable changes and commit locally with the suggested
  message. Do not push.

**Verify:** `git log -1` shows the new commit; `git status` is clean; no `git push` was run.
