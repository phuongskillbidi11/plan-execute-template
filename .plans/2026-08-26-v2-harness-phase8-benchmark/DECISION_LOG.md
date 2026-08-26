# Decision Log — Phase 8

## 1. No `eng benchmark` CLI — documented command sequences instead

**Considered:** a real `eng benchmark run/list/report` command family, matching this
project's own convention of building everything as an `eng` subcommand.

**Rejected because:** the instruction is explicit — "Only implement this if it clearly
reduces manual work... A simple internal test runner may be sufficient... Do not turn the
benchmark framework into a permanent complex subsystem prematurely." With eight scenarios,
most run once as a validation snapshot rather than routinely, a generic runner engine would
be more code than the thing it orchestrates. It would also need to live somewhere — inside
`cli/` it becomes production surface area (a new command a user could invoke against their
own project, with its own compatibility burden forever); outside `cli/` it duplicates
argument-parsing/dispatch machinery `eng` already has for no benefit.

**Chosen:** `benchmarks/README.md` documents the exact, copy-pasteable command sequence per
scenario per mode — the same pattern every phase's own `tests.md` has used successfully six
times over in this repository. Zero new Go code. If a future phase needs routine re-runs
(e.g., in CI), the documented sequence is already a complete spec for what that command
should do.

## 2. Fixtures live in a new top-level `benchmarks/`, not under `harness/` or `.plans/`

**Considered:** `harness/benchmarks/` (ships with every install, discoverable by any
project using the harness) or `.plans/_benchmarks/` (reuses the existing plan-lifecycle
directory).

**Rejected because:** `eng install --from <path>` copies `harness/` wholesale into
`~/.engineering-harness/` (`cli/install.go`'s `copyTree`) — anything placed there reaches
every consumer project via `eng install`, which is exactly wrong for fixtures that exist
only to validate *this repository's own* harness, not to be shipped as harness content.
`.plans/` is live, mutable plan-lifecycle state (`eng plan new` scaffolds directly into it,
`eng workflow advance` mutates `plan.yaml` in place) — mixing static benchmark fixtures into
it risks a benchmark run being mistaken for a real in-flight plan by any future tooling that
walks `.plans/*`.

**Chosen:** `benchmarks/`, a sibling of `cli/`/`harness/`/`docs/`/`scripts/` — this
repository's own dev/validation tooling, structurally parallel to `scripts/` (V1's own
dev tooling) and never touched by `eng install`.

## 3. Mode A is one illustrative run per scenario, not a statistical sample

**Considered:** running Mode A multiple times per scenario to average out LLM
non-determinism before comparing to Modes B/C.

**Rejected because:** each Mode A run is a full fresh-subagent dispatch with real file
mutations to inspect — running each scenario even 3–5 times to get a defensible sample
would multiply Phase 8's cost several-fold for a phase whose own instructions explicitly
warn against building "a large benchmark platform" and ask to "keep Phase 8 lightweight."
The instruction also explicitly asks to "identify risks of non-deterministic LLM
comparisons" rather than solve them statistically.

**Chosen:** exactly one Mode A run per scenario it's used for (Quick Fix, Feature), labeled
explicitly as illustrative/single-run evidence everywhere it's reported — never presented
with the same evidentiary weight as Mode C's fully deterministic, mechanically-reproducible
results. If Phase 8's findings suggest Mode A vs. C is a genuinely close call worth
resolving statistically, that becomes a named item in the refinement backlog, not something
this phase tries to settle on one sample.

## 4. Categories 3–7 run Mode C only (or B+C for Legacy), not the full three-mode matrix

**Considered:** running all eight categories through all three modes for a uniform,
symmetric comparison table.

**Rejected because:** for Bug Fix/Large-Context/Cross-Domain/Tool-Routing/Failure-Safety,
the property under test is *internal to the harness* (does the debugging skill get
selected; does the context bundle stay small relative to the repo; does routing match the
instruction's own worked example; does a denied capability actually stay uninvoked; does a
Verifier FAIL actually block completion) — none of these claims are meaningfully sharpened
by also running a baseline agent or V1 side by side, and V1 has no skill/tool/context
routing to compare against at all for most of them. Spending a non-deterministic Mode A run
on a scenario whose entire point is a deterministic internal guarantee would add cost
without adding evidence.

**Chosen:** the two scenarios where "does the extra workflow overhead pay for itself" is
literally the question (Quick Fix, Feature) get all three modes; the rest get exactly the
mode(s) that can actually falsify their specific claim.

## 5. Success criteria for Mode A vs. C on Quick Fix/Feature are deliberately not fixed
   pass/fail lines

**Considered:** defining a strict threshold in advance (e.g., "Mode C must produce fewer
files changed than Mode A" or "Mode C's approval gate must not add more than N steps").

**Rejected because:** the instruction's own Requirement 19 is explicit — "the benchmark may
reveal that V2 uses more steps... Treat these as valid findings. Do not optimize the report
to make the harness look successful." Pre-committing to a numeric threshold on the one
genuinely open, subjective question (is the extra process worth it) would function exactly
like optimizing the report backward from a desired conclusion, even if the threshold were
chosen in good faith before running anything.

**Chosen:** every other category has a hard pass/fail line fixed before execution (spec.md's
"Success criteria" section). Quick Fix/Feature's Mode A vs. C comparison instead has fixed
*measurements* to record (files changed, self-verification, unprompted planning artifacts,
step count) with the interpretation — whether the harness's extra steps paid for themselves
— left to the final report, grounded in what was actually observed.
