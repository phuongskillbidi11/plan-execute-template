# Skills — levels, metadata, routing, and how to add one

## What a skill is

One `SKILL.md` file under `harness/skills/<domain>/<skill-name>/` (or a project's own
`skills/<domain>/<skill-name>/`), YAML frontmatter plus a markdown body. The frontmatter is
what the harness reads mechanically; the body is what a role reads once the skill is
selected.

## The three levels

- **`level: engineering`** — reusable across every domain (`engineering/debugging`,
  `engineering/testing`). `domain: engineering`.
- **`level: domain`** — methodology specific to one domain but not one vendor/technology
  (`automation/plc`).
- **`level: technology`** — a specific technology, protocol, or vendor
  (`automation/modbus`, `automation/siemens-s7`, `embedded/esp32`).

`level` is metadata for humans and `eng skills list`; nothing in the router currently
branches on it directly (routing runs on tags/triggers/domain/requires/recommends).

## What a skill pack is

A directory under `harness/skills/<domain>/` (or a project's `skills/<domain>/`, or a
private root — see below) containing one or more related skills. There's no separate
manifest file for a "pack" — the domain directory itself is the pack.

## Metadata schema (all optional except `name`)

```yaml
name: modbus                        # required — the skill's identity within its domain
domain: automation                  # groups the skill; also namespaces it (see below)
level: technology                   # engineering | domain | technology
description: One sentence.
tags: [modbus, register]            # scored against a request's text
triggers: [modbus, "40001"]         # scored against a request's text
version: "1.0.0"
requires: []                        # hard dependency — always included when this skill is
recommends: [networking/tcp-ip]     # soft — included only if the context budget allows
capabilities: []                    # free-form, reserved for future capability-based routing
conflicts: []                       # surfaced by `eng skills validate`, not enforced by the router
when_to_use: ...
when_not_to_use: ...
```

A skill with **no frontmatter at all** still resolves via the legacy `# Skill: name` /
`## Purpose` heading convention (`domain: unknown`, every new field at its zero value) —
nothing here requires migrating an old skill.

## Namespacing and collisions

A skill's *qualified name* is `domain/name` (e.g. `automation/modbus`) unless `name:`
already contains a `/` (a self-namespaced name like `company/internal-api`) or `domain` is
empty/`unknown` (a legacy skill, which stays addressed by its bare name). Two skills in
different domains may share a bare name on purpose — `automation/modbus` and a future
`networking/modbus` would not collide. `eng skills validate` warns only when the exact same
qualified name is authored twice under one source root — a real mistake.

**Legacy-vs-qualified bare-name collapse (Phase 9):** a bare-name collision is treated
differently when one side is a legacy (frontmatter-less) skill and the other is qualified — that
combination collapses to whichever tier has precedence (`local` > `private` > `global`), since a
legacy skill sharing a global skill's exact bare name is almost always the same conceptual skill
re-declared without frontmatter (e.g. a pre-harness project-local copy), not a deliberate
cross-domain reuse. A group made entirely of qualified skills (the `automation/modbus` /
`networking/modbus` case above) is never collapsed this way. `eng skills validate` reports which
source won as a `shadowed by ...` warning. See `cli/internal/skills/skills.go`'s
`collapseLegacyDuplicates`.

## Dependencies

`requires:` is transitive and never dropped by the context budget, even when the skill
that needed it was — a `requires` edge means "this skill's advice doesn't make sense
without the other one," not "this is loosely related." Use `recommends:` for that instead.
Cycles and unknown targets are hard errors from `eng skills validate` and from the router
itself (`eng context bundle`/`eng adapter prompt` will report the error rather than
silently drop the broken skill).

## Routing precedence

```
explicit project-enabled skills (.agent/project.yaml's enabled_skills)
      ↓  (never dropped)
required dependencies (transitive)
      ↓  (never dropped)
strong request matches (weighted score >= a minimum threshold — see below)
      ↓  (best score first)
project domain-profile fills (.agent/project.yaml's domains:)
      ↓
recommended related skills
      ↓  (dropped first if the budget is tight)
budget cutoff
```

See it explain itself for any request: `eng context skills "<request text>"`.

**Weighted scoring (Phase 9):** a `tags:`/`triggers:` match is weighted higher than a
`description:` word match, and matching is word/phrase-boundary based (not raw substring
containment) — a description word can no longer accidentally match as a substring of an
unrelated request word, and a single generic description-word match alone is no longer enough
to select a skill (a single tag/trigger match still is). See
`cli/internal/skillmatch/skillmatch.go`'s `TagTriggerWeight`/`DescriptionWordWeight`/
`MinMatchScore` constants for the exact current values — deliberately not restated here, since
they're implementation detail that could drift out of sync with this doc.

## Project profiles (`domains:`)

```yaml
# .agent/project.yaml
domains:
  - embedded
  - automation
```

Optional. Fills the router's domain-profile tier with every skill in those domains that
wasn't already pulled in by a stronger match. Absent (every project before this field
existed) means that tier is simply empty — zero behavior change.

## Skill sources and precedence

```
global (~/.engineering-harness/skills/, from `eng install`)
      <
private (optional, .agent/project.yaml's private_skills_path)
      <
project-local (./skills/)
```

A private pack needs no registry — point `private_skills_path` at any local directory or
git checkout laid out the same way as `harness/skills/`.

## Inspecting routing and validity

- `eng skills list` — every resolved skill, one line each, with source/domain/level.
- `eng skills validate` — metadata/dependency issues; exits non-zero only on an error
  (a legacy skill's missing metadata is always a warning, never an error).
- `eng context skills "<request>"` — what would be selected for a request, and why.
- `eng doctor` — a four-line skill summary; run the two commands above for detail.

## Adding a new skill

1. `mkdir -p harness/skills/<domain>/<name>` (or `skills/<domain>/<name>` in a project for
   a project-local skill).
2. Write `SKILL.md` with frontmatter (copy an existing skill under
   `harness/skills/` as a starting template) plus a body explaining the actual method —
   not what the skill does (the `description:` covers that), but how to apply it.
3. `eng skills validate` — fix anything it flags as an error.
4. If it composes with another skill, add `requires:`/`recommends:` rather than repeating
   that skill's content.
5. If it's meant to be shipped globally, it needs to live under this repo's own
   `harness/skills/` and reach users via `eng install --from <path>`; a private/company
   skill never needs to touch this repo at all — see "Skill sources" above.
