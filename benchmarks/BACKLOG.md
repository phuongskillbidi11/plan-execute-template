# Refinement Backlog

Every entry below is a real weakness a benchmark run or real-world dogfooding actually
surfaced — none is hypothetical. Severity uses the Phase 8 instruction's P0–P3 scale:

- **P0** — blocks normal use
- **P1** — significant friction / undermines a headline claim, but a workaround exists
- **P2** — moderate friction, non-obvious, workaround exists
- **P3** — future enhancement, no current friction

Phase 8 deliberately did not fix any of these (per its own bounded-fix rule: core harness
behavior is not changed merely to make a benchmark pass or look better). **Phase 9 resolved
every P1/P2 item below** — each entry states its resolution and the evidence, right below the
original Phase 8 finding. Historical Phase 8 analysis is left intact rather than deleted, so the
before/after is visible in one place.

---

## P1-1 — Quick Fix triage misclassifies numeric/parameter-tweak requests

**Source:** `benchmarks/results/quick-fix-timeout-harness-v2.yaml`

`eng workflow start "Increase the reconnect timeout from 1000 ms to 1500 ms."` — the exact
request text the Phase 8 instruction itself uses as its canonical Quick Fix example — auto-
triages to risk level `feature`, not `quick-fix`. Reproduced directly via `eng triage` on the
same input. Root cause: Phase 5's quick-fix trigger keyword list (`typo`, `rename`, `comment`,
`formatting`, `"small change"`) has no entry that matches numeric-value/parameter-tweak phrasing.

**Impact:** defeats one of Phase 5's most user-facing value propositions — "quick fixes should be
cheap" — for a plausible and common real-world phrasing, unless the user or Planner knows to pass
`--risk quick-fix` explicitly.

**Workaround:** explicit `--risk quick-fix` on `eng plan new`/`eng workflow start` bypasses
triage entirely and the underlying Quick Fix state machine behaves correctly once bypassed
(`mechanism_proof` in the same result file: TRIAGED → EXECUTING → VERIFYING → COMPLETED, 1 file,
PASS).

