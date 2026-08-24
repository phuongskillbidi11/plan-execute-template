# Spec — [Feature Name]

> **Planner note:** Write this entire file before touching tasks.md.
> Get explicit confirmation from the user before proceeding to tasks.md.
> This file answers WHAT and WHY. tasks.md answers HOW.

---

## Goal

[One paragraph. What user-visible outcome does this sprint achieve?
Write from the user's perspective, not the implementation's perspective.]

**Done looks like:** [One sentence describing the observable end state]

---

## Background

[Why is this needed now? What problem does it solve?
What happens if we don't do it? Keep to 2–3 sentences.]

---

## Design decisions

[For each significant decision, explain what was chosen and why.
Copy key decisions to DECISION_LOG.md as well.]

### Decision 1 — [Title]
- **Chosen:** [approach]
- **Why:** [reasoning]
- **Rejected alternatives:** [what else was considered and why rejected]

---

## Scope

### In scope
- [Bullet list of exactly what this sprint covers]

### Out of scope (explicitly excluded)
- [Bullet list of things that might seem related but are NOT in this sprint]
- [This section is required — Karpathy principle: simplicity first]

---

## Affected files

[List every file that will be created or modified.
Claude Code uses this to avoid touching unrelated files.]

| File | Change type | Reason |
|------|-------------|--------|
| `path/to/file.ext` | Create | [why] |
| `path/to/file.ext` | Modify | [what changes] |

> If this sprint adds a new module/file to the source tree, or changes what an existing
> `docs/src-map.md` entry describes, list `docs/src-map.md` here too (Modify) — its own
> update becomes `tasks.md`'s last task. See `docs/src-map.md`'s own "How to use this file".

---

## Risks and unknowns

[What might go wrong? What assumptions are being made?
What needs to be verified before implementation starts?]

- [Risk 1]
- [Risk 2]

---

## Open questions

[Questions that need answers before or during implementation.
Check these off as they are resolved.]

- [ ] [Question 1]
- [ ] [Question 2]

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|-----------|-----------|-------|-------|
| **Think Before Planning** | spec.md written first; user-observable goal stated | /10 | |
| **Simplicity First** | ≥ 3 out-of-scope items with reasoning | /10 | |
| **Surgical Changes** | Every file listed with change type and exact reason | /10 | |
| **Goal-Driven Execution** | Each scope item traces to an acceptance criterion | /10 | |

**Total: /40 → /10** (threshold: 7/10)

---

**User confirmation received:** [ ] Yes
**Confirmed on:** [DATE]
