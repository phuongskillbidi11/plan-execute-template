# Gotchas — failures that already cost time, most of them silent

> **Read this before writing any `spec.md`**, alongside `docs/src-map.md`. This file exists
> because the second time a mistake happens is more expensive than the first: the first time
> it costs debugging time, the second time it costs debugging time *plus* the embarrassment of
> having already paid for the lesson once. A gotcha that silently produces wrong output or a
> half-working feature is worse than one that crashes loudly — crashes get noticed.

---

## How to use this file

**As Planner, before `spec.md`:** skim this file. If your feature touches an area with an
entry here, design around the gotcha explicitly rather than rediscovering it live.

**As Planner, when a plan's own Executor run hits a real defect** — not a task written
ambiguously, but a genuine "the obvious approach silently does the wrong thing" — add an entry
here as part of closing out that plan. The bar for an entry: would a reasonable engineer
new to this codebase make the same mistake? If yes, it belongs here. If the mistake was
specific to one plan's own bad task wording, it belongs in that plan's own `DECISION_LOG.md`
instead, not here.

**Format:** one entry per gotcha. State the trap, the symptom it produces, and the fix or
rule that avoids it — in that order, so a reader scanning for "have I hit this" can match on
the symptom.

```markdown
### [Short name for the gotcha]

**Trap:** [The thing that looks like it should work, or that most people would reach for.]

**Symptom:** [What actually happens — especially if it's silent: wrong output, not a crash.]

**Fix / rule:** [What to do instead, and why it isn't obvious from reading the code alone.]

**From:** `.plans/YYYY-MM-DD-feature-name/` (optional — only if a specific plan discovered it)
```

---

## Entries

<!--
Delete this comment once the project has real entries. Add one gotcha per confirmed,
non-obvious failure — not for every bug, only ones a fresh engineer would plausibly repeat.
-->

_Nothing documented yet._
