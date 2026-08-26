# Phase 8 Tasks — Dogfooding, Benchmarking & Validation

Execute in order. Tasks 4–11 are execution-and-recording tasks, not code-diff tasks — each
runs real commands against a real fixture copy and writes down what actually happened. No
number in any committed result file may be invented; every field traces to a specific
command's real output, quoted or paraphrased in that scenario's `notes:`.

---

## Task 1 — `benchmarks/` skeleton and methodology doc

- [x] **1.1** Create the directory skeleton: `benchmarks/scenarios/`, `benchmarks/fixtures/`,
  `benchmarks/results/`.

- [x] **1.2** Create `benchmarks/README.md` covering: the three comparison modes and what
  each can/cannot prove (verbatim from spec.md's "What cannot be measured reliably"), the
  scenario/result YAML schemas, how to run each mode (the documented command sequence per
  scenario — filled in as each scenario task below is executed, so this file grows through
  the phase rather than being finalized up front), what "structural proxy" means and why no
  result contains a `tokens:` field, and how to add a new scenario.

**Verify:** directories exist; `benchmarks/README.md` is non-empty and names all three
modes.

---

## Task 2 — Scenario definitions

- [x] **2.1** Create one scenario YAML per category under `benchmarks/scenarios/` (schema
  from spec.md's "Result format" section, mirrored for inputs):

  `quick-fix-timeout.yaml`, `feature-csv-export.yaml`, `bug-lastn-off-by-one.yaml`,
  `large-context-auth-validation.yaml`, `cross-domain-esp32-siemens-modbus.yaml`,
  `tool-routing-git-status.yaml`, `legacy-v1-compat.yaml`, and five small
  `failure-*.yaml` files (`failure-reviewer-reject.yaml`, `failure-verifier-fail.yaml`,
  `failure-drift.yaml`, `failure-approval-missing.yaml`, `failure-tool-denied.yaml`).

  Each names: `category`, `request` (the exact natural-language text used across every mode
  that scenario runs), `fixture` (path, or `none` for scenarios 5/6 which use this repo
  itself), `modes` (which of `baseline`/`v1`/`harness-v2` actually run), and `expected`/
  `forbidden` lists matching spec.md's per-category success criteria exactly — copied, not
  paraphrased, so a result can be checked against the same wording that was fixed in advance.

**Verify:** 12 scenario files exist; each parses as valid YAML (`python3 -c "import yaml,
sys; yaml.safe_load(open(sys.argv[1]))"` per file, or equivalent).

---

## Task 3 — Fixtures

- [x] **3.1** Create `benchmarks/fixtures/quick-fix/` — a minimal Go program
  (`go.mod` + `main.go`) with a single named constant, `ReconnectTimeoutMS = 1000`, printed
  by `main()`. This is the fixture the scenario's request ("Increase the reconnect timeout
  from 1000 ms to 1500 ms") targets — deliberately the exact shape spec.md's own Success
  Criteria for Quick Fix names.

- [x] **3.2** Create `benchmarks/fixtures/feature/` — a minimal Go CLI (`go.mod` +
  `main.go`) with an in-memory `[]Record{ID, Name}` slice and two existing subcommands,
  `list` and `add`. The scenario's request asks for a third, `export`, printing CSV — small
  enough to implement in one sitting, large enough that "add one feature to existing code"
  is a real claim, not a trivial one.

- [x] **3.3** Create `benchmarks/fixtures/bug/` — `go.mod` + `lastn.go` (a `LastN(items
  []int, n int) []int` helper with a real, deterministic off-by-one:
  `items[len(items)-n-1:]` instead of `items[len(items)-n:]`) + `lastn_test.go` (a test
  that fails against the bug with a clear, non-panicking mismatch — `go test` must fail
  before the fix and pass after).

- [x] **3.4** Create `benchmarks/fixtures/large-context/` — `go.mod` plus five small
  packages (`api/`, `db/`, `cache/`, `auth/`, `utils/`, each one short file, ~10–20 lines)
  and a `cli/main.go` wiring them together, plus a `docs/src-map.md` with one `### ` section
  per package (matching this repository's own `docs/src-map.md` convention exactly, so
  `eng context project` has real sections to route against). The scenario's request
  ("add input validation to the auth token check so an empty token is rejected") should
  only concern `auth/` and its one `docs/src-map.md` section.

- [x] **3.5** Create `benchmarks/fixtures/legacy/` — `CLAUDE.md` (minimal), an empty
  `.plans/` (with `.gitkeep`), `skills/example/SKILL.md` (legacy heading style, no
  frontmatter), `.planner-executor/config.yaml`, and a trivial `go.mod` + `main.go` — the
  same V1-shaped recipe already proven, byte-for-byte, in every phase's own Legacy E2E test
  this session.

**Verify:** every fixture's Go code builds standalone (`cd benchmarks/fixtures/<name> && go
build ./...`); `benchmarks/fixtures/bug`'s test fails as designed
(`go test ./... ; test $? -ne 0`); no fixture directory contains a `.git/` (fixtures are
plain file trees — `git init` happens only in a scratch copy at run time).

---

## Task 4 — Run & record Category 1: Quick Fix (Mode A, B, C)

- [x] **4.1 — Mode C.** Copy `fixtures/quick-fix` to a scratch dir, `git init`, `eng init`,
  `eng workflow start "Increase the reconnect timeout from 1000 ms to 1500 ms."`, make the
  edit, `eng workflow advance` through to `COMPLETED`. Record real observed values (state
  transitions from `eng workflow status`/`events.jsonl`, `git diff --stat` file count,
  `context-manifest.yaml` skill/doc counts if a context bundle was pulled, verification
  verdict) into `benchmarks/results/quick-fix-timeout-harness-v2.yaml`.

- [x] **4.2 — Mode B.** Copy `fixtures/quick-fix` to a scratch dir, `git init`, create
  `.planner-executor/config.yaml`, `./scripts/plan-executor.sh new bump-timeout`, hand-author
  `spec.md`/`tasks.md`/`tests.md` as the Planner role would for a change this size, act as
  Executor, make the edit, `./scripts/plan-executor.sh test`. Record real observed values
  (files scaffolded, manual steps counted, test result) into
  `benchmarks/results/quick-fix-timeout-v1.yaml`.

- [x] **4.3 — Mode A.** Dispatch one fresh `general-purpose` `Agent` at a scratch copy of
  `fixtures/quick-fix` with only the raw request text and an instruction to use its own
  judgment, no assumed workflow. After it finishes, inspect the real `git diff` in that
  copy. Record what it actually did (files changed, whether it wrote any planning artifact
  unprompted, whether it ran `go build`/`go test` itself) into
  `benchmarks/results/quick-fix-timeout-baseline.yaml`, explicitly labeled single-run/
  illustrative per spec.md.

**Verify:** all three result files exist and satisfy spec.md's Quick Fix success criteria
for Mode C specifically (Mode A/B have no fixed pass/fail line — see Decision 5); the
three-mode comparison numbers get pulled forward into Task 12 unedited.

---

## Task 5 — Run & record Category 2: Feature / Spec-First (Mode A, B, C)

- [x] **5.1 — Mode C.** Copy `fixtures/feature` to a scratch dir, `git init`, `eng init`
  (this project gets `planning_mode: spec_first` by default per Phase 5), `eng workflow
  start "Add an export command that prints all records as CSV (id,name) to stdout."`.
  Confirm the state reaches `NEEDS_SPEC_APPROVAL` **before** `tasks.md`/`tests.md` exist
  (`ls` the plan dir at that point and record it). Write `spec.md`, `eng plan approve-spec`,
  confirm `SPEC_APPROVED`, write `tasks.md`/`tests.md`, advance through review/approval (if
  triggered)/execute/verify to `COMPLETED`. Record into
  `benchmarks/results/feature-csv-export-harness-v2.yaml`.

- [x] **5.2 — Mode B.** Same fixture, V1 flow: `plan-executor.sh new csv-export`, author
  `spec.md`/`tasks.md`/`tests.md` directly (V1 has no separate spec-approval gate — note
  this explicitly as a structural difference, not a defect, in the result's `notes:`),
  implement, test. Record into `benchmarks/results/feature-csv-export-v1.yaml`.

- [x] **5.3 — Mode A.** One fresh `general-purpose` `Agent` dispatch at a scratch copy of
  `fixtures/feature` with only the raw request text. Record whether it drafted any spec/plan
  before coding, whether it stopped for approval (it cannot meaningfully "stop" inside a
  single autonomous dispatch — record this limitation explicitly rather than papering over
  it), what it changed, and whether it added a test. Record into
  `benchmarks/results/feature-csv-export-baseline.yaml`, labeled single-run/illustrative.

**Verify:** the Mode C result shows `tasks.md`/`tests.md` genuinely absent at the
`NEEDS_SPEC_APPROVAL` checkpoint (this is the one hard, falsifiable claim in this category)
and present only after `SPEC_APPROVED`.

---

## Task 6 — Run & record Category 3: Bug Fix (Mode C)

- [x] **6.1** Copy `fixtures/bug` to a scratch dir, `git init`, `eng init`. Confirm
  `go test ./...` fails first (the seeded bug is real). Run
  `eng workflow start "LastN is returning the wrong elements — debug and fix it."` and
  inspect `eng context skills`' output for this request — record whether
  `engineering/debugging` is selected and why (its `triggers:` include "debug"/"fails"). Fix
  the bug, advance to verify, confirm `go test ./...` passes and `eng verify` reports
  `PASS`. Record into `benchmarks/results/bug-lastn-off-by-one-harness-v2.yaml`, including a
  `notes:` paragraph discussing (not measuring) what Mode A/B would plausibly differ on —
  explicitly marked as discussion, not a run.

**Verify:** the result shows `engineering/debugging` in the selected-skills list with a
real `selected because ...` reason string, and `verification.verdict: PASS`.

---

## Task 7 — Run & record Category 4: Large Context (Mode C)

- [x] **7.1** Copy `fixtures/large-context` to a scratch dir, `git init`, `eng init`, run
  `eng context skills "add input validation to the auth token check so an empty token is
  rejected"` and `eng context project` with the same request. Record the fixture's total
  file/package count (a plain `find`/`ls` count) alongside the bundle's selected file/doc
  count, so the ratio is the actual evidence, not an assertion. Confirm only the `auth/`
  section of `docs/src-map.md` is matched. Record into
  `benchmarks/results/large-context-auth-validation-harness-v2.yaml`.

**Verify:** the result's `context.docs` count is 1 (only the `auth/` section), and the
fixture's total tracked-file count is recorded alongside it so the ratio is visible without
recomputation.

---

## Task 8 — Run & record Category 5: Cross-Domain (Mode C)

- [x] **8.1** From this repository itself (already `eng init`-ed via every prior phase's
  testing this session — or a fresh scratch copy if a clean state is preferred), run
  `eng context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"` — the instruction's own
  headline example, and the same request already proven in Phase 6's committed
  `harness/evals/embedded/esp32-siemens-modbus.yaml` router eval. Record the real selected
  skill list and each one's reason string into
  `benchmarks/results/cross-domain-esp32-siemens-modbus-harness-v2.yaml`. This captures an
  already-proven Phase 6 guarantee as a durable Phase 8 artifact rather than re-deriving it.

**Verify:** selected skills include (at minimum) `esp32`, `siemens-s7`, `plc`, `modbus`,
`tcp-ip` — the exact set Phase 6's own committed eval scenario already asserts.

---

## Task 9 — Run & record Category 6: Tool-Routing (Mode C)

- [x] **9.1** From this repository (a real git repo), scaffold a throwaway plan
  (`eng plan new bench-tool-routing --risk quick-fix` in a scratch copy, or reuse an
  existing scratch plan dir), run `eng tools invoke executor git.status <plan-dir>`, and
  inspect the resulting `tool_invocation` audit event in that plan's `events.jsonl`. Record
  the exact event fields (confirming no raw command output appears inline — only
  `log_path`) into `benchmarks/results/tool-routing-git-status-harness-v2.yaml`.

**Verify:** the audit event's field set matches exactly `adapter`, `capability`, `role`,
`result`, `reason`, `log_path`, `type`, `at` — nothing else.

---

## Task 10 — Run & record Category 7: Failure/Safety (Mode C, 5 sub-cases)

- [x] **10.1** Reviewer REJECT: scaffold a `--risk architecture` plan (review required),
  write minimal spec/tasks/tests, `eng plan review <dir> --verdict REJECT`,
  `eng workflow advance` — confirm state becomes `NEEDS_REPLAN`, never `APPROVED`.

- [x] **10.2** Verifier FAIL: scaffold a plan, get it to `EXECUTING`, leave the fixture's
  test failing, run `eng verify` — confirm verdict `FAIL` and state does not reach
  `COMPLETED`.

- [x] **10.3** Drift: scaffold and advance a plan to `APPROVED`, modify a tracked file
  outside `write_scope` before executing, run `eng plan drift`/`eng workflow advance` —
  confirm `NEEDS_REPLAN`.

- [x] **10.4** Approval missing: scaffold a `--risk high-risk` plan, attempt
  `eng workflow advance` past `NEEDS_APPROVAL` without running `eng plan approve` — confirm
  it stays at `NEEDS_APPROVAL`.

- [x] **10.5** Tool capability denied: `eng tools invoke planner git.push <plan-dir>` (role
  toolbox/risk-ceiling denial, reusing Phase 7's own proven E2E case) — confirm `REFUSED
  (DENIED)`, non-zero exit, and that no `git push` process ever actually ran (no network
  error in the output — only the policy refusal message).

  Record all five outcomes together into `benchmarks/results/failure-safety-harness-v2.yaml`
  (one file, five sub-sections — five separate files would be pure overhead for cases this
  small).

**Verify:** every sub-case's recorded end state matches spec.md's Failure/Safety success
criteria exactly; none reaches a state or invocation the criteria forbid.

---

## Task 11 — Run & record Category 8: Legacy (Mode B + C)

- [x] **11.1 — Mode B.** Copy `fixtures/legacy` to a scratch dir, `git init`, run
  `./scripts/load_skill.sh list` and `./scripts/plan-executor.sh new smoke-check` against
  it — confirm both behave exactly as documented (V1 unaware anything past it exists).

- [x] **11.2 — Mode C.** Same fixture copy, add `.agent/project.yaml` with `mode: hybrid`
  (or run `eng init` fresh in `hybrid` mode per `project.DetectMode`'s existing legacy
  detection), confirm `eng doctor`/`eng workflow start` treat it exactly as every prior
  phase's own Legacy E2E test already proved (auto_plan path, no forced migration).

  Record both into `benchmarks/results/legacy-v1-compat.yaml` (one file, two mode sections).

**Verify:** Mode B's output is unchanged from a pre-Phase-1 run (nothing about V1's own
behavior has ever been touched by any phase); Mode C shows zero required migration steps.

---

## Task 12 — Comparison tables, context-efficiency findings, scorecard

- [x] **12.1** Create `benchmarks/COMPARISON.md`: one human-readable table per category that
  ran more than one mode (Quick Fix, Feature, Legacy), in the exact style spec.md's
  "Compare V2 Against V1" section shows — metric rows, one column per mode that actually
  ran, every cell sourced from a committed result file (no new numbers invented here).

- [x] **12.2** Create `benchmarks/CONTEXT_EFFICIENCY.md`: pull the Category 4/5 results
  together with the explicit ratio (selected/total) and a short paragraph judging whether
  Phase 4's "large knowledge base ≠ large prompt" principle held in practice, grounded in
  those two results only.

- [x] **12.3** Create `benchmarks/SCORECARD.md`: the ten-dimension scorecard from
  Requirement 20 (Setup UX, Quick Fix UX, Feature Planning, Context Efficiency, Skill
  Routing, Tool Routing, Safety, Legacy Compatibility, Cross-Domain Usability,
  Maintainability), each rated with the specific committed result file(s) it's based on —
  never an unsourced rating.

**Verify:** every claim in all three files cites a specific `benchmarks/results/*.yaml`
file by name.

---

## Task 13 — Refinement backlog and decision gate

- [x] **13.1** Create `benchmarks/BACKLOG.md`: every real weakness Tasks 4–11 actually
  surfaced (not hypothetical ones), each classified P0–P3 per Requirement 22's scale, each
  citing the specific result file that surfaced it.

- [x] **13.2** In `benchmarks/SCORECARD.md`, add a closing section stating exactly one of
  `READY_TO_EXPAND` or `CORE_REFINEMENT_REQUIRED`, with the reasoning grounded in the
  scorecard ratings and backlog severities above it — decided after Tasks 4–11 are complete,
  never before.

**Verify:** the decision line is present and is exactly one of the two literal strings.

---

## Task 14 — Full regression re-verification

- [x] **14.1** Run `cd cli && go build ./... && go vet ./... && go test ./...` — confirm
  every Phase 1–7 package still passes unmodified (Phase 8 adds no Go code, so this should
  be a no-op confirmation, not a fix-up step).

- [x] **14.2** Re-run the V1 regression script check
  (`./scripts/load_skill.sh list && ./scripts/plan-executor.sh new smoke-test-phase8 &&
  ./scripts/plan-executor.sh list | grep smoke-test-phase8`, then clean up the scaffolded
  plan) and confirm identical behavior to every prior phase's own run of this check.

- [x] **14.3** Re-run Phase 6's router eval integration test
  (`go test ./... -run TestRouterEvalScenarios -v`) and confirm all three scenarios still
  pass — the one place Phase 8's own Category 5 benchmark and an existing automated test
  cover overlapping ground; both should agree.

**Verify:** all three sub-steps pass with no diff from Phase 7's own final verification
output.
