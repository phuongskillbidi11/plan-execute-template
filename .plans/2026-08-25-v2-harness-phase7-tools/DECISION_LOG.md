# Decision Log — Phase 7

## 1. Revise `tooladapter.Adapter`'s interface rather than adding a parallel one

**Considered:** leave the Phase 5 interface (`Capability() string` singular,
`PermissionLevel() string` adapter-wide) untouched and add a second, richer interface
(`AdapterV2` or similar) for new adapters only.

**Rejected because:** Phase 5's own DECISION_LOG is explicit that this interface is
"foundation only" and `GitAdapter` exists "to prove the interface compiles and is testable,
not as a real capability gate." A second parallel interface would mean the router has to
know about two adapter shapes forever, and every future adapter author has to guess which
one to implement. Since the only real consumers (`GitAdapter`, its two test files) are
small and entirely internal to this repo, revising the one interface and fixing its two
consumers in the same task is less total risk than permanently forking it.

**Chosen:** `Adapter` gains `Provider() string`, `Version() string`,
`Capabilities() []toolcap.Capability` (plural, risk-tagged, replacing singular
`Capability()`), and `Invoke(capability string, args []string, dir string) (string, error)`;
`PermissionLevel()` is removed (superseded by per-capability `Risk`, which is strictly more
expressive — an adapter can expose both a `READ` and a `WRITE` capability, which one
adapter-wide string could never say). `Doctor()` is kept unchanged (name and meaning).

## 2. Do not repurpose `project.Config.RequireApproval`

**Considered:** rename/reuse the existing top-level `RequireApproval []string` field for
the new capability require-approval list, since the name is an almost-exact match.

**Rejected because:** it has zero readers in the codebase (confirmed by grep) and zero
documentation of intended contents — it could have meant "risk levels that require
approval" (`architecture`, `high-risk`) just as plausibly as "capability names." Guessing
wrong and silently changing behavior for a field some existing (even if hypothetical)
project.yaml already populated is a worse outcome than adding one clearly-named, clearly
nested field. The instruction's own example schema is nested (`tools: {allow,
require_approval, deny}`) anyway, which a flat top-level field can't represent without a
second, confusing top-level list.

**Chosen:** a new `Tools toolpolicy.Policy` field, nested exactly as the instruction's
example shows. `Config.RequireApproval` is left exactly as it was — untouched, still
unused, still available for a future phase to define if its original intent is ever
recovered or decided.

## 3. `Skill.Capabilities` stays a flat `[]string`; capabilities from skills are always
   treated as soft (recommended) signals

**Considered:** change `Skill.Capabilities` from `[]string` to a struct
(`{Requires []string; Recommends []string}`) matching the instruction's
`capabilities: {recommends: [...]}` example literally.

**Rejected because:** every one of the 11 skills this repo ships today writes
`capabilities: []` — an empty YAML *sequence*. Redefining the Go field to a struct type
would make `yaml.v3` fail to unmarshal every one of those files (sequence vs. mapping type
mismatch), a real regression for zero actual benefit, since nothing in Phase 6 or 7
needs a hard "this skill cannot function without capability X" concept — a skill's
usefulness has never depended on an external tool being available.

**Chosen:** keep the field exactly as Phase 6 shipped it. Every capability name a skill
lists is a *recommendation* fed into the tool router (mirroring exactly how a skill's own
`recommends:` list is a soft signal into the skill router, not a hard one) — consistent
terminology, zero schema risk.

## 4. Two role-permission tables, not one combined table

**Considered:** fold the new risk-ceiling concept into the existing `RolePermissions` map
by changing its value type from `[]string` (adapter names) to something richer per adapter
(e.g. `map[string]toolcap.Risk`) that also encodes the max risk allowed for that adapter.

**Rejected because:** Requirement 5's model is risk-tier-first, not adapter-first — "Planner
→ READ only" applies to *every* adapter equally, not one at a time. Encoding it per-adapter
would mean listing every adapter × role combination by hand and manually keeping every new
adapter's entry in sync across every role, which is exactly the kind of maintenance trap a
single risk-ceiling-per-role table avoids. It would also silently reintroduce the collapse
Requirement 5 explicitly forbids: "adapter available" and "role allowed" would live in the
same map cell again, just with an extra field, rather than being two genuinely independent
checks.

**Chosen:** `RolePermissions` (coarse, adapter-toolbox, Phase 5, unchanged) and the new
`RoleMaxRisk` (fine, risk-ceiling) are two separate maps in the same file, both consulted by
`toolpolicy.Decide`. A capability is usable by a role only if it passes *both* — the
adapter must be in that role's toolbox *and* the capability's risk must be at or below that
role's ceiling.

## 5. Provider precedence: alphabetical among available adapters, no config field yet

**Considered:** add a `tools.preferred_provider: {capability: adapter_name}` map to
project.yaml now, per Requirement 30's suggested model, even though no two shipped Phase 7
adapters collide on a capability name.

**Rejected because:** there is nothing real to configure yet — `git.*` and `github.*` and
`docs.search` never overlap across `GitAdapter`/`GitHubAdapter`/`ReferenceMCPAdapter`.
Adding a config field with no adapter that could ever need it is exactly the kind of
speculative infrastructure this project's own conventions (and Requirement 2's "do not
over-generalize into a large plugin framework yet") argue against.

**Chosen:** the router's tie-break logic (deterministic: sort available candidate adapters
by `Name()`, take the first) is implemented and tested now with two synthetic fake adapters
that *do* collide on a capability name, so the mechanism is proven — but no project.yaml
field exists to override it. Adding `tools.preferred_provider` later, when a second real
adapter for the same capability actually ships, is a small, additive, low-risk change.

## 6. GitHub over Docker as the second reference adapter

**Considered:** Docker inspect/read-only, per the instruction's own second suggestion.

**Rejected because:** every `eng doctor`/`eng capabilities list` run in this environment
this whole session reports `docker unavailable` — there's no running Docker daemon to
actually exercise a Docker adapter's `Available()`/`Invoke()` path in this environment,
which would leave the "reference" adapter permanently untested end-to-end here.

**Chosen:** GitHub, via the `gh` CLI — confirmed installed and authenticated in this
environment (`gh auth status` shows an active, keyring-stored token) — giving a real,
live-testable external adapter without the harness ever touching a credential itself.

## 7. MCP registry is descriptive-only; the mock adapter is matched by name, not driven

**Considered:** have `internal/mcpregistry`'s loaded entries dynamically instantiate
adapters via a generic transport dispatcher (e.g. `transport: mock` → look up a registered
Go constructor by string).

**Rejected because:** with exactly one entry and one Go implementation, a dispatch registry
is speculative machinery for a problem that doesn't exist yet (one name, one mapping).
Requirement 11 asks for a "minimal" foundation, not a plugin-loading mechanism.

**Chosen:** `harness/mcp/servers.yaml` is read for display/inspection (`eng doctor`, `eng
capabilities explain`) and to seed `ReferenceMCPAdapter`'s declared capability list (so the
registry file is the source of truth for *what* it exposes, not a redundant copy); the one
entry whose `name` is `docs-search` is matched, by name, to the one Go constructor that
exists. A second real adapter would need one new `case` in that small match, not a new
subsystem — deferred until there's a second one to justify it.
