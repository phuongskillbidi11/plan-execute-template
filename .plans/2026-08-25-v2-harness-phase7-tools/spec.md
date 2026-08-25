# Phase 7 — Tool/MCP Adapter Runtime & Permission Enforcement

## Goal

Phase 6 answered "what does the agent know" (skills). Phase 7 answers "what external
capabilities can the agent safely use" — turning the Phase 5 `internal/tooladapter`/
`internal/toolrouter` foundation (deliberately unpopulated, per its own doc comments) into
an enforced runtime: fine-grained, risk-classified capabilities; role permission that's
actually checked, not just reported; a project-level allow/require-approval/deny policy;
deterministic routing with an explanation; a real invocation boundary that writes a compact
audit event and fails closed on missing approval; and one genuine external reference adapter
proving the whole pipeline, without building a plugin marketplace.

## What already exists (Phase 1–6) — reused, not rebuilt

Read in full before writing any code for this plan:

- `cli/internal/tooladapter/tooladapter.go` — `Adapter` interface (`Name`, `Capability()
  string` singular, `Available`, `PermissionLevel() string`, `Doctor`) and `GitAdapter`, both
  explicitly documented as "foundation only... not a real capability gate" (Phase 5
  DECISION_LOG entry). This is the one interface Phase 7 is expected to evolve — see Decision
  1 for why that's a deliberate revision of a foundation, not a break to a stable contract.
- `cli/internal/toolrouter/toolrouter.go` — `Filter(required []string, adapters
  []tooladapter.Adapter) []tooladapter.Adapter`, "the entire Tool Router for Phase 5... it
  exposes nothing to any agent session." Kept unchanged; Phase 7 adds `Route`, the new
  authoritative path.
- `cli/internal/capabilities/capabilities.go` — binary-on-PATH detection (`Known`, `Detect`,
  `DescribeAll`) for `git`/`claude`/`codex`/`docker`. This is a *different, lower* layer
  than Phase 7's capability model — "is this binary on PATH" vs. "is this specific
  operation permitted" — and stays as-is except for one additive entry (`gh`).
- `cli/internal/agent/permissions.go` — `RolePermissions map[Role][]string` (coarse,
  adapter-name-keyed, e.g. `RoleExecutor: {"git", "claude", "codex", "docker"}`) and
  `RoleMayUse`, explicitly "reporting-only — nothing yet enforces this against a real tool
  invocation" (Phase 5 Decision 11). Phase 7 both *enforces* this and adds a second,
  independent axis (risk-tier ceiling per role) alongside it.
- `cli/internal/project/project.go` — `Config.RequireApproval []string` exists, is never
  read by any code, and its intended shape/semantics were never documented or tested. See
  Decision 2 for why this plan does *not* repurpose it.
