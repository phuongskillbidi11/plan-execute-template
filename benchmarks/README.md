# Benchmarks — Phase 8 dogfooding and validation

This directory answers one question: is the Global Engineering Harness (V2, Phases 1–7)
actually better in real usage than (a) a baseline Claude Code session with no harness, or
(b) the original V1 Plan-Execute Template — or is it only more complex?

It is this repository's own validation tooling. It is **not** part of `harness/`, so it is
never copied into a consumer project by `eng install`.

## The three comparison modes

- **`baseline`** — a fresh `general-purpose` Claude Code subagent, genuinely unaware of this
  conversation or the harness, given only a plain fixture directory and the raw request. It
  is told not to assume any particular workflow. **Single run per scenario — illustrative,
  not statistical.** LLM behavior is non-deterministic; do not read a `baseline` result as a
  reproducible measurement, only as one real, honest data point.
- **`v1`** — the real `scripts/plan-executor.sh` / `scripts/load_skill.sh`, run against a
  fixture with a real `.planner-executor/config.yaml`. Deterministic in its mechanics
  (scaffolding, task-completion counting); the spec/tasks/tests *content* is authored by
  whoever is acting as Planner, exactly as `CLAUDE.md` already prescribes.
- **`harness-v2`** — the real `eng` binary, built from this repository's current `cli/`, run
  against a fixture initialized with `eng init`, following
  `harness/core/runtime/METHOD.md`'s documented sequence. Fully deterministic in its
  mechanics (state transitions, skill/tool routing, verification verdicts).

## What can and cannot be measured

