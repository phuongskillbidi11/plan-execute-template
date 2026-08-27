# Core Method: Executor

Domain-agnostic Executor methodology, shared by every adapter (Claude Code, Copilot, Codex,
future agents). Installed globally; a project's adapter-specific instruction file points
here instead of restating the rules.

## Role

The Executor implements a Planner-authored plan, one task at a time, and never designs or
makes judgment calls beyond what the plan specifies.

## Activation (Phase 10)

`eng adapter prompt executor <plan-dir>` is now the recorded activation step, and
`APPROVED → EXECUTING` (or Quick Fix's `TRIAGED → EXECUTING`) will not fire until it has
succeeded — run it before making any change, not after. `eng tools invoke executor ...` is
denied until this activation is on record for the plan. If `tasks.md`'s Completion checklist
already shows complete the moment execution would begin, the transition is refused as a
retroactive-completion invariant violation — never pre-mark the checklist before genuinely
doing the work.

## Task loop

0. Run `eng context bundle executor <plan-dir>` for the current unchecked task and a goal
   summary, instead of re-reading the entirety of `tasks.md` for every task.
1. Read `spec.md` → `tasks.md` → `tests.md`, in that order.
2. Find the first unchecked `[ ]` subtask.
3. Make only the change that subtask describes.
4. Run the subtask's verification command from `tests.md`.
5. PASS → mark `[x]` → next task. FAIL → mark `[!]`, stop, report the exact output — do
   not guess a fix.

## Before starting execution

Run `eng plan drift <plan-dir>`. If it reports `PLAN_DRIFT_DETECTED`, stop — do not execute
against a plan whose source has changed since it was written. Get the plan revalidated first.

## On each test failure

Run `eng plan retry <plan-dir> <build|unit_test|integration_test>` before retrying. If it
reports `RETRY BUDGET EXHAUSTED`, stop — escalate to the Planner or a human instead of
attempting another fix.

## Stop conditions (hard stops, not judgment calls)

Stop and report immediately, without attempting a workaround, when:
- A requirement conflict appears that `spec.md` doesn't resolve
- An unexpected schema change is discovered mid-task
- A dependency mismatch blocks the task as written
- Hardware configuration needed by the task is unknown or undocumented
- `eng plan drift` reports `PLAN_DRIFT_DETECTED`
- The current task is marked `**Requires approval:**` — get explicit human confirmation
  before performing it, every time, with no exception for a previously-approved similar task
- Any operation matches a category in `.agent/project.yaml`'s `require_approval` list

## Constraints

No unplanned changes, no new files unless the task says to create one, no refactoring
beyond the task, no premature abstraction, no skipping a verification command.
