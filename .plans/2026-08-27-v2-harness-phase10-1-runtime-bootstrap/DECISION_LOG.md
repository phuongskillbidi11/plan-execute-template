# Phase 10.1 Decision Log

## Decision 1 — `--append-system-prompt`, not `--system-prompt`, not env-var-only

`--system-prompt <prompt>` replaces Claude Code's entire default system prompt — rejected: it
would discard built-in tool-use behavior, safety instructions, and every other default Claude
Code ships with, for the sake of adding ~12 lines of harness identity. `--append-system-prompt`
adds to the default without replacing it — the correct, minimal-footprint channel. Verified as a
real flag via `claude --help` in this environment, not guessed. Continuing to rely on
`ENG_HOME`/env vars alone (Phase 9/10's existing mechanism) was rejected as the *sole* channel
because env vars are not self-announcing — a session must already suspect a harness exists to
think to inspect them, which is exactly the chicken-and-egg problem this phase fixes.

## Decision 2 — bootstrap text is machine-generated from `bootstrapStatus`, never free-form prose authored per-call

`renderBootstrapPrompt` takes a struct, not a caller-supplied string — every fact in the prompt
traces back to a specific field read from `eng doctor`'s own data sources (`project.Load`,
`registeredAdapters`, `harnessVersion()`). This is what makes it a *trusted* boundary distinct
from "user claims harness exists" (instruction section 7): the harness's own process, not chat
text, produced every line, and the function is unit-testable precisely because it has no
free-text input to smuggle anything through.

## Decision 3 — no new `eng runtime status` command

Considered adding one (the instruction explicitly floats it). Rejected in favor of reuse: `eng
doctor` already reports harness install/version/project mode/workflow/tools, and `eng workflow
status <plan-dir>` already reports state/active-role/next-role for a specific plan. The only
genuinely missing piece was "which plan(s), if any, are unfinished project-wide" — added as
`scanPlans`, a small internal function `cmdStart` calls, not a new user-facing command. A human
who wants the same information manually already has it via the two existing commands; adding a
third that partially overlaps both would be exactly the kind of scope creep this instruction's
"prefer reuse" guidance warns against.

## Decision 4 — multiple unfinished plans: surface in the prompt, don't fail `eng start`

Considered making `eng start` itself refuse to launch when `.plans/` contains more than one
non-terminal plan. Rejected: `eng start` launching a normal, usable session is the load-bearing
happy path — failing it over stale abandoned plans (a common, benign state in any real project)
would be a worse regression than the bug this phase fixes. Instead, the bootstrap prompt lists
every unfinished plan (capped at 5) and instructs the agent to ask the human rather than guess —
"fail clearly instead of guessing" is satisfied at the decision point that matters (which plan to
act on), not by refusing to start a session at all.

## Decision 5 — version scheme: `0.10.1-beta`, dropping the `phase<N>-<topic>` pattern

Every prior version string (`0.7.0-phase7-tools`, referenced `0.8.0-phase9`/`0.9.0-fresh` in
existing test literals) encoded a phase name. Per the instruction's explicit direction ("do not
keep phase names indefinitely if product versioning is now more appropriate"), Phase 10.1 is the
natural boundary to switch: `0.10.1-beta` reads as "tenth major harness milestone plus one
corrective patch, still pre-1.0." Chose `-beta` over `-phase10.1` specifically to stop coupling
the version string to the internal planning-phase numbering, which is an implementation detail of
how this repo's own work is organized, not something an end user of the harness needs to parse.
Future phases should bump this coherently (e.g. `0.11.0` for Phase 11) rather than reintroducing
phase-name suffixes.

## Decision 6 — `gatherBootstrapStatus` re-reads the same sources `eng doctor` reads, never a cached/separate copy

Rejected caching `eng doctor`'s output and reusing it, or having `cmdStart` shell out to `eng
doctor` as a subprocess and parse its text output. Both would introduce a second source of truth
(a cache that can go stale, or a text-parsing dependency on `eng doctor`'s exact print format).
Instead `gatherBootstrapStatus` calls the same underlying functions (`project.Load`,
`registeredAdapters`, `harnessVersion()`) `cmdDoctor` itself calls — by construction, the
bootstrap prompt and `eng doctor`'s own next output cannot disagree, satisfying instruction
section 12's explicit "do not say Codex is merely present but not wired if the installed adapter
proves otherwise" requirement without any synchronization mechanism needed.

## Decision 7 — `scanPlans` reads `plan.yaml` directly via `planmeta.Load`, not `eng workflow status` per plan

Iterating `.plans/*` and calling `eng workflow status` as a subprocess per directory would work
but is slow (one process spawn per plan) and re-parses printed text instead of structured data.
`planmeta.Load` (already Phase 3+'s own accessor) plus `workflow.Terminal(state)` (already Phase
10's own accessor) is the direct, already-tested path to exactly the one fact needed (is this
plan's state terminal) — no new parsing, no new process spawns, bounded to however many plan
directories genuinely exist.
