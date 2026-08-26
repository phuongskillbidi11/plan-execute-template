# Phase 8 — Dogfooding, Benchmarking & Validation

## Goal

Phases 1–7 built a Global Engineering Harness. Phase 8 does not add capability — it answers
whether the thing built so far is actually better than (a) a baseline Claude Code session
with no harness and (b) the original V1 Plan-Execute Template, or whether it is only more
complex. The output is evidence — durable, reviewable result artifacts and a scorecard —
plus one explicit decision: `READY_TO_EXPAND` or `CORE_REFINEMENT_REQUIRED`.

## What already exists — the observability this phase reuses, not rebuilds

Every metric this phase needs already has a real, existing producer. Nothing new is
instrumented; Phase 8 only reads what Phases 1–7 already write:

| Signal | Existing producer |
|---|---|
| Workflow state selected | `plan.yaml` (`state`, `risk_level`), read via `eng workflow status` |
| Skills/docs selected + reasons | `<plan-dir>/context-manifest.yaml`, written by `eng context bundle`/`eng adapter prompt` (Phase 4/6) |
| Tool routing + reasons | `eng capabilities explain` (Phase 7) |
| Verification verdict | `<plan-dir>/verify-report.md`, `plan.yaml`'s `verification.verdict` (Phase 2) |
| Tool invocation audit | `<plan-dir>/events.jsonl`'s `tool_invocation` events (Phase 7) |
| Quick Fix completion | `events.jsonl`'s `quick_fix` event (Phase 5) |
| Full lifecycle history | `events.jsonl` (Phase 3/5) |
| Files actually changed | `git diff --stat` against the plan's stamped `git_sha` (the same check `eng verify` already runs) |
| Context bundle size | byte/line count of the text `eng context bundle`/`eng adapter prompt` prints — a structural proxy, not a token count |

V1's own observability is much thinner by design: `scripts/plan-executor.sh list/status`
count `- [ ]`/`- [x]` lines in `tasks.md`; there is no context routing, no skill routing, no
tool routing, no machine-readable verdict beyond whatever the human/Executor reports. This
asymmetry is itself a finding, not a flaw in the benchmark — V1 was never designed to
produce this telemetry.

## What cannot be measured reliably

**Token usage.** Nothing in this stack — `eng`, the CLI harness, or this conversation —
has access to a real per-request token count from the model runtime. Phase 4/5's own
`contextcfg`/log-compaction work already established the rule "do not hard-code token
counts to a single model" for the identical reason. Phase 8 follows the instruction's own
explicit rule: never invent a number. Every result file either omits `tokens:` entirely or
uses a labeled structural proxy (`context_bundle_lines`, `context_bundle_bytes`) that is
never called a token count.

**True A/B/A′ statistical comparison.** A baseline "Claude Code without the harness" run is
a single LLM episode — non-deterministic by nature, and this phase can only afford one run
per scenario, not a distribution. Baseline (Mode A) results are therefore treated as
**illustrative structural evidence** (what files did it touch, did it plan first, did it
self-verify), never as a statistically powered comparison. This is stated explicitly next
to every Mode A result, not buried in a footnote.

## Comparison modes

- **Mode A — Baseline Agent.** A fresh `general-purpose` subagent (via the `Agent` tool,
  genuinely no memory of this conversation or the harness) is pointed at a plain fixture
  directory and given only the raw natural-language request — the same request a Mode C
  run gets. It is told explicitly not to assume any particular workflow. Its resulting git
  diff, and whether it produced any planning artifact unprompted, are the evidence.
- **Mode B — V1 Plan-Execute.** The real `scripts/plan-executor.sh` and
  `scripts/load_skill.sh`, run against a fixture with a real `.planner-executor/config.yaml`
  — this session acts as Planner (writing `spec.md`/`tasks.md`/`tests.md` by hand, exactly
  as `CLAUDE.md`'s Planner role already prescribes) and then as Executor (implementing from
  `tasks.md`), mirroring the documented V1 workflow precisely rather than a caricature of it.
