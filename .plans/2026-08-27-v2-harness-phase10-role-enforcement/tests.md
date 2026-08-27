# Phase 10 Acceptance Tests

## T1 — Reproduced bypass, now blocked

```bash
# against benchmarks/fixtures/investigation-bypass/, plan advanced to APPROVED,
# tasks.md bottom checklist hand-checked while still APPROVED
eng workflow advance <plan-dir>
```
**Pass:** does NOT reach `EXECUTING`; reports the invariant-violation reason naming pre-execution
completion.

## T2 — Role-state package

```bash
cd cli && go test ./internal/rolestate/... -v
```
**Pass:** all cases pass — round-trip, missing-file default, every state-to-role mapping row
(positive and negative), `NextRole` for every non-terminal state.

## T3 — Workflow gates

```bash
cd cli && go test ./internal/workflow/... -v
```
**Pass:** new `ExecutorActivated`/temporal-invariant/`RoleVerification` cases pass; every
pre-existing Phase 3/5/9 transition test still passes unmodified.

## T4 — Activation boundary

```bash
# TRIAGED state
eng adapter prompt executor <plan-dir>
```
**Pass:** non-zero exit, no prompt/context printed, `role_activation_denied` event recorded.

```bash
# APPROVED state
eng adapter prompt executor <plan-dir>
eng workflow status <plan-dir>
```
**Pass:** succeeds, `role-state.yaml` shows `current_role: executor`, `role_activated` event
recorded, `context-manifest-executor.yaml` exists.

## T5 — Tool invocation gated by active role

```bash
# only planner activated so far
eng tools invoke executor git.status <plan-dir>
```
**Pass:** `DENIED`, reason names "role not active."

```bash
eng adapter prompt executor <plan-dir>
eng tools invoke executor git.status <plan-dir>
```
**Pass:** `ALLOWED` (same as Phase 7's existing behavior, now reachable only after activation).

## T6 — Mechanical vs. role verification

```bash
# workflow.verifier enabled (default), plan reaches VERIFYING, eng verify PASSes
eng workflow status <plan-dir>
```
**Pass:** reports mechanical verification PASS, role verification pending, state still
`VERIFYING`.

```bash
eng adapter prompt verifier <plan-dir>
eng plan verify-review <plan-dir> --verdict PASS
eng workflow advance <plan-dir>
```
**Pass:** now reaches `COMPLETED`.

```bash
# workflow.verifier: false project
```
**Pass:** mechanical PASS alone reaches `COMPLETED`, byte-identical to Phase 9 behavior.

## T7 — Per-role context manifests

```bash
eng adapter prompt planner <plan-dir> "..."
eng adapter prompt executor <plan-dir>
ls <plan-dir> | grep context-manifest
```
**Pass:** both `context-manifest-planner.yaml` and `context-manifest-executor.yaml` exist;
`context-manifest.yaml` (unqualified) also exists and matches the most recent activation.

## T8 — `eng doctor` / `eng capabilities` Codex semantics

```bash
eng doctor
eng capabilities list --verbose
```
**Pass:** `codex` reports `installed`/`wired`/`invokable` as distinct fields, not one flag,
matching this environment's real state (installed, wired, invokable — codex is authenticated
here).

## T9 — Real, bounded Codex invocation

```bash
eng adapter prompt planner <plan-dir> "..."
eng tools invoke planner codex.inspect <plan-dir> "<short bounded prompt>"
```
**Pass:** succeeds against the real `codex` binary; output is bounded in the printed
result; `.agent/logs/codex-*.log` exists; a `tool_invocation` audit event is recorded.

## T10 — Full regression

```bash
cd cli && go build ./... && go vet ./... && go test ./...
```
**Pass:** zero failures.

```bash
go test ./... -run TestRouterEvalScenarios -v
```
**Pass:** unchanged, all scenarios pass.

Quick Fix / Spec-First / Legacy / Hybrid E2E (Phase 8/9's own proven sequences, re-run against
scratch fixtures): **Pass:** each reaches its expected terminal state; Quick Fix and any
Verifier-enabled flow now include exactly one new required activation step, everything else
unchanged.

## T11 — Acceptance criteria

Every criterion in spec.md's "Acceptance criteria" section and the instruction's section 43/44
is satisfied by one of T1–T10 above, or by a specific `benchmarks/results/*.yaml` file cited in
the final Phase 10 report. No criterion is marked satisfied without a specific test name or
result file backing it.
