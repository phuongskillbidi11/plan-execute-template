# Global Engineering Harness

> Originated from `plan-execute-template`, a per-project Planner/Executor workflow. This repo
> now *is* that harness, installed once and linked into any project — the original template
> still works unmodified inside it (see [Legacy compatibility](#legacy-compatibility)).

![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Works with Claude Code](https://img.shields.io/badge/Claude-Code-orange)
![Language agnostic](https://img.shields.io/badge/Language-Agnostic-lightgrey)
![Status: internal dogfooding](https://img.shields.io/badge/Status-internal%20dogfooding-yellow)

Install once, initialize any project once, then use a natural-language-first engineering
workflow with planning, review, execution, verification, context routing, multi-domain skills,
and tool permission gates.

Written for developers across Software, Backend/Frontend/Desktop, Embedded, Automation/PLC/
SCADA, IT/Networking, DevOps, Security, and Electronics — nothing here assumes any one domain.

---

## What this project is

```text
V1 (the original template):
  clone/copy the whole template into every project repository

V2 (this repo, current):
  install the harness once, globally  →  eng install
  +
  link a thin per-project config      →  eng init  (writes only .agent/project.yaml)
```

`eng install` copies the harness payload — methodology, skills, workflow templates, hooks —
into `~/.engineering-harness/`. A project never gets its own copy of that tree. `eng init`
writes exactly one file. Everything else (skills, context routing, workflow state, tool
permissions) is resolved at run time from the global install plus your thin project config.

## Why it exists

The V1 template worked, but cloning it into every repository meant:

- repeated cloning/copying, and drift between each project's copy
- duplicated skills and workflow files with no shared source of truth
- noisy project git diffs from template files living inside the project itself
- no separation strong enough to prevent Planner/Executor drift on its own
- context bloat — every session re-reading everything instead of what's relevant
- no real review/verification gate — a verdict was a file, not something that blocked progress
- poor reuse across domains (a `web-ui` skill didn't help an embedded project, and vice versa)
- no control over what an AI session could actually touch outside the conversation

Phases 1–7 built a harness that keeps V1's Planner/Executor separation and Karpathy principles,
and adds: a deterministic lifecycle state machine, selective context routing, a multi-domain
skill router, and an audited tool/capability permission layer.

## Architecture overview

```text
Developer
   ↓
eng start
   ↓
Runtime (harness/core/runtime/METHOD.md)
   ↓
Triage
   ↓
Context Manager
   ↓
Skill Router
   ↓
Workflow
   ├─ Quick Fix
   └─ Spec-First Feature
   ↓
Planner → Plan Reviewer → Executor → Verifier
   ↓
Capability / Tool Router
   ↓
Adapters (git, GitHub read-only, reference MCP)
```

```text
Skill            = methodology / knowledge
Agent Adapter    = integration with a coding agent (Claude Code, ...)
Tool Adapter     = integration with an external capability (git, GitHub, an MCP server, ...)
Harness          = orchestration, context, state, policy, routing
```

Full diagram, request-flow walkthrough, context-engineering principles, and the security model
live in **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)**.

## Current phase / maturity

**Internal dogfooding / team-usable beta.** Not production-hardened. Phase 8 ran a real,
evidence-based benchmark suite (`benchmarks/`) comparing this harness against a baseline
unharnessed agent and against the original V1 template across 8 categories, scoring 7 of 10
scorecard dimensions Strong, 3 Adequate, 0 Weak, 0 blocking (P0) issues, and surfacing three
real refinement items. **Phase 9 fixed all three**, plus two more real defects found during
subsequent real-world dogfooding (a skill-routing false positive on non-PLC serial/RS485
projects, and a global/local duplicate-skill resolution bug) — see
[`benchmarks/SCORECARD.md`](benchmarks/SCORECARD.md) for the Phase 8 baseline and
[`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) for what Phase 9 closed vs. what's still open.
Phase 9 also added 9 new skills (C#/.NET desktop, Qt/CMake, serial/protocol engineering,
embedded Linux, reverse engineering) for the real project types the harness is now used on —
see [Skill authoring](#skill-authoring) below.

Real-world dogfooding after Phase 9 found a genuine architectural gap: the Planner → Plan
Reviewer → Executor → Verifier lifecycle was a documented convention, not something the code
actually enforced — a plan could reach `COMPLETED` with a PASS verdict and an **empty git
diff**, because the workflow state machine had no way to tell "completed during execution"
from "already marked complete before execution began." **Phase 10 fixed this** — role
activation is now a validated, recorded step, and the state machine provably refuses to
retroactively legitimize work that happened outside it. See
[`docs/ARCHITECTURE.md#role-runtime-enforcement-phase-10`](docs/ARCHITECTURE.md#role-runtime-enforcement-phase-10)
for the design and [`benchmarks/results/investigation-bypass-blocked.yaml`](benchmarks/results/investigation-bypass-blocked.yaml)
for the real reproduced-and-blocked proof.

Dogfooding immediately after Phase 10 found the next layer of the same problem: a fresh,
harness-launched Claude Code session had no way to *learn* any of the above — it checked only
project-local `.claude/`/`CLAUDE.md`, found nothing, and correctly (given what it actually had)
concluded no harness existed. **Phase 10.1 fixed this** — `eng start` now launches the session
with a small, trusted, machine-generated identity block appended to its system prompt. See
[`docs/ARCHITECTURE.md#session-bootstrap-phase-101`](docs/ARCHITECTURE.md#session-bootstrap-phase-101).

See [Known limitations](#known-limitations) for what's still genuinely open, and
[`benchmarks/README.md`](benchmarks/README.md) for how to reproduce any of this yourself.

---

## Installation

```bash
cd cli
go build -o eng .              # ./eng.exe on Windows
./eng install --from .         # copies harness/ to ~/.engineering-harness/, plus the eng binary
```

Then add the printed bin directory to PATH (or pass `--add-to-path` to do it automatically):

```bash
# Linux / macOS
export PATH="$HOME/.engineering-harness/bin:$PATH"

# Windows
setx PATH "%PATH%;C:\Users\<you>\.engineering-harness\bin"
```

Full per-platform detail (including what `--add-to-path` does on each OS) is in
[`docs/USAGE.md#installation`](docs/USAGE.md#installation).

## First project setup

```bash
cd my-project
eng init
eng doctor
```

`eng init` creates `.agent/project.yaml` only:

```text
my-project/
├── src/
├── tests/
├── .agent/
│   └── project.yaml
└── .plans/            ← appears once you scaffold your first plan
```

The full skill/workflow library is **not** copied into your project — it's resolved from the
global install. See [`docs/USAGE.md#first-project-setup`](docs/USAGE.md#first-project-setup).

## Daily usage

```bash
cd my-project
eng start
```

Then just describe what you want, in plain language:

```text
Add CSV export to locker history.
```

That session follows the documented runtime routing sequence automatically — triage, context
selection, skill routing, workflow state transitions, approval gates — and stops to ask you at
every point that needs a human decision. **You should not normally need to run** `eng plan
new`, `eng context bundle`, or `eng workflow advance` yourself; those are low-level/debug
commands the session uses on your behalf. See
[Normal user vs. advanced user](#normal-user-vs-advanced-user).

## Quick Fix vs. Spec-First Feature

```text
Quick Fix:
  small, localized, low-risk request → edit → eng verify → compact event recorded
  (skips PLANNED/REVIEWED/APPROVED entirely)

Spec-First Feature (the default for any eng init project):
  request → draft spec.md → STOP for approval → write tasks.md/tests.md
  → review → (execution approval, if risk requires it) → execute → verify
```

A one-line change can still be high-risk (a triage keyword match doesn't override your own
judgment — scaffold explicitly with `--risk high-risk` when you know better). A Quick Fix that
turns out broader than expected escalates via `eng plan escalate` rather than continuing under
the wrong workflow. Full detail, including the exact command sequence for each path and the
current triage-misclassification limitation, is in
[`docs/USAGE.md#quick-fix-workflow`](docs/USAGE.md#quick-fix-workflow) and
[`docs/USAGE.md#spec-first-feature-workflow`](docs/USAGE.md#spec-first-feature-workflow).

## How it all fits together

- **Context** — selective skill/doc loading keeps large knowledge bases from becoming large
  prompts. See [`docs/ARCHITECTURE.md#context-engineering`](docs/ARCHITECTURE.md#context-engineering).
- **Skills** — three levels (`engineering`/`domain`/`technology`), dependency-aware, routed by
  request text + project domain profile. See [`docs/skills.md`](docs/skills.md).
- **Workflow** — a deterministic state machine (`Facts → Decide → Decision`) drives every plan
  from `TRIAGED` to `COMPLETED`. See [`docs/USAGE.md#workflow-states`](docs/USAGE.md#workflow-states).
- **Tools** — every external capability (`git.status`, `github.pr.read`, ...) is risk-classified,
  role-permission-checked, and audited before it runs. See [`docs/tools.md`](docs/tools.md).

### Multi-domain example

Request: `ESP32 reads Siemens S7-1200 over Modbus TCP`

```text
eng context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"

Selected: karpathy-guidelines (explicit), esp32, siemens-s7, modbus, tcp-ip (matched),
          plc (pulled in only via siemens-s7's required dependency — not a text match)
```

This is the exact scenario `harness/evals/embedded/esp32-siemens-modbus.yaml` asserts and
Phase 8's benchmark reproduced against the real router — no unrelated-domain skill
(`software/cpp`, `devops/docker`, `it/linux`, ...) was selected for this request.

## Tool / MCP runtime

Every external capability is `<adapter>.<operation>` (e.g. `git.status`), risk-classified
(`READ < WRITE < DESTRUCTIVE < HIGH_RISK`), checked against both a role's toolbox and its risk
ceiling, then checked against project policy (`deny`/`require_approval`/`allow`) — in that
fixed order. Since Phase 10, the invoking role must also actually be the plan's currently
*activated* role (`eng adapter prompt <role> <plan-dir>` records this) before any of that even
runs. `eng tools invoke` is the only sanctioned invocation path; every outcome, allowed or
refused, is written to an audit trail.

**Real adapters today:** `git`, `github` (read-only, via the `gh` CLI), `codex` (read-only —
`codex.inspect`/`codex.review`/`codex.verify`, a second-opinion delegation adapter, no write
capability), and a deterministic reference MCP adapter (`docs-search`) that greps local docs —
no live network transport. **Not implemented:** a full MCP JSON-RPC transport, a tool
installer/marketplace, or any live industrial-control write adapter (PLC/Modbus/OPC UA). Full
model in [`docs/tools.md`](docs/tools.md) and
[`docs/ARCHITECTURE.md#tool--capability-model-phase-7`](docs/ARCHITECTURE.md#tool--capability-model-phase-7).

## Important `eng` commands

The five you'll actually type:

| Command | Purpose |
|---|---|
| `eng install --from <path> [--add-to-path]` | One-time: install the harness payload |
| `eng init` | One-time per project: create `.agent/project.yaml` |
| `eng doctor` | Check install/project/skill/tool status |
| `eng start` | Doctor, then launch your agent — describe requests in plain language from here |
| `eng workflow status [dir]` | See a plan's state when the session stops and asks |

Full reference (every `eng` command, its exact syntax, an example, and whether normal users
need it) is in **[`docs/USAGE.md#command-reference`](docs/USAGE.md#command-reference)**.

## Normal user vs. advanced user

**Normal developer:** `eng init`, `eng doctor`, `eng start`, then natural language.

**Advanced / debug / CI:** `eng plan`, `eng workflow`, `eng context`, `eng tools`,
`eng capabilities`, `eng hooks` — scripting CI checks, debugging why a skill wasn't selected,
inspecting exactly what a role would see, or manually driving the state machine. See
[`docs/USAGE.md#normal-vs-advanced-usage`](docs/USAGE.md#normal-vs-advanced-usage).

## Legacy compatibility

Existing V1 projects (`CLAUDE.md`/`.plans/`, no `.agent/`) are never forced to migrate. `eng
doctor` classifies every project as `legacy` (pure V1, fully compatible, no action required),
`hybrid` (V1 files present, `.agent/project.yaml` also present — both work), `modern` (started
with `eng init`), `none`, or `broken` (unparseable config). Running `eng init` inside a legacy
project auto-detects it and sets `mode: hybrid` without touching any existing file. **There is
no `eng migrate` command** — none is needed, and none is claimed. Verified end-to-end in
Phase 8's benchmark: `benchmarks/results/legacy-v1-compat.yaml`. Full detail, including the
original V1 command reference (`scripts/plan-executor.sh`, `scripts/load_skill.sh`, ...), is in
[`docs/USAGE.md#legacy-v1-workflow-reference`](docs/USAGE.md#legacy-v1-workflow-reference).

## Project & context configuration

`.agent/project.yaml` (stack, enabled skills, domain profile, workflow mode, retry budget, tool
policy) and the optional `.agent/context.yaml` (skill/doc budget, log retention) are documented
field-by-field, with real defaults, in
[`docs/USAGE.md#project-configuration-agentprojectyaml`](docs/USAGE.md#project-configuration-agentprojectyaml)
and [`docs/USAGE.md#context-configuration-agentcontextyaml`](docs/USAGE.md#context-configuration-agentcontextyaml).

## Skill authoring

Skills are YAML-frontmatter Markdown files with three levels (`engineering`/`domain`/
`technology`), optional hard (`requires`) and soft (`recommends`) dependencies, and a defined
source precedence (global < private < project-local). A skill with no frontmatter still
resolves via the legacy heading convention — nothing requires migrating an old skill. Full
authoring guide, routing precedence, and namespacing rules: **[`docs/skills.md`](docs/skills.md)**.

## Security / safety model

- Planner and Plan Reviewer are capped at `READ` — neither can invoke a `WRITE`-or-above
  capability regardless of project config.
- Executor is capped at `WRITE`; anything `DESTRUCTIVE`/`HIGH_RISK` (e.g. `git.force_push`) is
  hard-denied at the policy layer for every role.
- Verifier never silently fixes anything — it reports `PASS`/`FAIL` and sees only its
  `write_scope`, never a general edit surface.
- One `approved_at` field on `plan.yaml` drives both execution-risk approval and
  `tools.require_approval` — there's no second, easy-to-forget approval concept to configure.
- Every tool invocation, allowed or refused, is audited (`adapter`, `capability`, `role`,
  `result`, `reason`, `log_path`) — raw arguments/output never enter the event itself.
- No field anywhere in `tools:`/`harness/mcp/servers.yaml`/a capability declaration can hold a
  secret by the shape of the type, not just convention — credentials must never go into tracked
  project files.

Full detail: [`docs/ARCHITECTURE.md#security--safety-model`](docs/ARCHITECTURE.md#security--safety-model).

## Benchmark / validation

Phase 8 (`benchmarks/`) validated the harness with real command output only — no invented token
counts. Token telemetry isn't available anywhere in this stack, so context-size claims use
structural proxies instead (skill/doc counts, context-bundle line counts, files changed) —
never a fabricated `tokens:` field. Read:

- [`benchmarks/README.md`](benchmarks/README.md) — the three comparison modes, what can/cannot
  be measured, how to reproduce any scenario
- [`benchmarks/COMPARISON.md`](benchmarks/COMPARISON.md) — V1 vs. V2 tables (Quick Fix, Feature,
  Legacy)
- [`benchmarks/CONTEXT_EFFICIENCY.md`](benchmarks/CONTEXT_EFFICIENCY.md) — the skill-vs-doc
  routing findings
- [`benchmarks/SCORECARD.md`](benchmarks/SCORECARD.md) — the ten-dimension scorecard and the
  `READY_TO_EXPAND` decision
- [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) — every real weakness found, P0–P3

## Known limitations

**Resolved in Phase 9** (kept here only as a pointer, not re-explained — see
[`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) for the fix evidence): Quick Fix triage
misclassification of parameter-tweak phrasing, doc-context over-matching relative to skill
routing, the dual `tasks.md` completion-convention confusion, a skill-routing false positive on
non-PLC serial/RS485 projects, and a global/local duplicate-skill resolution bug.

**Resolved in Phase 10:** the workflow state machine could be advanced after real work had
already happened outside it (role activation was a documented convention, not a runtime-checked
fact) — see [`docs/ARCHITECTURE.md#role-runtime-enforcement-phase-10`](docs/ARCHITECTURE.md#role-runtime-enforcement-phase-10)
and [`benchmarks/results/investigation-bypass-blocked.yaml`](benchmarks/results/investigation-bypass-blocked.yaml).

**Resolved in Phase 10.1:** a fresh, harness-launched session had no channel to learn the
harness existed — see [`docs/ARCHITECTURE.md#session-bootstrap-phase-101`](docs/ARCHITECTURE.md#session-bootstrap-phase-101).

Still open:

- **No live MCP JSON-RPC transport** — the one MCP-style adapter shipped (`docs-search`) is a
  deterministic local mock, not a real server connection.
- **No tool installer or marketplace** — adapters are added by implementing
  `tooladapter.Adapter` and registering it in `cli/tools_cmd.go`, not installed dynamically.
- **No live industrial-control write adapters** — no PLC/Modbus/OPC UA write capability exists;
  Phase 7's risk tiers exist so a future one has a safety model to plug into, not because one
  ships today.
- **No version-constrained skill package manager** — skills are resolved by directory tree +
  precedence tier, not a versioned dependency resolver.
- **No `eng migrate` command** — not needed today (see Legacy compatibility), but also not
  built; a legacy project's `mode: hybrid` is the only "upgrade" step that exists.
- **A request relevant to more skills than the context budget allows still gets budget-cut** —
  an inherent, expected consequence of a bounded skill budget (`max_skills`, default 5, which
  counts explicitly-enabled skills against it too), not a routing defect. See
  [`benchmarks/results/qt-cmake-embedded-linux-harness-v2.yaml`](benchmarks/results/qt-cmake-embedded-linux-harness-v2.yaml)
  for a concrete case.
- **`automation/plc`'s bare `automation` tag is a latent P2-2-class risk, not yet fixed** —
  observed alongside the `embedded/esp32` defect Phase 9 did fix, but nothing in Phase 9's own
  scenarios exercises it, so it's documented rather than fixed speculatively.
  [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md).
- **Role enforcement cannot intercept a Claude Code session's own native tool calls** — `eng` is
  a CLI a session chooses to invoke; there is no wrapper/sandbox interposed on Read/Edit/Write/
  Bash/browser-automation MCP tools. Phase 10's guarantee is that the workflow state machine
  cannot retroactively legitimize work that happened outside it, not that such work is prevented
  from happening in the first place — see `docs/ARCHITECTURE.md`'s enforced-vs-instructional
  table.
- **No fresh-session isolation for the Verifier role** — recommended in docs for stronger
  independence from the Executor's own session, but not spawned or enforced by any code.
- **`codex.execute` (write) does not exist** — Codex delegation is read-only by design; a future
  write capability would need its own capability name, risk tier, and policy decision.

## Roadmap

See [`ROADMAP.md`](ROADMAP.md). Short version: continued real-world dogfooding on role
enforcement and the newly covered project types (C#/.NET, Qt/CMake, embedded Linux) → address
the remaining latent risks above if real usage shows they matter → then further MCP/skill-
distribution expansion.

## Backward compatibility promise

V1 compatibility is a **tested, hard design requirement**, not an aspiration — every phase's
own test suite, plus Phase 8's dedicated legacy benchmark, re-verifies that `scripts/
plan-executor.sh`, `scripts/load_skill.sh`, and a project's existing `CLAUDE.md`/`.plans/`
behave identically to a pre-harness run. This commitment covers *today's* V1 surface as
currently implemented in this repository — it is not a promise that every future harness
change will automatically preserve compatibility without its own regression pass; each phase
re-verifies it explicitly rather than assuming it.

---

## Contributing

Contributions follow the same workflow this harness enforces: open an issue, a plan gets
created under `.plans/`, implementation follows the plan. See
[`docs/USAGE.md`](docs/USAGE.md) for the exact commands and
[`harness/core/principles/karpathy.md`](harness/core/principles/karpathy.md) for the engineering
principles every plan is checked against.

## License

MIT. You are free to use, copy, modify, and distribute this project in any project, commercial
or otherwise.

## Acknowledgements

Workflow philosophy based on [Andrej Karpathy's](https://karpathy.ai) engineering principles:
think before coding, prefer simplicity, make surgical changes, define done by user-visible
outcomes. Built with [Claude Code](https://claude.ai/code).
