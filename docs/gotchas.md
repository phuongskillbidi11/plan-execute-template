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

### `eng hooks run` commands assume `eng` itself is on PATH

**Trap:** `harness/hooks/default.yaml`'s built-in commands are written as `eng scan`,
`eng plan drift .`, `eng verify .` — plain invocations of the CLI by name, the way you'd expect
any installed tool to be called from a hook.

**Symptom:** Running `eng hooks run <stage>` from a shell that only has the locally-built
`cli/eng` binary (not on PATH) fails with `sh: line 1: eng: command not found` for every hook
that shells out to `eng` itself — silent-looking until you notice the hook's `->` line printed
the command that then immediately errored.

**Fix / rule:** `eng install` only installs the harness *payload* (`core/`, `skills/`,
`profiles/`, `templates/`, `hooks/`) into `~/.engineering-harness/` — it does not, and was
never designed in Phase 1 or Phase 2 to, put the `eng` binary itself on PATH. Anyone using
`eng hooks run` must put `eng` on PATH themselves (e.g. `go install` it, or add `cli/` to
PATH during development) before hooks that reference `eng` by name will work. A future phase
that builds a real `eng install`/`eng update` release pipeline should either put the binary on
PATH as part of that install, or have `hooks/default.yaml`'s commands resolve `eng` via an
absolute path instead of assuming PATH.

**From:** `.plans/2026-08-24-v2-harness-phase2/`

**Resolved in Phase 3:** `eng install` now copies the running binary to
`~/.engineering-harness/bin/` and prints (or, with `--add-to-path`, applies) the correct
PATH setup for the current platform. See
`.plans/2026-08-24-v2-harness-phase3/spec.md` Decision 8.

### Go's `flag` package silently drops flags placed after a positional argument

**Trap:** Writing a command as `eng plan review <plan-dir> --verdict PASS` looks natural —
most CLIs accept flags in any position — but Go's standard `flag` package stops parsing at the
first non-flag token. `<plan-dir>` is non-flag, so parsing stops there and `--verdict PASS`
is silently left in `flagset.Args()`, never consumed.

**Symptom:** The flag's default value is used instead, with no error — `eng plan new my-plan
--risk high-risk` silently created a `feature`-risk plan (the default), and
`eng plan review <dir> --verdict PASS` failed its own "verdict must be PASS or REJECT" check
because `--verdict` was never actually parsed. Both looked like they ran successfully until the
resulting `plan.yaml` was inspected.

**Fix / rule:** Every `eng plan <subcommand>` that mixes a positional plan directory with
flags (`new`, `review`, `approve`, `block`, `cancel`) runs its arguments through
`reorderFlagsFirst` (`cli/plan_cmd.go`) before calling `flagset.Parse`, so flags and
positional arguments work in either order. Any *new* `eng` subcommand that accepts both a flag
and a positional argument must do the same — plain `flagset.Parse(args)` is only safe when
flags are guaranteed to come first, which no user reliably does by habit.

**From:** `.plans/2026-08-24-v2-harness-phase3/`
