# Tests: V2 Harness Phase 5 (Runtime Integration)

Run each test after completing the corresponding task. Stop and report on first failure.

---

## T1 — Workflow state machine extensions (after Task 1)

```bash
cd cli && go test ./internal/workflow/... -v
```

**Pass:** All tests pass, including the five new ones (`TestQuickFixSkipsStraightToExecuting`,
`TestSpecFirstRequiresSpecApprovalBeforeTasks`, `TestNeedsSpecApprovalGate`,
`TestSpecApprovedWaitsForTasksAndTests`, `TestAutoPlanPathUnaffectedByNewFields`) **and** every
pre-existing Phase 3 test, unmodified.
**Fail:** Any `FAIL`. If a pre-existing Phase 3 test fails, the `TRIAGED` case restructure
broke the `auto_plan` path — treat as the highest-priority failure in this plan.

---

## T2 — `project.Workflow` planning-mode fields (after Task 2)

```bash
cd cli && go test ./internal/project/... -v
```

**Pass:** All tests pass, including `TestLegacyProjectYAMLWithoutPlanningModeStillLoads`.
**Fail:** Any `FAIL`. That specific test failing means a Phase 1-4 `project.yaml` would
silently get spec-first behavior it never asked for — the second-highest-priority failure in
this plan, after T1's auto_plan-path check.

---

## T3 — `plan.yaml` spec-approval fields and structured events (after Task 3)

```bash
cd cli && go test ./internal/planmeta/... -v
```

**Pass:** All tests pass, including `TestAppendStructuredEventWritesFlatJSON`.
**Fail:** Any `FAIL` — paste output.

---

## T4 — `workflow_cmd.go` facts (after Task 4)

```bash
cd cli && go build ./... 2>&1
```

**Pass:** Compiles cleanly (no dedicated unit tests for `cli/*.go` per this repo's own
convention — `gatherFacts`/`specReady`/`tasksAndTestsReady` are exercised by the end-to-end
walkthroughs in T-QUICKFIX and T-SPECFIRST below).
**Fail:** Any compile error.

---

## T5 — `eng plan approve-spec` / `eng plan escalate` / quick-fix template branch (after Task 5)

```bash
cd cli && go build ./... 2>&1 && echo BUILD_OK
```

**Pass:** `BUILD_OK` — full integration coverage is in T-QUICKFIX/T-SPECFIRST below.
**Fail:** Any compile error.

---

## T6 — Minimal quick-fix templates present (after Task 6)

```bash
cd "$(git rev-parse --show-toplevel)" && \
test -f harness/templates/quickfix/spec.md && \
test -f harness/templates/quickfix/tasks.md && \
test -f harness/templates/quickfix/tests.md && \
test -f harness/templates/quickfix/plan.yaml && \
echo "ALL PRESENT"
wc -l < harness/templates/quickfix/spec.md
```

**Pass:** `ALL PRESENT` prints; `spec.md` is short (under 10 lines) — confirming it's genuinely
minimal, not a copy of the full template.
**Fail:** Any missing file, or `spec.md` is as long as the full `harness/templates/plan/spec.md`.

---

## T7 — `eng init` writes `planning_mode: spec_first` for new projects only (after Task 7)

```bash
REPO="$(git rev-parse --show-toplevel)"
cd cli && go build -o eng . && ./eng install --from .. && cd ..
rm -rf /tmp/eng-test-p5-init && mkdir /tmp/eng-test-p5-init && cd /tmp/eng-test-p5-init
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init
grep -A3 "^workflow:" .agent/project.yaml
```

**Pass:** `workflow:` block includes `planning_mode: spec_first`.
**Fail:** Field missing or wrong value.

---

## T8 — `buildContextBundle` and the fallback-to-full retry (after Task 8)

```bash
cd /tmp/eng-test-p5-init
"$REPO/cli/eng" plan new smoke --risk feature
PLAN=$(ls -d .plans/*-smoke)
sed -i 's/\[Feature Name\]/Smoke/' "$PLAN/spec.md"

# No docs/src-map.md or docs/gotchas.md exist yet, and no skill matches a
# nonsense request — this should trigger the bounded fallback.
"$REPO/cli/eng" context bundle planner "$PLAN" "zzznonsensequery12345"
grep "fallback_to_full" "$PLAN/context-manifest.yaml"
```

**Pass:** Output includes "(no matches under 'selective' strategy — fell back to 'full' for
this call)"; `context-manifest.yaml` contains `fallback_to_full: true`.
**Fail:** No fallback attempted, or the command errors instead of degrading gracefully.

