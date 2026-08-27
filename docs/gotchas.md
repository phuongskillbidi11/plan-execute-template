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

### `enabled_skills` entries didn't match resolved skill names

**Trap:** `eng init` writes `enabled_skills` as a domain-qualified path (e.g.
`engineering/karpathy-guidelines`) — that's how it's displayed everywhere (`eng skills list`,
`eng doctor`) and how a reader would naturally write a new entry by hand.

**Symptom:** `eng context skills`'s "always include enabled_skills, even beyond max_skills"
guarantee silently failed for exactly the entry `eng init` itself creates: a resolved
`skills.Skill.Name` is its bare frontmatter `name:` field (`karpathy-guidelines`, no domain
prefix), so a naive exact-string lookup against `engineering/karpathy-guidelines` never
matched. This went unnoticed through three phases because `enabled_skills` had never actually
been *read* by any code before Phase 4's `skillmatch.Select` — it was written but never
consumed.

**Fix / rule:** `skillmatch.Select` (`cli/internal/skillmatch/skillmatch.go`) registers both
the full `mustInclude` entry and the substring after its last `/` as matching keys, so either
`karpathy-guidelines` or `engineering/karpathy-guidelines` in `enabled_skills` correctly
guarantees inclusion. Any future code that matches against `enabled_skills` should do the same
normalization rather than assuming either form.

**From:** `.plans/2026-08-24-v2-harness-phase4-context/`

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

### `eng start` launching an agent whose PATH never actually had the harness `bin/` directory

**Trap:** `eng install --add-to-path`'s Windows path uses `setx`, which only affects *new*
terminal sessions — a session already open when `--add-to-path` (or the printed manual `setx`
line) runs keeps its old PATH for the rest of that session. Running `eng start` from that same
stale session (especially via the binary's full path, e.g.
`C:\Users\<you>\.engineering-harness\bin\eng.exe start`, which bypasses PATH lookup entirely and
so never itself proves PATH is correct) launches `claude` with that same stale, harness-less
PATH inherited.

**Symptom:** Inside the launched Claude Code session, running `eng doctor` from the Bash tool
fails with `bash: eng: command not found` — easy to misread as "the harness isn't installed,"
when the harness is installed fine and only the PATH propagation was stale.

**Fix / rule:** `eng start` (Phase 9 onward) no longer relies on the inherited PATH being
correct — it explicitly prepends both the running binary's own directory and the installed
`bin/` directory to the launched session's `PATH` (via `buildChildEnv` in `cli/start_cmd.go`),
and sets `ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION` as explicit environment variables. A
session that still can't resolve `eng` in some nested shell it spawns itself (e.g. that shell's
own profile script resets `PATH`) should fall back to `"$ENG_HOME/bin/eng"` — see
`harness/core/runtime/METHOD.md`.

**From:** `.plans/2026-08-26-v2-harness-phase9-core-refinement/`

### The workflow state machine could "catch up" after real work already happened

**Trap:** `workflow.Decide` is a pure function of *current* file/flag state (`tasks.md`'s
Completion checklist, `plan.yaml`'s fields) — it has no concept of *when* something became
true relative to *when* a state was entered. Nothing stopped an agent from doing real work
(e.g. a browser/UI investigation, which produces no tracked-file diff) while a plan was still
`APPROVED`, then marking `tasks.md`'s Completion checklist complete afterward.

**Symptom:** `eng workflow advance` transitioned `APPROVED → EXECUTING` (drift detection found
nothing, since the real work never touched a tracked file) and then, on the very next call,
`EXECUTING → VERIFYING → COMPLETED` with a PASS verdict — while `eng verify`'s own report
showed an **empty git diff since the plan's `git_sha`**. The state machine wasn't gating
execution before it happened; it was rubber-stamping work that had already happened outside it.
Reproduced deterministically in `benchmarks/fixtures/investigation-bypass/`.

**Fix / rule:** Phase 10 added two things this trap needed: (1) `APPROVED → EXECUTING` (and
Quick Fix's `TRIAGED → EXECUTING`) now requires the executor role to have actually been
activated first (`eng adapter prompt executor <plan-dir>`, which now validates and records
activation instead of just printing a prompt); (2) if `tasks.md`'s Completion checklist is
*already* fully checked at the exact moment that transition would fire, `Decide` refuses with
an explicit "this looks like retroactive completion" reason instead of proceeding — no override
flag exists on purpose. See `docs/ARCHITECTURE.md`'s "Role runtime enforcement" section and
`benchmarks/results/investigation-bypass-blocked.yaml` for the real before/after proof. This
does **not** prevent the premature edit itself — a Claude Code session's own native tools are
outside what a CLI harness can technically intercept — it prevents the workflow from
retroactively legitimizing one once detected.

**From:** `.plans/2026-08-27-v2-harness-phase10-role-enforcement/`
