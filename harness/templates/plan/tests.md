# Tests — [Feature Name]

> **Executor instructions:** Run every test in this file after ALL tasks are complete.
> A sprint is not done until every test below shows ✅.
> Do not mark tasks.md completion checklist until this file is fully green.

---

## Build gate (run first — if this fails, stop immediately)

```bash
[build command, e.g. dotnet build ToeicAgent.sln OR npm run build]
```
**Pass:** Exit code 0, zero errors
**Fail:** Any compile error → report to Planner before proceeding

---

## Unit tests

```bash
[test command, e.g. dotnet test OR pytest OR npm test]
```
**Pass:** All tests pass, exit code 0
**Fail:** Any test failure → paste full output to Planner

---

## Functional tests

### Test F-1 — [Name]
**Setup:** [any prerequisite state]
```bash
[exact command or step-by-step action]
```
**Pass:** [exact observable outcome]
**Fail:** [what failure looks like]
**Result:** [ ] Pass / [ ] Fail

---

### Test F-2 — [Name]
**Setup:** [prerequisite]
```bash
[command]
```
**Pass:** [outcome]
**Fail:** [failure]
**Result:** [ ] Pass / [ ] Fail

---

## Regression tests (ensure nothing broke)

### Test R-1 — [Previously working feature still works]
```bash
[command]
```
**Pass:** [condition]
**Result:** [ ] Pass / [ ] Fail

---

## Sprint sign-off

- [ ] Build gate: ✅
- [ ] All unit tests: ✅
- [ ] All functional tests: ✅
- [ ] All regression tests: ✅
- [ ] `CLAUDE.md` `## Current state` section updated
- [ ] `DECISION_LOG.md` updated with any new decisions
- [ ] `sprint-summary.md` written

**Sign-off date:** [DATE]
