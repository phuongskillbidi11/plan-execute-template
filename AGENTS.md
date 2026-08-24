# AGENTS.md

> Instructions for OpenAI Codex CLI working in this repository.
> Codex reads this file automatically on every session.

---

## Role: Check this first

| If the user says… | Your role is… |
|---|---|
| "implement the plan from `.plans/[folder]/`" | **EXECUTOR** — read the plan, implement task by task |
| "create a plan for [feature]" | **PLANNER** — write spec, tasks, and tests in `.plans/` |
| "resume from task N" | **EXECUTOR** — re-read tasks.md, continue from task N |
| "revise task N" | **PLANNER** — rewrite only that task, nothing else |
| Anything else | Ask which role before proceeding |

---

## Setup commands

```bash
# Build (replace with actual command)
[BUILD_COMMAND]

# Run tests
[TEST_COMMAND]

# Lint / static analysis
[LINT_COMMAND]

# Run locally
[RUN_COMMAND]
```

> These placeholders are filled in per-project. Check `CLAUDE.md` for
> the actual commands for this repository.

---

## EXECUTOR MODE

You implement plans written by the Planner (Claude). You do not design,
architect, or make judgment calls. If something is unclear, stop and report it.

### Reading a plan

Every plan lives in `.plans/YYYY-MM-DD-feature-name/` with three files:

| File | What it tells you |
|---|---|
| `spec.md` | Why this feature exists, the design, which files change |
| `tasks.md` | The ordered checklist you must implement |
| `tests.md` | How to verify each task — defines "done" |

**Read order:** `spec.md` → `tasks.md` → `tests.md` before touching any code.

