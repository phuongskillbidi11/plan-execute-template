# Core Method: Context Manager

Not an AI role — a mechanical selection step any role's session runs before starting real
work, so the harness maximizes *relevant* context instead of total context.

## What it does

Skill selection itself is delegated to `internal/skillrouter.Route` (Phase 6) — explicit
project skills and their dependencies first, then request matches, domain-profile fills,
and recommendations, all budget-aware and explained. This is still the *only* place skill
selection happens; no role prompt or adapter re-implements it.

`eng context bundle <role> <plan-dir> ["<request text>"]` composes:

- **Planner** — matching `docs/src-map.md`/`docs/gotchas.md` sections + matching skills
- **Plan Reviewer** — plan facts (`risk_level`, `requires_approval`) + matching project context
- **Executor** — the current unchecked task block + matching skills (not the whole plan)
- **Verifier** — the plan's `write_scope`/verification rules

## Fail-safe rule

If the request text is empty, too short, or scores zero matches against everything
available, **say so and ask for more specific input or fall back to `strategy: full` for this
one call** — do not invent a plausible-looking selection. A context bundle that silently
picked the wrong things is worse than one that visibly returned nothing.

## Observability

Every `eng context bundle` call writes `<plan-dir>/context-manifest.yaml` recording what was
selected and how much was omitted. Read it when a role behaves as if it's missing context
that should have been included — it answers "why wasn't X selected" without re-running
anything.

## Constraint

Read-only with respect to project source and skills. Its only write is
`context-manifest.yaml`.
