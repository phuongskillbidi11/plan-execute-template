# Tools — the adapter model, capability/risk model, policy, and how to add one

## The four responsibilities, kept separate

```
Skill            = methodology / knowledge / how to perform a task
Agent Adapter    = integration with a coding agent (Claude Code, ...)
Tool/MCP Adapter = integration with an external capability (git, GitHub, an MCP server, ...)
Harness          = routing, lifecycle, permissions, state, context, orchestration
```

`internal/agent.Adapter` launches/talks to a coding agent. `internal/tooladapter.Adapter`
exposes an external capability. Neither knows about the other; the harness (`internal/
toolrouter`, `internal/toolpolicy`, `cli/context_cmd.go`) is what connects a role, a
request, and a policy to a concrete adapter invocation.

## Capability naming and the risk model

A capability is `<adapter>.<operation>` — `git.status`, `github.pr.read`, `docs.search`.
Each `tooladapter.Adapter` declares its own `[]toolcap.Capability{Name, Risk}`; there is no
separate global capability catalog to keep in sync.

```
READ < WRITE < DESTRUCTIVE < HIGH_RISK
```

`toolcap.RiskRank` gives each tier an integer so comparisons stay a single `<=` check. No
Phase 7 capability is `DESTRUCTIVE` or `HIGH_RISK` except `git.force_push` (`DESTRUCTIVE`,
hard-denied) — the tiers exist so a future PLC-write/Modbus-write/production-deploy adapter
has a safety model to plug into, not to gate anything that exists yet.

## Two independent role-permission axes

1. **`agent.RolePermissions`** (coarse) — is this *adapter* even in this role's toolbox at
   all? Verifier never touches `claude`/`codex`; every role currently includes `git`,
   `github`, and `mcp-docs`.
2. **`agent.RoleMaxRisk`** (fine) — the highest *risk tier* this role may invoke without an
   extra approval. Planner/Plan Reviewer/Verifier are capped at `READ`; Executor at `WRITE`.
   An adapter being in a role's toolbox does **not** by itself grant every risk tier that
   adapter exposes — both checks must pass.

## Project tool policy (`tools:`)

```yaml
# .agent/project.yaml
tools:
  allow:
    - git.status
  require_approval:
    - github.issue.comment
  deny:
    - git.force_push
```

All three lists are optional; every field is a capability *name*, never a secret. Precedence,
evaluated in this fixed order by `toolpolicy.Decide`:

```
1. built-in hard deny            (unconditional — no project config can override)
2. project tools.deny            → DENIED
3. role's adapter-toolbox check  → DENIED if this role doesn't use this adapter at all
4. role's risk-ceiling check     → DENIED if this role's max risk tier is below this capability's risk
5. project tools.require_approval → ALLOWED if the plan is approved, else NEEDS_APPROVAL
6. project tools.allow           → ALLOWED
7. risk == READ, nothing matched → ALLOWED (today's ambient behavior — a project with no
   tools: block at all gets exactly this for every read)
8. risk >= WRITE, nothing matched → NEEDS_APPROVAL (never silently allowed)
```

"Approved" for step 5/8 is exactly `plan.yaml`'s existing `approved_at != ""` — the same
field `eng plan approve` sets. There is no separate tool-specific approval state.

`Config.RequireApproval` (a pre-existing, top-level, never-read field) was deliberately
*not* reused for `tools.require_approval` — see `.plans/2026-08-25-v2-harness-phase7-tools/
DECISION_LOG.md` entry 2 for why.

## Secrets and credentials

No field in `tools:`, in `harness/mcp/servers.yaml`, or on `toolcap.Capability` can hold a
secret — structurally, by the shape of those types, not just by convention. `GitHubAdapter`
delegates entirely to the `gh` CLI's own authentication (a token in the OS keyring from `gh
auth login`); the harness never reads, stores, or prints it.

A future adapter that needs its own credential should use a *reference*, never a value:

```yaml
# a hypothetical future adapter's config — not implemented by anything in Phase 7
credential_env: SOME_SERVICE_TOKEN
```

naming an environment variable the adapter reads at invocation time, never a literal secret
written into a tracked file.

## The MCP registry — static, local, no live transport

`harness/mcp/servers.yaml` declares known MCP-style servers: name, transport, capabilities,
permission category. Phase 7 ships one entry, `docs-search` (`transport: mock`), backed by
`tooladapter.ReferenceMCPAdapter` — a real, deterministic, network-free implementation (it
greps this repo's `docs/*.md`) that proves the full lifecycle (discovery → health →
capability declaration → routing → permission → invocation → audit → context exposure)
without a live MCP JSON-RPC connection, which is out of Phase 7's scope. A future real MCP
server needs one new registry entry and one new `tooladapter.Adapter` implementation
matched by name in `registeredAdapters` (`cli/tools_cmd.go`) — not a new subsystem.

## Inspecting routing

```bash
eng capabilities explain <role> <plan-dir> ["<request text>"]
```

Shows what the skills selected for that request would need, and the routing verdict
(`ALLOWED`/`NEEDS_APPROVAL`/`BLOCKED`) with a reason for each. `eng doctor`'s `Tools:`
section shows a bounded, secret-free summary (name, availability, capability count) for
every registered adapter.

## Invoking a capability

```bash
eng tools invoke <role> <capability> <plan-dir> [args...]
```

The only sanctioned invocation path — `toolpolicy.Decide` always runs first; a refusal
(`DENIED`/`NEEDS_APPROVAL`) writes a compact audit event and exits non-zero *before*
touching the adapter. On success, the adapter's full output goes to `.agent/logs/tool-*.log`
(reusing the same `writeFullLog`/`summarizeOutput` compaction Phase 4/5 already established
for test output) and a bounded `tool_invocation` event — `adapter`, `capability`, `role`,
`result`, `reason`, `log_path` — is appended to the plan's `events.jsonl` via the existing
`planmeta.AppendStructuredEvent`. Raw arguments/output never go into the event itself.

## Adding a new adapter

1. Implement `tooladapter.Adapter` (`Name`, `Provider`, `Version`, `Capabilities`,
   `Available`, `Doctor`, `Invoke`) — see `cli/internal/tooladapter/github.go` for a small,
   real example.
2. Add its `Name()` to every role's entry in `agent.RolePermissions` that should be able to
   use it (or a subset, if that's the intent).
3. Register it in `registeredAdapters` (`cli/tools_cmd.go`).
4. If any capability is `WRITE` or above, decide whether it belongs in a project's
   `tools.allow`/`tools.require_approval` by default — nothing is silently allowed past
   `READ`.