**Suggested direction (not authorized/implemented in Phase 8):** broaden the quick-fix trigger
keyword list to include numeric-value-change phrasing ("increase," "decrease," "change X from Y
to Z," "bump"), or add a size-based heuristic (single named constant, single file touched) as a
secondary triage signal alongside keyword matching.

**Status: RESOLVED in Phase 9.** `cli/triage_cmd.go` gained `looksLikeParameterTweak` (a
change-verb word + digit-presence check, consulted only after every risk-elevating category has
already had a chance to match, so it can never override a risk-elevating classification) plus
broadened `high-risk`/`architecture` word lists (security/auth, hardware-control, public-API
terms). `eng triage "Increase the reconnect timeout from 1000 ms to 1500 ms."` now returns
`quick-fix`. See `benchmarks/results/quick-fix-timeout-triage-fixed.yaml` and
`cli/triage_cmd_test.go`.

---

## P1-2 — Docs/project-context routing over-matches relative to skill routing

**Source:** `benchmarks/results/large-context-auth-validation-harness-v2.yaml`,
`benchmarks/CONTEXT_EFFICIENCY.md`

`eng context project` (`internal/docsearch.Match`) matched 4 of 6 `docs/src-map.md` sections for
a request that should concern only 1 (`auth/`) — a 67% false-positive rate against this fixture's
own section count. The same request through `eng context skills` (`internal/skillmatch.Score`)
produced 0 false positives.

**Impact:** undermines half of Phase 4's "large knowledge base ≠ large prompt" claim — it holds
for skills, measurably less well for docs, on the one case tested. Bloats context bundles with
irrelevant doc sections, increasing both bundle size and (as an unverifiable but plausible
consequence) risk of an implementer skimming/misreading unrelated sections.

**Workaround:** none currently — this is the default and only doc-routing path.

**Suggested direction (not authorized/implemented in Phase 8):** align `internal/docsearch.Match`
with `internal/skillmatch.Score`'s weighted tag/trigger/description-word model rather than its
current apparent simple word-overlap approach; re-run this same scenario after any change to
confirm the false-positive rate actually drops.

**Status: RESOLVED in Phase 9.** `internal/docsearch.Match` gained word-boundary (tokenized)
phrase matching instead of raw substring containment, plus weighted scoring (title words worth
more than body words) with a minimum-score inclusion threshold — the same design pattern
applied to `internal/skillmatch.Score` (see P2-2 below). Re-running the exact same scenario:
`docs_matched` dropped from 4/6 to **1/6** (the single genuinely relevant section) — better than
this phase's own "materially reduced" acceptance bar. See
`benchmarks/results/doc-context-overmatch-fixed.yaml` and `cli/internal/docsearch/
docsearch_test.go`.

---

## P2-1 — `tasks.md` has two unsynchronized completion-tracking conventions

**Source:** `benchmarks/results/feature-csv-export-harness-v2.yaml` (first observed),
recurred in `benchmarks/results/bug-lastn-off-by-one-harness-v2.yaml`

The rich plan template's `tasks.md` carries a per-task `**Status:** \`[x]\`` marker AND a
separate bottom "Completion checklist" section using `- [ ]` list items.
`cli/workflow_cmd.go`'s `tasksComplete()` scans only for lines literally starting with
`"- [ ]"` — it reads the bottom checklist, not the per-task Status markers. Marking every
individual task's Status field to `[x]` alone leaves `eng workflow advance` reporting "tasks.md
still has unchecked items" until the separate bottom checklist is also marked.

**Impact:** non-obvious UX trap — an Executor could reasonably believe marking each task's own
Status field is sufficient and get stuck without an explicit hint pointing at the bottom
checklist specifically. Recurred a second time independently (Category 3, Bug Fix) without any
memory of the first occurrence carried over, i.e. it's a structural gap, not a one-off authoring
slip.

**Workaround:** mark both the per-task Status fields and the bottom Completion checklist.

**Suggested direction (not authorized/implemented in Phase 8):** either drop the per-task Status
marker (the bottom checklist alone is sufficient and is what's actually enforced) or make
`tasksComplete()` recognize both conventions; whichever is chosen, the "Next: tasks.md still has
unchecked items" message could name the specific unchecked line(s) to make the trap self-
correcting.

**Status: RESOLVED in Phase 9** — chose the smaller, fully backward-compatible option: the
bottom checklist stays the sole machine-authoritative source (zero change to `tasksComplete`'s
logic, so no existing plan's completion state changes), and the confusion is fixed at its two
real sources instead. `harness/templates/plan/tasks.md`'s header now states explicitly which
marker gates advance. `eng workflow advance` now names the specific unchecked checklist line(s)
instead of a generic message. See `cli/workflow_cmd_test.go` and Phase 9 DECISION_LOG.md
Decision 5.

---

## P1-3 — `eng start` launched an agent whose child environment's PATH could be stale

**Source:** Real-world dogfooding report (not a Phase 8 benchmark finding).

`eng.exe start` invoked via its full path (bypassing PATH lookup entirely) launched `claude`
with the same inherited, potentially-stale PATH — on Windows, `eng install --add-to-path`'s
`setx` only affects new terminal sessions, so a session already open when it ran kept its old
PATH. Inside the launched Claude Code session, `eng doctor` failed with
`bash: eng: command not found`, easily misread as "the harness isn't installed."

**Status: RESOLVED in Phase 9.** `cmdStart` now builds the child process's environment
explicitly via a new pure `buildChildEnv` function: it prepends both the running binary's own
directory and the installed `bin/` directory (deduplicated) to `PATH` — never relying on the
inherited PATH being correct, and never destroying whatever PATH already existed — and sets
`ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION` as explicit environment variables. See
`cli/start_cmd_test.go`, `docs/gotchas.md`, and `harness/core/runtime/METHOD.md`'s new
`$ENG_HOME/bin/eng` fallback note.

---

## P1-4 — `workflow: {triage,plan_review,verifier}: false` was indistinguishable from unset

**Source:** Real-world dogfooding report (not a Phase 8 benchmark finding) — confirmed by direct
inspection of `cli/internal/project/project.go`.

`Workflow.enabled()` returned `w.Triage || w.PlanReview || w.Verifier`, and
`Config.EffectiveWorkflow()` fell back to an all-`true` `Workflow{}` whenever `enabled()` was
`false` — so a project.yaml with every field explicitly set to `false` silently behaved
identically to a project.yaml with no `workflow:` block at all.

**Status: RESOLVED in Phase 9.** `Triage`/`PlanReview`/`Verifier` are now `*bool` (mirroring
`RequireSpecApproval`'s existing pattern), each defaulting to `true` only when its own pointer is
`nil`; the group-level `enabled()`/`EffectiveWorkflow()` fallback was deleted as unnecessary once
each field resolves independently. `eng doctor` now reports each field's resolved value plus
whether it's `(explicit)` or `(default)`. See `cli/internal/project/project_test.go`'s four
required cases (omitted/all-true/mixed/all-false).

---

## P2-2 — RS485/serial vocabulary falsely implied PLC/Siemens/Modbus in skill routing

**Source:** Real-world dogfooding report (a C# Windows UHF RFID application using RS485/USB-HID)
— confirmed by direct reproduction.

`eng context skills` on a C#/WinForms/RS485/USB-HID request selected `automation/modbus`,
`automation/siemens-s7`, and `automation/plc` — none of which have any real bearing on the
project. Root cause: `internal/skillmatch.Score`'s old uniform, unweighted, substring-based
matching let (a) a description word be a false-positive substring of an unrelated request word
(`"form"` inside `"WinForms"`), and (b) a single generic-vocabulary description-word match
(`"framing"`, `"protocol"`) count as a full match on its own, with `automation/plc` then
force-included via `automation/siemens-s7`'s `requires:` edge.

**Status: RESOLVED in Phase 9.** Same fix as P1-2, applied to `skillmatch.Score`: word-boundary
tokenized matching plus weighted scoring (`TagTriggerWeight`/`DescriptionWordWeight`) with a
`MinMatchScore` threshold. A "C++"/"C#" token-collision (`software/cpp`'s `"c++"` tag) was also
found and fixed during this same reproduction. A new reusable `software/serial-communications`
skill (deliberately containing no Modbus-specific content — see Phase 9 DECISION_LOG.md Decision
8) now routes correctly instead. See `benchmarks/results/skill-routing-rs485-not-plc.yaml`,
`cli/internal/skillmatch/skillmatch_test.go`, and
`harness/evals/software/csharp-winforms-rs485.yaml`.

---

## P2-3 — Global/local duplicate skill resolution (legacy vs. qualified same-name skill)

**Source:** Real-world dogfooding report (`engineering/karpathy-guidelines` global +
project-local legacy `karpathy-guidelines`) — also reproduced live against this repository's own
working tree, which had the identical real instance of the bug.

`skills.ResolveWithPrivate` merges by `QualifiedName()`, and a legacy (frontmatter-less) skill's
qualified name is its bare `Name` — never `"engineering/karpathy-guidelines"` — so the two never
collided in the merge map and both resolved as distinct entries.

**Status: RESOLVED in Phase 9.** Added a bare-name collision-collapse pass to
`ResolveWithPrivate`, triggered only when a bare-name group mixes at least one legacy entry with
at least one qualified entry (never when a group is made entirely of genuinely distinct,
qualified, cross-domain skills sharing a bare name — e.g. `automation/modbus` vs. a future
`networking/modbus`, which must stay separate). `skillvalidate.Validate` now also reports which
source won as a `shadowed by ...` warning. See
`benchmarks/results/skill-dedup-legacy-global.yaml` and `cli/internal/skills/skills_test.go`.

---

## P3 — Broader context-efficiency sweep across more domains/phrasings

**Source:** `benchmarks/CONTEXT_EFFICIENCY.md` verdict section

P1-2 above is evidence from one scenario, not a statistical claim. A genuinely broader sweep
(more fixtures across more domains, more request phrasings per fixture) would strengthen or
narrow the finding, but is out of Phase 8's scope (Phase 8 deliberately avoided building a large
benchmark platform). Reasonable candidate for a future phase.

---

## P2-4 — `automation/plc`'s bare `automation` tag is a latent P2-2-class risk (open)

**Source:** Phase 9 DECISION_LOG.md Decision 11, observed while fixing the analogous
`embedded/esp32` `embedded` tag defect.

`automation/plc`'s frontmatter carries `automation` as a bare tag — the same class of
over-generic curated-tag signal that caused both the P2-2 (RS485) and the `embedded/esp32`
(found during Task 8 dogfooding, see `benchmarks/results/qt-cmake-embedded-linux-harness-v2.
yaml`) false positives. Not fixed in Phase 9: nothing in Phase 9's own scenarios exercises it,
so per the same bounded-fix discipline used throughout this phase, it's documented rather than
fixed speculatively. A future phase should either remove the tag or replace it with a more
specific one, then add a regression scenario proving the fix (the same pattern P2-2/`esp32`
followed).

---

## P3-1 — Budget-vs-breadth tension when a request genuinely concerns more skills than the budget (open, expected behavior)

**Source:** `benchmarks/results/qt-cmake-embedded-linux-harness-v2.yaml`

Under the real default `max_skills: 5` budget (which counts the explicitly-enabled
`karpathy-guidelines` skill against it too), a request genuinely relevant to 5 domain skills at
once has one of them budget-cut. This is an inherent, expected consequence of a bounded context
budget — not a routing defect, and not something Phase 9 was asked to fix. Left here as an
honest, documented observation rather than silently omitted.