- **Mode C — V2 Global Engineering Harness.** The real `eng` binary built from this
  repository's current `cli/`, run against a fixture initialized with `eng init`, following
  `harness/core/runtime/METHOD.md`'s documented sequence exactly as any Claude Code session
  using the harness would.

## Scenario matrix (bounded, per Requirement 1's "keep Phase 8 lightweight")

| # | Category | Fixture | Modes run | What it validates |
|---|---|---|---|---|
| 1 | Quick Fix | `benchmarks/fixtures/quick-fix/` | A, B, C | Phase 5 Quick Fix cheapness |
| 2 | Normal Feature | `benchmarks/fixtures/feature/` | A, B, C | Phase 5 Spec-First + approval gate |
| 3 | Bug Fix | `benchmarks/fixtures/bug/` | C (A/B discussed qualitatively) | debugging skill selection + Verifier |
| 4 | Large Context | `benchmarks/fixtures/large-context/` | C only | Phase 4 context efficiency |
| 5 | Cross-Domain | this repo's own `harness/skills/` tree | C only | Phase 6 skill routing on real shipped skills |
| 6 | Tool-Routing | this repo (a real git repo) | C only | Phase 7 capability routing + audit |
| 7 | Failure/Safety | `benchmarks/fixtures/quick-fix/` (reused) | C only, 5 sub-cases | deterministic state-machine safety |
| 8 | Legacy | `benchmarks/fixtures/legacy/` | B, C | no V1 regression under the harness |

Categories 3–7 are deterministic, mechanical, and cheap to run for real — every mode run
for them is Mode C (or B for legacy), so there is no reason to spend a non-deterministic
Mode A run on them; the two categories where "how much workflow overhead does a human
actually feel" is the entire question (Quick Fix, Feature) get the full three-mode
treatment, per Requirement 10/11's own framing of those two as "one of the most important
user-facing validations."

## Success criteria — defined now, not after seeing results

**Quick Fix (Mode C):** no `spec.md`/`tasks.md`/`tests.md` beyond the minimal quick-fix
template; state reaches `EXECUTING` directly from `TRIAGED` (never `PLANNED`/`REVIEWED`/
`APPROVED`); exactly one file changed; verification `PASS`; a `quick_fix` event recorded.
Fail if any full-feature artifact appears, or state passes through `PLANNED`/`REVIEWED`.

**Feature / Spec-First (Mode C):** state reaches `NEEDS_SPEC_APPROVAL` before `tasks.md`/
`tests.md` exist; `eng plan approve-spec` is required before `SPEC_APPROVED`; `tasks.md`/
`tests.md` are not written until after that approval; the plan reaches `PLANNED` and
completes review/execute/verify. Fail if `tasks.md` is ever present before spec approval.

**Context efficiency (Categories 4/5, Mode C):** the context bundle names only skills/docs
relevant to the request; for the large-context fixture, the bundle's file/skill/doc count is
a small fraction of the fixture's total file count; for the cross-domain scenario, the
selected skill set matches (at minimum) the skills the instruction's own headline example
names. Fail if an unrelated package/skill is pulled in with no `recommends`/`requires`
justification recorded in the routing explanation.

**Tool routing (Category 6, Mode C):** a read-only request routes to `ALLOWED` and produces
exactly one `tool_invocation` audit event with a `log_path`, never raw output inline in
`events.jsonl`. Fail if the event contains more than the documented compact field set.

**Failure/safety (Category 7, Mode C):** Reviewer `REJECT` → `NEEDS_REPLAN`; Verifier `FAIL`
→ `NEEDS_FIX` (or `FAILED` once retries are exhausted), never `COMPLETED`; drift after
approval → `NEEDS_REPLAN`; a `NEEDS_APPROVAL` capability is never actually invoked before
approval (no adapter output, only a refusal audit event); a hard-denied capability is denied
regardless of role/policy. Fail if any case reaches a state or invocation it shouldn't.

