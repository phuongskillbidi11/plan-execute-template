# Tests: V2 Harness Phase 3

Run each test after completing the corresponding task. Stop and report on first failure.

**Caution before you start:** T15 covers `eng install --add-to-path`, which mutates a real
user-scope PATH variable (Windows `setx`) or shell profile file (macOS/Linux `~/.bashrc`/
`~/.zshrc`). Do not run `--add-to-path` against your own development machine as part of routine
verification — test it against an isolated `HOME`/environment, or verify the logic by reading
the code path rather than executing the mutating branch on your real machine. The print-only
default (no flag) is what should be exercised end-to-end.

---

## T1 — `executil` (after Task 1)

```bash
cd cli && go test ./internal/executil/... -v
```

**Pass:** All five tests pass (`TestUnmarshalScalarIsShell`, `TestUnmarshalStructuredForm`,
`TestRunShellMode`, `TestRunStructuredMode`, `TestEmpty`).
**Fail:** Any `FAIL` — paste full output.

---

## T2 — `planmeta` lifecycle fields and events (after Task 2)

```bash
cd cli && go test ./internal/planmeta/... -v
```

**Pass:** All five tests pass, including `TestLegacyStatusMigratesToState` (a Phase-2-style
`status: executing` with no `state` field must load as `state: EXECUTING`) and
`TestAppendEventWritesJSONLine`.
**Fail:** Any `FAIL`. If `TestLegacyStatusMigratesToState` fails, a real Phase-2 plan.yaml
would silently lose its progress under Phase 3 — treat as high priority.

---

## T3 — `project.Stack` structured/shell compatibility (after Task 3)

```bash
cd cli && go test ./internal/project/... -v
```

**Pass:** All tests pass, including the Phase 1 `TestSaveLoadRoundTrip` (now asserting on
`Stack.Test.Shell`) and the new `TestPlainStringStackCommandStillParses`.
**Fail:** Any `FAIL`. `TestPlainStringStackCommandStillParses` failing means an existing
Phase 1/2 `.agent/project.yaml` would break under Phase 3 — treat as the highest-priority
failure in this entire plan.

---

## T4 — Lifecycle transition table (after Task 4)

```bash
cd cli && go test ./internal/workflow/... -run 'Test[^P]' -v
```

**Pass:** All seven `workflow.go` tests pass (transition rules + `TestTerminalStates`).
**Fail:** Any `FAIL` — paste output and cross-check against `spec.md`'s Decision 6 table;
the test failing means the code doesn't match the documented table.

---

## T5 — Workflow profiles (after Task 5)

```bash
cd cli && go test ./internal/workflow/... -v
cd .. && for f in quick-fix bug-fix feature architecture high-risk; do
  test -f "harness/workflows/$f.yaml" && echo "$f OK"
done
```

**Pass:** All workflow package tests pass; all five `<name> OK` lines print.
**Fail:** Any missing profile file or test failure.

---

## T6 — Capability registry (after Task 6)

```bash
cd cli && go test ./internal/capabilities/... -v
```

**Pass:** `TestDetectMissingBinary` and `TestDetectAllCoversKnownSet` pass;
`TestDetectGitIsUsuallyPresent` either passes or is skipped (never fails outright).
**Fail:** Any real `FAIL`.

---

## T7 — Agent adapter (after Task 7)

```bash
cd cli && go test ./internal/agent/... -v
test -f ../harness/adapters/claude-code/ADAPTER.md && echo "ADAPTER DOC OK"
```

**Pass:** All three agent tests pass; `ADAPTER DOC OK` prints.
**Fail:** Any `FAIL` or missing file.

---

## T8 — `plan_cmd.go` compiles with the new subcommands (after Task 8)

```bash
cd cli && go build ./... 2>&1
```

**Pass:** Fails only on genuinely unrelated later tasks (e.g. missing `cmdWorkflow` before
Task 11) — at this point in the sequence, expect errors naming symbols from later tasks
(`workflow.StateTriaged` undefined is fine if Task 4 isn't done yet; check the error
specifically references `cli/internal/workflow` not yet existing, not a syntax error in
`plan_cmd.go` itself). If Tasks 1–7 are already done, this should build cleanly.
**Fail:** A syntax error, or an error unrelated to a not-yet-created symbol from a later task.

---

## T9 — `eng verify` persists a machine-readable verdict (after Task 9)

