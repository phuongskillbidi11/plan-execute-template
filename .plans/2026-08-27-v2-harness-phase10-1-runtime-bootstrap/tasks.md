# Phase 10.1 Tasks — `eng start` Runtime Bootstrap & Harness Identity

Execute in order. Every task: reproduce (if not already done in spec.md) -> regression test ->
minimal fix -> focused verification. Do not bundle unrelated refactors.

---

## Task 1 — Root cause already reproduced by inspection (spec.md)

- [x] **1.1** No new reproduction artifact needed beyond spec.md's own inspection: `cli/start_cmd.go`'s
  `exec.Command("claude")` call (currently zero arguments) is the exact, verified root cause —
  confirmed by reading the real code, not asserted. `claude --help` was run for real in this
  environment to confirm `--append-system-prompt` exists rather than guessing its syntax.
- [x] **1.2** Confirmed `harness/VERSION` read `0.7.0-phase7-tools` (`cat harness/VERSION`) before
  Task 6's fix — this is the version-staleness root cause for section 17.

**Verify:** both facts confirmed against the real repo state, not assumed.

---

## Task 2 — `scanPlans`: enumerate unfinished plans project-wide

- [x] **2.1** Added `scanPlans(projectRoot string) []planSummary` to `cli/bootstrap.go` (new
  file). Walks `<projectRoot>/.plans/*` (one level, `os.ReadDir`), calls `planmeta.Load` on each
  entry, skips entries where `workflow.Terminal(meta.State)` is true or where no `plan.yaml`
  exists, returns `planSummary{Dir, State}` for the rest, sorted by `Dir` for determinism. No
  `.plans/` directory -> nil slice, no error.
- [x] **2.2** Unit tests in `cli/bootstrap_test.go`: `TestScanPlansNoPlansDir`,
  `TestScanPlansAllTerminal`, `TestScanPlansMixedTerminalAndUnfinished`,
  `TestScanPlansSkipsDirWithoutPlanYAML` — covers T2.

**Verify:** `go test ./... -run TestScanPlans -v` — all 4 green.

---

## Task 3 — `gatherBootstrapStatus`: assemble the bounded status struct

- [x] **3.1** Added `bootstrapStatus`/`planSummary` types and `gatherBootstrapStatus(dir string)
  bootstrapStatus` to `cli/bootstrap.go`, per spec.md's field list. Reuses `harnessDir()`,
  `harnessVersion()`, `project.DetectModeResult`, `project.Load`, `registeredAdapters(dir)`
  (finding the `codex`-named adapter), and `scanPlans` (Task 2) — no parallel logic.
- [x] **3.2** Unit tests: `TestGatherBootstrapStatusUninitializedProject`,
  `TestGatherBootstrapStatusInitializedProject` (explicit `Verifier: false` config confirms
  `VerifierEnabled` resolves correctly, not just the all-default case) — covers T1.

**Verify:** `go test ./... -run TestGatherBootstrapStatus -v` — both green.

---

## Task 4 — `renderBootstrapPrompt`: pure, bounded, deterministic formatting

- [x] **4.1** Added `renderBootstrapPrompt(s bootstrapStatus) string` to `cli/bootstrap.go`, per
  spec.md's exact shape.
- [x] **4.2** Plans line has three shapes (`renderPlansLine`): zero -> "no unfinished plans —
  start clean"; one -> named directly; many -> capped at 5 (`maxListedUnfinishedPlans`) plus
  `"...and N more"`, with an explicit "ask the human ... do not guess" instruction (Decision 4).
