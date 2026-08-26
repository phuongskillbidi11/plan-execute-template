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

**Internal dogfooding / early team use.** Not production-hardened. Phase 8 ran a real,
evidence-based benchmark suite (`benchmarks/`) comparing this harness against a baseline
unharnessed agent and against the original V1 template across 8 categories. Result: **7 of 10
scorecard dimensions rated Strong, 3 rated Adequate, 0 Weak, 0 blocking (P0) issues** — see
[`benchmarks/SCORECARD.md`](benchmarks/SCORECARD.md). The benchmark's own decision:
`READY_TO_EXPAND`, with three real, documented refinement items outstanding (not silently
fixed — tracked in [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md)):

- **Quick Fix triage misclassification** — plausible real-world phrasing routes to the full
  feature workflow instead of Quick Fix (workaround: `eng plan new --risk quick-fix` explicitly)
- **Doc-context over-matching** — `eng context project` selected 4 of 6 doc sections where only
  1 was relevant in the one case tested, vs. 0 false positives for skill routing on the same
  request
- **Dual task-completion conventions** — `tasks.md`'s per-task `Status:` marker and its bottom
  "Completion checklist" are unsynchronized; only the bottom one gates `eng workflow advance`

See [Known limitations](#known-limitations) for the short version, and
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
fixed order. `eng tools invoke` is the only sanctioned invocation path; every outcome, allowed
or refused, is written to an audit trail.

**Real adapters today:** `git`, `github` (read-only, via the `gh` CLI), and a deterministic
reference MCP adapter (`docs-search`) that greps local docs — no live network transport. **Not
implemented:** a full MCP JSON-RPC transport, a tool installer/marketplace, or any live
industrial-control write adapter (PLC/Modbus/OPC UA). Full model in
[`docs/tools.md`](docs/tools.md) and [`docs/ARCHITECTURE.md#tool--capability-model-phase-7`](docs/ARCHITECTURE.md#tool--capability-model-phase-7).

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

- **Quick Fix triage misclassifies some plausible phrasing** (numeric/parameter-tweak requests)
  — workaround: scaffold explicitly with `--risk quick-fix`. [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) P1-1.
- **Doc-context routing over-matches relative to skill routing** — measured 67% false-positive
  rate on one large-context benchmark case. [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) P1-2.
- **`tasks.md` has two unsynchronized completion conventions** (per-task `Status:` marker vs.
  the bottom checklist that actually gates `eng workflow advance`) — mark both.
  [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md) P2-1.
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

## Roadmap

See [`ROADMAP.md`](ROADMAP.md). Short version: **Core Refinement** (fix the Phase 8 P1/P2
backlog items above) → continued real-world dogfooding → then further MCP/skill-distribution
expansion, only once the refinement backlog is addressed.

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