```bash
REPO="$(git rev-parse --show-toplevel)"
cd cli && go build -o eng . && cd ..
rm -rf /tmp/eng-test-p3-verify && mkdir /tmp/eng-test-p3-verify && cd /tmp/eng-test-p3-verify
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" plan new smoke --risk feature
PLAN=$(ls -d .plans/*-smoke)
"$REPO/cli/eng" verify "$PLAN"
grep -A2 "^verification:" "$PLAN/plan.yaml"
grep -c '"type":"verified"' "$PLAN/events.jsonl"
```

**Pass:** `plan.yaml` contains a `verification:` block with a `verdict:` of `PASS` or `FAIL`
and a non-empty `verified_at:`; `events.jsonl` contains at least one `"type":"verified"` line.
**Fail:** No `verification:` block written, or no matching event line.

---

## T10 — `eng hooks run` with a structured command (after Task 10)

```bash
cd /tmp/eng-test-p3-verify
mkdir -p .agent
cat > .agent/hooks.yaml <<'EOF'
before_execute: [native_check]
commands:
  native_check:
    command: go
    args: ["version"]
EOF
PATH="$PATH:$REPO/cli" "$REPO/cli/eng" hooks run before_execute
```

**Pass:** Output shows `[before_execute] native_check      -> go version` followed by real
`go version ...` output — proving the structured `{command, args}` form executes without a
shell. (This test deliberately does not also list `drift_check` here: that hook runs
`eng plan drift .` against the *current working directory*, which is the project root, not a
plan directory — it would fail for reasons unrelated to structured-command execution. Testing
`eng plan drift` itself belongs to T11/T11c below, run from inside an actual plan directory.)
**Fail:** `native_check` errors, or its command line is printed as if it went through `sh -c`
(e.g. a shell-escaping artifact) instead of running `go` directly.

---

## T11 — End-to-end orchestrator walkthrough (after Task 11)

This is the plan's centerpiece test — it exercises the full `TRIAGED → PLANNED → REVIEWED →
APPROVED → EXECUTING → VERIFYING → COMPLETED` path plus the approval gate and drift gate.

```bash
REPO="$(git rev-parse --show-toplevel)"
cd cli && go build -o eng . && cd ..
rm -rf /tmp/eng-test-p3-e2e && mkdir /tmp/eng-test-p3-e2e && cd /tmp/eng-test-p3-e2e
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
ENG="$REPO/cli/eng"

