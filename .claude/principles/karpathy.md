# Karpathy Engineering Principles

> Source: Andrej Karpathy's engineering philosophy
> Reference: https://github.com/multica-ai/andrej-karpathy-skills
> Extended for the plan-execute-template workflow.

---

## The 4 principles (verbatim from source)

> **Think Before Coding:** Don't assume. Don't hide confusion. Surface tradeoffs.
> State assumptions explicitly. If uncertain, ask. If multiple interpretations
> exist, present them — don't pick silently. If simpler approach exists, say so.
> If something is unclear, stop. Name what's confusing. Ask.

> **Simplicity First:** Minimum code that solves the problem. Nothing speculative.
> No features beyond what was asked. No abstractions for single-use code.
> No "flexibility" that wasn't requested. No error handling for impossible scenarios.
> If you write 200 lines and it could be 50, rewrite it.
> Ask: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

> **Surgical Changes:** Touch only what you must. Clean up only your own mess.
> Don't improve adjacent code. Don't refactor things that aren't broken.
> Match existing style. Mention unrelated dead code — don't delete it.
> Every changed line traces directly to the user's request.

> **Goal-Driven Execution:** Define success criteria. Loop until verified.
> "Add validation" → "Write tests for invalid inputs, then make them pass"
> "Fix the bug" → "Write a test that reproduces it, then make it pass"
> Strong criteria let you loop independently. Weak criteria require
> constant clarification.

---

## Applied to planning

### Think Before Coding
- Restate feature in your own words before writing spec.md
- Can you write "Done looks like" in one sentence? If not → ask user
- Trigger phrases signaling hidden confusion (replace with questions):
  "I'll assume...", "probably means...", "I think they want..."

### Simplicity First
- Out of scope section MANDATORY — minimum 3 explicit exclusions
- For every abstraction: used in 3+ places? If no → inline it
- For every new dependency: can stdlib handle this?

### Surgical Changes
- Every task: exactly 1 file + 1 symbol
- "Update various files" = invalid task — split it
- Adjacent code not broken → mention in DECISION_LOG, leave alone

### Goal-Driven Execution
- Write success criteria BEFORE implementation approach
- Criteria must be binary: pass or fail, no judgment calls
- "make it work" = invalid criterion → rewrite as exact command

---

## The key insight
> "LLMs are exceptionally good at looping until they meet specific goals.
> Give success criteria, not instructions." — Karpathy

This is why tests.md exists.
