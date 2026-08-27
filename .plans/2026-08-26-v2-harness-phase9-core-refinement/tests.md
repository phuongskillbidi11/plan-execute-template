# Phase 9 Acceptance Tests

Each item names the exact command(s) to run and the pass condition. All are real commands
against the real `eng` binary or `go test`, never asserted without being run.

## T1 — Quick Fix triage (P1-1)

```bash
cd cli && go build -o eng . && cd ..
./cli/eng triage "Increase the reconnect timeout from 1000 ms to 1500 ms."
```
**Pass:** `Suggested level: quick-fix`.

```bash
./cli/eng triage "Increase the session timeout for authenticated users from 30 to 60 minutes."
```
**Pass:** `Suggested level:` is `high-risk` (or another non-quick-fix elevated level) — never
`quick-fix`.

```bash
cd cli && go test ./... -run Triage -v
```
**Pass:** all cases pass, including the two above and every pre-existing keyword case.

## T2 — Skill/doc matching precision (P1-2 / P2-2)

```bash
cd cli && go test ./internal/skillmatch/... ./internal/docsearch/... ./internal/skillrouter/... -v
go test ./... -run TestRouterEvalScenarios -v
```
**Pass:** all green, including the pre-existing esp32-siemens-modbus/docker-linux-ci/cpp-debug
scenarios.

```bash
./cli/eng context skills "Maintain a C# WinForms UHF RFID configuration tool using RS485 and USB-HID binary protocol framing."
```
**Pass:** does NOT list `siemens-s7`, `plc`, or `modbus` in the selected set.

```bash
# against benchmarks/fixtures/large-context/, from a scratch copy
./cli/eng context project "add input validation to the auth token check so an empty token is rejected"
```
**Pass:** matched section count is materially below the Phase 8-recorded 4/6 (target 1–2/6).

## T3 — `eng start` child environment (P1-3)

```bash
cd cli && go test ./... -run BuildChildEnv -v
```
**Pass:** all cases pass (PATH prepended not replaced, case-insensitive detection, ENG_* set,
no duplication on re-application).

Manual: `eng start --init` in a scratch fixture with no `claude` on PATH prints the existing
"not found on PATH" message unchanged, proving the new `c.Env` wiring doesn't break the
no-agent path.

## T4 — Workflow config semantics (P1-4)

```bash
cd cli && go test ./internal/project/... -v
```
**Pass:** new cases for omitted/all-true/mixed/all-false workflow blocks each resolve exactly as
specified; all-false resolves to all three `Enabled()` accessors returning `false`.

```bash
./cli/eng doctor   # run against a fixture with an explicit all-false workflow: block
```
**Pass:** output shows `disabled (explicit)` for all three lines, not silently enabled.

## T5 — Task completion source of truth (P2-1)

Manual, against a scratch plan with every per-task `**Status:**` at `[x]` but the bottom
checklist still `- [ ]`:
```bash
./cli/eng workflow advance <plan-dir>
```
**Pass:** output names the specific unchecked checklist line(s), not just "tasks.md still has
unchecked items."

```bash
cd cli && go test ./... -run Workflow -v
```
**Pass:** new `uncheckedChecklistLines` test case passes; `tasksComplete`'s existing behavior is
unchanged (verified by the existing Phase 3/5 workflow tests still passing).

## T6 — Skill dedup (P2-3)

```bash
cd cli && go test ./internal/skills/... ./internal/skillvalidate/... -v
```
**Pass:** legacy-vs-qualified collapse case passes; genuinely-distinct-domain same-bare-name case
stays uncollapsed.

```bash
# against benchmarks/fixtures/legacy-duplicate-skill/
./cli/eng skills list
```
**Pass:** exactly one `karpathy-guidelines` entry, sourced `local`.

## T7 — New skill taxonomy

```bash
./cli/eng skills validate
```
**Pass:** 0 new errors introduced by the 9 new skill files.

```bash
./cli/eng skills list | grep -E "csharp-dotnet|winforms|qt|cmake|protocol-engineering|serial-communications|embedded-linux|cross-compilation|reverse-engineering"
```
**Pass:** all 9 new skills listed.

## T8 — Example A/B/C routing

```bash
cd cli && go test ./... -run TestRouterEvalScenarios -v
```
**Pass:** `csharp-winforms-rs485` scenario passes (expected skills present, forbidden skills
absent); `qt-cmake-embedded-linux` scenario passes; `esp32-siemens-modbus` still passes
unchanged.

## T9 — Full regression

```bash
cd cli && go build ./... && go vet ./... && go test ./...
```
**Pass:** zero failures.

```bash
./scripts/load_skill.sh list
./scripts/plan-executor.sh new smoke-test-phase9
./scripts/plan-executor.sh list | grep smoke-test-phase9
```
**Pass:** identical behavior to every prior phase's own run of this check (then clean up the
scaffolded plan).

## T10 — Benchmark acceptance criteria (spec.md's 10 items)

Each of spec.md's ten numbered acceptance criteria is satisfied by one of T1–T8 above, or by a
specific `benchmarks/results/*.yaml` file cited in the final Phase 9 report. No criterion is
marked satisfied without a specific test name or result file backing it.
