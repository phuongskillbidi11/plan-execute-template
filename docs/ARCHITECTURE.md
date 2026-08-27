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

## Session bootstrap (Phase 10.1)

Everything below in this document — role enforcement, state-role mapping, the enforced-vs-
instructional boundary — only matters to a session that already knows to consult it. Real
dogfooding immediately after Phase 10 shipped found that a fresh, harness-launched Claude Code
session had no way to learn any of it: `eng start` set `ENG_HOME`/`ENG_PROJECT_ROOT`/
`ENG_VERSION` as environment variables (Phase 9) and printed a pointer to
`core/runtime/METHOD.md`, but env vars aren't self-announcing and printed terminal output never
reaches the launched agent's own context — so the session concluded, correctly given what it
actually had, that no harness existed.

`eng start` (`cli/bootstrap.go`) now also launches `claude` with `--append-system-prompt
"<bootstrap identity>"` — a small, deterministic status block built from the same data `eng
doctor` reads (`gatherBootstrapStatus`), appended to Claude Code's own default system prompt via
a real, verified CLI flag (not a shell-string hack — `os/exec` never invokes a shell for this
call on any platform). This is a **trusted** channel, distinct from anything said in chat: every
line traces back to a specific field the harness's own process read before the session's first
turn, not free-form text a session or a user could fabricate. It tells the agent to verify
current state through `eng` rather than trust the snapshot alone, states plainly that a missing
project-local `CLAUDE.md`/`.claude/` does not mean the harness is absent, and instructs it not
to auto-resume a `COMPLETED` plan. See [`docs/USAGE.md`](USAGE.md#how-a-fresh-session-knows-its-harness-managed-phase-101)
for the exact prompt shape and the global-vs-project-vs-session hierarchy.

## Role runtime enforcement (Phase 10)

Before Phase 10, `Planner → Plan Reviewer → Executor → Verifier` was a documented convention —
four `core/*/METHOD.md` prose files an agent was expected to follow, with nothing in code
checking that it actually did. Real dogfooding found the state machine could be advanced
*after* the real work had already happened (a plan reached `COMPLETED` with a PASS verdict and
an **empty git diff**, because `tasks.md`'s completion checklist had been marked before
`EXECUTING` was ever entered) — the workflow was descriptive, not controlling. Phase 10 closes
this with two small, additive mechanisms:

**A persisted role runtime model** (`cli/internal/rolestate`, one file per plan,
`role-state.yaml`): `current_role`, `activated_at`, `activated_for_state`,
`prompt_generated_at`, `context_manifest`. A pure, deterministic state-to-role table
(`rolestate.AllowedForState`/`NextRole`) answers "is this role compatible with the current
workflow state" — the same table drives both the activation boundary below and `eng workflow
status`'s `Active role:`/`Next role:` lines.

**`eng adapter prompt <role> <plan-dir>` is the activation boundary**, not a new command — every
`core/*/METHOD.md` file already instructs running it before acting in a role, so making that
existing call validate and record the activation required no new command surface. A role/state
mismatch refuses (fail closed, non-zero exit, no prompt printed) before anything is composed;
success writes `role-state.yaml`, a per-role `context-manifest-<role>.yaml` (never overwritten
by a different role's later activation — the unqualified `context-manifest.yaml` still exists
as the last-activation pointer), and a `role_activated` event.

**Two new hard gates in `workflow.Decide`:**
1. `APPROVED → EXECUTING` (and Quick Fix's `TRIAGED → EXECUTING`) now requires the executor role
   to have been activated first — `Facts.ExecutorActivated`.
2. If `tasks.md`'s Completion checklist is *already* fully checked at the exact instant a
   transition into `EXECUTING` would otherwise fire, `Decide` refuses with
   `Action: "invariant_violation"` instead of proceeding — under every normal flow this is
   false (the checklist starts unchecked); it being true is the reproduced bypass signature.
   No override flag exists; remediation is to uncheck the checklist and let the Executor
   genuinely complete it under `EXECUTING`.

**Mechanical verification and role verification are now genuinely separate.** `eng verify`
(git diff / build / test) still runs automatically on `EXECUTING → VERIFYING`, unchanged. When
`workflow.verifier` resolves enabled (Phase 9's `VerifierEnabled()` accessor — Phase 10 is its
first real consumer), `VERIFYING → COMPLETED` additionally requires `eng plan verify-review
<plan-dir> --verdict PASS`, the Verifier role's own independent judgment
(`plan.yaml`'s `role_verification` field, distinct from the mechanical `verification` field).
Disabled, `COMPLETED` is reachable on mechanical `PASS` alone — byte-identical to Phase 9.

**What this does and does not enforce — read before assuming more than is true:**

| | Enforced | Instructional only |
|---|---|---|
| `eng tools invoke <role> <capability> <plan-dir>` | Yes — checks `role-state.yaml`'s current role *and* state compatibility before `toolpolicy.Decide` even runs | — |
| `eng adapter prompt <role> <plan-dir>` | Yes — refuses on role/state mismatch, records success | — |
| `eng workflow advance`'s state transitions | Yes — the two new gates above | — |
| A Claude Code session's own native tools (Read/Edit/Write/Bash/browser-automation MCP tools) | — | **Not enforceable.** `eng` is a CLI a session *chooses* to invoke; there is no wrapper, sandbox, or daemon interposed on a session's own tool calls. A session that edits a file directly while `APPROVED` cannot be stopped by anything in this repo. |

Given that boundary, Phase 10's actual guarantee is precisely calibrated: the harness cannot
prevent a premature edit from happening, but it now provably refuses to let the state machine
retroactively legitimize one once detected — see
[`benchmarks/results/investigation-bypass-blocked.yaml`](../benchmarks/results/investigation-bypass-blocked.yaml)
for the real before/after proof. `eng doctor`'s `Tools:` section reflects the same honesty
principle generically: every adapter reports `installed`/`wired`/`invokable` as three distinct
booleans, not one conflated `available` flag.

Full command reference: [`docs/USAGE.md`](USAGE.md#role-activation).

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
  Phase 10 adds one more check ahead of `toolpolicy.Decide`: the invoking role must actually be
  the plan's currently-activated role (`role-state.yaml`) — closes the "claim any role in the
  CLI argument" gap the role-vs-capability checks alone never covered.
- Today's real adapters: `git`, `github` (read-only, via the `gh` CLI), `codex` (read-only —
  `codex.inspect`/`codex.review`/`codex.verify`, Phase 10's second-opinion delegation adapter,
  no write capability), and a deterministic reference MCP adapter (`docs-search`, greps
  `docs/*.md` locally — no live network transport). **No live MCP JSON-RPC transport, no tool
  marketplace, and no write adapter for PLC/Modbus/OPC UA exists yet** — see
  [Known Limitations in the README](../README.md#known-limitations).

## Security / safety model

- **Planner and Plan Reviewer are read-capped** — `RoleMaxRisk` limits both to `READ`; neither
  can invoke a `WRITE`-or-above capability regardless of project policy.
- **Executor is write-capped, not unlimited** — `RoleMaxRisk` allows up to `WRITE`;
  anything `DESTRUCTIVE`/`HIGH_RISK` (e.g. `git.force_push`) is hard-denied at the policy layer
  regardless of role or project config.
- **Verifier never silently fixes** — its context bundle exposes only `write_scope`; it reports
  a verdict (`PASS`/`FAIL`) and never edits files. `RoleMaxRisk` caps it at `READ`, same as
  Planner/Plan Reviewer — it cannot invoke a `WRITE`-or-above capability regardless of what its
  own role verdict says.
- **A role must be activated to invoke a tool as that role** (Phase 10) — `eng tools invoke
  executor ...` is denied unless `eng adapter prompt executor <plan-dir>` has actually succeeded
  for that plan since its last replan cycle. Claiming a role in the CLI argument is no longer
  sufficient on its own.
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

**Internal dogfooding / team-usable beta** — not production-hardened. Phase 8's benchmark suite
found and Phase 9 fixed every real-world gap it surfaced; real dogfooding after that found the
workflow state machine could be advanced after work had already happened outside it (see
[Role runtime enforcement](#role-runtime-enforcement-phase-10) above) — fixed in Phase 10, with
a real reproduced-and-blocked proof, not just a design claim. Dogfooding immediately after that
found a fresh harness-launched session had no way to *learn* any of the above (see
[Session bootstrap](#session-bootstrap-phase-101) above) — fixed in Phase 10.1. See
[`benchmarks/SCORECARD.md`](../benchmarks/SCORECARD.md) and
[`benchmarks/BACKLOG.md`](../benchmarks/BACKLOG.md) for the full evidence, and the README's
[Known Limitations](../README.md#known-limitations) for what's still genuinely open.
