# Tests: V2 Harness Phase 2

Run each test after completing the corresponding task. Stop and report on first failure.
T-SH is a prerequisite gate — run it before Task 7 (the first task that shells out).

---

## T-SH — POSIX shell available for `sh -c` (before Task 7)

```bash
sh -c "echo ok"
```

**Pass:** Prints `ok`.
**Fail:** `sh: command not found` — install Git for Windows (provides `sh.exe`) or otherwise
put a POSIX shell on PATH before proceeding to Task 7/8's `eng verify`/`eng hooks run`. This is
the same dependency V1's own scripts already carried (README's Bash 4+ requirement) — not a
new one introduced by this plan.

---

## T1 — Parse-error surfacing and Config extension (after Task 1)

```bash
cd cli && go test ./internal/project/... -run 'TestDetectModeResultBroken|TestEffectiveWorkflowDefaultsAllTrue|TestEffectiveRetryBudgetDefault|TestConfigVersionDefaultsToOneOnLoad' -v
```

**Pass:** All four tests print `PASS`.
**Fail:** Any `FAIL` — paste the full output. If `TestDetectModeResultBroken` fails, the core
fix this task exists for didn't land; treat as high priority.

Then confirm the full existing suite still passes (no regression from the rewrite):

```bash
go test ./internal/project/... 2>&1
```

**Pass:** `ok eng/internal/project` — all tests (Phase 1's three plus Task 1's four) pass.

---

## T2 — `planmeta` (after Task 2)

```bash
cd cli && go test ./internal/planmeta/... -v
```

**Pass:** `TestSaveLoadRoundTrip` and `TestDefaultBudget` both pass.
**Fail:** Any `FAIL` — paste output.

---

## T3 — `gitutil` (after Task 3)

```bash
cd cli && go test ./internal/gitutil/... -v
```

**Pass:** `TestHeadSHA` and `TestChangedFilesSince` both pass.
**Fail:** If both fail with "git: command not found," `git` itself isn't on PATH in the test
environment — install/PATH it before continuing (this repo is already a git repository, so
`git` must exist for any of Phase 2 to function). If only `TestChangedFilesSince` fails, re-read
its comment about untracked vs. modified files before assuming it's wrong.

---

## T4 — `hooks` (after Task 4)

```bash
cd cli && go test ./internal/hooks/... -v
```

**Pass:** `TestLoadGlobalDefault` and `TestProjectOverrideReplacesGlobal` both pass.
**Fail:** Any `FAIL` — paste output. If the override test fails by showing *both* hooks present,
the "full replace, no merge" design (Decision 5) was implemented as a merge instead — fix to
match the decision.

---

## T5 — Templates and hooks default present (after Task 5)

```bash
cd "$(git rev-parse --show-toplevel)" && \
test -f harness/templates/plan/plan.yaml && \
test -f harness/templates/plan/review.md && \
test -f harness/templates/plan/verify-report.md && \
test -f harness/hooks/default.yaml && \
echo "ALL PRESENT"
grep -q "^before_plan:" harness/hooks/default.yaml && echo "HOOKS SCHEMA OK"
```

**Pass:** Both `ALL PRESENT` and `HOOKS SCHEMA OK` print.
**Fail:** Any missing file or the grep fails — report which.

---

## T6 — Full build succeeds (after Task 10.3)

```bash
cd cli && go mod tidy && go build -o eng . 2>&1 && echo BUILD_OK
```

**Pass:** `BUILD_OK` prints, no compiler errors. In particular, there must be **no**
"copyTree redeclared" error — confirm Task 6.2 removed the duplicate before this test.
**Fail:** Any compile error — paste it in full, including file and line.

---

## T7 — `eng plan new` scaffolds and stamps `plan.yaml`

```bash
REPO="$(git rev-parse --show-toplevel)"
cd /tmp && rm -rf eng-test-p2 && mkdir eng-test-p2 && cd eng-test-p2
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" plan new smoke-feature --risk feature
cat .plans/*-smoke-feature/plan.yaml
```

**Pass:** Output includes `Scaffolded ... — risk: feature, git_sha: <40-char sha>`;
`plan.yaml` contains `risk_level: feature`, a non-empty `planned_at.git_sha` matching
`git rev-parse HEAD` in that repo, and `status: planned`. The plan folder also contains
`review.md` and `verify-report.md` copied from the templates.
**Fail:** Missing fields, empty git_sha, or missing template files — report which.

---

## T8 — `eng plan drift` reports OK, then detects drift

```bash
cd /tmp/eng-test-p2
"$REPO/cli/eng" plan drift .plans/*-smoke-feature
echo "exit: $?"

echo "package x" > go.mod   # unrelated-looking edit, but go.mod has no write_scope declared yet
"$REPO/cli/eng" plan drift .plans/*-smoke-feature
echo "exit: $?"
```

