# Phase 6 Tests — Multi-Domain Skill Ecosystem + Skill Router Evolution

## Per-task unit/build gates

Run after each corresponding task in `tasks.md`. All of these are also re-run as one batch
in the final regression gate (T16 below).

## T1 — Hook plan-directory fix

```bash
REPO="$(git rev-parse --show-toplevel)"
export PATH="$PATH:/c/Program Files/Go/bin"
cd "$REPO/cli" && go build -o eng . && echo BUILD_OK

rm -rf /tmp/eng-test-p6-hooks && mkdir -p /tmp/eng-test-p6-hooks && cd /tmp/eng-test-p6-hooks
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init

# Old shape, no plan-dir arg, run from project root with no plan.yaml there — unchanged
# behavior: still fails exactly as it always has (this is not being "fixed", only made
# escapable via the new optional argument).
"$REPO/cli/eng" hooks run before_execute; echo "no-arg-exit=$?"

# New shape: create a plan, pass its directory explicitly, run from project root.
"$REPO/cli/eng" plan new demo --risk feature
PLAN=$(ls -d .plans/*-demo)
"$REPO/cli/eng" hooks run before_execute "$PLAN"; echo "with-plan-dir-exit=$?"
```

**Pass:** `no-arg-exit` is non-zero with a "no plan.yaml found" message — byte-identical to
Phase 2–5 behavior (proving zero regression for every existing invocation shape).
`with-plan-dir-exit` is `0`, and the printed command line reads
`eng plan drift <the real plan dir>`, not `eng plan drift .` — proving the new argument is
actually threaded through the `${plan_dir}` substitution.
**Fail:** the no-arg case's exit code or message changed, or the with-plan-dir case still
prints `eng plan drift .`.

---

## T2 — Skill metadata schema + `QualifiedName`/`ResolveWithPrivate`

```bash
cd "$REPO/cli" && go test ./internal/skills/... -v
```

**Pass:** every test passes, including the four pre-existing ones
(`TestParseFrontmatter`, `TestParseLegacyHeading`, `TestResolveLocalOverridesGlobal`,
`TestResolveMissingRoots`) unmodified, proving the rename/refactor didn't touch existing
behavior.
**Fail:** any pre-existing test fails, or a new collision/precedence test fails.

---

## T3 — `Domains`/`PrivateSkillsPath`

```bash
cd "$REPO/cli" && go test ./internal/project/... -v
```

**Pass:** all tests pass, including every pre-Phase-6 `project` test.

---

## T4 — `internal/skillgraph`

```bash
cd "$REPO/cli" && go test ./internal/skillgraph/... -v
```

**Pass:** all 6 tests pass — transitive closure, diamond dedup, cycle detection, unknown
skill error, deterministic order, bare-name requires.

---

## T5 — `internal/skillrouter`

```bash
cd "$REPO/cli" && go test ./internal/skillrouter/... -v
```

**Pass:** all 8 tests pass — explicit never dropped, required ignores budget, unknown
required errors, recommends dropped/kept by budget, higher score wins budget, domain-profile
fill, deterministic tie-break order.

---

## T6 — `context_cmd.go` router integration

```bash
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"
```

**Pass:** output lists `esp32`, `siemens-s7`, `plc`, `modbus`, `tcp-ip` (assuming
`eng install --from ..` has already propagated the Task 10 skill files to
`~/.engineering-harness/skills/` — see T10 below, run before this), each followed by a
`selected because ...` line naming a real reason (`matched request text`,
`required by automation/siemens-s7`, `recommended by automation/modbus`, or similar).
**Fail:** `plc` is missing (dependency expansion broken), or no reason line is printed.

---

## T7 — `internal/skillvalidate`

```bash
cd "$REPO/cli" && go test ./internal/skillvalidate/... -v
```

**Pass:** all 9 tests pass — legacy warns not errors, missing description warns, unknown
requires errors, unknown recommends warns only, duplicate qualified name warns, same bare
name across domains does *not* warn, cycle errors, bad version warns, `Report.Errors()`
excludes warnings.