**Available and used:** workflow state (`plan.yaml`, `eng workflow status`), skill/doc
selection and reasons (`context-manifest.yaml`, `eng context skills`), tool routing and
reasons (`eng capabilities explain`), verification verdict (`verify-report.md`), tool
invocation audit (`events.jsonl`'s `tool_invocation` events), files actually changed
(`git diff --stat`), and a structural proxy for context size — **lines/bytes of the printed
context bundle, never a token count.**

**Not available, never invented:** real per-request token usage. Nothing in this stack has
access to the model runtime's actual tokenizer count. Every result file either omits
numeric context-size fields entirely or labels them explicitly as a structural proxy
(`context_bundle_lines`) — never as `tokens`.

## Directory layout

```
benchmarks/
├── README.md              this file
├── scenarios/              *.yaml — one per benchmark, request + expected/forbidden
├── fixtures/                static template projects, copied fresh before each run
├── results/                 one *.yaml per (scenario, mode) — real observed evidence
├── COMPARISON.md            V1-vs-V2 tables for the categories that ran more than one mode
├── CONTEXT_EFFICIENCY.md    Phase 4 "large knowledge base != large prompt" findings
├── SCORECARD.md             ten-dimension scorecard + the final decision gate
└── BACKLOG.md               real weaknesses found, P0-P3
```

Fixtures are never mutated in place — every run copies a fixture into a scratch directory
first (the same discipline every phase's own end-to-end tests already use in this
repository), so `benchmarks/fixtures/*` always stays a clean, reusable starting point.

## Scenario schema

```yaml
name: quick-fix-timeout
category: quick-fix
request: >
  Increase the reconnect timeout from 1000 ms to 1500 ms.
fixture: fixtures/quick-fix   # or "none" for scenarios that use this repo itself
modes: [baseline, v1, harness-v2]
expected:
  workflow: quick-fix
  max_files_changed: 1
  verification: pass
forbidden:
  - full_feature_plan
```

## Result schema

```yaml
scenario: quick-fix-timeout
mode: harness-v2              # baseline | v1 | harness-v2
workflow_selected: quick-fix
context:
  skills: 2
  docs: 1
  files: 1
  context_bundle_lines: 42    # structural proxy - never a token count
implementation:
  files_changed: 1
  unexpected_files_changed: 0
verification:
  verdict: PASS
human_interventions: 0
notes: >
  Free-text observation. Every number above must be traceable to a specific command's
  real output, referenced here.
```

## How to run each scenario

See each scenario's own section below. The commands shown are exactly what was run for Phase
8 — copy-paste reproducible against a fresh scratch copy of the named fixture. Every command
below assumes `$REPO` is this repository's root and `$SCRATCH` is a throwaway directory
outside it; every scenario starts with a fresh `cp -r`+`git init` of its fixture, never a
mutation of the committed fixture itself.

### quick-fix-timeout (`fixtures/quick-fix`)

```bash
# Mode C
cp -r "$REPO/benchmarks/fixtures/quick-fix/." "$SCRATCH/qf-c/" && cd "$SCRATCH/qf-c"
git init -q && git add -A && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" workflow start "Increase the reconnect timeout from 1000 ms to 1500 ms."
# observe: triage suggests "feature", not "quick-fix" — see finding in the result file.
# to exercise the Quick Fix mechanism itself:
"$REPO/cli/eng" plan new bump-timeout --risk quick-fix
# edit the constant, then:
"$REPO/cli/eng" workflow advance <plan-dir>   # repeat to COMPLETED

# Mode B
cp -r "$REPO/benchmarks/fixtures/quick-fix/." "$SCRATCH/qf-b/" && cd "$SCRATCH/qf-b"
git init -q && git add -A && git commit -q -m init
bash "$REPO/scripts/plan-executor.sh" new bump-timeout
# hand-author spec.md/tasks.md/tests.md as Planner, then edit main.go, mark tasks done
go build ./... && go run . | grep "1500 ms"

# Mode A
# dispatch one general-purpose Agent at a fresh scratch copy with only the raw request text
```

### feature-csv-export (`fixtures/feature`)

```bash
# Mode C
cp -r "$REPO/benchmarks/fixtures/feature/." "$SCRATCH/fx-c/" && cd "$SCRATCH/fx-c"
git init -q && git add -A && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" workflow start "Add an export command that prints all records as CSV (id,name) to stdout."
ls <plan-dir>   # confirm tasks.md/tests.md are still placeholder pre-approval
# write spec.md, then:
"$REPO/cli/eng" plan approve-spec <plan-dir> --by planner
# now write tasks.md/tests.md, advance through review/approval, implement ONLY once EXECUTING,
# eng verify, eng workflow advance to COMPLETED

# Mode B — same fixture, bash scripts/plan-executor.sh new csv-export, author all 3 files
#          up front (no separate spec-approval gate in V1), implement, test.
# Mode A — one general-purpose Agent dispatch, raw request text only.
```

### bug-lastn-off-by-one (`fixtures/bug`)

```bash
cp -r "$REPO/benchmarks/fixtures/bug/." "$SCRATCH/bug-c/" && cd "$SCRATCH/bug-c"
git init -q && git add -A && git commit -q -m init
go test ./...   # confirm it fails first
"$REPO/cli/eng" init
"$REPO/cli/eng" workflow start "LastN is returning the wrong elements — debug and fix it."
"$REPO/cli/eng" context skills "LastN is returning the wrong elements — debug and fix it."
# fix items[len(items)-n-1:] -> items[len(items)-n:], advance to EXECUTING first, then edit
"$REPO/cli/eng" verify <plan-dir>
```

### large-context-auth-validation (`fixtures/large-context`)

```bash
cp -r "$REPO/benchmarks/fixtures/large-context/." "$SCRATCH/lc-c/" && cd "$SCRATCH/lc-c"
git init -q && git add -A && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" context skills "add input validation to the auth token check so an empty token is rejected"
"$REPO/cli/eng" context project "add input validation to the auth token check so an empty token is rejected"
```

### cross-domain-esp32-siemens-modbus (this repository itself)

```bash
cd "$REPO"
"$REPO/cli/eng" context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"
```

### tool-routing-git-status (this repository itself)

```bash
cd "$REPO"
"$REPO/cli/eng" plan new bench-tool-routing --risk quick-fix   # scratch scaffold, or reuse an existing one
"$REPO/cli/eng" tools invoke executor git.status <plan-dir>
cat <plan-dir>/events.jsonl   # inspect the tool_invocation event's fields
```

### failure-* (5 sub-cases, `fixtures/quick-fix`)

```bash
# 10.1 reviewer REJECT
"$REPO/cli/eng" plan new r --risk architecture   # then write minimal spec/tasks/tests
"$REPO/cli/eng" plan review <plan-dir> --verdict REJECT
"$REPO/cli/eng" workflow advance <plan-dir>       # -> NEEDS_REPLAN

# 10.2 verifier FAIL
# advance a plan to EXECUTING, leave the fixture's test failing, then:
"$REPO/cli/eng" verify <plan-dir>                 # -> FAIL, no COMPLETED

# 10.3 drift after approval
# advance a plan cleanly to APPROVED, THEN edit a tracked file, then:
"$REPO/cli/eng" plan drift <plan-dir>             # -> PLAN_DRIFT_DETECTED
"$REPO/cli/eng" workflow advance <plan-dir>       # -> NEEDS_REPLAN

# 10.4 approval missing
"$REPO/cli/eng" plan new a --risk high-risk       # advance to REVIEWED
"$REPO/cli/eng" workflow advance <plan-dir>       # -> NEEDS_APPROVAL, repeat: stays there

# 10.5 tool capability denied
"$REPO/cli/eng" tools invoke planner git.push <plan-dir>   # -> REFUSED (DENIED), exit 1
```

### legacy-v1-compat (`fixtures/legacy`)

```bash
# Mode B
cp -r "$REPO/benchmarks/fixtures/legacy/." "$SCRATCH/lg-b/" && cd "$SCRATCH/lg-b"
cp -r "$REPO/scripts" ./scripts
git init -q && git add -A && git commit -q -m init
bash scripts/load_skill.sh list
bash scripts/plan-executor.sh new smoke-check

# Mode C
cp -r "$REPO/benchmarks/fixtures/legacy/." "$SCRATCH/lg-c/" && cd "$SCRATCH/lg-c"
git init -q && git add -A && git commit -q -m init
"$REPO/cli/eng" init            # auto-detects mode: hybrid, leaves CLAUDE.md/.plans/skills untouched
"$REPO/cli/eng" doctor
"$REPO/cli/eng" workflow start "Print a friendly greeting on startup."
```

## Adding a new scenario

1. Add `benchmarks/scenarios/<name>.yaml` (schema above).
2. Add a fixture under `benchmarks/fixtures/<category>/` if one doesn't already fit.
3. Run it for whichever modes are meaningful for that scenario's category (see
   `.plans/2026-08-26-v2-harness-phase8-benchmark/spec.md`'s scenario matrix for how each
   category's mode selection was justified — not every scenario needs all three).
4. Write `benchmarks/results/<name>-<mode>.yaml` from real, observed command output.
5. Append the exact commands you ran to this file's "How to run each scenario" section.