(`$REPO` from T7's shell — re-export if running in a fresh shell.)

**Pass:** First call prints `OK — no changes since plan was created` and exits `0`. Second call
(after editing `go.mod`, with an empty `write_scope` in `plan.yaml`) prints
`PLAN_DRIFT_DETECTED` listing `go.mod`, and exits `1` — an empty `write_scope` means *any*
change counts as relevant.
**Fail:** Wrong verdict or wrong exit code either time.

---

## T9 — `eng plan retry` tracks budget and reports exhaustion

```bash
cd /tmp/eng-test-p2
PLAN=$(ls -d .plans/*-smoke-feature)
"$REPO/cli/eng" plan retry "$PLAN" unit_test; echo "exit: $?"
"$REPO/cli/eng" plan retry "$PLAN" unit_test; echo "exit: $?"
"$REPO/cli/eng" plan retry "$PLAN" unit_test; echo "exit: $?"
grep unit_test "$PLAN/plan.yaml"
```

**Pass:** First two calls print `RETRY 1/2 for unit_test — proceed` / `RETRY 2/2 ... — proceed`
and exit `0`; the third prints `RETRY BUDGET EXHAUSTED for unit_test (3/2) — escalate to
Planner or human` and exits `1`. `plan.yaml`'s `retry.unit_test` reads `3` afterward.
**Fail:** Budget not enforced at the right count, or the counter isn't persisted between calls.

---

## T10 — `eng verify` writes a report and never touches source

```bash
cd /tmp/eng-test-p2
PLAN=$(ls -d .plans/*-smoke-feature)
md5sum go.mod > /tmp/before-verify.md5
"$REPO/cli/eng" verify "$PLAN"
echo "exit: $?"
md5sum go.mod > /tmp/after-verify.md5
diff /tmp/before-verify.md5 /tmp/after-verify.md5 && echo "SOURCE UNCHANGED"
cat "$PLAN/verify-report.md"
```

**Pass:** `verify-report.md` exists and contains a `## Verdict:` line (`PASS` or `FAIL`
depending on whether `go.mod`'s content is still valid Go and the project's test command
succeeds — either verdict is acceptable here, this test checks the *mechanism*, not a specific
verdict); `SOURCE UNCHANGED` prints (the Verifier must never modify `go.mod` or any source
file); exit code matches the printed verdict (`0` for PASS, `1` for FAIL).
**Fail:** No report written, or `go.mod` was modified by `eng verify` itself, or exit code
doesn't match the printed verdict.

---

## T11 — `eng hooks run` executes shell hooks and flags manual steps

```bash
cd /tmp/eng-test-p2
"$REPO/cli/eng" hooks run before_execute
echo "---"
"$REPO/cli/eng" hooks run after_plan
```

**Pass:** `before_execute` runs `eng plan drift .` (a real shell command — prints its `->`
line followed by drift output). `after_plan` prints a line for `plan_review` ending in
"manual step — no shell command; perform via the documented role" and does **not** attempt to
execute anything for it.
**Fail:** A manual-step hook (empty command in `default.yaml`) is executed as a shell command
(would likely error), or a real hook is skipped.

---

## T12 — `eng triage` heuristic classification

```bash
"$REPO/cli/eng" triage "fix the login bug"
echo "---"
"$REPO/cli/eng" triage "deploy the new release to production"
echo "---"
"$REPO/cli/eng" triage "add a recommendations endpoint"
```

**Pass:** First call suggests level `bug`; second suggests `high-risk`; third (no keyword
match) falls through to `feature`. All three end with the
"(heuristic hint only — the Planner makes the final call)" line.
**Fail:** Wrong level for any of the three, or the non-authoritative disclaimer is missing.

---

## T13 — Phase 1 regression gate: `eng init`/`doctor`/`scan`/`skills list` unchanged

```bash
cd /tmp && rm -rf eng-test-p1regress && mkdir eng-test-p1regress && cd eng-test-p1regress
touch go.mod
"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
```

**Pass:** Behavior matches the V2 Foundation plan's T8/T10/T11/T12 exactly — `init` reports
`mode: modern, stack: go`; `doctor` reports `Project mode:      modern` (not `broken`, not
mis-parsed) and lists `karpathy-guidelines`; `scan` reports `Stack: go`; `skills list` shows
`karpathy-guidelines [global]`. Nothing about Task 1's `Config` struct changes should alter any
of this output.
**Fail:** Any output differs from the Foundation plan's documented behavior — this means the
`Config`/`DetectMode` changes in Task 1 broke backward compatibility with Phase 1's own
contract; treat as high priority.

---

## T14 — V1 regression gate (unchanged from the Foundation plan's T13)

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase2-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase2-regression && \
rm -rf .plans/*-smoke-test-phase2-regression
```

**Pass:** Identical output to every prior run of this test — same three skills listed, same
scaffold behavior. Phase 2 added zero lines to any V1 script.
**Fail:** Any difference — Phase 2's changes are entirely additive; V1 script behavior must be
provably untouched.

---

## T15 — A Phase-1-only `project.yaml` (no Phase 2 fields) still loads correctly

```bash
cd /tmp && rm -rf eng-test-p1only && mkdir eng-test-p1only && cd eng-test-p1only
mkdir -p .agent
cat > .agent/project.yaml <<'EOF'
project_name: eng-test-p1only
mode: modern
harness_profile: software
stack:
    type: go
    build_cmd: go build ./...
    test_cmd: go test ./...
enabled_skills:
    - engineering/karpathy-guidelines
EOF
"$REPO/cli/eng" doctor
```

**Pass:** `doctor` runs without error and reports `Project mode:      modern` — a file written
before `config_version`/`workflow`/`retry_budget`/`require_approval` existed loads cleanly, with
those fields defaulting sensibly (verified indirectly here; directly by T1's
`TestConfigVersionDefaultsToOneOnLoad`).
**Fail:** Any parse error or crash — this would mean Task 1's `Config` extension broke loading
of a real Phase 1 file, the single most important compatibility guarantee of this plan.
