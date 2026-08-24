# Sprint Summary — [Feature Name]

> **When to write:** Immediately after all tests.md checks are green.
> **Who writes it:** The Planner (Claude), with input from the Executor.
> **Who reads it:** Claude Code at the start of the NEXT sprint planning session.
>
> Paste this summary into the next planning prompt under "Previous sprint context"
> so Claude Code starts with accurate state instead of assumptions.

---

## Sprint info

| Field | Value |
|-------|-------|
| Sprint name | [e.g. Sprint 1 — DB + Core Scaffold] |
| Plan folder | [e.g. .plans/2026-05-17-sprint-1-db-core/] |
| Start date | [YYYY-MM-DD] |
| End date | [YYYY-MM-DD] |
| Tests | [e.g. 9/9 pass] |

---

## Outcome

**Status:** [ ] Complete / [ ] Partial / [ ] Abandoned

### What was built (matches tasks.md `[x]` items)
- [List everything that was completed and verified]

### What was skipped or deferred
| Item | Reason | Deferred to |
|------|--------|------------|
| [task or feature] | [why skipped] | [Sprint N / Backlog] |

---

## Discoveries (not in the spec)

> Things that were harder, easier, or different than expected.
> This is the most valuable section for future planning.

- [e.g. "Library X requires explicit connection disposal — added using blocks everywhere"]
- [e.g. "DI setup required manual ServiceCollection — not built into the framework"]

---

## Tech debt created

> Things that work but are not clean. Must be addressed before they block future sprints.

| Debt item | Risk if ignored | Target sprint |
|-----------|----------------|---------------|
| [e.g. No error handling in repositories] | [Crashes on DB failure] | Sprint 2 |

---

## Lessons learned

> What would you do differently if starting this sprint again?

- [e.g. "Seed data should be validated against schema before first run"]
- [e.g. "Write the interface before the implementation, not after"]

---

## What the next sprint must NOT assume

- [e.g. "No WinForms code exists yet"]
- [e.g. "Service X is a skeleton only — not implemented"]
