# Tasks — [Feature Name]

> **Executor instructions:** Complete groups in order unless noted parallel.
> After each task, run the verification command. Mark status before moving on.
>
> **Status markers:**
> - `[ ]` — not started
> - `[~]` — in progress (currently being worked on)
> - `[x]` — complete (verification command passed)
> - `[!]` — failed (stop here, report error to Planner)
>
> **Rule:** Only one task may be `[~]` at a time.
> Never mark `[x]` without running the verification command first.
> When a task becomes `[!]`, write the exact error below the task line
> and stop. Do not attempt the next task.

---

## Group 1 — [Group name, e.g. "Project scaffold"]

### Task 1.1 — [Short imperative title]
**File:** `path/to/file.ext`
**Symbol:** `ClassName.MethodName()`
**Action:** [Exact description of what to create/modify]

```
// Paste the exact signature or key lines here if helpful
```

**Verification:**
```bash
[exact command to run]
```
**Pass:** [what success looks like — exit code, output string, etc.]
**Fail:** [what failure looks like]

**Status:** `[ ]`
**Error (if [!]):**
> _Leave blank until task fails_

---

### Task 1.2 — [Short imperative title]
**File:** `path/to/file.ext`
**Symbol:** `ClassName`
**Action:** [Exact description]

**Verification:**
```bash
[exact command]
```
**Pass:** [success condition]
**Fail:** [failure condition]

**Status:** `[ ]`
**Error (if [!]):**
> _Leave blank until task fails_

---

## Group 2 — [Group name]

### Task 2.1 — [Title]
**File:** `path/to/file.ext`
**Symbol:** `MethodName()`
**Action:** [Description]

**Verification:**
```bash
[command]
```
**Pass:** [condition]
**Fail:** [condition]

**Status:** `[ ]`
**Error (if [!]):**
> _Leave blank until task fails_

---

## Completion checklist

- [ ] All tasks marked `[x]`
- [ ] No tasks marked `[!]`
- [ ] Build passes
- [ ] All tests pass
- [ ] Sprint summary written to `.plans/[folder]/sprint-summary.md`

---

## Rollback procedures

> **When to use:** A task is marked `[!]` AND the codebase is in a broken state
> (build fails, DB corrupted, file partially written).
> Run the relevant rollback before relaying the error to the Planner.
> After rollback, the codebase should be back to the last `[x]` state.

### General rollback (always safe to run)
```bash
# Restore any accidentally modified files using git
git diff --name-only          # see what changed
git checkout -- [file-path]   # restore a specific file
```

---

## Per-task rollback field

> Each task block optionally includes a `**Rollback:**` field.
> The Planner fills this in for tasks that modify shared state
> (DB schema, global config, package manifests, file renames).
> Simple file creation tasks do not need a rollback field.

### Example task block with rollback field:

### Task 1.3 — Seed initial data
**File:** `src/database/seeder.py`
**Symbol:** `Seeder.seed_questions()`
**Action:** Read `seed/questions.json` and insert all rows into the `questions`
table inside a single transaction.

**Rollback:** If this task fails and leaves partial rows, run:
`DELETE FROM questions;` in the DB console. Then re-run from Task 1.1.

**Verification:**
```bash
python -m pytest tests/test_seeder.py
```
**Pass:** All seeder tests pass
**Fail:** Any test failure or exception during seed

**Status:** `[ ]`
**Error (if [!]):**
> _Leave blank until task fails_