```bash
"$REPO/cli/eng" context manifest "$PLAN"
```

**Pass:** Prints the same manifest content (proving `eng context manifest` reads it correctly).
**Fail:** Errors or prints nothing.

---

## T9 — `eng adapter prompt` folds in the context bundle (after Task 9)

```bash
cd /tmp/eng-test-p5-init
PLAN=$(ls -d .plans/*-smoke)
"$REPO/cli/eng" adapter prompt executor "$PLAN" "add a health check"
```

**Pass:** Output contains **both** the full `core/executor/METHOD.md` content (proving Phase 3's
contract still holds — this is the same assertion Phase 3's own T12 made) **and**, after it, a
`# Context bundle for role: executor` section with a `## Skills` block — proving the two are
now composed in one call.
**Fail:** Only the METHOD.md content appears, or the command errors.

---

## T10/T11 — Quick-fix structured event and log pruning wiring (after Task 10)

Covered end-to-end in T-QUICKFIX below (the structured event) and here (pruning is invoked,
not yet exercised at scale):

```bash
cd /tmp/eng-test-p5-init
PLAN=$(ls -d .plans/*-smoke)
sed -i 's/test_cmd:.*/test_cmd: "seq 1 5"/' .agent/project.yaml
"$REPO/cli/eng" verify "$PLAN"
ls .agent/logs/*.log | wc -l
```

**Pass:** Command succeeds; at least one log file exists in `.agent/logs/` (pruning ran but had
nothing to prune yet, since fewer files exist than any default limit).
**Fail:** Command errors, or no log file was written.

---

## T12 — `internal/logprune` unit tests (after Task 11)

```bash
cd cli && go test ./internal/logprune/... -v
```

**Pass:** All four tests pass, including `TestPruneRespectsMaxFilesButKeepsMostRecent` (the
single most important test in this task — confirms the "never delete the most recent file"
safety rule).
**Fail:** Any `FAIL`.

---

## T13 — `eng logs prune` end-to-end (after Task 12)

```bash
cd /tmp/eng-test-p5-init
for i in 1 2 3 4 5; do
  "$REPO/cli/eng" verify "$PLAN" >/dev/null
  sleep 1
done
ls .agent/logs/*.log | wc -l
"$REPO/cli/eng" logs prune --dry-run
"$REPO/cli/eng" logs prune
```

**Pass:** `logs prune --dry-run` reports what *would* be deleted without changing the file
count; `logs prune` (no flag) then actually deletes according to the configured limits (or
reports 0 deletions if under every limit, which is fine at this small scale) and never reports
deleting the most recently created log.
**Fail:** `--dry-run` actually deletes something, or the most recent log is ever listed as
deleted.

---

## T14 — Capability Registry `Describe`/`DescribeAll` (after Task 14)

```bash
cd cli && go test ./internal/capabilities/... -v
```

**Pass:** All tests pass, including the three new ones. `Detect`/`DetectAll`'s existing tests
(Phase 3) also still pass unmodified.
**Fail:** Any `FAIL`.

---

## T15 — `eng capabilities list --verbose --role` (after Task 15)

```bash
"$REPO/cli/eng" capabilities list
echo "---"
"$REPO/cli/eng" capabilities list --verbose
echo "---"
"$REPO/cli/eng" capabilities list --role planner
echo "---"
"$REPO/cli/eng" capabilities list --role executor
```

**Pass:** Plain `list` output is unchanged from Phase 3 (four lines, name+status only);
`--verbose` adds `provider=`/`version=` columns; `--role planner` shows only `git`
(per `RolePermissions`); `--role executor` shows `git`, `claude`, `codex`, `docker` (filtered
to whichever are actually available).
**Fail:** Plain `list` output changed shape (regression), or `--role` filtering doesn't match
`RolePermissions`.

---

## T16 — Role-based permissions (after Task 16)

```bash
cd cli && go test ./internal/agent/... -v
```

**Pass:** All tests pass, including the three new permission tests, alongside every existing
Phase 3 `internal/agent` test.
**Fail:** Any `FAIL`.

---

## T17 — Tool Adapter interface (after Task 17)

```bash
cd cli && go test ./internal/tooladapter/... -v
```

**Pass:** All three tests pass, including `TestGitAdapterImplementsAdapter` (a compile-time
interface-satisfaction check).
**Fail:** Any `FAIL` or compile error.

---

## T18 — Tool Router (after Task 18)

```bash
cd cli && go test ./internal/toolrouter/... -v
```

**Pass:** All three tests pass.
**Fail:** Any `FAIL`.

---

## T19 — Adapter directory reorganization (after Task 19)

```bash
cd "$(git rev-parse --show-toplevel)" && \
test -f harness/adapters/agents/claude-code/ADAPTER.md && \
test ! -e harness/adapters/claude-code && \
test -f harness/adapters/tools/README.md && \
echo "REORG OK"
```

**Pass:** `REORG OK` prints — new path exists, old path is gone, tools placeholder exists.
**Fail:** Old path still exists (incomplete move), or new path missing.

Then re-run T9 (`eng adapter prompt`) once more to confirm the move didn't break anything —
`internal/agent.ClaudeCodeAdapter` never read the moved file, so this should be unaffected.

---

## T20 — `eng start` safe first-run handling (after Task 20)

```bash
cd cli && go build -o eng . && cd ..
rm -rf /tmp/eng-test-p5-start && mkdir /tmp/eng-test-p5-start && cd /tmp/eng-test-p5-start
git init -q
"$REPO/cli/eng" start
```

**Pass:** Prints "This project isn't initialized for the harness yet." and the `--init` hint,
then returns — does **not** run `eng doctor`, does **not** attempt to launch an agent, and does
**not** create `.agent/project.yaml`.
**Fail:** Any file was created, or doctor/launch ran anyway.

```bash
"$REPO/cli/eng" start --init
```

**Pass:** `.agent/project.yaml` now exists (created via the `--init` path); doctor output
follows; the "consult ... core/runtime/METHOD.md" banner prints before any launch attempt.
**Fail:** `--init` didn't create the file, or the banner is missing.

---

## T21 — Triage: gotcha match raises but never lowers the suggested level (after Task 21)

```bash
cd /tmp/eng-test-p5-init
mkdir -p docs
cat > docs/gotchas.md <<'EOF'
### Reconnect timeout silently truncates to 32-bit ms

**Trap:** setting a reconnect timeout above ~24 days silently wraps.
EOF
"$REPO/cli/eng" triage "fix the reconnect timeout bug"
```

**Pass:** Level is raised to `architecture` (the keyword match alone would say `bug`, but the
gotcha match — "reconnect timeout" appears in both the request and the gotcha title — raises
it), with a workflow description mentioning the matched gotcha title.
**Fail:** Level stays `bug` (gotcha match not applied), or the level is raised past
`architecture` to `high-risk` (over-inference not permitted by design).

```bash
"$REPO/cli/eng" triage "deploy to production" # keyword-only high-risk, no gotcha match
```

**Pass:** Level is still `high-risk` — a request that already keyword-matches a *higher* level
than any gotcha would suggest is never lowered.
**Fail:** Level drops below `high-risk`.

---

## T22 — Runtime Router methodology present (after Task 22)

```bash
cd "$(git rev-parse --show-toplevel)" && \
test -f harness/core/runtime/METHOD.md && \
grep -q "eng workflow start" harness/core/runtime/METHOD.md && \
grep -q "Quick Fix path" harness/core/runtime/METHOD.md && \
grep -q "Spec-First path" harness/core/runtime/METHOD.md && \
echo "METHOD DOC OK"
```

**Pass:** `METHOD DOC OK` prints.
**Fail:** Missing file or missing section.

---

## T23 — Full build (after Task 23.3)

```bash
export PATH="$PATH:/c/Program Files/Go/bin"
cd cli && go vet ./... 2>&1 && go build -o eng . 2>&1 && go test ./... 2>&1 && echo ALL_GO_CHECKS_PASS
```

**Pass:** `ALL_GO_CHECKS_PASS`; every package's tests pass; `go vet` reports nothing.
**Fail:** Any compile error, vet warning, or test failure.

---

## T-QUICKFIX — End-to-end Quick Fix walkthrough

```bash
REPO="$(git rev-parse --show-toplevel)"
rm -rf /tmp/eng-test-p5-qf && mkdir /tmp/eng-test-p5-qf && cd /tmp/eng-test-p5-qf
git init -q && git config user.email t@example.com && git config user.name t
echo "timeout = 1000" > connection.cfg
git add connection.cfg && git commit -q -m init
"$REPO/cli/eng" init

"$REPO/cli/eng" plan new bump-timeout --risk quick-fix
PLAN=$(ls -d .plans/*-bump-timeout)
cat "$PLAN/plan.yaml"
```

**Pass (part 1):** `plan.yaml` shows `risk_level: quick-fix`, `state: TRIAGED`,
`requires_approval: false`; the plan directory contains only `spec.md`, `tasks.md`, `tests.md`,
`plan.yaml` (no `review.md`, no `DECISION_LOG.md`, no `sprint-summary.md`, no `errors.log`) —
confirming the minimal template set was used, not the full one.

```bash
"$REPO/cli/eng" workflow advance "$PLAN"
```

**Pass (part 2):** Reports staying in `TRIAGED` — "waiting on the minimal quick-fix plan" —
since the templates still contain the `[Feature Name]` placeholder.

```bash
sed -i 's/\[Feature Name\]/Bump reconnect timeout/' "$PLAN/spec.md" "$PLAN/tasks.md" "$PLAN/tests.md"
"$REPO/cli/eng" workflow advance "$PLAN"
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass (part 3):** Reports `TRIAGED -> EXECUTING (quick-fix: minimal plan present, skipping
review/approval)` — **critically, never** passing through `PLANNED`, `REVIEWED`, `APPROVED`,
or any spec-approval state. `workflow status` confirms `State: EXECUTING`.

```bash
sed -i 's/timeout = 1000/timeout = 1500/' connection.cfg
sed -i 's/^- \[ \]/- [x]/' "$PLAN/tasks.md"
sed -i 's/test_cmd:.*/test_cmd: "echo ok"/' .agent/project.yaml
"$REPO/cli/eng" workflow advance "$PLAN"
cat "$PLAN/events.jsonl"
```

**Pass (part 4):** Output shows `EXECUTING -> VERIFYING` then `VERIFYING -> COMPLETED`.
`events.jsonl` contains a `quick_fix`-typed line with `"verification":"PASS"` and a `files`
array listing `connection.cfg`. **No** `spec.md` beyond the one-line goal was ever required.
**Fail:** The plan ever entered `PLANNED`/`REVIEWED`/`APPROVED`, or no `quick_fix` event was
recorded.

### Escalation sub-test

```bash
"$REPO/cli/eng" plan new too-big-for-quickfix --risk quick-fix
PLAN2=$(ls -d .plans/*-too-big-for-quickfix)
"$REPO/cli/eng" plan escalate "$PLAN2" --to feature --reason "touches the public API"
cat "$PLAN2/plan.yaml" | grep -E "risk_level|^state"
```

**Pass:** `risk_level: feature`, `state: TRIAGED` — the plan is back at the top of the normal
lifecycle, not silently continuing as a quick fix.
**Fail:** `risk_level` unchanged, or state left at something other than `TRIAGED`.

---

## T-SPECFIRST — End-to-end Spec-First walkthrough

```bash
cd /tmp/eng-test-p5-qf
"$REPO/cli/eng" plan new add-export --risk feature
PLAN=$(ls -d .plans/*-add-export)
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass (part 1):** `State: TRIAGED`; profile shows `feature`; since this project's
`.agent/project.yaml` was written by this Phase 5 `eng init`, `planning_mode` is `spec_first`.

```bash
"$REPO/cli/eng" workflow advance "$PLAN"   # still TRIAGED — spec.md still has the placeholder
sed -i 's/\[Feature Name\]/Add CSV Export/' "$PLAN/spec.md"
"$REPO/cli/eng" workflow advance "$PLAN"
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass (part 2):** `TRIAGED -> NEEDS_SPEC_APPROVAL (spec.md written — waiting on \`eng plan
approve-spec\`)` — **critically, tasks.md and tests.md are not required yet** (they may still
contain the full-template placeholder at this point, and that must not block this transition).
`workflow status` confirms `State: NEEDS_SPEC_APPROVAL`.

```bash
"$REPO/cli/eng" workflow advance "$PLAN"   # must stay blocked
"$REPO/cli/eng" plan approve-spec "$PLAN" --by reviewer
"$REPO/cli/eng" workflow advance "$PLAN"
```

**Pass (part 3):** The first `workflow advance` reports staying in `NEEDS_SPEC_APPROVAL`.
After `eng plan approve-spec`, the next `workflow advance` reports
`NEEDS_SPEC_APPROVAL -> SPEC_APPROVED (spec approved)`.

```bash
"$REPO/cli/eng" workflow advance "$PLAN"   # still SPEC_APPROVED — tasks/tests not written
sed -i 's/\[Feature Name\]/Add CSV Export/' "$PLAN/tasks.md" "$PLAN/tests.md"
"$REPO/cli/eng" workflow advance "$PLAN"
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass (part 4):** First call stays in `SPEC_APPROVED` ("waiting on Planner to write
tasks.md/tests.md"). After writing them, `SPEC_APPROVED -> PLANNED (tasks.md/tests.md are
present)`. From here the lifecycle continues exactly as Phase 3 documented (review, approval if
required, execution, verification) — confirm by checking `plan.yaml`:

```bash
grep -E "spec_approved_at|spec_approved_by|^requires_approval|^\s*approved_at" "$PLAN/plan.yaml"
```

**Pass:** `spec_approved_at`/`spec_approved_by` are populated; the plain `approved_at` field
(execution approval) is **absent** — confirming the two approval concepts stayed separate;
this plan never needed execution approval since `--risk feature` doesn't require it.
**Fail:** `approved_at` got populated by the spec-approval step (the two concepts merged —
the single most important thing this feature must not do), or the plan reached `PLANNED`
before `tasks.md`/`tests.md` were actually written.

---

## T-LEGACY — A hand-written Phase-1-4-shaped project never gets spec_first behavior

This is the compatibility test for Decision 6 — the highest-risk decision in this plan.

```bash
rm -rf /tmp/eng-test-p5-legacy && mkdir /tmp/eng-test-p5-legacy && cd /tmp/eng-test-p5-legacy
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
mkdir -p .agent
cat > .agent/project.yaml <<'EOF'
project_name: eng-test-p5-legacy
mode: modern
harness_profile: software
stack:
    type: go
    build_cmd: go build ./...
    test_cmd: echo ok
enabled_skills:
    - engineering/karpathy-guidelines
EOF
# Note: no `workflow:` block at all — simulates a Phase 1-3 project.yaml.

"$REPO/cli/eng" plan new legacy-check --risk feature
PLAN=$(ls -d .plans/*-legacy-check)
sed -i 's/\[Feature Name\]/Legacy Check/' "$PLAN/spec.md" "$PLAN/tasks.md" "$PLAN/tests.md"
"$REPO/cli/eng" workflow advance "$PLAN"
```

**Pass:** Reports `TRIAGED -> PLANNED (spec.md/tasks.md/tests.md are present)` — the **single-
step** transition, exactly as Phase 3/4 always behaved — **not** `NEEDS_SPEC_APPROVAL`. This
proves a project.yaml with no `workflow.planning_mode` field takes the `auto_plan` path
regardless of what a freshly-`eng init`-ed project would now default to.
**Fail:** The plan enters `NEEDS_SPEC_APPROVAL` — this would mean an existing, real Phase 1-4
project's behavior silently changed under this harness upgrade, the single most important
compatibility guarantee in this entire plan.

---

## T24 — Full Phase 1-4 and V1 regression gates

```bash
REPO="$(git rev-parse --show-toplevel)"
rm -rf /tmp/eng-test-p5-regress && mkdir /tmp/eng-test-p5-regress && cd /tmp/eng-test-p5-regress
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init

"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
"$REPO/cli/eng" capabilities list
"$REPO/cli/eng" context skills "add a feature"
"$REPO/cli/eng" context project "add a feature"
"$REPO/cli/eng" plan new regress --risk feature
PLAN=$(ls -d .plans/*-regress)
"$REPO/cli/eng" plan drift "$PLAN"
"$REPO/cli/eng" plan retry "$PLAN" unit_test
"$REPO/cli/eng" triage "fix a bug"
"$REPO/cli/eng" hooks run before_execute
```

**Pass:** Every command behaves exactly as documented in its own phase's plan — Phase 5 only
ever *adds* subcommands/flags, never changes an existing one's meaning (the one deliberate,
called-out exception is `eng adapter prompt`'s output, which now includes more content than
before, per Decision 2 — its existing content is still present, just not alone anymore).
**Fail:** Any command errors, or an existing flag/argument shape stops working.

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase5-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase5-regression && \
rm -rf .plans/*-smoke-test-phase5-regression && \
echo "V1 REGRESSION CHECK OK"
```

**Pass:** Identical output to every prior run across all four phases.
**Fail:** Any difference.

---

## Cleanup

```bash
rm -rf /tmp/eng-test-p5-init /tmp/eng-test-p5-start /tmp/eng-test-p5-qf /tmp/eng-test-p5-legacy /tmp/eng-test-p5-regress
```
