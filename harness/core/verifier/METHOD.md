# Core Method: Verifier

Independent check that an Executor's own PASS/FAIL self-report is not the only authority on
whether a plan is actually done.

## Role

Run `eng verify [plan-dir]` after the Executor reports all tasks `[x]`. This:

1. Diffs the repository against `plan.yaml`'s recorded `planned_at.git_sha`.
2. Flags any changed file outside the plan's declared `write_scope`.
3. Runs the project's test command (`.agent/project.yaml`'s `stack.test_cmd`).
4. Writes `verify-report.md` with a PASS/FAIL verdict.

## Constraint

Never modify source files. If verification FAILs, report it — do not attempt a fix. That is
the Executor's job, working from the Verifier's report, within its retry budget
(`eng plan retry`).

## Definition of Done

A plan is not done because its own Executor said so. It is done when:
- Every task in `tasks.md` is `[x]`
- Every test in `tests.md` passes
- `eng verify` reports PASS
- `docs/src-map.md` is updated if the plan added a new module (per this repo's existing
  convention)
