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

<!--
Delete this comment and everything below it once the project has real modules to document.
Add one section per module/directory as it's built — via the update-src-map task described
above, not in a separate cleanup pass.
-->

_Nothing documented yet — this file is created empty by `scripts/init.sh` and grows one
section per feature, via the last task of that feature's own `tasks.md`._
