---
name: testing
domain: engineering
level: engineering
description: What makes a test worth writing — deterministic, isolated, and testing behavior rather than implementation.
tags: [testing, unit-test, regression, verification]
triggers: [test, testing, "unit test", "regression test", coverage]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Writing or reviewing any automated test, in any language or framework.
when_not_to_use: Manual/exploratory QA sessions — this skill is about automated, repeatable tests.
---

# Skill: testing

## Purpose

Domain-independent rules for what makes a test worth keeping in a suite, as opposed to a
test that merely exists.

## Rules

1. **Deterministic** — no real network calls, no wall-clock time, no unseeded randomness. A
   flaky test is worse than no test: it teaches people to ignore red CI.
2. **Isolated** — a test doesn't depend on another test's side effects or run order.
3. **Test behavior, not implementation** — a test that breaks every time a private function
   is renamed is testing the wrong thing.
4. **One clear failure message** — when it fails, the assertion should say what was expected
   and what happened, not just "false is not true."
5. **A regression test accompanies every real bug fix** — the test should fail against the
   old code and pass against the fix.

## Anti-patterns

- Asserting on incidental output (log text, formatting) instead of the actual contract.
- A test suite that takes so long nobody runs it locally.
- Mocking so much of the system that the test no longer exercises real integration points.
