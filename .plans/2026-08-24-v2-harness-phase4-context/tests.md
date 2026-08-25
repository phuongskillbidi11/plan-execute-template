# Tests: V2 Harness Phase 4 (Context Engineering)

Run each test after completing the corresponding task. Stop and report on first failure.

---

## T1 — `contextcfg` defaults, override, and the unset-vs-false fix (after Task 1)

```bash
cd cli && go test ./internal/contextcfg/... -v
```

**Pass:** All four tests pass, including `TestUnsetBoolDoesNotResetToFalse` and
`TestExplicitFalseIsRespected`.
**Fail:** Any `FAIL`. If `TestUnsetBoolDoesNotResetToFalse` fails, the exact bug caught during
planning (Go's bool zero-value ambiguity) has regressed — treat as high priority.

---

## T2 — `skillmatch` scoring and the required-skill guarantee (after Task 2)

```bash
cd cli && go test ./internal/skillmatch/... -v
```

**Pass:** All five tests pass, including `TestSelectAlwaysIncludesRequiredEvenBeyondCap`.
**Fail:** Any `FAIL`. If the required-skill test fails, a project's `enabled_skills` could be
silently dropped by the new filtering layer — this is the single most important compatibility
guarantee in this plan; treat as high priority.

---

## T3 — `docsearch` section parsing and matching (after Task 3)

```bash
cd cli && go test ./internal/docsearch/... -v
```

**Pass:** All three tests pass.
**Fail:** Any `FAIL` — paste output.

---

## T4 — `taskscope` extraction (after Task 4)

```bash
cd cli && go test ./internal/taskscope/... -v
```

**Pass:** All three tests pass.
**Fail:** Any `FAIL` — paste output.

---

## T5 — Global default config and methodology doc present (after Task 5)

```bash
cd "$(git rev-parse --show-toplevel)" && \
test -f harness/context/default.yaml && \
test -f harness/core/context-manager/METHOD.md && \
echo "ALL PRESENT"
grep -q "^strategy: selective" harness/context/default.yaml && echo "DEFAULT SCHEMA OK"
```

**Pass:** Both `ALL PRESENT` and `DEFAULT SCHEMA OK` print.
**Fail:** Any missing file or the grep fails.

---

## T6/T7 — Full build (after Task 7.3)

```bash
cd cli && go vet ./... 2>&1 && go build -o eng . 2>&1 && echo BUILD_OK
```

**Pass:** `BUILD_OK` prints, no compile errors.
**Fail:** Any compile error — paste it in full, including file and line. In particular check
for an import cycle: `context_cmd.go` imports `internal/skills`, `internal/project`,
`internal/planmeta`, `internal/taskscope`, `internal/docsearch`, `internal/skillmatch`,
`internal/contextcfg` — none of those internal packages should import back from `cli` or from
each other in a way that cycles.

---

## T8 — `eng context skills` end-to-end

```bash
REPO="$(git rev-parse --show-toplevel)"
cd cli && go build -o eng . && ./eng install --from .. && cd ..
rm -rf /tmp/eng-test-p4 && mkdir /tmp/eng-test-p4 && cd /tmp/eng-test-p4
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" context skills "improve planning methodology"
```

**Pass:** Output shows `Selected N/1 skills` (only `karpathy-guidelines` is installed at this
point) with `karpathy-guidelines` listed — its `tags: [planning, methodology, universal]`
frontmatter (Phase 1) should score a match against this request text, finally making those
fields load-bearing.
**Fail:** Zero skills selected despite an obvious tag match, or a crash.

---

## T9 — `eng context skills` never drops `enabled_skills`

```bash
cd /tmp/eng-test-p4
mkdir -p skills/unrelated-skill
cat > skills/unrelated-skill/SKILL.md <<'EOF'
---
name: unrelated-skill
domain: unknown
description: Has nothing to do with the request text below.
tags: [nothing-matching-here]
---
EOF
# Add it to enabled_skills manually for this test:
sed -i '/enabled_skills:/a\    - unrelated-skill' .agent/project.yaml
REPO="$(git rev-parse --show-toplevel)"
"$REPO/cli/eng" context skills "completely different topic entirely"
```

**Pass:** `unrelated-skill` still appears in the selected list even though nothing in the
request matches its tags/description — proving `enabled_skills` is never silently dropped.
**Fail:** `unrelated-skill` is missing from the output.

---

## T10 — `eng context project` matches relevant sections only

```bash
REPO="$(git rev-parse --show-toplevel)"
cd /tmp/eng-test-p4
mkdir -p docs
cat > docs/src-map.md <<'EOF'
## Modules

### `src/auth/` — authentication module

What it does: handles login and session tokens.

### `src/billing/` — billing module

What it does: handles invoices and payment processing.
EOF
"$REPO/cli/eng" context project "fix a login bug"
```

**Pass:** Output includes the `src/auth/` section and excludes (or ranks far below) the
`src/billing/` section — `login` in the request matches `auth`'s body text, not billing's.
**Fail:** Both sections returned with no differentiation, or neither returned.

---

## T11 — `eng context task` returns only the current task

```bash
REPO="$(git rev-parse --show-toplevel)"
cd /tmp/eng-test-p4
"$REPO/cli/eng" plan new smoke --risk feature
PLAN=$(ls -d .plans/*-smoke)
sed -i 's/\[Feature Name\]/Smoke/' "$PLAN/spec.md"
cat >> "$PLAN/tasks.md" <<'EOF'

## Task 1 — Already done

- [x] **1.1** finished

## Task 2 — Still pending

- [ ] **2.1** not yet done
EOF
"$REPO/cli/eng" context task "$PLAN"
```

**Pass:** Output's "## Current task" section contains "Task 2" and does **not** contain
"Task 1" — only the first unchecked block is returned. "## Goal summary" contains a short
excerpt from `spec.md`'s Goal paragraph, not the whole file.
**Fail:** Both tasks shown, or the wrong task selected, or the goal summary includes unrelated
sections (e.g. "Design decisions").

---

## T12 — `eng context bundle` writes a manifest and composes per role

```bash
REPO="$(git rev-parse --show-toplevel)"
cd /tmp/eng-test-p4
PLAN=$(ls -d .plans/*-smoke)
"$REPO/cli/eng" context bundle executor "$PLAN"
cat "$PLAN/context-manifest.yaml"
```

**Pass:** Command prints "## Task scope" (Task 2's block) and "## Skills" sections; exits 0;
`context-manifest.yaml` exists in the plan directory and contains `role: executor`,
`plan: <plan-name>`, and a `skills:` list.
**Fail:** Missing manifest file, wrong role composed (e.g. Planner's project-context section
appears instead of task scope), or a crash.

---

## T13 — `eng verify` log compaction: bounded report, full log on disk

```bash
REPO="$(git rev-parse --show-toplevel)"
cd /tmp/eng-test-p4
PLAN=$(ls -d .plans/*-smoke)
mkdir -p .agent
cat > .agent/context.yaml <<'EOF'
max_log_lines: 10
EOF
# Override test_cmd with a command that deterministically prints 50 lines —
# do NOT rely on `go test ./...`'s incidental (and possibly short, e.g. a
# 2-3 line module-not-found error) output to exceed max_log_lines; that
# would make this test's pass/fail depend on accidental message length
# rather than the compaction logic itself.
sed -i 's/test_cmd:.*/test_cmd: "seq 1 50"/' .agent/project.yaml
"$REPO/cli/eng" verify "$PLAN"
LOGFILE=$(ls .agent/logs/verify-*.log | tail -1)
echo "log file: $LOGFILE"
wc -l < "$LOGFILE"
grep -c "lines omitted" "$PLAN/verify-report.md"
```

**Pass:** `.agent/logs/verify-*.log` exists and contains the **full** 50-line output;
`wc -l` reports 50. `verify-report.md`'s embedded output block is bounded to `max_log_lines`
(10) lines total (5 head + 5 tail) and contains a "... [40 lines omitted, see full log] ..."
marker plus a "Full output: `<path>`" line. `plan.yaml`'s `verification.verdict` field is
still populated (`PASS`, since `seq` exits 0) — the compaction must not have broken the
machine-readable path.
**Fail:** Full log missing or not 50 lines, report not bounded to ~10 lines, no omitted
marker, or `verification.verdict` empty — the last case would mean this change broke
`eng workflow advance`'s gating logic, the single most important thing to not regress here.

---

## T14 — `eng init` adds a `.gitignore` entry for `.agent/logs/` without disturbing existing content

```bash
REPO="$(git rev-parse --show-toplevel)"
rm -rf /tmp/eng-test-p4-gitignore && mkdir /tmp/eng-test-p4-gitignore && cd /tmp/eng-test-p4-gitignore
git init -q
echo "node_modules/" > .gitignore
touch go.mod
"$REPO/cli/eng" init
cat .gitignore
echo "--- re-run after deleting project.yaml (simulates eng init run twice) ---"
rm .agent/project.yaml
"$REPO/cli/eng" init
grep -c "\.agent/logs/" .gitignore
```

**Pass:** After the first run, `.gitignore` contains both the original `node_modules/` line
(untouched, still first) and a new `.agent/logs/` line appended after it. After the second
run, `grep -c` reports exactly `1` — the entry was not duplicated.
**Fail:** `node_modules/` line removed or reordered, `.agent/logs/` missing after the first
run, or the count is `2` (duplicated) after the second run.

---

## T15 — Phase 1/2/3 regression gate

```bash
REPO="$(git rev-parse --show-toplevel)"
rm -rf /tmp/eng-test-p4-regress && mkdir /tmp/eng-test-p4-regress && cd /tmp/eng-test-p4-regress
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init

"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
"$REPO/cli/eng" capabilities list
"$REPO/cli/eng" plan new regress --risk feature
PLAN=$(ls -d .plans/*-regress)
"$REPO/cli/eng" plan drift "$PLAN"
"$REPO/cli/eng" plan retry "$PLAN" unit_test
"$REPO/cli/eng" triage "fix a bug"
sed -i 's/\[Feature Name\]/Regress/' "$PLAN/spec.md"
"$REPO/cli/eng" workflow advance "$PLAN"
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass:** Every command behaves exactly as documented in the Phase 1/2/3 plans — in
particular, `eng skills list` (unfiltered, unchanged) still returns every resolved skill even
though `eng context skills` (new, filtered) now also exists; `eng workflow advance` still
transitions `TRIAGED -> PLANNED` correctly, proving Phase 4 didn't disturb the state machine.
**Fail:** Any command errors or behaves differently than its own phase's documentation.

---

## T16 — V1 regression gate (unchanged since Phase 1)

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase4-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase4-regression && \
rm -rf .plans/*-smoke-test-phase4-regression
```

**Pass:** Identical output to every prior run of this test.
**Fail:** Any difference — Phase 4 must be provably additive to V1 as well.

---

## Cleanup

```bash
rm -rf /tmp/eng-test-p4 /tmp/eng-test-p4-gitignore /tmp/eng-test-p4-regress
```
