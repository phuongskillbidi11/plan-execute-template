# Relay Prompt Template

> **Multi-executor note:** This template works for any executor
> (Copilot, Codex, Claude Code in executor mode). Replace "Copilot"
> in the prompt with the actual executor name being used.

> **When to use:** Copy this template when a task is marked `[!]` in tasks.md.
> Fill in the blanks and paste the completed prompt into Claude Code.
> Claude Code will revise ONLY the failing task — no other tasks will change.

---

## Template (copy everything below this line)

---

You are the Planner. A task has failed during execution.

**Plan folder:** `.plans/[FOLDER-NAME]/`
**Failed task:** Task [NUMBER] — [TASK TITLE]
**File involved:** `[EXACT FILE PATH]`

**What the Executor attempted:**
[Copy from errors.log — "What was attempted" field]

**Exact error:**
```
[Copy from errors.log — "Exact error output" field — do not summarize]
```

**Last confirmed working state:**
[Copy from errors.log — "Last working state" field]
[List which tasks are `[x]` and must not be changed]

**Attempted fixes that did not work:**
[Copy from errors.log — "Attempted fixes" field, or write "None attempted"]

**Request:**
Please revise ONLY Task [NUMBER] in `.plans/[FOLDER]/tasks.md`.
- Do not change any other tasks
- Do not change spec.md or tests.md unless the error reveals a spec problem
- If the error reveals a spec problem, explain it first and ask for my confirmation before changing spec.md
- Provide the revised task block in full so I can copy it directly into tasks.md

---

## Example (filled in)

---

You are the Planner. A task has failed during execution.

**Plan folder:** `.plans/2026-05-17-sprint-1-db-core/`
**Failed task:** Task 1.3 — Seed questions from JSON file
**File involved:** `ToeicAgent.Data/Database/DbInitializer.cs`

**What the Executor attempted:**
Read `questions_part5.json` using `System.Text.Json` and insert rows into
the `questions` table inside `EnsureCreatedAsync()`.

**Exact error:**
```
System.IO.FileNotFoundException: Could not find file 'questions_part5.json'.
File path used: 'questions_part5.json'
Stack trace:
   at System.IO.File.ReadAllText(String path)
   at ToeicAgent.Data.Database.DbInitializer.SeedQuestionsAsync()
```

**Last confirmed working state:**
Tasks 1.1 and 1.2 are `[x]` — tables are created, connection works.
The JSON file exists at `ToeicAgent.Data/Seed/questions_part5.json`.

**Attempted fixes that did not work:**
- Changed path to `./Seed/questions_part5.json` → same error
- Added `Directory.GetCurrentDirectory()` debug log → shows wrong working dir

**Request:**
Please revise ONLY Task 1.3 in `.plans/2026-05-17-sprint-1-db-core/tasks.md`.
Do not change any other tasks.
Provide the revised task block in full so I can copy it directly into tasks.md.
