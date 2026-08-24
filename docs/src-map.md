# src-map — what already exists

> **Read this before writing any `spec.md`.** Its entire purpose is to stop the Planner
> (and, downstream, the Executor) from re-discovering or re-inventing code that already
> exists. An incomplete `src-map.md` is what lets the next plan duplicate a module, rename
> something that already has a name, or rebuild a helper that's one directory over.
>
> **Keep this file honest, not exhaustive.** One paragraph per module is enough: what it
> does, the one or two decisions that would surprise a reader, and which `.plans/` phase
> created it. Do not paste code here — link to it.

---

## How to use this file

**As Planner, before `spec.md`:** read every section whose area your feature touches. If a
function, class, or module already does roughly what you're about to design, use it — extend
or call it, don't parallel-build it. If you genuinely need to replace something documented
here, say so explicitly in `spec.md`'s Design decisions section and explain why the existing
version doesn't work, rather than silently duplicating it.

**As Planner, after a plan is confirmed and about to become `tasks.md`:** if the plan adds a
new file, module, or directory under the project's own source tree — or changes what an
existing entry below does — add a task whose only job is updating this file. Put it last in
`tasks.md`, after the feature's own tasks, so the map always describes what actually landed,
not what was planned. A one-line addition is enough; do not let this become its own multi-hour
task.

**As Executor:** treat a `docs/src-map.md`-update task exactly like any other task in
`tasks.md` — implement only what it says, run its verification command, mark it `[x]`.

---

## Format

Group by directory or module. For each one, cover:

- **What it does** — one or two sentences, plain language.
- **Key files** — the two or three files a reader actually needs, not a full listing (the
  filesystem already shows that).
- **Notable design decisions** — anything a reasonable reader would NOT guess by looking at
  the code for thirty seconds (a workaround, a deliberately-not-generic choice, a constraint
  from an external system). Skip this line entirely if there's nothing non-obvious.
- **Where it came from** — the `.plans/` folder that introduced or most recently changed it,
  so a reader who wants the full reasoning knows exactly where to look.

```markdown
### `path/to/module/` — one-line description

What it does: ...

Key files: `foo.ext` (...), `bar.ext` (...)

Notable: ... (omit this line if nothing is non-obvious)

From: `.plans/YYYY-MM-DD-feature-name/`
```

---

## Modules

### `cli/` — the `eng` CLI

What it does: Go CLI (`eng install`, `eng init`, `eng doctor`, `eng scan`,
`eng skills list`) that installs the harness payload globally and links a thin
`.agent/project.yaml` into any project.

Key files: `cli/main.go` (dispatch), `cli/internal/project/project.go` (mode detection),
`cli/internal/skills/skills.go` (global+local skill resolution)

From: `.plans/2026-08-24-v2-harness-foundation/`

### `harness/` — the installable harness payload

What it does: source tree copied by `eng install` into `~/.engineering-harness/` — core
Planner/Executor methodology, the first skill (`engineering/karpathy-guidelines`), the
`software` profile, and the plan templates.

Notable: skills here use YAML frontmatter for metadata; project-local skills without
frontmatter still resolve via the legacy `# Skill:` heading convention — see
`cli/internal/skills/skills.go`.

From: `.plans/2026-08-24-v2-harness-foundation/`

### `cli/internal/planmeta/`, `cli/internal/gitutil/`, `cli/internal/hooks/` — Phase 2 plan lifecycle state

What it does: `planmeta` reads/writes each plan's `plan.yaml` (git SHA, risk level, write
scope, retry counters/budget); `gitutil` wraps `git rev-parse`/`git diff` for drift checks;
`hooks` loads the lifecycle hook table (`harness/hooks/default.yaml`, project-overridable via
`.agent/hooks.yaml`).

Key files: `cli/plan_cmd.go` (`eng plan new/drift/retry`), `cli/verify_cmd.go` (`eng verify`),
`cli/hooks_cmd.go` (`eng hooks run`)

Notable: `eng verify` and `eng hooks run` shell out via `sh -c` — requires a POSIX shell on
PATH (Git Bash's `sh.exe` on Windows), same dependency V1's own scripts always had. Separately,
`harness/hooks/default.yaml`'s built-in commands (`eng scan`, `eng plan drift .`, `eng verify
.`) assume the `eng` binary itself is on PATH — nothing in Phase 1 or Phase 2 installs `eng`
onto PATH (only `eng install` puts the harness *payload* in `~/.engineering-harness/`); a
project using `eng hooks run` today must put `eng` on PATH itself.

From: `.plans/2026-08-24-v2-harness-phase2/`

### `harness/core/triage/`, `harness/core/plan-reviewer/`, `harness/core/verifier/` — new roles

What it does: methodology docs for the three roles Phase 2 adds around the existing
Planner/Executor loop. None of them are enforced by code — `review.md` and
`verify-report.md` are the enforcement surface (a file with a clear verdict), not a gate a
script can close.

From: `.plans/2026-08-24-v2-harness-phase2/`