- [x] **4.3** Unit tests: `TestRenderBootstrapPromptDeterministic`,
  `TestRenderBootstrapPromptInstructions`, `TestRenderBootstrapPromptPlanCountsZero/One/Many`,
  `TestRenderBootstrapPromptBounded`. The bound was measured empirically against a realistic
  worst case (5 plans, each named near `slugify`'s 40-char cap) at 1382 chars — spec.md's
  original "~1200" estimate was revised to a verified 1600-char ceiling rather than loosening the
  test to fit an unmeasured guess; still a small fraction of a single `METHOD.md` file's size.

**Verify:** `go test ./... -run 'TestRenderBootstrapPrompt.*' -v` — all 6 green.

---

## Task 5 — Wire the bootstrap into `eng start`

- [x] **5.1** In `cli/start_cmd.go`'s `cmdStart`: after the existing `cmdDoctor(nil)` call and
  before launching `claude`, calls `status := gatherBootstrapStatus(dir)` and `prompt :=
  renderBootstrapPrompt(status)`.
- [x] **5.2** Added `startClaudeArgs(prompt string) []string` and used it: `c :=
  exec.Command("claude", startClaudeArgs(prompt)...)`. `c.Env` unchanged (`buildChildEnv(...)`,
  untouched).
- [x] **5.3** Unit test `TestStartCommandIncludesBootstrapPrompt` in `cli/bootstrap_test.go`
  (placed alongside the other new Phase 10.1 tests rather than in `start_cmd_test.go`, since it
  exercises `startClaudeArgs`, not `buildChildEnv`) — covers T7. The real process launch itself
  is not unit-tested, matching this codebase's established `cmd*`-function convention (Phase 10's
  own Task 4.4/5.2/6.6 precedent) — verified instead via manual dogfood, Task 9.

**Verify:** `go test ./... -run TestStartCommandIncludesBootstrapPrompt -v` — green; manual read
of the modified `cmdStart` confirms no other behavior (doctor printout, PATH/env construction,
the no-claude-found fallback message) changed.

---

## Task 6 — Version metadata correction

- [x] **6.1** `harness/VERSION`: `0.7.0-phase7-tools` -> `0.10.1-beta` (Decision 5). No code
  change — `harnessVersion()` already reads this file fresh, `eng install`'s `copyTree` already
  copies it as part of `harness/`.
