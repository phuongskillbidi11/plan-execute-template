# Plan Quality Rubric

> Self-evaluate every plan before presenting spec.md to user.
> Score < 7/10 → revise before presenting.
> Include score table at bottom of spec.md.

---

## Scoring (10 points)

### spec.md (4 points)
- **[1] Goal clarity** — user-visible outcome in one sentence, not impl detail
- **[1] Out of scope** — 3+ explicit exclusions with one-line reasons
- **[1] Design decisions** — each: chosen + why + rejected alternatives
- **[1] Risks** — 2+ specific risks with mitigation notes

### tasks.md (4 points)
- **[1] Granularity** — every task: 1 file + 1 symbol, completable <30 min
- **[1] Verification** — every task: exact shell command + pass/fail conditions
- **[1] Rollback** — every state-changing task has rollback procedure
- **[1] Order** — each task only depends on outputs of prior tasks

### tests.md (2 points)
- **[1] Functional test** — at least one user-visible end-to-end check
- **[1] Binary criteria** — no "looks good" or "seems to work" anywhere

---

## Thresholds
- **< 7** — revise before presenting to user
- **7–8** — present with note on weak areas
- **9–10** — present without caveats

---

## Common failure patterns

| Pattern | Symptom | Fix |
|---------|---------|-----|
| "and then" task | Executor asks what to do | Split into two tasks |
| Vague criterion | "works correctly" | Rewrite as exact command + output |
| Missing out of scope | Executor adds unrequested features | Add 3+ explicit exclusions |
| Orphaned task | Task needs symbol not yet created | Reorder — interface before impl |
| 200-line function | Hard to review/test | Split into focused functions |
