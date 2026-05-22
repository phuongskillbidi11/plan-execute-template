# Skill: example

> This is a template skill. Copy this folder, rename it, and fill in the
> sections below. Then run `./scripts/update-manifest.sh` to register it.
> Add frontmatter metadata below the `# Skill:` line once you fill it in.

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

| File | Role |
|---|---|
| `path/to/file.ext` | What it does |
| `path/to/other.ext` | What it does |

---

## Constraints and gotchas

- [e.g., "Always source the environment before building: `. ~/env/activate`"]
- [e.g., "This module is shared — changes here affect both services A and B"]
- [e.g., "The config file is generated at build time — do not edit it directly"]

---

## Patterns

Show the code patterns, idioms, or conventions used in this area:

```
// Example pattern — rename to the actual language
function doThing(param) {
    // show the expected style, not pseudocode
}
```

---

## Step-by-step (common task)

1. [Step 1 — what to do and in which file]
2. [Step 2]
3. [Step 3 — including any verification step]

---

## Tests and verification

```bash
# Replace with the actual command
[TEST_COMMAND]
```

**Pass:** [what success looks like]
**Fail:** [what to report and to whom]
