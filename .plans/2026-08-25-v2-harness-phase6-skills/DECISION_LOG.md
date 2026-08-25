# Decision Log — Phase 6

## 1. Skill `level` is an explicit frontmatter field, not derived from directory depth

**Considered:** infer level from path depth (`skills/<domain>/<skill>` → always "domain or
technology", `skills/engineering/<skill>` → always "engineering").

**Rejected because:** `automation/plc` and `automation/modbus` sit in the exact same
directory (`harness/skills/automation/`) but are conceptually different levels — `plc` is
vendor-agnostic methodology (Level 2), `modbus` is a specific protocol (Level 3). Directory
depth cannot distinguish them. An explicit `level:` field costs one YAML line per skill and
is honest about what it means; depth-inference would have been wrong for the very first
real domain this plan adds.

**Chosen:** `level: engineering | domain | technology`, optional, defaults to empty string
(unclassified) for any skill that doesn't set it — including every pre-Phase-6 skill.
Nothing currently reads `level` for routing decisions; it exists for `eng skills list`
display and future filtering, per Requirement 3's "evaluate fields such as... level."

## 2. Recommends do not cascade through forced (budget-immune) required dependencies

**Considered:** after force-adding a required-but-missing skill past the budget, also
collect *its* `recommends` and try to fit them too.

**Rejected because:** this could cascade arbitrarily (a forced skill recommends another,
which if also force-added-later would recommend a third, ...) with no natural stopping
point, directly against Requirement 6's "the router should remain simple and explainable."
It also means a project with a deep `requires` chain could see its context bundle grow in a
way that's hard to predict from the request text alone.

**Chosen:** `recommends` is only collected from skills selected during the primary
pass (Tiers A/B/C, i.e., explicit, matched, or domain-profile skills) — never from a skill
that only entered the final set because something else required it past the budget. This is
a deliberate, bounded stopping rule. If a real project surfaces a case where this feels
wrong, it's a one-line change to `skillrouter.Route`'s final pass — recorded here so that
change is a conscious revision, not a surprise.

## 3. Skill sources: three tiers (global < private < local), not four

**Considered:** the instruction's four-tier model — built-in < user/global-installed <
company/private < project-local.

**Rejected because:** "built-in" and "user/global-installed" are the same location in this
architecture. `eng install --from <path>` always copies an entire `harness/` tree
(built-in skills included) into `~/.engineering-harness/`; there is no second mechanism by
which a user adds a skill to "their global set" that isn't just editing that same directory
or re-running install from a different source. Building a second global directory with
defined precedence over the first, when nothing populates it differently today, is
infrastructure for a distinction that doesn't exist yet — speculative complexity the
instruction's own Requirement 6 ("remain simple") and general "don't over-engineer"
guidance argue against.

**Chosen:** three tiers — `global` (`~/.engineering-harness/skills/`, both "built-in" and
whatever a user has put there by any means), `private` (new, optional, from
`Config.PrivateSkillsPath`), `local` (project's own `skills/`). Precedence
`global < private < local` — exactly the instruction's precedence order with the two
indistinguishable global tiers merged into one. If a future phase introduces a real
separate user-managed global location (e.g. `eng skills install` landing skills somewhere
that survives `eng install --from` without being overwritten), splitting this tier becomes
a one-function change (`ResolveWithPrivate` grows one more root parameter) — recorded here
so that's a traceable, deliberate step, not a redesign.

## 4. `Dependencies`/`dependencies:` renamed to `Requires`/`requires:` rather than added
   alongside it

**Considered:** keep `Dependencies` as-is (matching Requirement 3's "keep the schema
additive") and add a new, separate `Requires` field.

**Rejected because:** `Dependencies` is checked by grep to have zero readers anywhere in
the codebase and zero occurrences of the `dependencies:` YAML key in any committed
`SKILL.md`. It exists only as an unused struct field from an earlier phase. Renaming it
is not a behavior change for any existing consumer (there are none), and keeping two
differently-named fields that mean the same thing would be exactly the kind of
unnecessary-abstraction-vs-duplication tradeoff this project's own CLAUDE.md instructs
against ("three similar lines is better than a premature abstraction" — the inverse is
true here: two fields for one concept is worse than one correctly-named field). "Additive
and backward-compatible" is interpreted here as "no real skill file or reader breaks," which
holds.

## 5. `eng doctor`'s skill listing becomes a summary; full detail moves to existing commands

**Considered:** keep the full per-skill listing in `eng doctor` and additionally print a
summary line above it.

**Rejected because:** Requirement 15 explicitly says not to flood normal doctor output, and
"keep both" doesn't actually solve that — it just adds a line without removing the flood.
`eng skills list` already exists, is unchanged by this plan, and is the correct place for
per-skill detail; `eng skills validate` (new) is the correct place for issue detail.

**Chosen:** `eng doctor`'s skills section becomes exactly four lines (discovered / valid /
warnings / broken dependencies) plus a pointer to `eng skills list` / `eng skills validate`.
This changes doctor's printed text (a human-facing report, not a machine-readable file
anything parses) but changes no file format, so it carries no backward-compatibility risk.

## 6. Router evaluation is a Go test, not a new `eng` subcommand

**Considered:** `eng skills eval` or `eng eval run`, printing pass/fail per scenario.

**Rejected because:** Requirement 16 explicitly frames this as "a small deterministic
evaluation foundation," not a feature end users invoke — the audience is this project's own
CI/regression suite, not a project adopting the harness. A Go test that fails loudly in
`go test ./...` is the more honest fit, and avoids the command-proliferation Requirement 8
already warns against elsewhere in this same phase. If a future phase wants scenario
results surfaced to a human interactively, that's a straightforward addition once there's a
real audience asking for it.
