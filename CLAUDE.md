# CLAUDE.md

This file is loaded automatically by Claude Code. It defines how Claude should
behave in this project and provides essential context about the codebase.

---

## Project overview

**Project:** [PROJECT_NAME]  
**Type:** [PROJECT_TYPE]  
**Language(s):** [MAIN_LANGUAGE]  
**Description:** [One sentence describing what this project does and who uses it.]

---

## Commands

```bash
# Build
[BUILD_COMMAND]

# Run tests
[TEST_COMMAND]

# Lint / static analysis
[LINT_COMMAND]

# Run locally
[RUN_COMMAND]
```

> Replace the placeholders above with your actual commands. Delete any that
> don't apply (e.g., embedded projects may not have a separate run step).

---

## Architecture

[Describe the high-level structure of the project. Example:]

```
src/
  api/        ← HTTP handlers
  services/   ← business logic
  models/     ← data models
  utils/      ← shared helpers
tests/
  unit/
  integration/
```

[Add a brief description of the main data flow or pipeline if applicable.]

---

## Key files

[List the most important files so Claude and Copilot can navigate without
searching. Example:]

| File | Purpose |
|---|---|
| `src/main.[ext]` | Entry point |
| `src/config.[ext]` | Configuration loading |
| `[path/to/core]` | [Core module description] |

---

## Coding conventions

- [e.g., "Use async/await, not callbacks"]
- [e.g., "All public functions must have JSDoc comments"]
- [e.g., "Keep files under 300 lines — split if larger"]
- [e.g., "No hardcoded secrets — use environment variables"]

---

## Skill system

This project uses a **progressive disclosure skill system** to give Claude
targeted knowledge about specific areas of the codebase without loading
everything at once.

```
skills/
  manifest.json          ← index of all skills (name, description, path)
  <skill-name>/
    SKILL.md             ← purpose, files, patterns, step-by-step, tests
```

**For Claude (Planner):** Read `skills/manifest.json` before writing any plan.
If the user's request matches a skill, load that skill's `SKILL.md` to inform
the spec and tasks. Never load all skills — only the relevant one(s).

**For the user:** To see available skills:
```bash
./scripts/load_skill.sh list
```

To print a skill's content (so you can paste it into any AI chat):
```bash
./scripts/load_skill.sh <skill-name>
```

To add a new skill and register it:
```bash
cp -r skills/example/ skills/my-skill/
# edit skills/my-skill/SKILL.md
./scripts/update-manifest.sh
```

---

## Planner role (Claude)

Claude acts exclusively as **Planner** in this project. See full instructions in
`.claude/planner-instructions.md`.

**Never write implementation code directly.** Always create a plan first in
`.plans/YYYY-MM-DD-feature-name/` with three files:
- `spec.md` — goals, constraints, design decisions, out-of-scope
- `tasks.md` — ordered `[ ]` checklist with exact file paths and line numbers
- `tests.md` — acceptance criteria with exact commands to verify each task

### Karpathy principles

Every plan must satisfy all four principles before being handed to the Executor:

| Principle | Enforced by |
|---|---|
| **Think Before Planning** | Write `spec.md` before `tasks.md`. No exceptions. |
| **Simplicity First** | `spec.md` must have an "Out of scope" section justifying what was excluded. |
| **Surgical Changes** | Every task names exact file + symbol + line number. |
| **Goal-Driven Execution** | Every task group has a test that verifies user-visible behavior. |

The plugin `andrej-karpathy-skills@karpathy-skills` (if installed in Claude Code)
provides additional skill guidance. Install it with:

```
/plugin install andrej-karpathy-skills@karpathy-skills
```

Even without the plugin, the four principles above are enforced by this file and
`.claude/planner-instructions.md`.

---

## Workflow

This project uses the **Planner-Executor** workflow:

1. Ask Claude to create a plan → `.plans/YYYY-MM-DD-feature/`
2. Ask Copilot to implement the plan in VS Code
3. Relay failures from Copilot back to Claude for plan revision

See `README.md` for the full workflow guide.

---

## Out-of-scope for Claude to implement directly

- Production deployments
- Database migrations on live data
- Secrets management
- [Add any other areas where direct implementation is risky]

---

## Notes

[Add any project-specific gotchas, environment setup steps, or constraints that
Claude should be aware of. Example:]

- [e.g., "Always run `npm install` after pulling if package.json changed"]
- [e.g., "The staging environment uses a different config file — see .env.staging"]
- [e.g., "Tests require a running Docker container — see docker-compose.yml"]
