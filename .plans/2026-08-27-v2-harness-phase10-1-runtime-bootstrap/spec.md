# Phase 10.1 — `eng start` Runtime Bootstrap & Harness Identity

Focused corrective patch, not an expansion phase. Fixes a P1 UX/runtime-integration gap found
during real dogfooding immediately after Phase 10 shipped: a harness-launched Claude Code
session has no way to learn it is harness-managed, and concluded the harness — including all of
Phase 10's role enforcement — was fictional.

## Reproduced root cause (by inspection of the real, current implementation)

### 1. `cmdStart` (`cli/start_cmd.go`) never gives the launched session anything

Traced the exact call that launches the agent:

```go
c := exec.Command("claude")
c.Stdin = os.Stdin
c.Stdout = os.Stdout
c.Stderr = os.Stderr
c.Env = buildChildEnv(os.Environ(), engBinDirs(), harnessDir(), dir, harnessVersion())
```

Three facts, confirmed by reading the code, not assumed:

- **No arguments are passed to `claude`.** `exec.Command("claude")` — zero-length arg list. Every
  `fmt.Println` line `cmdStart` prints before this (`eng doctor`'s own output, the pointer to
  `core/runtime/METHOD.md`) goes to `os.Stdout`, which is the *same terminal* the interactive
  `claude` TUI then takes over — visible to the human eye, structurally invisible to the launched
  agent's own context. Terminal scrollback is not conversation history.
