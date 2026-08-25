# Phase 6 — Multi-Domain Skill Ecosystem + Skill Router Evolution

## Goal

Turn the harness's skill system from "one flat pile of skills, keyword-scored" into a
three-level, dependency-aware, multi-domain skill ecosystem — without making the core
workflow engine know anything about any specific domain, and without regressing any
Phase 1–5 behavior.

A user describing "ESP32 reads Siemens S7-1200 over Modbus TCP" should get
`embedded/esp32`, `automation/siemens-s7` (which pulls in `automation/plc` as a hard
dependency), `automation/modbus`, and `networking/tcp-ip` — and nothing else — without any
`if domain == "embedded"` logic anywhere in `internal/workflow`, `internal/agent`, or
`cli/`.

## What already exists (Phase 1–5) — reused, not rebuilt

Read in full before writing any code for this plan:

- `cli/internal/skills/skills.go` — `Skill` struct already has `Tags`, `Triggers`,
  `Version`, `Dependencies` (defined, **never read** anywhere — dead field), `Conflicts`
  (same), `WhenToUse`/`WhenNotToUse`. `ParseSkillFile` prefers YAML frontmatter, falls back
  to the legacy `# Skill: name` / `## Purpose` heading convention. `Resolve(global, local)`
  merges by bare `Name`, local overrides global.
- `cli/internal/skillmatch/skillmatch.go` — `Score` (substring match against
  tags/triggers/description words) and `Select` (must-include normalization for
  domain-qualified `enabled_skills` entries — the Phase 4 gotcha fix — plus score-ranked
  fill up to a budget).
- `cli/internal/contextcfg/` — `.agent/context.yaml` budget (`strategy`, `max_skills`, …).
- `cli/context_cmd.go` — `selectSkills(dir, request, cfg)` is the **only** call site of
  `skillmatch.Select`; `buildContextBundle` (Phase 5) is the one authoritative
  context-assembly path shared by `eng context bundle` and `eng adapter prompt`.
- `cli/internal/project/project.go` — `Config.EnabledSkills []string`,
  `Config.HarnessProfile string` (single, free-text, written by `eng init`, **never read**
  by anything — vestigial V1 field, left untouched by this plan).
- `harness/skills/engineering/karpathy-guidelines/SKILL.md` — the one real frontmatter
  example in the repo today: `name`, `domain`, `description`, `tags`, `triggers`, `version`,
  `dependencies: []`, `conflicts: []`, `when_to_use`, `when_not_to_use`.
- `cli/doctor.go` — prints every resolved skill, one line each (`eng doctor`).
- `cli/skills_cmd.go` — `eng skills list` only.
- `cli/install.go` — `eng install --from <path>` copies `harness/` wholesale via
  `copyTree` (additive — never deletes files already in `~/.engineering-harness`). Any new
  file under `harness/skills/**` or `harness/evals/**` in this repo reaches the global
  install automatically on the next `eng install --from ..`; no `install.go` change needed.
- `harness/core/context-manager/METHOD.md` — documents the context-assembly flow in prose;
  gets a one-line addition pointing at the router, no structural change.
- `cli/hooks_cmd.go` + `harness/hooks/default.yaml` — the known Phase 5 gotcha:
  `drift_check`/`verify` hard-code `eng plan drift .` / `eng verify .`, so `eng hooks run
  <stage>` only works when invoked from a directory that itself contains `plan.yaml`.

Everything above is reused as-is except where a task explicitly names it.

## Design decisions

1. **Three-level model is metadata, not a rigid directory rule.** `harness/skills/<domain>/
   <skill>/SKILL.md` — the directory segment after `skills/` is the skill's `domain:`
   value. A skill's *level* (`engineering` | `domain` | `technology`) is a new, independent
   frontmatter field (`level:`), not inferred from directory depth — `automation/plc` and
   `automation/modbus` sit in the same directory but are `level: domain` (vendor-agnostic
   methodology) and `level: technology` (a specific protocol) respectively. See Decision Log
   entry 1 for why level isn't derived automatically.

