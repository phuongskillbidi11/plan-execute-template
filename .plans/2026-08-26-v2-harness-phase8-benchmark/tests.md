# Phase 8 Tests — Dogfooding, Benchmarking & Validation

Most acceptance criteria live inline in `tasks.md`'s own "Verify" lines, since Phase 8's
tasks are execution-and-recording steps, not code diffs. This file is the consolidated
gate list plus the final regression sweep.

## T1 — Skeleton and scenarios

```bash
ls benchmarks/scenarios/*.yaml | wc -l   # expect 12
ls benchmarks/fixtures/                  # expect quick-fix, feature, bug, large-context, legacy
test -s benchmarks/README.md && echo OK
```

**Pass:** 12 scenario files, 5 fixture directories, non-empty README.

## T2 — Fixtures build and the seeded bug actually fails

```bash
for d in benchmarks/fixtures/quick-fix benchmarks/fixtures/feature benchmarks/fixtures/large-context; do
  (cd "$d" && go build ./... && echo "OK: $d")
done
(cd benchmarks/fixtures/bug && go test ./... ; echo "expected non-zero: $?")
```

**Pass:** every fixture except `bug` builds clean; `bug`'s test suite fails (non-zero exit)
before any fix is applied.

## T3 — Category 1 (Quick Fix) success criteria

**Pass (Mode C):** `benchmarks/results/quick-fix-timeout-harness-v2.yaml` shows
`workflow_selected: quick-fix`, `implementation.files_changed: 1`,
`verification.verdict: PASS`, and no `spec.md`/`tasks.md`/`tests.md` beyond the minimal
quick-fix template ever existed for this plan.
**Fail:** any full-feature artifact, or `PLANNED`/`REVIEWED`/`APPROVED` ever appears in that
plan's `events.jsonl`.

## T4 — Category 2 (Feature/Spec-First) success criteria

**Pass (Mode C):** `benchmarks/results/feature-csv-export-harness-v2.yaml` records that
`tasks.md`/`tests.md` were confirmed absent at the `NEEDS_SPEC_APPROVAL` checkpoint and
present only after `eng plan approve-spec`, and the plan reaches `COMPLETED`.
**Fail:** `tasks.md` exists before spec approval in the recorded evidence.

## T5 — Category 3 (Bug Fix)

**Pass:** `benchmarks/results/bug-lastn-off-by-one-harness-v2.yaml` shows
`engineering/debugging` selected with a real reason string, and `verification.verdict:
PASS` after the fix.

## T6 — Category 4 (Large Context)

**Pass:** `benchmarks/results/large-context-auth-validation-harness-v2.yaml` shows
`context.docs: 1` (only the `auth/` section matched) against a recorded fixture total file
count large enough to make the ratio meaningful (at least 5 packages/sections).

## T7 — Category 5 (Cross-Domain)

**Pass:** `benchmarks/results/cross-domain-esp32-siemens-modbus-harness-v2.yaml`'s selected
skill list is a superset of `{esp32, siemens-s7, plc, modbus, tcp-ip}` — the same set Phase
6's own committed `harness/evals/embedded/esp32-siemens-modbus.yaml` asserts.

## T8 — Category 6 (Tool-Routing)

**Pass:** `benchmarks/results/tool-routing-git-status-harness-v2.yaml`'s recorded audit
event has exactly the field set `{type, at, adapter, capability, role, result, reason,
log_path}` — no raw command output inline.

## T9 — Category 7 (Failure/Safety)

**Pass:** `benchmarks/results/failure-safety-harness-v2.yaml` records, for all five
sub-cases: Reviewer REJECT → `NEEDS_REPLAN`; Verifier FAIL → not `COMPLETED`; drift →
`NEEDS_REPLAN`; missing approval → stays at `NEEDS_APPROVAL`; denied tool capability →
`REFUSED (DENIED)` with no adapter invocation evidence (no network/process output, only the
policy-refusal message).

## T10 — Category 8 (Legacy)

**Pass:** `benchmarks/results/legacy-v1-compat.yaml` shows Mode B behavior identical to
every prior phase's own Legacy E2E run, and Mode C requiring zero migration steps.

## T11 — Comparison/scorecard/backlog artifacts exist and are sourced

```bash
test -s benchmarks/COMPARISON.md && echo OK
test -s benchmarks/CONTEXT_EFFICIENCY.md && echo OK
test -s benchmarks/SCORECARD.md && echo OK
test -s benchmarks/BACKLOG.md && echo OK
grep -E "READY_TO_EXPAND|CORE_REFINEMENT_REQUIRED" benchmarks/SCORECARD.md && echo "decision present"
```

**Pass:** all four files exist and are non-empty; the scorecard contains exactly one of the
two literal decision strings.

## T12 — Full regression re-verification

```bash
cd cli && go build ./... && go vet ./... && go test ./... 2>&1
```

**Pass:** identical (zero-diff) outcome to Phase 7's own final verification — Phase 8 adds
no Go code to `cli/`.

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase8-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase8-regression && \
rm -rf .plans/*-smoke-test-phase8-regression && \
echo "V1 REGRESSION CHECK OK"
```

**Pass:** identical output to every prior phase's own run of this check.

```bash
cd "$REPO/cli" && go test ./... -run TestRouterEvalScenarios -v
```

**Pass:** all three Phase 6 router eval scenarios still pass — cross-checks Category 5's
manual benchmark result against the existing automated test; both must agree.

## Cleanup

```bash
rm -rf /tmp/eng-test-p8-* 2>/dev/null
# scratch copies used during Tasks 4-11 live under the session scratchpad, not /tmp on
# Windows — clean up whichever scratch root was actually used, never a committed fixture.
```