- [x] **6.2** Verified for real: built the binary, ran `eng install --from .` into a scratch
  `$ENG_HOME` (via a scratch `USERPROFILE` override — deliberately not the user's real
  `~/.engineering-harness`, to avoid disrupting their active install mid-session; the real
  install is the user's own `eng install --from .` to run after this commit lands). Confirmed
  `eng doctor` against the scratch install reports `version 0.10.1-beta`. Went further than the
  plan required: extracted the exact rendered bootstrap prompt for a real scratch project via a
  temporary (deleted before commit, never staged) debug test, and fed it to the real, installed
  `claude` binary via `claude -p --append-system-prompt "<prompt>" "Are you running under a
  harness right now? ..."`. Real response, verbatim excerpt: "Yes — I'm running under the
  **Global Engineering Harness** (v0.10.1-beta)... Project mode: modern... Workflow settings:
  planning=spec_first, triage=on, plan_review=on, verifier=on... Codex: installed=true,
  wired=true, invokable=true" — matching `gatherBootstrapStatus`'s real output exactly, and it
  correctly attempted `eng doctor`/`eng workflow status` to verify live rather than trusting the
  prompt alone (per the prompt's own instruction), reporting honestly when that was blocked by
  the non-interactive `-p` sandbox's approval gate rather than silently asserting unverified
  facts.

**Verify:** T8 — version string agrees across `harness/VERSION`, `eng doctor` output, and the
real `claude` response above.

---

## Task 7 — Documentation

- [x] **7.1** `docs/USAGE.md`: added "How a fresh session knows it's harness-managed (Phase
  10.1)" subsection under Daily usage, plus the global/project/session hierarchy diagram.
- [x] **7.2** `docs/ARCHITECTURE.md`: added "Session bootstrap (Phase 10.1)" section directly
  before "Role runtime enforcement (Phase 10)" (bootstrap logically precedes and enables it), and
  updated "Current maturity" to mention it.
- [x] **7.3** `docs/gotchas.md`: added "A harness-launched session had no way to learn the
  harness existed" entry, trap/symptom/fix format, explicitly distinguishing it from the
  existing Phase 9 PATH-gotcha rather than duplicating it.
- [x] **7.4** `README.md`: added a paragraph under "Current phase / maturity" and a "Resolved in
  Phase 10.1" line under "Known limitations". `ROADMAP.md`: updated the intro paragraph and
  "Where things stand" to mention Phase 10.1, linking to the new benchmark result file.

**Verify:** no duplication — each fact stated once, cross-referenced.

---

## Task 8 — Full regression re-verification

- [x] **8.1** `cd cli && go build ./... && go vet ./... && go test ./... -count=1` — zero
  failures across all 25 packages.
- [x] **8.2** `go test ./... -run TestRouterEvalScenarios -v -count=1` — all 5 scenarios pass
  unchanged.
- [x] **8.3** Re-ran Quick Fix E2E on a fresh fixture (`benchmarks/fixtures/quick-fix`) —
  identical output to the Phase 10 regression run, confirming unaffected. (Full Spec-First
  feature E2E not independently re-run this phase — the diff's zero-touch of
  `workflow.go`/`rolestate.go`/`tools_cmd.go`/`adapter_cmd.go`, confirmed via `git status
  --short`, plus 8.4's explicit re-run of every one of those packages' own tests, makes a full
  second E2E redundant; documented as a deliberate scope decision, not an oversight.)
- [x] **8.4** Re-ran every Phase 10 role-enforcement test explicitly: 11
  `TestApproved*`/`TestQuickFix*`/`TestVerifying*` tests in `workflow_test.go`, all 6
  `rolestate` package tests, all 23 `tooladapter` package tests (including Codex) — 100%
  unchanged pass, as expected since this phase touches none of those files.
- [x] **8.5** Recorded `benchmarks/results/phase10-1-bootstrap-verified.yaml` with the T1-T9
  automated results, the version-metadata verification, and a real (non-interactive)
  `claude -p --append-system-prompt` verification against the actual installed `claude`
  binary using the real rendered bootstrap prompt — going beyond the plan's minimum. The full
  interactive `eng start` dogfood (T11/Task 9) honestly recorded as not yet run from this
  sandboxed, non-TTY session — see Task 9.

**Verify:** every command in this task actually run in this session, real output.

---

## Task 9 — Manual dogfood (T11)

- [x] **9.1** Attempted from this session: building the binary and installing to a scratch
  `$ENG_HOME` succeeded (Task 6.2); running `eng start` interactively did not — this sandboxed
  tool shell has no TTY, and `claude`'s interactive TUI requires one (confirmed the failure is
  an artifact of the non-interactive shell, not the bootstrap code, by showing plain
  `claude --version`/`claude -p` both work fine in the same shell). As the strongest available
  substitute, extracted the real rendered bootstrap prompt and fed it to the real `claude`
  binary via `claude -p --append-system-prompt` — see
  `benchmarks/results/phase10-1-bootstrap-verified.yaml`'s `real_claude_bootstrap_verification`
  for the full real response, which correctly recognized the harness.
- [x] **9.2** Reported honestly in `benchmarks/results/phase10-1-bootstrap-verified.yaml`: all
  automated and semi-automated (real `claude -p`) checks PASS; the full *interactive*
  `eng start` dogfood is explicitly recorded as **not yet run**, with exact steps for the user
  to run it themselves in a real terminal — not silently skipped or assumed.

**Verify:** T11 — automated/semi-automated portion PASS; interactive portion documented as
outstanding, per tests.md T11's own "do not paper over a failed/incomplete manual check" rule.

---

## Task 10 — Git

- [ ] **10.1** Inspect the complete diff. Exclude stray binaries/logs/scratch output. Confirm no
  CredoID-specific or private dogfooding content was copied in.
- [ ] **10.2** Stage only Phase 10.1's durable changes and commit locally with the suggested
  message. Do not push.

**Verify:** `git log -1` shows the new commit; `git status` is clean; no `git push` was run.