"$ENG" init
"$ENG" workflow start "add a health check endpoint"
PLAN=$(ls -d .plans/*-add-a-health-check-endpoint)
echo "--- status right after creation ---"
"$ENG" workflow status "$PLAN"
```

**Pass (part 1):** `State: TRIAGED`; `Next:` line says "waiting on Planner to write
spec.md/tasks.md/tests.md" — **not** "spec.md/tasks.md/tests.md are present" — proving the
`filesReady` placeholder check (Task 11.2's fix) works: the scaffolded template content alone
does not count as a real plan.
**Fail:** `Next:` claims the plan files are ready before anyone wrote them — the placeholder
check regressed; this was the exact bug caught during planning review.

```bash
# Simulate the Planner actually writing the plan:
sed -i 's/\[Feature Name\]/Health Check Endpoint/' "$PLAN/spec.md"
"$ENG" workflow advance "$PLAN"
```

**Pass (part 2):** `TRIAGED -> PLANNED (spec.md/tasks.md/tests.md are present)`;
`Next action: run \`eng adapter prompt plan-reviewer ...\``.

```bash
"$ENG" plan review "$PLAN" --verdict PASS
"$ENG" workflow advance "$PLAN"
```

**Pass (part 3):** `PLANNED -> REVIEWED (review verdict PASS)`, then since `--risk feature`
does not require approval by default: `REVIEWED -> APPROVED (approval not required or already
granted)`, then (no drift yet) `APPROVED -> EXECUTING (no drift detected...)` — **note:**
`eng workflow advance` applies exactly one transition per call per Decision 6, so run it three
times in a row and confirm each single-step transition individually:

```bash
"$ENG" workflow status "$PLAN"
```

**Pass:** `State: EXECUTING`.

```bash
# Simulate the Executor finishing all tasks:
sed -i 's/^- \[ \]/- [x]/' "$PLAN/tasks.md"
"$ENG" workflow advance "$PLAN"
"$ENG" workflow status "$PLAN"
```

**Pass (part 4):** Output shows `EXECUTING -> VERIFYING (all tasks.md items checked off)`,
`Running eng verify automatically...`, a verify report, and then either
`VERIFYING -> COMPLETED (eng verify reported PASS)` or `VERIFYING -> NEEDS_FIX (...)` depending
on whether `go test ./...` actually passes in this throwaway repo (it will likely FAIL since
there's no real Go module content — either outcome is acceptable; this test checks the
mechanism, not the verdict). Final `workflow status` reflects whichever state resulted.
**Fail:** State does not change at all after tasks.md is fully checked, or `plan.yaml`'s
`verification` block is empty after this step.

```bash
cat "$PLAN/events.jsonl"
```

**Pass:** Contains, in order, `triaged`, one or more `state_changed` lines, `reviewed`,
`approved` (only if this plan required it — `--risk feature` does not, so no `approved` line
is expected here), and `verified` — a complete, append-only history of everything that
happened, per Decision 1/goal #10.

---

## T11b — Approval gate actually blocks execution (after Task 11)

```bash
cd /tmp/eng-test-p3-e2e
"$REPO/cli/eng" plan new smoke-highrisk --risk high-risk
PLAN=$(ls -d .plans/*-smoke-highrisk)
sed -i 's/\[Feature Name\]/Smoke High Risk/' "$PLAN/spec.md"
"$REPO/cli/eng" workflow advance "$PLAN"          # TRIAGED -> PLANNED
"$REPO/cli/eng" plan review "$PLAN" --verdict PASS
"$REPO/cli/eng" workflow advance "$PLAN"          # PLANNED -> REVIEWED
"$REPO/cli/eng" workflow advance "$PLAN"          # REVIEWED -> ??? must NOT be APPROVED
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass:** The third `workflow advance` reports
`REVIEWED -> NEEDS_APPROVAL (run \`eng plan approve\` before execution can begin)`, and
`workflow status` confirms `State: NEEDS_APPROVAL` and
`Requires approval: true (NOT yet approved)`. Running `workflow advance` again without
approving must keep reporting `NEEDS_APPROVAL`, never silently reach `EXECUTING`.
**Fail:** The plan reaches `APPROVED`/`EXECUTING` without `eng plan approve` ever being run —
this is the single most important test in this plan; it verifies Requirement 6's actual
enforcement, not just its documentation.

```bash
"$REPO/cli/eng" plan approve "$PLAN" --by tester
"$REPO/cli/eng" workflow advance "$PLAN"
```

**Pass:** `NEEDS_APPROVAL -> APPROVED (approval granted)`.

---

## T11c — Drift before execution forces `NEEDS_REPLAN` (after Task 11)

```bash
cd /tmp/eng-test-p3-e2e
"$REPO/cli/eng" plan new smoke-drift --risk feature
PLAN=$(ls -d .plans/*-smoke-drift)
sed -i 's/\[Feature Name\]/Smoke Drift/' "$PLAN/spec.md"
"$REPO/cli/eng" workflow advance "$PLAN"   # -> PLANNED
"$REPO/cli/eng" plan review "$PLAN" --verdict PASS
"$REPO/cli/eng" workflow advance "$PLAN"   # -> REVIEWED
"$REPO/cli/eng" workflow advance "$PLAN"   # -> APPROVED (no approval required for "feature")

echo "unrelated change" >> go.mod   # drift: go.mod changed since this plan's git_sha
"$REPO/cli/eng" workflow advance "$PLAN"
```

**Pass:** Reports `APPROVED -> NEEDS_REPLAN (PLAN_DRIFT_DETECTED before execution started)` —
the plan never silently proceeds to `EXECUTING` on top of drifted source.
**Fail:** State reaches `EXECUTING` despite the drift.

---

## T12 — `eng adapter prompt` (after Task 12)

```bash
"$REPO/cli/eng" adapter prompt executor /tmp/eng-test-p3-e2e/.plans/*-smoke-drift
```

**Pass:** Prints the full contents of `harness/core/executor/METHOD.md` followed by a block
naming the `executor` role and the absolute plan directory path. If `claude` is not on this
machine's PATH, a `note:` line says so first but the prompt still prints.
**Fail:** Missing method content, wrong role name, or the command errors out instead of
degrading gracefully when `claude` is absent.

---

## T13 — `eng capabilities list` (after Task 13)

```bash
"$REPO/cli/eng" capabilities list
```

**Pass:** Prints exactly four lines (`git`, `claude`, `codex`, `docker`), each `available` or
`unavailable`; `git` should read `available` on any machine capable of running this repo's own
tests.
**Fail:** Missing a known capability, or a crash.

---

## T14 — `eng doctor` Capabilities section (after Task 14)

```bash
cd /tmp/eng-test-p3-e2e && "$REPO/cli/eng" doctor
```

**Pass:** Output ends with a `Capabilities:` section listing the same four tools as T13.
**Fail:** Section missing, or `eng doctor`'s prior output (harness install/mode/skills) is
disturbed by this change — confirm against Phase 1/2's documented `doctor` output shape.

---

## T15 — `eng install` — print-only path is the one to test automatically

```bash
cd "$REPO/cli" && go build -o eng . && ./eng install --from ..
ls "$HOME/.engineering-harness/bin/"
```

**Pass:** Output includes "Copied eng binary to ..." and a "To use \`eng\` from any terminal,
add this to your PATH:" block with the platform-correct line (`setx ...` on Windows,
`export PATH=...` elsewhere); `$HOME/.engineering-harness/bin/` contains `eng` (or `eng.exe`).
**Fail:** Binary not copied, or PATH instructions missing/wrong for this platform.

**Do not run this next command against your real machine as part of routine verification** —
it mutates your real user PATH. If you choose to verify `--add-to-path` deliberately, do so
once, understanding the consequence, and confirm you can find/undo the change afterward
(Windows: `setx PATH "<value without the harness bin dir>"`; Unix: remove the appended lines
from `~/.bashrc`/`~/.zshrc`).

```bash
# ./eng install --from .. --add-to-path   # deliberate, manual verification only
```

---

## T16 — `eng start` — test the fallback path only if `claude` is unavailable here

```bash
"$REPO/cli/eng" capabilities list | grep claude
```

**If `claude unavailable`:**
```bash
cd /tmp/eng-test-p3-e2e && "$REPO/cli/eng" start
```
**Pass:** Runs `eng doctor`'s full output, then prints the "`claude` was not found on PATH..."
fallback message and returns — does not hang.

**If `claude available` on this machine:** do not run `eng start` as part of an automated
sweep — it will exec into an interactive session and block. Verify manually in an interactive
terminal instead: confirm doctor output prints, then Claude Code launches attached to the
current terminal.

---

## T17 — Full build (after Task 17.3)

```bash
cd cli && go vet ./... 2>&1 && go build -o eng . 2>&1 && go test ./... 2>&1 && echo ALL_GO_CHECKS_PASS
```

**Pass:** `ALL_GO_CHECKS_PASS` prints; every package's tests pass; `go vet` reports nothing
(in particular, no unused-import warning from `adapter_cmd.go`'s placeholder line — confirm it
was removed per Task 12.1's note if `path/filepath` ended up unused).
**Fail:** Any compile error, vet warning, or test failure — paste the full output.

---

## T18 — Phase 1 and Phase 2 regression gate (after Task 18)

```bash
cd "$REPO/cli" && go build -o eng . && cd /tmp && rm -rf eng-test-p3-regress && mkdir eng-test-p3-regress && cd eng-test-p3-regress
touch go.mod
"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
"$REPO/cli/eng" plan new regress-check --risk feature
PLAN=$(ls -d .plans/*-regress-check)
"$REPO/cli/eng" plan drift "$PLAN"
"$REPO/cli/eng" plan retry "$PLAN" unit_test
"$REPO/cli/eng" verify "$PLAN"
"$REPO/cli/eng" hooks run before_execute
"$REPO/cli/eng" triage "fix a bug"
```

**Pass:** Every command behaves exactly as documented in the Phase 1/Phase 2 plans — `init`
reports `mode: modern, stack: go`; `doctor` reports `modern` (not `broken`) plus the new
Capabilities section; `plan new`/`drift`/`retry`/`verify`/`hooks run`/`triage` all still work
using their Phase 1/2 argument shapes (Phase 3 only added new optional flags and subcommands,
never changed an existing one's meaning).
**Fail:** Any command errors, or behaves differently than its Phase 1/2 documentation —
Phase 3 must be 100% additive to the CLI surface Phase 1/2 already shipped.

---

## T19 — V1 regression gate (unchanged since Phase 1)

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase3-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase3-regression && \
rm -rf .plans/*-smoke-test-phase3-regression
```

**Pass:** Identical output to every prior run of this test across Phase 1 and Phase 2 — same
three skills listed, same scaffold behavior. Phase 3 added zero lines to any V1 script.
**Fail:** Any difference.

---

## Cleanup

```bash
rm -rf /tmp/eng-test-p3-verify /tmp/eng-test-p3-e2e /tmp/eng-test-p3-regress
```
