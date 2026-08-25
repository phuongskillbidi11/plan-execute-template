---
name: debugging
domain: engineering
level: engineering
description: Systematic root-cause debugging methodology — reproduce, isolate, form a hypothesis, verify — independent of language or platform.
tags: [debugging, root-cause, troubleshooting]
triggers: [debug, bug, crash, broken, "not working", error, fails, reproduce]
version: "1.0.0"
requires: []
recommends: []
capabilities: [docs.search]
conflicts: []
when_to_use: Any time something behaves unexpectedly and the cause isn't already known.
when_not_to_use: A requirement that's still being designed — that's planning, not debugging.
---

# Skill: debugging

## Purpose

A domain-independent method for finding the actual cause of a defect instead of guessing at
fixes.

## Method

1. **Reproduce** — get a reliable, minimal repro before touching any code.
2. **Isolate** — bisect the change/input space until the smallest failing case is known.
3. **Form a hypothesis** — state what you believe is wrong and how you'd know if you're right.
   Searching `docs/gotchas.md`/prior notes for a similar symptom (via `docs.search`, when
   available) often shortcuts straight to a known cause.
4. **Verify** — test the hypothesis directly (a log line, a debugger breakpoint, a targeted
   assertion) before writing the fix.
5. **Fix at the cause**, not the symptom — and add a regression test that would have caught it.

## Anti-patterns

- Changing code to see if it helps without a hypothesis.
- Fixing the first suspicious-looking line instead of the line the reproduction actually
  implicates.
- Declaring victory because the symptom disappeared, without knowing why.
