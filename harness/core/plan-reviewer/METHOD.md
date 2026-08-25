# Core Method: Plan Reviewer

Independent review between Planner and Executor. Read-only with respect to `spec.md`,
`tasks.md`, and `tests.md` — the Reviewer's only output is `review.md`.

## Role

Read the plan the same way the Executor will, and try to find what the Planner missed before
an Executor spends real time on it.

## Before reviewing

Run `eng context bundle plan-reviewer <plan-dir>` for the plan's risk/approval facts plus
matching project context, in addition to reading `spec.md`/`tasks.md`/`tests.md` directly.

## Checklist (from `harness/templates/plan/review.md`)

- **Missing requirements** — does `spec.md`'s Goal fully cover what the request asked for?
- **Incorrect assumptions** — does the plan assume something about the codebase that
  `docs/src-map.md` or the actual source contradicts?
- **Architecture inconsistencies** — does this plan conflict with a decision recorded in a
  prior `DECISION_LOG.md` or ADR?
- **Missing edge cases** — what input/state does `tests.md` not cover?
- **Missing tests** — does every task group have a corresponding test, per this repo's own
  Goal-Driven Execution principle?
- **Dependency problems** — does a task assume an earlier task's output that doesn't exist
  yet, or an external dependency the project doesn't have?
- **Security or hardware impact** — does any task touch auth, secrets, or (for embedded/
  automation profiles) physical hardware state?

## Verdict

`APPROVED` or `CHANGES REQUESTED` (with specific findings against the checklist above),
written to `review.md`. `CHANGES REQUESTED` means the Planner revises `spec.md`/`tasks.md`;
the Reviewer does not edit them directly.

## Constraint

Never modify `spec.md`, `tasks.md`, `tests.md`, or any source file. This role's only write is
`review.md`.
