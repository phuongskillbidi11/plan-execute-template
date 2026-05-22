# Pre-Planning Thinking Checklist

> Run ALL items before writing the first word of spec.md.
> Internal process — not shown to user.

---

## Phase 1 — Understand codebase state (5 min)

- [ ] Read CLAUDE.md → ## Current state — what exists right now?
- [ ] Read previous sprint-summary.md — what was discovered?
- [ ] Load relevant skills from skills/manifest.json
- [ ] Check DECISION_LOG.md from past sprints — what's decided?

**Gate:** Can I describe codebase state in 3 sentences? If no → read more.

---

## Phase 2 — Understand the request (5 min)

- [ ] Restate feature in my own words — does it match what was asked?
- [ ] What user-visible outcome exists after this sprint that didn't before?
- [ ] Does everything needed already exist? (check Current state)
- [ ] What existing working code must NOT be touched?

**Gate:** Can I write "Done looks like" in one sentence? If no → ask user.

---

## Phase 3 — Apply Karpathy principles (5 min)

- [ ] What is MINIMUM implementation delivering value?
- [ ] What would I cut with half the time?
- [ ] Out of scope section written with 3+ items? (do BEFORE anything else)
- [ ] Any task touching more than 2 files? → split or justify

**Gate:** Out of scope has 3+ items? If no → think harder.

---

## Phase 4 — Validate plan (3 min)

- [ ] Score with plan-quality.md rubric — 7/10+?
- [ ] Any "and then" in any task? → split it
- [ ] Every criterion verifiable by exact command?
- [ ] Every state-changing task has rollback?

**Gate:** Score ≥ 7 AND no "and then" AND all criteria binary? If no → revise.

---

## When to ask vs decide

**Ask user:** ambiguous requirement, missing dependency with multiple fixes,
major long-term consequence decision.

**Decide yourself:** pure impl detail, established codebase pattern,
reversible within sprint, Karpathy principle points clearly one way.

**Never ask:** questions answerable from CLAUDE.md/skills, more than one
question at a time, decisions already in DECISION_LOG.md.
