# Phase 9 Tasks — Core Refinement & Real-World Skill Coverage

Execute in order. Every fix task follows: preserve failing evidence → regression test → smallest
clean fix → rerun focused package tests → rerun/record the related benchmark scenario. Do not
bundle unrelated refactors into a fix task.

---

## Task 1 — P1-1: Quick Fix triage classifier

- [x] **1.1** Preserve evidence: `eng triage "Increase the reconnect timeout from 1000 ms to
  1500 ms."` currently returns `feature` (already reproduced in Phase 8; re-confirm on current
  `main`).
- [x] **1.2** Add `cli/triage_cmd_test.go` (new file — none exists today) with table-driven
  cases: the canonical timeout example → `quick-fix`; a request combining a size-hint verb with
  an auth/security word (e.g. "increase the session timeout for authenticated users from 30 to
  60 minutes") → stays high-risk, never `quick-fix`; a request combining a size-hint verb with a
  hardware-control word → stays high-risk; existing typo/rename/bug/architecture keyword cases
  still classify as before (no regression on current behavior).
- [x] **1.3** In `cli/triage_cmd.go`: extend the `high-risk` word list with security/auth
  (`authenticat`, `authoriz`, `credential`, `password`, `login`, `permission`, `access control`,
  `auth token`) and hardware-control (`actuator`, `motor control`, `relay output`, `move axis`,
  `gpio write`) terms; extend `architecture` with `breaking change`/`public api` (schema/database
  migration wording is already covered by the pre-existing bare `migration` high-risk keyword,
  checked first — a separate entry there would be unreachable). Add
  `looksLikeParameterTweak(lower string) bool` (change-verb word list + digit presence) and
  consult it in `keywordLevel` only after the existing table loop finds nothing.
- [x] **1.4** Run `go test ./internal/... ./... -run Triage -v` (package + root `eng` package,
  since `triage_cmd_test.go` lives in `package main`). Confirm all new + existing cases pass.
- [x] **1.5** Record `benchmarks/results/quick-fix-timeout-triage-fixed.yaml`: real
  `eng triage` output for the canonical example (now `quick-fix`) and for one risk-elevating
  counter-example (stays elevated), both with real command output quoted.

**Verify:** `eng triage "Increase the reconnect timeout from 1000 ms to 1500 ms."` prints
`Suggested level: quick-fix`; a request naming authentication/production/hardware alongside a
size-hint verb does not.

---

## Task 2 — P1-2 / P2-2: word-boundary + weighted skill/doc matching

- [x] **2.1** Preserve evidence: capture real `eng context skills` output for the RS485/WinForms
  request (run directly from the repo root — this is sufficient to exercise the real shipped
  `harness/skills` tree without needing a project fixture; the dedicated fixture is still
  created in Task 8 for the persisted eval scenario) showing `modbus`/`siemens-s7`/`plc`/
  `tcp-ip`/`debugging`/`linux` incorrectly selected on current `main`; capture real
  `eng context project` output on the existing `benchmarks/fixtures/large-context/` fixture
  showing the pre-existing 4/6 over-match (re-confirmed on current `main`, unchanged from
  Phase 8).
- [x] **2.2** In `cli/internal/skillmatch/skillmatch.go`: add a tokenizer (letters/digits only,
  lowercased — plus a `symbolLanguageNames` pre-normalization for `c++`/`c#`, a real collision
  found during reproduction: both degenerate to a bare `c` token once punctuation is stripped)
  and a `containsPhrase(tokens []string, phrase string) bool` helper matching a (possibly
  multi-word) phrase as a contiguous token subsequence. Rewrote `Score` to: tag/trigger match →
  weight `TagTriggerWeight = 3` each; **deduplicated** description words, length > 3 (the
  original threshold — word-boundary tokenizing, not word length, is what kills the substring
  class of bug), weight `DescriptionWordWeight = 1` each. Exported `MinMatchScore = 2`. Updated
  `Select` (and `skillrouter.Route`'s Tier-B call site) to require `Score(...) >= MinMatchScore`
  instead of `> 0`.
- [x] **2.3** In `cli/internal/docsearch/docsearch.go`: applied the same tokenizer; score title
  words at weight `TitleWordWeight = 2`, deduplicated body words (length > 2, the original
  threshold) at weight `BodyWordWeight = 1`; exported `MinMatchScore = 2`; `Match` includes a
  section only when its total score `>= MinMatchScore`.
- [x] **2.4** Updated `cli/internal/skillrouter/skillrouter_test.go`:
  `TestRecommendsIncludedWhenBudgetAllows`'s fixture needed the real fix (not
  `TestDomainProfileFillAfterStrongMatches`, which turned out to still pass via its Tier-C
  domain-profile fill regardless — documented in DECISION_LOG.md Decision 4). Added new cases in
  `skillmatch_test.go`: `TestScoreIgnoresSubstringInsideUnrelatedWord`,
  `TestSingleDescriptionWordAloneDoesNotClearThreshold`,
  `TestSingleTagOrTriggerAloneClearsThreshold`,
  `TestScoreDoesNotDoubleCountRepeatedDescriptionWord`, `TestScoreDistinguishesCSharpFromCpp`
  (the c++/c# collision found during reproduction).
- [x] **2.5** Updated `cli/internal/docsearch/docsearch_test.go`: added
  `TestMatchIgnoresSubstringInsideUnrelatedWord`, `TestMatchSingleBodyWordAloneDoesNotClear
  Threshold`, `TestMatchSingleTitleWordAloneClearsThreshold`. Existing
  `TestMatchRanksByScoreAndCaps`/`TestMatchNoCapReturnsAll` pass unchanged, fixtures untouched.
- [x] **2.6** Run `go test ./internal/skillmatch/... ./internal/docsearch/... ./internal/
  skillrouter/...` — all green.
- [x] **2.7** Re-run `go test ./... -run TestRouterEvalScenarios -v` (Phase 6's existing eval
  integration test) — still passes unchanged (esp32-siemens-modbus, docker-linux-ci, cpp-debug).
- [x] **2.8** Re-ran the RS485/WinForms and large-context reproductions from 2.1 against the
  fixed binary; recorded the real before/after into
  `benchmarks/results/skill-routing-rs485-not-plc.yaml` and
  `benchmarks/results/doc-context-overmatch-fixed.yaml`.

**Verify:** RS485/WinForms request no longer selects `automation/plc`/`automation/siemens-s7`;
large-context auth-validation request's `docs_matched` count is materially below 4/6 (target
1–2/6); `TestRouterEvalScenarios` still fully passes.

---

## Task 3 — P1-3: `eng start` child environment

- [x] **3.1** Preserve evidence (documented, not literally reproducible in this environment
  without a real Windows session mid-bug): recorded the exact reported symptom and root-cause
  analysis from spec.md's P1-3 section — `eng.exe` invoked by full path bypasses PATH lookup
  entirely, so its own inherited environment (and everything it later spawns) never gained the
  harness `bin/` directory.
- [x] **3.2** Added `cli/start_cmd_test.go` testing the pure `buildChildEnv(base []string,
  binDirs []string, harnessHome, projectRoot, version string) []string` function: PATH prepended
  (not replaced), deduplicated bin dirs, works with no pre-existing PATH, case-insensitive PATH
  detection (Windows `Path=`), `ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION` set and replace (not
  duplicate) any pre-existing same-named entries.
- [x] **3.3** Implemented `buildChildEnv` plus `engBinDirs()`/`dedupStrings()` in
  `cli/start_cmd.go`. Added a shared `harnessVersion() string` helper in `cli/install.go` (reads
  `~/.engineering-harness/VERSION`, trimmed) and updated `doctor.go` to use it instead of its own
  inline read. Wired `cmdStart` to set `c.Env = buildChildEnv(os.Environ(), engBinDirs(),
  harnessDir(), dir, harnessVersion())` before `c.Run()`.
- [x] **3.4** `go build ./... && go vet ./... && go test ./...` — all packages pass, including
  every existing doctor-adjacent test; `doctor.go`'s only change is calling the shared helper
  (identical output).
- [x] **3.5** Manual verification: ran `eng start --init` against a fresh scratch fixture (no
  prior `.agent/project.yaml`). Startup sequence (init, doctor, capabilities, METHOD.md pointer)
  printed unchanged; `claude` happened to be on PATH in this environment (a nested Claude Code
  session) so it attempted a real launch through the new `c.Env` — it reached the real `claude`
  CLI and got a `--print`/stdin argument error from running non-interactively, not a
  "command not found" error, confirming the child process was launched with a working PATH via
  the new wiring. The `err != nil` fallback path ("Could not launch `claude` automatically...
  Run it yourself in this directory.") printed correctly, exactly as before.

**Verify:** `buildChildEnv`'s unit tests pass (6/6); `eng start` still behaves identically at
every printed step; the one live launch attempt in this environment reached the real `claude`
binary rather than failing on PATH resolution, and the graceful-failure path is unchanged.

---

## Task 4 — P1-4: workflow config `false` vs. unset

- [x] **4.1** Preserve evidence: a short Go snippet/test demonstrating today's bug — `Workflow{
  Triage: false, PlanReview: false, Verifier: false}` (all explicit) resolves through
  `EffectiveWorkflow()` to all-`true` on current `main`. Capture this as the "before" state in
  the task notes before changing anything.
- [x] **4.2** In `cli/internal/project/project.go`: change `Workflow.Triage`/`PlanReview`/
  `Verifier` from `bool` to `*bool` (mirroring `RequireSpecApproval`'s existing pattern and YAML
  tag style exactly). Add `TriageEnabled()`, `PlanReviewEnabled()`, `VerifierEnabled()` methods
  (nil → `true`, non-nil → literal). Delete `enabled()` and `EffectiveWorkflow()` — no longer
  needed once each field defaults independently.
- [x] **4.3** Update `cli/workflow_cmd.go`'s `gatherFacts`: replace
  `cfg.EffectiveWorkflow().PlanReview` with `cfg.Workflow.PlanReviewEnabled()`.
- [x] **4.4** Update `cli/internal/project/project_test.go`: replace
  `TestEffectiveWorkflowDefaultsAllTrue` with a test asserting all three accessors default `true`
  on a zero-value `Workflow{}`. Add: all-`true` explicit round-trips and resolves `true`;
  mixed `{true,false,true}` round-trips and each field resolves to its literal value; all-`false`
  explicit round-trips and **all three resolve to `false`** (the regression test for the bug
  itself).
- [x] **4.5** In `cli/doctor.go`: add a `Workflow:` section reporting each field as
  `enabled`/`disabled` plus `(explicit)`/`(default)`, and the resolved `planning_mode` — the
  exact shape shown in the instruction's "Project Config Validation" example.
- [x] **4.6** Run `go build ./... && go vet ./... && go test ./internal/project/... ./...`.
- [x] **4.7** Manually verify `eng doctor` against a fixture with an explicit
  `workflow: {triage: false, plan_review: false, verifier: false}` block — confirm the new
  output says `disabled (explicit)` for all three, not the old silent-default behavior.

**Verify:** the four required cases (omitted, all-true, mixed, all-false) each resolve exactly
as written, with a passing regression test per case; no existing `planning_mode`/legacy/hybrid/
modern test regresses.

---

## Task 5 — P2-1: `tasks.md` completion source of truth

- [x] **5.1** Preserve evidence: quoted the current confusing "Still in EXECUTING — tasks.md
  still has unchecked items" message with no indication of which line, from a real
  `eng workflow advance` run against a scratch plan where every per-task `**Status:**` was
  `[x]` but the bottom checklist was not.
- [x] **5.2** Edited `harness/templates/plan/tasks.md`'s header note to state explicitly: the
  per-task `**Status:**` marker is for human tracking; only the bottom **Completion checklist**
  gates `eng workflow advance`. No change to the checklist items themselves.
- [x] **5.3** In `cli/workflow_cmd.go`: added `uncheckedChecklistLines(planDir string) []string`
  (`tasksComplete` now delegates to it — same untrimmed prefix detection as before, byte-
  identical gating logic) and `printUncheckedChecklistLines(state, planDir string)`, called from
  both `workflowStatus` and `workflowAdvance`'s "Still in ..." branch when the state is
  `EXECUTING`/`NEEDS_FIX`.
- [x] **5.4** Added `cli/workflow_cmd_test.go` (new file) with
  `TestUncheckedChecklistLinesNamesTheBlockingLines`,
  `TestUncheckedChecklistLinesEmptyWhenAllChecked`,
  `TestTasksCompleteIgnoresPerTaskStatusMarker` (the explicit zero-behavior-change regression
  test), `TestTasksCompleteFalseWhenChecklistItemUnchecked`.
- [x] **5.5** `go build ./... && go test ./...` — all green. Manually re-ran the Task 5.1
  scenario against the exact same scratch plan: the new output names all 7 unchecked
  Completion-checklist lines individually.

**Verify:** the bottom checklist remains the only thing `tasksComplete` gates on (zero behavior
change for any existing plan); `eng workflow advance`'s message now names the specific unchecked
line(s) instead of a generic sentence.

---

## Task 6 — P2-3: global/local duplicate skill resolution

- [x] **6.1** Create `benchmarks/fixtures/legacy-duplicate-skill/` (a minimal project with
  `skills/karpathy-guidelines/SKILL.md` using the legacy `# Skill: karpathy-guidelines` heading
  convention, no frontmatter). Preserve evidence: `eng skills list` against this fixture (with
  `eng init`'d harness pointed at the real global skills) currently returns **two** entries for
  the same conceptual skill — confirm on current `main`.
- [x] **6.2** Add `cli/internal/skills/skills_test.go` cases (extend the existing file — check
  what's there first) for `ResolveWithPrivate`: a legacy local skill sharing a bare name with a
  qualified global skill collapses to one entry, sourced from `local`; two qualified skills in
  genuinely different domains sharing a bare name are **not** collapsed (regression guard for
  Decision 6); a private-tier legacy skill loses to a qualified local skill of the same bare
  name (tier precedence still `local > private > global` after collapsing).
- [x] **6.3** Implement the bare-name collision-collapse pass in
  `cli/internal/skills/skills.go`'s `ResolveWithPrivate`, per spec.md's fix strategy.
- [x] **6.4** Extend `cli/internal/skillvalidate/skillvalidate.go`'s `Validate` to report (as a
  `SeverityWarning` `Issue`) each raw discovered skill file whose `QualifiedName()` either isn't
  present in the final resolved set, or maps to a different file's `Path` — i.e. "shadowed by a
  higher-precedence source," covering both ordinary tier-precedence shadowing and the new
  collapse case uniformly.
- [x] **6.5** Run `go build ./... && go test ./internal/skills/... ./internal/skillvalidate/...
  ./internal/skillrouter/... -v`.
- [x] **6.6** Record `benchmarks/results/skill-dedup-legacy-global.yaml`: real
  `eng skills list`/`eng skills validate` output against the fixture, before (2 entries) and
  after (1 entry, sourced correctly) the fix.

**Verify:** exactly one `karpathy-guidelines` entry resolves against the fixture, sourced from
`local`; `automation/modbus` vs. a synthetic same-bare-name different-domain skill in the test
suite remain both present.

---

## Task 7 — New skill taxonomy: authoring

- [x] **7.1** Create `harness/skills/software/csharp-dotnet/SKILL.md` — frontmatter per spec.md's
  taxonomy table (`tags`/`triggers` covering `c#`, `.net`, `dotnet`, `.net framework`, `csproj`,
  `nuget`); body covering .NET Framework vs. modern .NET awareness, target-framework inspection,
  P/Invoke basics, x86/x64 compatibility, managed/unmanaged boundary, testing legacy desktop
  apps.
- [x] **7.2** Create `harness/skills/software/winforms/SKILL.md` — `requires:
  [software/csharp-dotnet]`; triggers covering `winforms`, `windows forms`, `.designer.cs`;
  body covering WinForms lifecycle, UI-thread rules (`Invoke`/`BeginInvoke`, cross-thread
  control access), designer-generated-code caution.
- [x] **7.3** Create `harness/skills/software/qt/SKILL.md` — `requires: [software/cpp]`;
  triggers covering `qt`, `qobject`, `qwidget`, `signal/slot`, `qthread`, `qml`; body covering
  QObject ownership, signals/slots, the event loop, Qt Widgets basics, QThread/thread affinity,
  model/view, Qt Network, UI responsiveness, AUTOMOC/AUTOUIC/AUTORCC.
- [x] **7.4** Create `harness/skills/software/cmake/SKILL.md` — triggers covering `cmake`,
  `cmakelists.txt`, `toolchain file`, `find_package`; body covering targets, include/link
  propagation, toolchain files, build types, cross compilation, dependency discovery (
  `find_package`), out-of-source builds.
- [x] **7.5** Create `harness/skills/engineering/protocol-engineering/SKILL.md` — `level:
  engineering`, `domain: engineering`; triggers covering `framing`, `checksum`, `crc`,
  `endianness`, `binary protocol`; body covering framing, partial reads, buffering, timeouts,
  CRC/checksum, endianness, retries, device state, transport-vs-protocol separation, hardware
  verification, capture/replay, protocol documentation practice.
- [x] **7.6** Create `harness/skills/software/serial-communications/SKILL.md` —
  `recommends: [engineering/protocol-engineering]`; triggers covering `rs485`, `rs232`,
  `usb-hid`, `serial port`, `com port`, `baud rate`; explicit `when_not_to_use` naming
  `automation/modbus` for actual Modbus addressing; body per spec.md Decision 8 — deliberately
  no Modbus-specific content.
- [x] **7.7** Create `harness/skills/embedded/embedded-linux/SKILL.md` — `level: domain`,
  `domain: embedded`; triggers covering `embedded linux`, `arm linux`, `aarch64`, `sysroot`,
  `systemd`; body covering ARM/aarch64 awareness, sysroot, target filesystem, shared-library
  dependency inspection (`ldd`/`readelf`), runtime loader paths, SSH/SCP deployment concepts,
  systemd service awareness, serial/GPIO permissions, resource constraints, remote debugging.
- [x] **7.8** Create `harness/skills/embedded/cross-compilation/SKILL.md` — `level: technology`,
  `domain: embedded`; triggers covering `cross compile`, `cross-compilation`, `toolchain file`,
  `target triple`; body covering CMake toolchain files, sysroot, cross-toolchain triples,
  common cross-compile pitfalls (host vs. target tool confusion).
- [x] **7.9** Create `harness/skills/engineering/reverse-engineering/SKILL.md` — `level:
  engineering`, `domain: engineering`; triggers covering `decompiled`, `reverse engineering`,
  `ghidra`, `disassembly`; body strongly emphasizing preserve-verified-behavior, don't
  blindly-clean-up-decompiled-output, evidence-based hypotheses, binary/source boundary
  awareness, known-good-behavior-as-source-of-truth, document uncertainty, small reversible
  changes; explicit note this is for legitimate interoperability/protocol-analysis/maintenance,
  not a malware-analysis framework.
- [x] **7.10** Run `eng skills validate` against the real `harness/skills` tree — zero new
  errors; warnings only where expected (legacy skills, if any touched — none should be).

**Verify:** 9 new `SKILL.md` files exist, each with valid frontmatter; `eng skills validate`
reports 0 new errors; `eng skills list` shows all 9 alongside the existing 11 (20 total).

---

## Task 8 — Routing fixtures and eval scenarios (Examples A/B/C)

- [x] **8.1** Created `benchmarks/fixtures/csharp-winforms-rfid/` (a real `.csproj`
  WinForms/net48 project + `Program.cs` — `eng init` correctly detects `stack: csharp`).
- [x] **8.2** Created `benchmarks/fixtures/qt-cmake-embedded-linux/` (`CMakeLists.txt` + a
  trivial `main.cpp` — `eng init` correctly detects `stack: c-cpp`).
- [x] **8.3** Added `skilleval.Scenario`'s optional `ForbiddenSkills []string` field and
  extended `TestRouterEvalScenarios` to also fail if any forbidden skill was selected.
- [x] **8.4** Added `harness/evals/software/csharp-winforms-rs485.yaml` — Example A.
- [x] **8.5** Added `harness/evals/embedded/qt-cmake-embedded-linux.yaml` — Example B
  (also gained `forbidden_skills: [esp32]` once Task 8.6/8.7 surfaced a real, additional
  defect — see below).
- [x] **8.6** Confirmed `harness/evals/embedded/esp32-siemens-modbus.yaml` (Example C) needs no
  content change — re-ran as part of 8.7, unchanged pass. While dogfooding Example B under the
  real default budget (not the eval's own no-cap mode), found a genuine additional P2-2-class
  defect: `embedded/esp32` carried a bare `embedded` tag that false-matched Example B's request
  and won a budget slot ahead of the genuinely-relevant `embedded/cross-compilation`. Fixed
  in-scope (single-line tag removal, DECISION_LOG.md Decision 11) since it directly blocked
  this phase's own Example B acceptance criterion.
- [x] **8.7** `go test ./... -run TestRouterEvalScenarios -v` — all 5 scenarios (3 pre-existing
  + 2 new) pass, including both forbidden-skill assertions.
- [x] **8.8** Recorded `benchmarks/scenarios/csharp-winforms-rs485.yaml` and
  `benchmarks/scenarios/qt-cmake-embedded-linux.yaml` plus
  `benchmarks/results/csharp-winforms-rs485-harness-v2.yaml` and
  `benchmarks/results/qt-cmake-embedded-linux-harness-v2.yaml` — the latter documents both the
  no-cap router-capability result and the real-default-budget result (where
  `cross-compilation` is honestly reported as budget-cut, not mis-routed — an expected
  consequence of a bounded budget for a 5-skill-relevant request, not a defect).

**Verify:** both new eval scenarios pass as part of the automated Go test suite (not just the
manual benchmark record); the benchmark result files quote real command output.

---

## Task 9 — Documentation updates

- [x] **9.1** `README.md`: update the Known Limitations section — remove the three Phase 8
  items now fixed (or mark them resolved with a pointer to this phase), keep only what's still
  genuinely a limitation after Phase 9 (see Task 10's honest accounting). Update the skill
  ecosystem mention to note the expanded taxonomy count.
  the multi-domain example if still accurate (Example C is unchanged, so no edit needed there).
- [x] **9.2** `docs/skills.md`: no structural change to the routing-precedence model (unchanged),
  but add a short note on the new bare-name collision-collapse rule under "Namespacing and
  collisions," and a one-line mention of the weighted-scoring change under "Routing precedence"
  (link to the actual `skillmatch.go` constants rather than restating numbers that could drift).
- [x] **9.3** `docs/USAGE.md`: update the `.agent/project.yaml` schema block's `workflow:`
  section to show the new `*bool` semantics (unset vs. explicit `true`/`false`) instead of the
  old ambiguous description; add a short troubleshooting note for the `eng start` PATH issue
  (what `ENG_HOME` is, how to fall back to `$ENG_HOME/bin/eng` if `eng` alone isn't found in a
  spawned shell); update the "tasks.md" mention to reflect the bottom-checklist-is-authoritative
  clarification.
- [x] **9.4** `harness/core/runtime/METHOD.md`: add one line noting that if `eng` isn't found on
  PATH inside a spawned shell, `$ENG_HOME/bin/eng` (or `%ENG_HOME%\bin\eng.exe` on Windows) is a
  reliable fallback, since `ENG_HOME` is now always set by `eng start`.
- [x] **9.5** `docs/gotchas.md`: add one new entry for the `eng start` child-PATH gotcha (trap /
  symptom / fix, per the file's existing format) — this is exactly the kind of real,
  non-obvious, silently-wrong-behavior defect the file exists to record.
- [x] **9.6** `ROADMAP.md`: move the Phase 9 backlog items out of "Now: Core Refinement" (since
  they're now done) into a brief "Recently completed" note, and restate what's next per the
  honest post-Phase-9 limitations.

**Verify:** no documentation duplication introduced (each fact stated once, cross-referenced
elsewhere); every command/field shown was checked against the actual Phase 9 code before being
documented.

---

## Task 10 — Full regression re-verification

- [x] **10.1** `cd cli && go build ./... && go vet ./... && go test ./...` — zero failures.
- [x] **10.2** Re-run the V1 regression script check (`./scripts/load_skill.sh list`,
  `./scripts/plan-executor.sh new smoke-test-phase9`, `./scripts/plan-executor.sh list | grep`,
  cleanup) — identical behavior to every prior phase.
- [x] **10.3** Re-run Quick Fix E2E, Spec-First E2E, Legacy E2E, and Hybrid-project behavior
  manually against a scratch fixture each (reusing Phase 8's proven command sequences) —
  confirm no regression from the P1-4/P2-1 changes specifically (workflow state machine
  behavior for a project with no `workflow:` block at all must be byte-for-byte unchanged).
- [x] **10.4** Re-run `go test ./... -run TestRouterEvalScenarios -v` one final time after all
  skill/matching changes are in.
- [x] **10.5** Re-run every Phase 8 benchmark scenario whose result could plausibly be affected
  by this phase's changes (`quick-fix-timeout-*`, `large-context-auth-validation-*`,
  `cross-domain-esp32-siemens-modbus-*`) and confirm each still matches its Phase 8 success
  criteria (or improves on it, per this phase's own acceptance criteria) — record any changed
  numbers in a short `benchmarks/results/*-phase9-reverify.yaml` note rather than silently
  overwriting the original Phase 8 result file.
- [x] **10.6** Confirm all ten Benchmark acceptance criteria from spec.md are met, with a
  specific result file or test name cited for each.

**Verify:** every command in this task's own bullet list was actually run in this session, with
real output, not asserted.

---

## Task 11 — Git

- [x] **11.1** Inspect the complete diff (`git status`, `git diff --stat`). Exclude any stray
  compiled binaries, logs, or scratch output the same way Phase 8's `.gitignore` discipline
  required. Confirm no local user paths, credentials, or private dogfooding source were copied
  into the repo.
- [x] **11.2** Stage only Phase 9's durable changes (code, tests, skills, fixtures, benchmark
  scenarios/results, docs, the plan folder itself) and commit locally with the suggested
  message. Do not push.

**Verify:** `git log -1` shows the new commit; `git status` shows a clean working tree; no
`git push` was run.
