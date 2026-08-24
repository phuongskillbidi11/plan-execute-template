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

1. Read the project's own context docs (`docs/src-map.md`, `docs/gotchas.md`, or
   `docs/context/*` if present) — do not re-invent what's already documented.
2. Resolve enabled skills (`eng skills list`) and load only the ones relevant to the request.
3. Read prior `.plans/*/DECISION_LOG.md` entries touching the same area.
