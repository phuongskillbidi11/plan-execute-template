# Core Method: Verifier

Independent check that an Executor's own PASS/FAIL self-report is not the only authority on
whether a plan is actually done.

## Role

Run `eng verify [plan-dir]` after the Executor reports all tasks `[x]`. This:

1. Diffs the repository against `plan.yaml`'s recorded `planned_at.git_sha`.
2. Flags any changed file outside the plan's declared `write_scope`.
3. Runs the project's test command (`.agent/project.yaml`'s `stack.test_cmd`).
4. Writes `verify-report.md` with a PASS/FAIL verdict.

## Activation and role verdict (Phase 10)

`eng adapter prompt verifier <plan-dir>` is the recorded activation step. Mechanical `eng
verify` is not the only gate to `COMPLETED` when `workflow.verifier` is enabled — this role's
own judgment, recorded via `eng plan verify-review <plan-dir> --verdict PASS|FAIL`, is required
too. Answer specifically: did the implementation actually satisfy the approved `spec.md`? Are
`tests.md`'s acceptance criteria genuinely met, not just mechanically green? Is the evidence
sufficient? Did execution stay in scope? A fresh Claude Code session for this role is
recommended, where practical, for stronger independence from the Executor's own session — not
enforced, but worth doing when it's easy.

## Full tool output

`eng verify`'s report shows a bounded head+tail summary of the test command's output. If
that isn't enough to diagnose a FAIL, the complete output is at
`.agent/logs/verify-<timestamp>.log` (the report names the exact path when truncated).

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
