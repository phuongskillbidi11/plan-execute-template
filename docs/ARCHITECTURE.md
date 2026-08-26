# Architecture

How a natural-language request turns into routed context, a state-machine-driven plan, and
audited tool calls. For command syntax see [`docs/USAGE.md`](USAGE.md); for the skill model see
[`docs/skills.md`](skills.md); for the tool/capability model see [`docs/tools.md`](tools.md).

## The four responsibilities, kept separate

```text
Skill            = methodology / knowledge — how to approach a task
Agent Adapter    = integration with a coding agent (Claude Code, ...)
Tool/MCP Adapter = integration with an external capability (git, GitHub, an MCP server, ...)
Harness          = routing, lifecycle, permissions, state, context, orchestration
```

None of the first three know about each other. The harness (`cli/internal/workflow`,
`cli/internal/skillrouter`, `cli/internal/toolrouter`, `cli/internal/toolpolicy`) is what
connects a role, a request, and a policy to a concrete decision.

## Request flow

```text
Developer
   │  describes a requirement in plain language
   ▼
eng start                       → eng doctor, then launches the configured agent
   │
   ▼
Runtime (harness/core/runtime/METHOD.md)
   │  the command sequence a Claude Code session follows automatically
   ▼
eng workflow start "<text>"
   │
   ▼
Triage (heuristic — keyword + docs/gotchas.md match)
   │  suggests a risk level: quick-fix | bug | feature | architecture | high-risk
   ▼
Context Manager (eng context skills / project / task / bundle)
   │  selects only the skills/docs/task-scope relevant to this request
   ▼
Skill Router (eng skills list / validate)
   │  explicit + required-dependency + matched + domain-profile + recommended skills
   ▼
Workflow (cli/internal/workflow — the state machine, see docs/USAGE.md#workflow-states)
   ├─ Quick Fix        (TRIAGED → EXECUTING → VERIFYING → COMPLETED)
   └─ Spec-First / Feature (TRIAGED → NEEDS_SPEC_APPROVAL → ... → COMPLETED)
   ▼
Planner → Plan Reviewer → Executor → Verifier
   │  each gets a role-specific eng adapter prompt / eng context bundle
   ▼
Capability / Tool Router (toolrouter.Route, toolpolicy.Decide)
   │  ALLOWED / NEEDS_APPROVAL / BLOCKED, per role + risk tier + project policy
   ▼
Adapters (git, GitHub read-only, reference MCP) — every invocation audited
```

Every arrow above is a real, inspectable `eng` command — nothing in this diagram is a black
box. Run `eng context skills "<request>"`, `eng capabilities explain <role> <plan-dir>
"<request>"`, or `eng workflow status <plan-dir>` at any point to see exactly what the harness
decided and why.

## Install vs. project — the V1 → V2 shift

```text
V1 (this repo, pre-harness):
  clone/copy the whole template into every project repository

V2 (this repo, current):
  install the harness once, globally (`eng install`)
  + link a thin per-project config (`eng init` writes only `.agent/project.yaml`)
```