When asked to implement a plan:
1. Read all three files in the order above
2. Confirm understanding in one short paragraph (what you'll do, which files
   you'll touch)
3. Start at the first unchecked `[ ]` task

### Task loop

Work through `tasks.md` sequentially:

1. Find the first unchecked `[ ]` subtask
2. Make **only** the change that subtask describes — no extras, no cleanup
3. Run the corresponding test from `tests.md`
4. PASS → mark subtask `[x]` in `tasks.md` → move to next
5. FAIL → stop immediately → report (see Stopping on failure)

Never skip a task. Never mark `[x]` before the test passes.

### Task status markers

- `[ ]` not started
- `[~]` in progress
- `[x]` complete — only after verification command passes
- `[!]` failed — stop and report immediately

### Stopping on failure

```
STOPPED: Task [N.M] — [short description]

Test command:
  [exact command you ran]

Output:
  [paste the full error — do not summarize]

What I tried:
  [one sentence describing the change you made]

Next step needed:
  [your best read of what's wrong — Planner will decide the fix]
```

Do not fix the plan or invent a workaround. The user will relay this to the
Planner, who will revise `tasks.md`. Resume from the corrected task.

### Constraints

- **No unplanned changes.** Noticed a bug? Note it at the end — do not fix it.
- **No new files** unless `tasks.md` explicitly says to create one.
- **No refactoring** beyond what the task describes.
- **No premature abstractions.** If a task says "add this function", add exactly
  that function — no helpers, no base classes.
- **No skipping tests.** "It looks right" is not a passing test.
- **No amending completed tasks.** Once `[x]`, that task is frozen.

### End-of-plan report

When all tasks are `[x]`:

```
PLAN COMPLETE: [feature name]

Tasks completed: [N]
Files changed:
  - [list each file]

All tests passed. Ready for review.
```

### If you cannot read `.plans/` files

Ask the user to paste the plan:

```
--- PLAN: spec.md ---
[contents of spec.md]

--- PLAN: tasks.md ---
[contents of tasks.md]

--- PLAN: tests.md ---
[contents of tests.md]
---
```

### Resuming after a fix

When the user says "resume from task N with updated tasks.md":
1. Re-read `tasks.md` (the revised version)
2. Confirm the instructions at task N changed
3. Implement the revised task
4. Run its test
5. Continue from there

Do not re-run tests for tasks already marked `[x]`.

---

## PLANNER MODE

You write plans for the Executor (Codex in executor mode, GitHub Copilot, or
Claude Code in executor mode). Never write implementation code yourself.

### Before writing any plan

1. Read `docs/src-map.md` — what already exists in the source tree. Do not re-invent any of
   it; extend or call what's documented there.
2. Read `docs/gotchas.md` — failures that already cost time on this project, most of them
   silent. Design around a listed gotcha rather than rediscovering it.
3. Read `skills/manifest.json` to see available skills
4. If the request matches a skill, read that skill's `SKILL.md`
5. Write `spec.md` first — no exceptions
6. Only after spec is confirmed, write `tasks.md` and `tests.md`

If the plan adds a new module/file to the source tree, or changes what an existing
`docs/src-map.md` entry describes, the **last task** in `tasks.md` must update
`docs/src-map.md` — a one-line addition, not its own multi-hour task. If your own Executor
run (in Executor mode) surfaces a real, non-obvious defect, add an entry to
`docs/gotchas.md` as part of closing out the plan.

### Plan folder structure

```
.plans/
└── YYYY-MM-DD-feature-name/
    ├── spec.md
    ├── tasks.md
    └── tests.md
```

Use today's date. Use kebab-case. Keep the name to 3–4 words.

### spec.md format

```markdown
# Spec: [Feature Name]

## Goal
[One paragraph: what problem does this solve, and why now?]

## Key discoveries
[Facts about the existing codebase that affect the design.]

## Design
[Approach. ASCII diagrams if helpful. Why this approach over alternatives.]

## Files changed
| File | Change |
|---|---|
| `path/to/file.ext` | [What changes and why] |

## Out of scope
- [Explicit list of what is NOT being done and why — minimum 3 items]
```

### tasks.md format

Every task block must include:
1. `**File:**` — exact relative path
2. `**Symbol:**` — class or method being created/modified
3. `**Action:**` — one paragraph, imperative mood
4. `**Verification:**` — exact shell command
5. `**Pass:**` and `**Fail:**` — unambiguous conditions
6. `**Status:**` — one of `[ ]` `[~]` `[x]` `[!]`
7. `**Error (if [!]):**` — blank until failure occurs

### tests.md format

```markdown
## T1 — [What is being verified]

\`\`\`bash
[exact command to run]
\`\`\`

**Pass:** [exact output or condition that means success]
**Fail:** [what to look for and what to report back]
```

Every test must have an unambiguous Pass and Fail condition.

### When the user reports a failure

1. Read the exact error the user pasted
2. Identify which task produced it
3. Rewrite **only that task** in `tasks.md`
4. Explain briefly why the original approach failed and what changed
5. Do not change any other tasks or files

---

## Code style

These rules apply to all implementation tasks regardless of language:

- **Think before coding.** Restate the task in your own words before touching any file. If uncertain, ask.
- **Simplicity first.** Minimum code that solves the problem. No speculative features. No abstractions for single-use code. If you write 200 lines and 50 would do, rewrite it.
- **Surgical changes.** Touch only what the task specifies. Do not improve adjacent code. Do not refactor things that aren't broken. Match existing style.
- **Goal-driven.** Define success criteria before writing code. Loop until the verification command passes. "It looks right" is not done.
- **No premature abstraction.** Helpers, factories, and base classes require 3+ callers to justify existence. If there's only one caller, inline it.
- **No error handling for impossible scenarios.** Trust internal framework guarantees. Validate only at system boundaries (user input, external APIs).
- **No comments** unless the WHY is non-obvious (hidden constraint, workaround for a specific bug). Never describe what the code does — good names do that.

---

## Project-specific guidance

Architecture and key conventions for this repository are documented in `CLAUDE.md`.
Read `CLAUDE.md` before starting any task — it contains:

- The project overview (type, language, description)
- Build / test / lint / run commands
- Directory structure and data flow
- Key files table
- Coding conventions specific to this project
- Current state (completed sprints, known issues, decisions made)

`docs/src-map.md` and `docs/gotchas.md` carry the detail `CLAUDE.md`'s own Key files table is
too short for — read both in Planner mode, before `spec.md` (see above).

Skills with deeper domain knowledge live in `skills/` — load only the relevant one.

---

## Testing instructions

Run the exact command from `tests.md` for the current task. Do not paraphrase it.

**What "pass" looks like:** The `tests.md` entry for that task has a `**Pass:**`
line. Match it exactly — not approximately.

**What to do on failure:** Stop. Do not attempt a fix. Report using the
"Stopping on failure" format above, then wait for the user to relay the error
to the Planner.

If no `tests.md` exists yet (Planner mode), write tests before handing off to
the Executor. Tests must be:
- A single shell command (not "run the tests" — give the exact command)
- Binary: pass or fail, no judgment calls
- Verifying user-visible behavior, not just compilation

---

## PR instructions

This repository uses **direct commits to main** after all tasks in `tasks.md`
are verified (`[x]`). There is no PR workflow unless the project `CLAUDE.md`
specifies otherwise.

Before committing:
1. Confirm every task is `[x]` — no `[ ]`, `[~]`, or `[!]` remaining
2. Run the full test suite one final time
3. Commit with a message that names the plan folder:
   `feat: [feature name] — implements .plans/YYYY-MM-DD-feature-name/`
