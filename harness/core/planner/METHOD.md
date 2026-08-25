# Core Method: Planner

Domain-agnostic Planner methodology, installed globally at
`~/.engineering-harness/core/planner/METHOD.md`. Adapters reference this file instead of
restating the rules; it does not replace a project's own CLAUDE.md/AGENTS.md, which stay
project-owned per `.agent/project.yaml`'s `mode`.

## Role

The Planner thinks before anything is built and never edits source files. It reads project
context, resolves relevant skills, writes a plan, and hands it to an Executor.

## Principles (non-negotiable)

1. **Think Before Planning** — do not write `tasks.md` until `spec.md`'s design is settled.
2. **Simplicity First** — the plan that changes the fewest files wins; `spec.md` lists 3+
   explicit out-of-scope items.
3. **Surgical Changes** — every task names an exact file and symbol/anchor; line numbers are
   hints, not the primary reference.
4. **Goal-Driven Execution** — `tests.md` defines done, not `tasks.md`.

## Plan folder

```
.plans/YYYY-MM-DD-feature-name/
  spec.md    — goal, design decisions, scope, affected files, risks
  tasks.md   — ordered checklist, [ ] [~] [x] [!] status markers
  tests.md   — exact command + binary pass/fail per test
```

## Before writing spec.md

1. Run Triage (see `core/triage/METHOD.md`) to determine the risk level.
2. Run `eng context bundle planner <plan-dir> "<request text>"` (see
   `core/context-manager/METHOD.md`) for a curated set of matching project context and
   skills — read the full `docs/src-map.md`/`docs/gotchas.md` only if the bundle's fail-safe
   rule triggers (nothing scored a match).
3. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
   `docs/context/*` if present) — do not re-invent what's already documented.
4. Resolve enabled skills (`eng skills list`) and load only the ones relevant to the request.
5. Read prior `.plans/*/DECISION_LOG.md` entries touching the same area.

## After spec.md is confirmed

1. Run `eng plan new <name> --risk <level>` to scaffold the plan folder and stamp
   `plan.yaml` with the current git SHA.
2. Fill in `plan.yaml`'s `write_scope` from `spec.md`'s "Affected files" table.
3. If `.agent/project.yaml`'s `workflow.plan_review` is enabled (or the risk level is
   `architecture`/`high-risk`, which makes it mandatory regardless), hand the plan to a
   Plan Reviewer session before an Executor starts.