`eng install --from <path>` copies `harness/` (methodology docs, skills, profiles, templates,
hooks) into `~/.engineering-harness/`. A project never gets its own copy of that tree — `eng
init` writes exactly one file, `.agent/project.yaml`. Skills, workflow templates, and hooks are
resolved from the global install (plus optional private/project-local tiers — see
[`docs/skills.md`](skills.md#skill-sources-and-precedence)) at run time, not copied in.

## Context engineering

**Large knowledge base ≠ large prompt.** The harness can ship dozens of skills and long
`docs/src-map.md`/`docs/gotchas.md` files without every request's prompt growing with them:

- **Selective skill loading** — `skillrouter.Route` scores a request's text against each
  skill's `tags`/`triggers`, expands hard `requires:` dependencies, fills a project's
  `domains:` profile, then adds `recommends:` only if the budget allows. See
  [`docs/skills.md`](skills.md#routing-precedence).
- **Bounded project-doc retrieval** — `docsearch.Match` keyword-matches
  `docs/src-map.md`/`docs/gotchas.md` sections against the request instead of including the
  whole file.
- **Task-scoped context** — `eng context task` extracts just the current unchecked task block
  and `spec.md`'s goal summary, not entire plan files.
- **Role-specific bundles** — `eng context bundle <role> <plan-dir>` composes only what that
  role needs (Planner gets project docs + skills + tools; Verifier gets `write_scope` only).
- **Bounded verification/tool output** — `eng verify` and `eng tools invoke` write full output
  to `.agent/logs/*.log` and show a head+tail summary (`max_log_lines`, default 300) in the
  report/audit trail, never the raw dump.
- **Log compaction** — `eng logs prune` applies `.agent/logs/` retention
  (`max_log_files`/`max_log_age_days`/`max_log_total_mb`); `eng verify` prunes automatically
  after every run.
- **Context manifest** — every `eng context bundle` call writes
  `<plan-dir>/context-manifest.yaml`, a durable record of exactly which skills/docs/tools were
  selected and why, independent of the conversational context that consumed it.

The budget itself is configurable per project — see [`docs/USAGE.md`](USAGE.md#context-configuration-agentcontextyaml).

**What this principle does *not* yet fully hold for:** Phase 8's benchmark found
`eng context skills` achieves 0 false positives on both a cross-domain and a single-domain
test request, but `eng context project`'s doc-section matching over-selected on the one case
tested (67% false-positive rate against that fixture). See
[`benchmarks/BACKLOG.md`](../benchmarks/BACKLOG.md) P1-2 and
[`benchmarks/CONTEXT_EFFICIENCY.md`](../benchmarks/CONTEXT_EFFICIENCY.md) for the evidence —
this is a real, open gap, not a solved problem.

## The lifecycle state machine

`cli/internal/workflow.Decide` is a pure function: `Facts → Decision`. It reads no files
itself — `cli/workflow_cmd.go`'s `gatherFacts` does the I/O (reading `plan.yaml`, `tasks.md`,
`.agent/project.yaml`), then `Decide` picks at most one transition per call. `eng workflow
advance` never chains multiple transitions silently past a state a human should see, and it
never writes plan content or invokes an agent unattended — every stage ends with a printed next
command and a stop.

See [`docs/USAGE.md`](USAGE.md#workflow-states) for the full state table and transition rules.

## Tool / capability model (Phase 7)

Full detail lives in [`docs/tools.md`](tools.md); summary:

- A capability is `<adapter>.<operation>` (`git.status`, `github.pr.read`). Each adapter
  declares its own capabilities and risk tier — `READ < WRITE < DESTRUCTIVE < HIGH_RISK`.
- Two independent role checks: is this adapter even in the role's toolbox
  (`agent.RolePermissions`), and is this capability's risk tier within the role's ceiling
  (`agent.RoleMaxRisk`)? Both must pass.
- `toolpolicy.Decide` evaluates project `tools:` policy (`deny` → role toolbox → role risk
  ceiling → `require_approval` → `allow` → a safe risk-based default) in a fixed order — see
  [`docs/tools.md`](tools.md#project-tool-policy-tools).
- `eng tools invoke` is the only sanctioned invocation path: it always runs `toolpolicy.Decide`
  first, and a refusal writes an audit event and exits non-zero *before* touching the adapter.
- Today's real adapters: `git`, `github` (read-only, via the `gh` CLI), and a deterministic
  reference MCP adapter (`docs-search`, greps `docs/*.md` locally — no live network transport).
  **No live MCP JSON-RPC transport, no tool marketplace, and no write adapter for
  PLC/Modbus/OPC UA exists yet** — see [Known Limitations in the README](../README.md#known-limitations).

## Security / safety model

- **Planner and Plan Reviewer are read-capped** — `RoleMaxRisk` limits both to `READ`; neither
  can invoke a `WRITE`-or-above capability regardless of project policy.
- **Executor is write-capped, not unlimited** — `RoleMaxRisk` allows up to `WRITE`;
  anything `DESTRUCTIVE`/`HIGH_RISK` (e.g. `git.force_push`) is hard-denied at the policy layer
  regardless of role or project config.
- **Verifier never silently fixes** — its context bundle exposes only `write_scope`; it reports
  a verdict (`PASS`/`FAIL`) and never edits files.
- **Approval is one field, reused consistently** — `plan.yaml`'s `approved_at` is both the
  execution-risk approval gate (`eng plan approve`) and the same signal `tools.require_approval`
  checks. There's no separate, easy-to-forget tool-approval concept.
- **Every tool invocation is audited** — `eng tools invoke` appends a `tool_invocation` event
  (`adapter`, `capability`, `role`, `result`, `reason`, `log_path`) to `events.jsonl` on every
  outcome, allowed or refused. Raw arguments/output never enter the event itself.
- **No credential can live in a tracked config field** — nothing in `tools:`,
  `harness/mcp/servers.yaml`, or a capability declaration can hold a secret value by the shape
  of those types, not just by convention. The `GitHubAdapter` delegates entirely to `gh`'s own
  OS-keyring auth; the harness never reads, stores, or prints a token.

## Current maturity

**Internal dogfooding / early team use** — not production-hardened. Phase 8's benchmark suite
(`benchmarks/`) ran real `eng`/V1 commands across 8 categories and found the core safety, skill
routing, and legacy-compatibility guarantees hold, while surfacing three concrete, unfixed
gaps — see [`benchmarks/SCORECARD.md`](../benchmarks/SCORECARD.md) and
[`benchmarks/BACKLOG.md`](../benchmarks/BACKLOG.md) for the full evidence, and the README's
[Known Limitations](../README.md#known-limitations) for the short version.