- `cli/internal/planmeta/planmeta.go` — `Meta.RequiresApproval`/`ApprovedAt`/`ApprovedBy`
  (Phase 3 execution-approval gate) and `AppendStructuredEvent(planDir, type, data)` (Phase
  5's flat-JSON audit primitive, already used for the `quick_fix` event). Both reused
  directly — no new approval state, no new audit package.
- `cli/verify_cmd.go` — `writeFullLog`/`summarizeOutput` (Phase 4/5 output-compaction
  pattern: full output to `.agent/logs/`, a bounded head+tail summary everywhere else).
  Reused as-is for tool invocation output.
- `cli/internal/skills/skills.go` — `Skill.Capabilities []string`, added in Phase 6,
  documented as "free-form, reserved for future capability-based routing," never consumed
  by anything until now. This is the exact hook Phase 7 Requirement 28 asks for — reused,
  not re-schema'd (see Decision 3).
- `cli/context_cmd.go` — `buildContextBundle`, the one authoritative context-assembly path.
  Gains one more section (`## Tools`) for the roles that already get a `## Skills` section,
  computed from the very capabilities those selected skills already declared.
- `harness/core/context-manager/METHOD.md`, `harness/core/runtime/METHOD.md` — get short,
  additive pointers to the new inspection commands; no structural rewrite.

## Design decisions

1. **The `tooladapter.Adapter` interface is revised, not frozen.** Phase 5's own
   DECISION_LOG calls it "foundation only... GitAdapter... exists to prove the interface
   compiles and is testable, not as a real capability gate." Phase 7 is explicitly the
   phase that turns it into a real gate, so its shape changing is expected evolution, not
   an unrelated refactor. The two real consumers (`GitAdapter`, the Phase 5
   `tooladapter_test.go`/`toolrouter_test.go` suites) are updated in the same task, not left
   broken.

2. **`Config.RequireApproval` is left untouched; a new nested `tools:` block is added
   instead.** The instruction's own example schema (`tools: {allow, require_approval,
   deny}`) is a nested block; the existing flat field has no documented intent, no reader,
   and no test — repurposing it would be guessing at semantics nobody ever specified,
   which is a bigger risk than adding a clearly-scoped new field. This is the same judgment
   call Phase 6 made about `HarnessProfile` vs. the new `Domains` field, applied
   consistently.

3. **`Skill.Capabilities` becomes the request→required-capabilities signal — unchanged
   shape.** Requirement 28 offers a nested `capabilities: {recommends: [...]}` example, but
   the field Phase 6 already shipped is a flat `[]string`, already scored by the skill
   router as an extra tag list, and already zero-value (`[]`) on every real skill.
   Redefining its Go type now would break every existing `SKILL.md`'s `capabilities: []`
   line (a sequence-vs-mapping YAML type mismatch) for a distinction (required vs.
   recommended) this plan doesn't need — every capability a skill names is treated as a
   *soft* signal into the tool router, exactly like a skill's `recommends:` is a soft
   signal into the skill router. One skill (`engineering/debugging`) gets a real,
   honestly-justified `capabilities: [docs.search]` entry to prove the wiring end-to-end;
   no skill is forced to declare one.

4. **Two independent role-permission axes, both in `internal/agent`.** Phase 5's
   `RolePermissions`/`RoleMayUse` (coarse: is this *adapter* even in this role's toolbox —
   e.g. Verifier never touches `claude`/`codex`) stays exactly as-is and gains one additive
   entry (`github`) per role. A new, second table, `RoleMaxRisk` (fine: what's the highest
   *risk tier* this role may invoke without extra approval — Planner/Reviewer/Verifier capped
   at `READ`, Executor at `WRITE`), lives alongside it in the same file, because Requirement
   5's example table ("Planner → READ only... Executor → READ + explicitly allowed WRITE")
   is expressed in risk terms, not adapter names, and conflating the two axes into one table
   would make an adapter's presence in `RolePermissions` implicitly grant every risk tier it
   exposes — exactly the "adapter available ⇏ role allowed" collapse Requirement 5
   explicitly forbids.

5. **Capability naming: `<adapter>.<operation>`, no central capability registry.** `git.status`,
   `git.diff`, `github.pr.read`, `docs.search`. Each adapter declares its own
   `[]toolcap.Capability{Name, Risk}` — there is no separate global "capability catalog" to
   keep in sync, because the adapter *is* the authority on what it exposes and at what risk;
   a central registry duplicating that would be one more place to drift out of sync with the
   code that actually implements it (`internal/capabilities.Known` already shows this
   drift risk is real — it's a static list a developer must remember to update).

6. **Risk model: `READ < WRITE < DESTRUCTIVE < HIGH_RISK`, a total order.** `RiskRank`
   gives each tier an integer so role ceilings and future comparisons stay a single `<=`
   check instead of a growing switch statement. No capability shipped in Phase 7 is
   `DESTRUCTIVE` or `HIGH_RISK` except `git.force_push` (`DESTRUCTIVE`, hard-denied) —
   Requirement 4 explicitly asks to establish the tiers before adapters that need the top
   two exist for real.

7. **Policy precedence — hard deny first, then project policy, then role, then a
   risk-based default (READ open, WRITE+ needs approval).** Concretely, `toolpolicy.Decide`
   evaluates in this fixed order:

   ```
   1. built-in hard deny (unconditional — no project config can override)
   2. project tools.deny            → DENIED
   3. role's adapter-toolbox check  → DENIED if this role doesn't use this adapter at all
   4. role's risk-ceiling check     → DENIED if this role's max risk tier is below this capability's risk
   5. project tools.require_approval → ALLOWED if plan.yaml is approved, else NEEDS_APPROVAL
   6. project tools.allow           → ALLOWED
   7. risk == READ (nothing else matched) → ALLOWED (preserves today's ambient "read tools
      just work" behavior for every project with no `tools:` block at all)
   8. risk >= WRITE (nothing else matched) → NEEDS_APPROVAL (never silently allowed)
   ```

   This is deliberately safe-by-default: a project with zero `tools:` configuration (every
   project before this phase) gets exactly today's behavior for reads, and every write-ish
   capability is gated behind the *existing* plan-approval mechanism rather than silently
   permitted — satisfying Requirement 29's "do not silently assume the tool is available"
   without inventing a second approval concept (Decision 8).

8. **Tool invocation reuses the Phase 3 execution-approval gate — no new approval state.**
   "Approved" for a `NEEDS_APPROVAL` capability means exactly `plan.yaml`'s existing
   `approved_at != ""` (the same field `eng plan approve` already sets). This is the
   literal meaning of Requirement 20 ("Phase 3/5 already established machine-readable
   approvals... connect tool invocation to those gates") — not a parallel tool-specific
   approval flag that could drift out of sync with the plan's own approval state.

9. **MCP is a static, local registry plus one deterministic mock adapter — no protocol
   client.** `harness/mcp/servers.yaml` declares known MCP-style servers (name, transport,
   capabilities, permission category — no credential fields exist in the schema at all,
   structurally, not just by convention). Phase 7 ships exactly one entry,
   `docs-search` (`transport: mock`), backed by `tooladapter.ReferenceMCPAdapter` — a real,
   deterministic, network-free implementation of the `Adapter` interface (greps this repo's
   `docs/*.md`) that proves discovery → health → capability declaration → routing →
   permission → invocation → audit → context exposure end to end, exactly as Requirement 16
   explicitly permits ("create a deterministic mock/reference adapter and document why")
   instead of implementing real MCP JSON-RPC transport, which is out of this phase's bound.

10. **Credentials never enter a tracked file — delegated entirely to each external tool's
    own store.** The one adapter needing real external auth, `GitHubAdapter`, shells out to
    the `gh` CLI (already installed and authenticated in this environment via its own OS
    keyring — verified: `gh auth status` reports a stored token, not a project file). The
    harness never reads, stores, or displays a token. `docs/tools.md` documents a
    `credential_env: <ENV_VAR_NAME>` *reference* pattern (naming an environment variable, never
    a value) for a future adapter that needs its own credential — not implemented by any
    Phase 7 adapter, since none needs it, per Requirement 13's "design a credential-reference
    model if needed, but do not build a secret manager."

11. **Two reference adapters, not one: `GitAdapter` (upgraded, local) and `GitHubAdapter`
    (new, genuinely external).** Requirement 15 asks to decide whether `GitAdapter` alone
    proves external integration — it doesn't; it's a local binary already unconditionally
    available throughout the harness, not evidence the routing/permission/audit pipeline
    works for something that actually calls out beyond the local machine. `GitHubAdapter`
    (read-only: `github.repo.read`/`github.pr.read`/`github.issue.read`, via `gh repo
    view`/`gh pr list`/`gh issue list`) is the real external adapter Requirement 14 asks
    for, chosen over Docker because this environment's Docker daemon is unavailable
    (`docker unavailable` in every `eng doctor` run this session) while `gh` is installed
    and authenticated — the lower-risk, actually-exercisable choice.

## Out of scope

- A general MCP JSON-RPC client/transport — `internal/mcpregistry` is a static descriptive
  registry only; `ReferenceMCPAdapter` is a deterministic mock, not a live server
  connection (Decision 9).
- `eng tools install <source>` / `eng mcp add <source>` — the `eng tools` command namespace
  is established (by `eng tools invoke`) so a future `install` verb fits naturally, but no
  installer is built.
- Automatic natural-language → capability detection from raw request text beyond the
  skill-router signal already established in Phase 6 (`Skill.Capabilities`) — building a
  second detection mechanism on top would duplicate Phase 6's own routing work; the
  `## Tools` section added to `buildContextBundle` is driven by *selected skills'*
  capabilities, which is the detection mechanism Requirement 28 sets up for.
- Provider-preference project config (`tools.preferred_provider`) — the router's tie-break
  (alphabetical among available adapters) is deterministic and tested, but no config field
  overrides it yet, since Phase 7 ships zero adapters that actually collide on a capability
  name.
- Any DESTRUCTIVE/HIGH_RISK-capability adapter (PLC write, Modbus write, OPC UA write,
  production deploy, arbitrary SSH exec, destructive DB ops) — explicitly excluded by the
  instruction; the risk model (Decision 6) exists precisely so a future phase can add one
  without redesigning the safety layer.
- Skill install / version pinning / community registry (Phase 6's own deferred items) —
  untouched, per the explicit instruction to not let them distract Phase 7.
- Any domain-specific conditional (`if ESP32`, `if Siemens`, `if DockerProject`) anywhere in
  `internal/toolpolicy`, `internal/toolrouter`, `internal/tooladapter`, or `internal/agent`
  (Requirement 27, hard constraint) — every domain fact flows through capability
  strings/risk tiers/adapter metadata/project policy, never a hard-coded vendor name in Go
  control flow.
- Public plugin marketplace, cloud tool registry, distributed execution cluster, vector
  database, autonomous tool-use swarm — explicitly excluded by the instruction.
