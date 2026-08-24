---
name: karpathy-guidelines
domain: engineering
description: Core behavioral rules preventing the most common LLM coding mistakes. Load alongside any domain skill — applies universally.
tags: [planning, methodology, universal]
triggers: []
version: "1.0.0"
dependencies: []
conflicts: []
when_to_use: Always — load first, before any domain-specific skill.
when_not_to_use: ""
---

# Skill: karpathy-guidelines

> Universal behavioral rules — load alongside any domain skill.
> Source: https://github.com/multica-ai/andrej-karpathy-skills
> Last updated: 2026-05-26 | Confidence: high | Status: stable

---

## Purpose

Core behavioral rules preventing the most common LLM coding mistakes.
Load alongside any domain skill — applies universally.

---

## When to use

Always. Load first, before any domain-specific skill.

---

## The 4 principles in practice

### 1. Think Before Coding
- Never pick an interpretation silently — present options and ask
- If simpler solution exists, say so before planning the complex one
- Stop and ask rather than proceed with hidden confusion

### 2. Simplicity First
- Out of scope section mandatory — minimum 3 items
- No new abstraction unless used in 3+ places
- No new dependency unless stdlib cannot handle it

### 3. Surgical Changes
- Every task: exactly 1 file + 1 symbol
- "Update various files" = invalid task — split it
- Broken-but-unrelated code: mention in DECISION_LOG, don't touch

### 4. Goal-Driven Execution
- Write success criteria BEFORE implementation approach
- Criteria must be binary — no judgment calls
- "make it work" = invalid → rewrite as exact command + expected output

---

## Anti-patterns

| Anti-pattern | Fix |
|---|---|
| Task with "and then" | Split into two tasks |
| "works correctly" criterion | Exact command + expected output |
| Missing out of scope | 3+ explicit exclusions |
| Task touches 3+ files | Split or justify in spec.md |

---

## Files involved

| File | Role |
|---|---|
| `.claude/principles/karpathy.md` | Full Karpathy principles with applied examples |
| `.claude/principles/plan-quality.md` | 10-point self-evaluation rubric for every plan |
| `.claude/principles/thinking-checklist.md` | Pre-planning checklist (run before writing spec.md) |
| `.claude/planner-instructions.md` | Claude's full role definition as Planner |
