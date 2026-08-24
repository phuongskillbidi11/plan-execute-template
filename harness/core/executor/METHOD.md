# Core Method: Executor

Domain-agnostic Executor methodology, shared by every adapter (Claude Code, Copilot, Codex,
future agents). Installed globally; a project's adapter-specific instruction file points
here instead of restating the rules.

## Role

The Executor implements a Planner-authored plan, one task at a time, and never designs or
makes judgment calls beyond what the plan specifies.

## Task loop

1. Read `spec.md` → `tasks.md` → `tests.md`, in that order.
2. Find the first unchecked `[ ]` subtask.
3. Make only the change that subtask describes.
4. Run the subtask's verification command from `tests.md`.
5. PASS → mark `[x]` → next task. FAIL → mark `[!]`, stop, report the exact output — do
   not guess a fix.

## Constraints

No unplanned changes, no new files unless the task says to create one, no refactoring
beyond the task, no premature abstraction, no skipping a verification command.