- **`ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION` are real (Phase 9 P1-3's fix), but env vars are
  not self-announcing.** A session only learns `ENG_HOME` exists if it already knows to run
  `printenv`/`echo $ENG_HOME` — which requires already suspecting a harness is present, the exact
  thing that's missing.
- **Nothing writes or points at a project-local `CLAUDE.md`.** Confirmed by reading `cmdInit`
  (`cli/init_cmd.go`): it writes only `.agent/project.yaml`. Claude Code's own CLAUDE.md
  auto-discovery (project-local, and any user-level `~/.claude/CLAUDE.md`) has nothing installed
  by this harness to discover, on purpose (Phase 9's "leave legacy files untouched" backward-
  compatibility promise) — but that promise had an unintended side effect: it left *zero* channel
  for a fresh session to learn about the harness at all.

**Conclusion:** the fresh session's reported conclusion ("This is a standard Claude Code
session") was not a hallucination or an LLM failure to reason — it was *correct* given the actual
information available to it. Nothing false was told; nothing true was told either. This is a
harness bug, not an agent-behavior bug.

### 2. The real `claude` CLI already has the exact channel this needs — verified, not guessed

Ran `claude --help` directly in this environment. Confirmed real, existing flags:

```
--append-system-prompt <prompt>   Append a system prompt to the default system prompt
--system-prompt <prompt>          System prompt to use for the session
```

`--append-system-prompt` is the correct one: it *adds* trusted, harness-authored context to
Claude's system prompt — not user-turn chat text, not a file the session has to go discover and
that could be spoofed by an untrusted local file — without replacing Claude Code's own default
system prompt (which `--system-prompt` would do, discarding built-in behavior; rejected for that
reason, see DECISION_LOG.md Decision 1). `exec.Command` passes this as a real argv entry, not a
shell string — no quoting/escaping hazard on any OS, and no shell-string hack needed (Go's
`os/exec` never invokes a shell for this call, on any platform).

### 3. `eng doctor` has no plan-scoped concept; `eng workflow status` requires an explicit plan-dir; nothing enumerates plans project-wide

Confirmed by reading `cli/doctor.go` and `cli/workflow_cmd.go`: `cmdDoctor` takes no plan
argument (Phase 10 Task 7.1's own decision — correct, unchanged here). `workflowStatus` requires
a plan directory and reports exactly one plan. **No existing code enumerates `.plans/*` to
determine whether zero, one, or several plans are unfinished** — the fact this phase's bootstrap
needs (“is there a plan the agent should already know about, or is this a clean start”) doesn't
exist anywhere yet.

### 4. `harness/VERSION` has been stale since Phase 7

```
$ cat harness/VERSION
0.7.0-phase7-tools
```

Confirmed: this file was never touched by Phases 8, 9, or 10 — `eng doctor` and the new
`ENG_VERSION` env var both faithfully report a three-phases-stale version number, undermining
trust in every other "verify through eng" instruction this phase adds (why would an agent trust
a runtime probe whose own version string is provably wrong?).

## Primary goal

```
eng start
  -> resolve harness + project
  -> gather a small, bounded runtime status
  -> launch Claude Code with that status appended to its system prompt (trusted, not user text)
  -> Claude immediately knows it is harness-managed, without being told in chat
  -> Claude verifies through eng (doctor/workflow status) rather than trusting the prompt alone
  -> normal harness workflow begins
```

The user must never need to manually remind Claude the harness exists.

## Design: bootstrap status + prompt (two pure, testable functions)

`cli/bootstrap.go` (new file, small, no new package — this is CLI glue, matching how
`start_cmd.go` itself is structured):

```go
type planSummary struct {
    Dir   string // relative to project root, e.g. ".plans/2026-08-27-add-x"
    State string
}

type bootstrapStatus struct {
    HarnessInstalled bool
    HarnessHome      string
    HarnessVersion   string
    ProjectRoot      string
    ProjectMode      string // "" | legacy | none | broken | modern | hybrid
    PlanningMode     string // e.g. "spec_first" — "" if project not initialized
    TriageEnabled      bool
    PlanReviewEnabled  bool
    VerifierEnabled    bool
    UnfinishedPlans  []planSummary // workflow.Terminal(state) == false
    CodexInstalled   bool
    CodexWired       bool
    CodexInvokable   bool
}

func gatherBootstrapStatus(dir string) bootstrapStatus   // filesystem reads only, no exec, no LLM
func renderBootstrapPrompt(s bootstrapStatus) string      // pure string formatting, no I/O
```

`gatherBootstrapStatus` reuses existing code paths exactly — no parallel implementation of
anything `eng doctor`/`eng workflow status` already compute:
- `harnessDir()`/`harnessVersion()` (existing, `cli/install.go`)
- `project.DetectModeResult(dir)` + `project.Load(dir)` (existing)
- `registeredAdapters(dir)` (existing, Phase 10) — find the adapter named `"codex"`, call
  `Available()`/`Doctor()` exactly like `cmdDoctor` already does, so the bootstrap can never
  disagree with what `eng doctor` itself would report a moment later.
- **New, small:** `scanPlans(dir) []planSummary` — walks `.plans/*/plan.yaml` via the existing
  `planmeta.Load`, keeps entries where `!workflow.Terminal(meta.State)`. This is the one genuinely
  new piece of logic (per section 3's gap above) — deliberately kept to "read plan.yaml files
  under `.plans/`, filter by terminal state," nothing more.

`renderBootstrapPrompt` is pure formatting — no filesystem, no exec — so it's fully unit-testable
against constructed `bootstrapStatus` values without a real project on disk.

## Bootstrap prompt shape (bounded — see acceptance criterion 8)

```
You are running under the Global Engineering Harness.

Harness home:    <ENG_HOME>
Harness version: <version>
Project root:    <project root>
Project mode:    <mode>
Workflow:        planning=<planning_mode> triage=<on/off> plan_review=<on/off> verifier=<on/off>
Codex:           installed=<bool> wired=<bool> invokable=<bool>
Plans:           <"no unfinished plans — start clean" | "1 unfinished: <dir> (<state>)" |
                  "N unfinished — ask the human which to resume, do not guess: <dir> (<state>), ...">

Before any workflow-sensitive action, verify current state through `eng` (e.g. `eng doctor`,
`eng workflow status <plan-dir>`) rather than trusting this summary alone — it is a snapshot
from session start. Role activation, state-role compatibility, the executor/verifier gates, and
tool policy (`eng tools invoke`) all remain enforced exactly as documented in
core/runtime/METHOD.md — this prompt does not bypass any of them. Do not conclude the harness is
absent from the lack of a project-local CLAUDE.md/.claude/ — the harness install above is the
source of truth. Do not auto-resume a COMPLETED plan; treat it as history unless the human
explicitly asks to resume it.
```

Every line is populated from `bootstrapStatus`, deterministic, no user chat text, no full
METHOD.md content, no skill list, no plan file contents. Roughly 12-14 lines / under 900
characters in the common case (0-1 unfinished plans), measured at up to ~1400 characters in the
worst realistic case (5 listed plans, each with a name near `slugify`'s own 40-character cap) —
see acceptance criterion 8 for the bound enforced by a test, not just eyeballed here. For scale:
a single role `METHOD.md` file alone runs several thousand characters — this prompt stays a
small fraction of that even at its worst case.

## `eng start` wiring

```go
status := gatherBootstrapStatus(dir)
prompt := renderBootstrapPrompt(status)
c := exec.Command("claude", "--append-system-prompt", prompt)
...
c.Env = buildChildEnv(...)  // unchanged from Phase 9/10 — env vars still set, still useful for
                             // the session's own later eng invocations and the METHOD.md PATH
                             // fallback; the prompt is additive, not a replacement.
```

No change to `buildChildEnv`, `engBinDirs`, or any Phase 9/10 gate — this phase only adds a
system-prompt argument to the existing launch call.

## Multiple unfinished plans — resolved as "surface, don't guess"

Per the instruction's explicit "If there are multiple unfinished plans, fail clearly instead of
guessing": `eng start` itself does **not** fail or block — a hard failure here would break normal
session launch over a bookkeeping ambiguity. Instead, the bootstrap prompt states the ambiguity
plainly (all unfinished plan dirs + states, capped — see below) and instructs the agent not to
guess, i.e. to ask the human which one (if any) to resume. This keeps `eng start` itself simple
and always-succeeds, while still satisfying "don't silently resume" and "don't guess" at the
point that actually matters (the agent's next action). See DECISION_LOG.md Decision 4.

To keep the prompt bounded even with many stale plans, list at most 5 unfinished plans by name,
with a `"...and N more"` suffix beyond that — never let a project with a long history of
abandoned plans blow the prompt budget.

## Version metadata correction

`harness/VERSION`: `0.7.0-phase7-tools` -> `0.10.1-beta` (Phase 10.1's own version — the harness
has shipped 10 phases plus this corrective patch; a coherent product-style version, not another
phase-name string, per the instruction's explicit direction). No code change needed beyond the
file content — `harnessVersion()` already reads it fresh from disk on every call, and `eng
install`'s `copyTree` already copies the whole `harness/` directory including `VERSION` — so a
plain `eng install --from .` after this fix propagates it with no new logic. `ENG_VERSION`
(env), `eng doctor`'s printed version, and the bootstrap prompt's `Harness version:` line all
read the same `harnessVersion()` call — cannot disagree by construction (single source of truth,
not three).

## Acceptance criteria (fixed before implementation)

1. `gatherBootstrapStatus` correctly reports `HarnessInstalled`, `HarnessVersion`, `ProjectMode`,
   `PlanningMode`, the three workflow-enabled bools, and Codex's installed/wired/invokable state
   — each matching what `eng doctor` reports for the same directory (no divergence).
2. `scanPlans` returns zero entries for a project with no `.plans/` dir, zero entries when every
   plan is terminal (COMPLETED/FAILED/CANCELLED/BLOCKED), and the correct non-terminal subset
   otherwise.
3. `renderBootstrapPrompt` output is deterministic for a given `bootstrapStatus` (same input,
   same output, no timestamps/randomness) and contains the harness home, version, project mode,
   and Codex state.
4. `renderBootstrapPrompt` explicitly instructs the agent to verify through `eng` rather than
   trust the prompt alone, and explicitly states that a missing project-local CLAUDE.md/.claude
   does not imply the harness is absent.
5. `renderBootstrapPrompt` explicitly instructs the agent not to auto-resume a COMPLETED plan.
6. With zero unfinished plans, the prompt says so plainly ("no unfinished plans"). With exactly
   one, it names it. With more than one, it lists them (capped at 5, `"...and N more"` beyond
   that) and says not to guess which to resume.
7. `cmdStart`'s launch of `claude` includes `--append-system-prompt` with the rendered prompt as
   an argv entry (verified via a test double / inspectable command construction — Go's `os/exec`
   never shells out this call, so no quoting hazard exists on any OS).
8. The rendered prompt stays bounded: under 1600 characters even with 5 unfinished plans listed,
   each near the maximum realistic plan-directory-name length (a hard ceiling check in a test —
   this is *not* "load all skills/docs/plans," it is a fixed-shape status summary, smaller than a
   single role `METHOD.md` file).
9. `harnessVersion()` (and therefore `eng doctor`, `ENG_VERSION`, and the bootstrap prompt) all
   report `0.10.1-beta` after this phase's `harness/VERSION` change and a fresh `eng install`.
10. Every Phase 10 enforcement mechanism (role activation, state-role compatibility, executor/
    verifier gates, tool policy, `verify-review`) is unaffected — this phase adds a prompt
    argument to a launch call, nothing else, and does not touch `workflow.go`, `rolestate.go`,
    `tools_cmd.go`, or `adapter_cmd.go`.
11. Manual dogfood (honestly reported, not asserted as a substitute for the above): a fresh
    Claude Code session launched via the rebuilt-and-reinstalled `eng start` states, without
    being told in chat, that it is running under the Global Engineering Harness, names the
    correct project mode/workflow/Codex state, and does not claim Phase 10 is fictional.

## Explicitly out of scope

Everything the instruction's section 20 lists: no marketplace, no new plugin system, no new
skills, no new MCP transports, no Codex write/execute capability, no agent swarm, no cloud
runtime, no vector DB, no memory platform. Also out of scope: a new `eng runtime status`
subcommand (Decision 3 — `gatherBootstrapStatus` is internal to `cmdStart`, not exposed as a new
command, since `eng doctor` + `eng workflow status` already cover the same ground for anyone
who wants it manually); any change to `workflow.Decide`, `rolestate`, or `toolpolicy` (Phase 10's
enforcement is correct and untouched); any change to how `eng init` handles legacy files.

## Fixture / test strategy

No new benchmark fixture needed — this phase's tests are unit tests against
`gatherBootstrapStatus`/`renderBootstrapPrompt`/`scanPlans` using the repo's own existing fixture
conventions (a temp dir with a hand-written `.agent/project.yaml` and zero or more
`.plans/*/plan.yaml` files), plus a manual dogfood pass per the instruction's own explicit
request to keep automated-vs-manual verification honestly separated (section 15/16). No
CredoID-specific content anywhere in this phase's fixtures — synthetic temp-dir plans only,
matching Phase 10's own established discipline.
