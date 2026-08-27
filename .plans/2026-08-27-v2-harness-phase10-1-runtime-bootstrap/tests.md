# Phase 10.1 Acceptance Tests

## T1 — `gatherBootstrapStatus` matches `eng doctor` for the same directory

```bash
cd cli && go test ./... -run TestGatherBootstrapStatus -v
```
**Pass:** a temp project with a known `.agent/project.yaml` (mode, planning_mode, workflow
bools) produces a `bootstrapStatus` whose fields equal what `project.Load`/`registeredAdapters`
independently report for the same dir.

## T2 — `scanPlans` terminal-state filtering

```bash
cd cli && go test ./... -run TestScanPlans -v
```
**Pass:** no `.plans/` dir -> empty slice, no error. All-terminal plans -> empty slice. A mix of
terminal and non-terminal plans -> only the non-terminal ones returned, correct `Dir`/`State`.

## T3 — `renderBootstrapPrompt` determinism and required content

```bash
cd cli && go test ./... -run TestRenderBootstrapPrompt -v
```
**Pass:** same `bootstrapStatus` in -> byte-identical output across repeated calls. Output
contains the harness home, version, project mode, and Codex installed/wired/invokable values
from the input struct.

## T4 — `renderBootstrapPrompt` verification/CLAUDE.md/no-auto-resume instructions present

```bash
cd cli && go test ./... -run TestRenderBootstrapPromptInstructions -v
```
**Pass:** output contains an instruction to verify through `eng`, a statement that a missing
project-local CLAUDE.md/.claude does not imply harness absence, and an instruction not to
auto-resume a COMPLETED plan.

## T5 — Unfinished-plan reporting: zero / one / many

```bash
cd cli && go test ./... -run TestRenderBootstrapPromptPlanCounts -v
```
**Pass:** zero unfinished -> "no unfinished plans" (or equivalent unambiguous phrase). Exactly
one -> names that plan directly. Six unfinished -> lists at most 5 by name plus an "...and 1
more" (or equivalent) suffix, and states not to guess which to resume.

## T6 — Bounded prompt size

```bash
cd cli && go test ./... -run TestRenderBootstrapPromptBounded -v
```
**Pass:** rendered prompt for a status with 5 listed unfinished plans (near-max-length names)
stays under the fixed character ceiling defined in spec.md (1600 chars) — a hard `len()`
assertion, not manual eyeballing.

## T7 — `cmdStart` passes `--append-system-prompt` to the launched `claude` process

```bash
cd cli && go test ./... -run TestStartCommandIncludesBootstrapPrompt -v
```
**Pass:** the constructed command args (via the same test-seam pattern used for `buildChildEnv`
— a pure function returning the args slice, not a real process launch) include
`--append-system-prompt` followed by the rendered prompt string.

## T8 — Version metadata agreement

```bash
cd cli && go build -o /tmp/eng.exe . && /tmp/eng.exe doctor 2>&1 | grep "0.10.1-beta"
cat ../harness/VERSION
```
**Pass:** `harness/VERSION` reads `0.10.1-beta`; a fresh `eng doctor` run against a harness
installed from this checkout reports the same string; `ENG_VERSION` in a launched child's env
(via `buildChildEnv`, already tested) carries the same value end-to-end.

## T9 — Full regression (no Phase 10 enforcement regressed)

```bash
cd cli && go build ./... && go vet ./... && go test ./... -count=1
go test ./... -run TestRouterEvalScenarios -v -count=1
```
**Pass:** zero failures across all packages, including every Phase 10 role-enforcement test
unmodified.

## T10 — Regression re-runs: Quick Fix E2E, Spec-First E2E, Legacy/Hybrid E2E

**Pass:** each reaches its Phase 10-proven terminal state via a fresh fixture copy, with no
behavior change beyond `eng start` itself now passing `--append-system-prompt` — none of these
E2E sequences invoke `eng start` directly (they call `eng workflow`/`eng adapter`/`eng plan`
directly), so this is confirming no regression was introduced elsewhere, not re-proving
`eng start` itself.

## T11 — Manual dogfood (honestly reported, not a substitute for T1-T9)

Steps, run for real after `go build` + `eng install --from .` (fresh install of this checkout):
1. `eng doctor` in a harness-initialized project — confirm version, mode, workflow, Codex line.
2. `eng start` — confirm it launches `claude`.
3. In the fresh Claude Code session, ask "are you running under a harness?" (or let it state so
   unprompted) — record verbatim what it says.

**Pass:** the session states it is running under the Global Engineering Harness, cites the
correct project mode/workflow/Codex state, and does not claim Phase 10 or role enforcement is
fictional — without having been told any of this in the chat text.
**Fail:** report exactly what the session said, and whether it's a prompt-content gap
(fixable by refining `renderBootstrapPrompt`) or an environment-plumbing gap (fixable in
`cmdStart`/`buildChildEnv`) — do not paper over a failed manual check.