**Legacy (Category 8, Mode B+C):** the V1 fixture's `plan-executor.sh`/`load_skill.sh`
behavior is byte-for-byte identical to a pre-Phase-1 run (already proven every phase this
session — Phase 8 just captures it as a durable artifact); the same fixture, given
`.agent/project.yaml` in `hybrid` mode, keeps behaving as `auto_plan` under `eng` with zero
migration required.

**Overall (qualitative, Mode A vs C on Quick Fix/Feature):** record — do not pre-judge —
whether Mode A produces more, fewer, or the same files changed; whether it self-verifies;
whether it invents a plan unprompted; whether Mode C's extra steps (triage, approval gate)
correspond to a real accuracy/safety benefit or only added friction. This is the one place
the plan does not fix a pass/fail line in advance, because it's the actual open question
the phase exists to answer.

## Fixture strategy and placement

A new top-level `benchmarks/` directory, a sibling of `cli/`, `harness/`, `docs/`,
`scripts/` — **not** under `harness/`, since `eng install` copies `harness/` wholesale into
every user's `~/.engineering-harness/` (confirmed in `cli/install.go`); benchmark fixtures
are this repository's own validation tooling and must never ship to a consumer project.
Not under `.plans/`, since that directory is live plan-lifecycle state, not a fixture
library. Fixtures are committed as static templates under `benchmarks/fixtures/<category>/`;
the documented run procedure always copies a fixture into a scratch directory before running
any mode against it, so running a benchmark never mutates the committed fixture (the same
discipline already used by every phase's own E2E tests in this session).

## Result format

One YAML file per `(scenario, mode)` under `benchmarks/results/`:

```yaml
scenario: quick-fix-timeout
mode: harness-v2            # baseline | v1 | harness-v2
workflow_selected: quick-fix
context:
  skills: 2
  docs: 1
  files: 1
  context_bundle_lines: 42   # structural proxy — never a token count
implementation:
  files_changed: 1
  unexpected_files_changed: 0
verification:
  verdict: PASS
human_interventions: 0
notes: >
  Free-text observation, always grounded in a specific command's real output.
```

No `tokens:` field appears anywhere in Phase 8's committed results. Every numeric field in a
committed result must be traceable to a specific command's real, observed output — recorded
in the scenario's own run notes, not asserted from memory.

## `eng benchmark` CLI — not built

Considered and rejected: eight scenarios, most single-run, do not justify a generic runner
engine. A documented, copy-pasteable command sequence per scenario (in
`benchmarks/README.md`, the exact style every phase's own `tests.md` already uses in this
repository) is simpler, more auditable, and adds zero new Go code to the production
`cli/` — directly satisfying "benchmark tooling must not regress production harness
behavior" by construction, not by testing. If a future phase needs to re-run this suite
routinely (e.g., in CI), promoting the documented sequence into a real `eng benchmark run`
command is a small, well-scoped follow-up with a known specification already written down.

## Out of scope

- Cloud telemetry, a benchmark SaaS, a database, distributed runners, an LLM evaluation
  platform, remote agents, a vector DB, a public leaderboard — explicitly excluded by the
  instruction.
- Any new external adapter, PLC/Modbus/OPC UA capability, or write-capable tool added
  purely to make a benchmark scenario more interesting — Category 6 uses only the already-
  shipped read-only `git.status`/`docs.search` capabilities.
- Statistically powered Mode A comparisons (multiple runs per scenario, variance analysis)
  — a single illustrative run per scenario is the bound this phase accepts, stated
  explicitly wherever Mode A results are reported.
- Any change to core harness behavior to make a benchmark pass — if a scenario reveals a
  real defect, it is recorded, severity-classified, and either fixed with a narrow, plan-
  scoped correction (only if bounded and directly implicated) or added to the refinement
  backlog, never silently patched to flatter the scorecard.
- A permanent `benchmarks/` CI job or scheduled re-run mechanism — this phase produces one
  dated snapshot of evidence, not an ongoing measurement system.