---

## T8 — `eng skills validate`

```bash
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" skills validate; echo "exit=$?"
```

**Pass:** `exit=0` (no errors expected against the shipped skill set once T10's files are
installed), and every real skill is discovered.
**Fail:** a non-zero exit with no real defect, or a shipped skill silently missing from the
count.

---

## T9 — `eng doctor` skill summary

```bash
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" doctor
```

**Pass:** output contains a `Skills:` section with exactly 4 detail lines (discovered /
valid / warnings / broken dependencies) plus the pointer to `eng skills list`/`eng skills
validate` — not a per-skill dump.
**Fail:** doctor still prints one line per skill (the old behavior wasn't actually replaced).

---

## T10 — Representative skill set

```bash
cd "$REPO/cli" && go build -o eng . && ./eng install --from ..
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" skills list | wc -l
"$REPO/cli/eng" skills validate
```

**Pass:** `skills list` reports 11 lines (10 new files — 9 from the suggested initial set
plus `automation/siemens-s7` to validate the headline dependency example — plus
`karpathy-guidelines`); `skills validate` reports `11 skill(s) discovered, 0 error(s), 0
warning(s)`.
**Fail:** fewer than 11 skills discovered (a file wasn't installed/parsed), or any error.

---

## T11 — Router evaluation scenarios

```bash
cd "$REPO/cli" && go test ./internal/skilleval/... -v
go test ./... -run TestRouterEvalScenarios -v
```

**Pass:** all `skilleval` unit tests pass, and every subtest under `TestRouterEvalScenarios`
(`esp32-siemens-modbus`, `cpp-debug`, `docker-linux-ci`) passes — the router selects every
`expected_skills` entry for its scenario's request against the real, committed
`harness/skills` tree.
**Fail:** any subtest fails — this means either a skill's `tags`/`triggers` don't actually
match its own scenario's request text, or dependency/recommend expansion is broken for real
shipped skills, not just synthetic test fixtures.

---

## T12 — `main.go` usage string

```bash
cd "$REPO/cli" && go build -o eng . && ./eng
```

**Pass:** usage output includes both `skills list` (mentioning "private") and
`skills validate` lines.

---

## T13 — `context-manager/METHOD.md`

Manual read: confirm the new paragraph mentions `internal/skillrouter.Route` and still says
skill selection happens in exactly one place.

---

## T14 — `docs/skills.md`

Manual read: confirm every item from Requirement 28's list (skill/pack/level/domain
structure/metadata/dependency behavior/routing precedence/project profiles/source
precedence/inspecting routing/creating a new skill) is covered.

---

## T15 — `docs/src-map.md`/`README.md`/`ROADMAP.md`

Manual read: confirm the new sections exist and reference
`.plans/2026-08-25-v2-harness-phase6-skills/`.

---

## T16 — Version bump

```bash
cat "$REPO/harness/VERSION"
```

**Pass:** `0.6.0-phase6-skills`.

---

## End-to-end walkthroughs

### E2E-QUICKFIX — Quick Fix stays light under the new router

```bash
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" plan new small-tweak --risk quick-fix
PLAN=$(ls -d .plans/*-small-tweak)
sed -i 's/\[Feature Name\]/Small tweak/' "$PLAN/spec.md" "$PLAN/tasks.md" "$PLAN/tests.md"
"$REPO/cli/eng" workflow advance "$PLAN"
"$REPO/cli/eng" adapter prompt executor "$PLAN" "fix a small c++ typo" | grep -A2 "## Skills"
```

**Pass:** state reaches `EXECUTING` exactly as in Phase 5's own Quick Fix walkthrough
(unaffected by this phase), and the folded-in skill section is short (at most
`max_skills`, default 5) — Quick Fix does not turn into a full domain-analysis dump.
**Fail:** Quick Fix's context bundle exceeds the configured budget, or the state machine
path changed from Phase 5's documented `TRIAGED -> EXECUTING`.

### E2E-SPECFIRST — Spec-First unaffected by routing changes

```bash
cd /tmp/eng-test-p6-hooks
"$REPO/cli/eng" plan new add-feature --risk feature
PLAN=$(ls -d .plans/*-add-feature)
"$REPO/cli/eng" workflow status "$PLAN"
```

**Pass:** `State: TRIAGED`, `planning_mode` is `spec_first` (this project's `.agent/
project.yaml` was written by this session's `eng init`) — identical to Phase 5's own
Spec-First walkthrough; the approval-state machine (`NEEDS_SPEC_APPROVAL`/`SPEC_APPROVED`)
is untouched by this plan.

### E2E-LEGACY — A hand-written pre-Phase-6 project.yaml is unaffected

```bash
rm -rf /tmp/eng-test-p6-legacy && mkdir -p /tmp/eng-test-p6-legacy && cd /tmp/eng-test-p6-legacy
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
mkdir -p .agent
cat > .agent/project.yaml <<'EOF'
project_name: eng-test-p6-legacy
mode: modern
harness_profile: software
stack:
    type: go
    build_cmd: go build ./...
    test_cmd: echo ok
enabled_skills:
    - engineering/karpathy-guidelines
EOF
# No `domains:`, no `private_skills_path:`, no `workflow:` block at all.

"$REPO/cli/eng" context skills "add a feature"
```

**Pass:** command succeeds; `karpathy-guidelines` is present in the selection (still
explicitly enabled, still normalized correctly); no error about missing `domains`/
`private_skills_path` (both are legitimately absent, and the router treats that exactly as
"no domain-profile tier, no private tier" per Decision 7/8) — proving a project.yaml that
predates every Phase 6 field works unchanged.

---

## Regression gates — Phase 1 through 5 and V1

```bash
cd "$REPO/cli" && go build ./... && go vet ./... && go test ./... 2>&1
```

**Pass:** builds clean, `go vet` clean, every package's tests pass — including
`internal/workflow`, `internal/project`, `internal/planmeta`, `internal/contextcfg`,
`internal/logprune`, `internal/tooladapter`, `internal/toolrouter`, `internal/agent`,
`internal/capabilities` (all untouched by this phase) alongside the new Phase 6 packages.

```bash
rm -rf /tmp/eng-test-p6-regress && mkdir -p /tmp/eng-test-p6-regress && cd /tmp/eng-test-p6-regress
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init

"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
"$REPO/cli/eng" skills validate
"$REPO/cli/eng" capabilities list
"$REPO/cli/eng" context skills "add a feature"
"$REPO/cli/eng" context project "add a feature"
"$REPO/cli/eng" plan new regress --risk feature
PLAN=$(ls -d .plans/*-regress)
"$REPO/cli/eng" plan drift "$PLAN"
"$REPO/cli/eng" plan retry "$PLAN" unit_test
"$REPO/cli/eng" triage "fix a bug"
"$REPO/cli/eng" hooks run before_execute "$PLAN"
"$REPO/cli/eng" logs prune --dry-run
```

**Pass:** every command runs and behaves as documented in its own phase's plan — Phase 6
only ever adds subcommands/fields (`skills validate`, `domains`, `private_skills_path`,
`hooks run`'s optional third argument), never changes an existing one's meaning. The
`hooks run before_execute "$PLAN"` line here should succeed (exit 0) — unlike the
project-root, no-arg case documented as an accepted pre-existing limitation in T1 above,
since a real plan directory is now passed explicitly.
**Fail:** any command errors unexpectedly, or an existing flag/argument shape stops working.

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase6-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase6-regression && \
rm -rf .plans/*-smoke-test-phase6-regression && \
echo "V1 REGRESSION CHECK OK"
```

**Pass:** identical output to every prior phase's run of this same check.

---

## Cleanup

```bash
rm -rf /tmp/eng-test-p6-hooks /tmp/eng-test-p6-legacy /tmp/eng-test-p6-regress
```