2. **`Dependencies []string` (dead field, `yaml:"dependencies"`) is renamed to
   `Requires []string` (`yaml:"requires"`).** Grep confirms it is defined but never read by
   any code, and no shipped `SKILL.md` uses the `dependencies:` key — renaming an unused
   field is not a behavior change for anyone. `Recommends []string` (soft, `yaml:"recommends"`)
   and `Capabilities []string` (`yaml:"capabilities"`, scored like an extra tag list) and
   `Level string` (`yaml:"level"`) are net-new, optional fields. `Conflicts` stays defined
   and gains one real use: `eng skills validate` warns when a declared conflict target isn't
   installed. Nothing about `ParseSkillFile`'s frontmatter-or-legacy-heading fallback
   changes — a skill with no frontmatter at all keeps resolving exactly as before, with every
   new field at its zero value.

3. **Skill identity for merge/collision purposes becomes domain-qualified.**
   `Skill.QualifiedName()` returns `Domain + "/" + Name` when `Domain` is set and `Name`
   doesn't already contain a `/`; otherwise it returns `Name` unchanged (this covers both a
   self-namespaced `name: company/internal-api` and every legacy skill, whose `Domain` is
   the literal string `"unknown"` from `parseLegacy`). `Resolve`/the new
   `ResolveWithPrivate` merge global/private/local by this qualified key instead of bare
   `Name`. Today there is exactly one skill per bare name across the whole repo, so this is
   a no-op for existing behavior and is covered by a regression test; it only changes what
   happens the day two different domains ship a same-named skill. `Skill.Name` itself is
   **not** rewritten — every existing consumer that prints or matches `.Name` (doctor,
   `eng skills list`, `skillmatch`'s must-include normalization) is unaffected.

4. **Dependency resolution is a new, separate package: `internal/skillgraph`.** It expands
   `Requires` transitively over the full resolved skill set, in deterministic order
   (alphabetical tie-break), detects cycles and reports the exact cycle, treats an unknown
   required skill name as a hard error (never silently dropped), and de-duplicates. It knows
   nothing about scoring, budgets, or requests — pure graph logic, independently testable.

5. **The router is a new, separate package: `internal/skillrouter`.** `skillmatch.Score`
   is reused unchanged for scoring; `skillmatch.Select` is left in place, still tested,
   simply no longer called by `context_cmd.go` (superseded, not deleted — nothing else in
   the codebase calls it, and removing a working, tested function to satisfy "no dead code"
   would be an unrelated cleanup this plan doesn't need). `skillrouter.Route` is the new
   single entry point `context_cmd.go`'s `selectSkills` calls. Precedence, exactly as
   specified by Phase 6 Requirement 7:

   ```
   explicit project-enabled skills          (never dropped)
         ↓
   required dependencies (transitive)       (never dropped)
         ↓
   strong request matches (score > 0)       (best-score-first; weakest cut first if budget runs out here)
         ↓
   profile/domain skills (Domain ∈ project's `domains:` list)
         ↓
   recommended related skills (from any already-selected skill's `recommends:`)
         ↓
   budget cutoff
   ```

   Concretely: Tier A (explicit ∪ their transitive `requires`) is never subject to the
   budget. Tiers B/C (score-ranked matches, then domain-profile fills) are filled greedily
   into the remaining budget, in that order. Tier D (`recommends` collected from every
   skill actually selected in A/B/C) fills whatever budget is left after that. **Then**,
   over the *final* selected set (A∪B∪C∪D), `skillgraph` expands `requires` one more time
   and force-adds anything still missing — ignoring the budget, per Requirement 4/18 ("explicit
   dependencies always included even when normal skill budget is reached"). A skill pulled
   in only by this last forced step does not itself contribute further `recommends` — see
   Decision Log entry 2 for why recommends deliberately do not cascade through forced
   dependencies.

6. **Every selection carries a one-line reason.** `skillrouter.Route` returns
   `Explanation{Skill, Reason}` alongside the skill list. `eng context skills` (extended,
   not a new command — Requirement 8 explicitly prefers this) prints it; `eng adapter
   prompt`'s folded-in context bundle gets it for free through the same `writeSkillSelection`
   helper both call.

7. **Project domains are a new, separate, plural field: `Config.Domains []string`
   (`yaml:"domains,omitempty"`).** This is deliberately not the same as the existing
   singular `HarnessProfile` field (which nothing reads today and this plan leaves alone —
   repurposing a dead field with unclear historical intent is a bigger risk than adding a
   clearly-named new one). Absent (the default for every existing project.yaml) means "no
   domain-profile tier" — Tier C of the router is simply empty, and routing behavior for
   every current project is unchanged.

8. **Skill sources become three tiers, not two: global < private < local.** The instruction
   asked for four (built-in / user-global / company-private / project-local); this repo has
   no real distinction between "built-in" and "user/global-installed" — both live in the
   same `eng install`-managed `~/.engineering-harness/skills/`, and inventing a second global
   location with no real installer to fill it would be speculative infrastructure with
   nothing behind it. Decision Log entry 3 records this simplification. The new tier is
   `private`, sourced from an optional `Config.PrivateSkillsPath string
   (yaml:"private_skills_path,omitempty")` resolved relative to the project root when
   relative; empty (the default) means the private tier is skipped entirely.
   `Resolve(global, local)` becomes a thin wrapper around
   `ResolveWithPrivate(global, "", local)` — byte-identical behavior, covered by a
   regression test — so nothing that already calls `Resolve` needs to change, and the three
   real call sites (`context_cmd.go`, `skills_cmd.go`, `doctor.go`) switch to
   `ResolveWithPrivate` so private packs are visible everywhere skills are, satisfying
   Requirement 19's "one authoritative... path" for resolution as well as selection.

9. **`eng skills validate`.** New subcommand under the existing `eng skills` command
   (`cli/skills_cmd.go`), backed by a new `internal/skillvalidate` package. Checks: missing
   `name`/`description`/`domain` (frontmatter-parsed skills only — **warning**, never
   error, for a legacy heading-only skill, exactly as Requirement 14 requires); a
   `requires`/`recommends` entry that names no resolved skill (**error** for `requires`,
   **warning** for `recommends`); a dependency cycle anywhere in the full installed graph
   (**error**, via `skillgraph`); two distinct files under the *same* root resolving to the
   *same qualified name* (**warning** — a genuine authoring mistake, since `Resolve` would
   silently keep only one; this is deliberately keyed on qualified name, not bare name, so
   the supported cross-domain pattern — `automation/modbus` and a future `networking/modbus`
   sharing a bare name on purpose — never triggers it); a `version` string that isn't roughly `\d+\.\d+(\.\d+)?`
   (**warning**). Exit code is non-zero only when there is at least one **error**.

10. **`eng doctor`'s skill section shrinks to a summary.** Today it prints every resolved
    skill inline; at 9+ skills that's already borderline and the instruction explicitly
    asks doctor to stay short. It becomes `N discovered / N valid / N warnings / N broken
    dependencies` (reusing `internal/skillvalidate`), with `eng skills list` (unchanged,
    full detail) and `eng skills validate` (full issue list) named as where to look next.
    This changes doctor's printed text but not any machine-readable file, so nothing
    downstream depends on the old shape.

11. **Router evaluation is a Go test, not a new CLI verb.** `harness/evals/<domain>/*.yaml`
    scenarios (`request`, `expected_skills`, optional `notes`) are loaded by a new
    `internal/skilleval` package and exercised by an integration-style test in `cli/`
    (package `main`) that runs the real `harness/skills` tree (via `../harness/skills`,
    the same relative-path convention already implicit in this repo's `$REPO/cli/eng`
    bash-test invocations) through `skillrouter.Route` and asserts `expected_skills ⊆
    selected`. Requirement 16 asks for "a small deterministic evaluation foundation," not a
    benchmark platform or a dedicated runner command — the Go test *is* the runner for
    Phase 6.

12. **Hook plan-directory fix is Task 1**, ahead of everything else, per the explicit
    instruction to include it as an early corrective task. `eng hooks run <stage>
    [plan-dir]` gains an optional third argument (default `"."`, i.e. today's exact
    behavior when omitted); `default.yaml`'s `drift_check`/`verify` commands change from
    the literal `.` to a new `${plan_dir}` template token, substituted the same way
    `${test_cmd}` already is. When the argument is omitted, `${plan_dir}` substitutes to
    `.` — byte-identical output to every existing invocation in Phase 2–5's own tests.

## Representative skill set (Requirement 2)

Nine new `SKILL.md` files, chosen to (a) match the instruction's own suggested initial set
almost exactly, (b) prove every mechanism this plan adds (level, domain, requires,
recommends, cross-domain composition), and (c) make the instruction's own headline example
— "ESP32 reads Siemens S7-1200 over Modbus TCP" — resolve exactly as the instruction
describes, end to end, as a real router test rather than a hypothetical:

| Skill | `domain` | `level` | `requires` | `recommends` |
|---|---|---|---|---|
| `engineering/debugging` | engineering | engineering | — | — |
| `engineering/testing` | engineering | engineering | — | — |
| `software/cpp` | software | technology | — | — |
| `embedded/esp32` | embedded | technology | — | — |
| `automation/plc` | automation | domain | — | — |
| `automation/modbus` | automation | technology | — | `networking/tcp-ip` |
| `automation/siemens-s7` | automation | technology | `automation/plc` | — |
| `networking/tcp-ip` | networking | technology | — | — |
| `devops/docker` | devops | technology | — | — |
| `it/linux` | it | technology | — | — |

`automation/siemens-s7`'s `requires: [automation/plc]` is the plan's one real hard-dependency
edge (proving forced, budget-immune inclusion); `automation/modbus`'s `recommends:
[networking/tcp-ip]` is the one real soft edge (proving budget-sensitive inclusion). This
is deliberately not exhaustive — it is the smallest set that exercises every new mechanism
once, per the instruction's own "do not blindly create dozens of empty skills."

## Out of scope

- Splitting "built-in" from "user/global-installed" skill locations (Decision 3) — no real
  installer distinguishes them today; revisit only if a real need appears.
- `eng skills install <source>` / `eng skills update` — architecture-compatible (the new
  `private` tier is exactly where a locally-cloned pack would live) but no command is built;
  Requirement 24 explicitly allows deferring this.
- Skill version *constraints* (`automation/modbus: "^2.0"` pinned in project.yaml) —
  `version:` metadata and a loose validation check exist; constraint solving does not.
- Active conflict avoidance during routing — `Conflicts` gains a validation warning only,
  not router-time exclusion logic; enforcing it would need a policy decision (block? warn?
  auto-pick one?) this plan doesn't have evidence to make yet.
- A dedicated `eng skills eval`/`eng eval run` command — the Go test suite is the Phase 6
  evaluation runner.
- Any domain-specific conditional anywhere in `internal/workflow`, `internal/agent`,
  `internal/contextcfg`, or any other core package (Requirement 26, hard constraint) — every
  domain fact this plan needs flows through `skills.Skill` metadata, `project.Config.Domains`,
  or request text, never a hard-coded domain/vendor name in Go control flow.
- Full MCP marketplace, live Modbus/PLC control, production deployment automation,
  distributed agent execution, vector databases/embeddings, cloud skill registry, automatic
  community marketplace, autonomous multi-agent swarm — explicitly excluded by the
  instruction.
- Rewriting `harness/core/*/METHOD.md` role protocols — Quick Fix and Spec-First already
  route every skill selection through `buildContextBundle` → `selectSkills`, so they inherit
  the new router with zero prose changes beyond one added sentence in
  `context-manager/METHOD.md`.
