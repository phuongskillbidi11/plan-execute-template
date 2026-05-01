# Skill: example

> This is a template skill. Copy this folder, rename it, and fill in the
> sections below. Then run `./scripts/update-manifest.sh` to register it.

---

## Purpose

Describe what this skill covers in one or two sentences. This first sentence is
extracted automatically as the manifest description, so make it self-contained.

---

## When to use

List the user requests or task types where Claude should load this skill before
planning:

- "I want to add a [feature]…"
- "How do I [task]…"
- Requests touching files in `[path/]`

---

## Files involved

List the key files this skill relates to. This helps Claude know where to look
and helps Copilot know which files a plan is likely to touch.

| File | Role |
|---|---|
| `path/to/file.ext` | What it does |
| `path/to/other.ext` | What it does |

---

## Constraints and gotchas

List anything a planner must know to avoid common mistakes:

- [e.g., "Always source the environment before building: `. ~/env/activate`"]
- [e.g., "This module is shared — changes here affect both services A and B"]
- [e.g., "The config file is generated at build time — do not edit it directly"]

---

## Patterns

Show the code patterns, idioms, or conventions used in this area. Copilot will
follow these when implementing tasks derived from this skill.

```[language]
// Example pattern — rename to the actual language
function doThing(param) {
    // show the expected style, not pseudocode
}
```

---

## Step-by-step (common task)

If there is a well-known multi-step process for this area, document it here.
Claude will use this to write tasks.md more accurately.

1. [Step 1 — what to do and in which file]
2. [Step 2]
3. [Step 3 — including any verification step]

---

## Tests and verification

How to verify this area is working correctly:

```bash
# Replace with the actual command
[TEST_COMMAND]
```

**Pass:** [what success looks like]  
**Fail:** [what to report and to whom]
