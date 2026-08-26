# Refinement Backlog

Every entry below is a real weakness Phase 8's benchmark runs (Tasks 4–11) actually surfaced —
none is hypothetical. Severity uses the Phase 8 instruction's P0–P3 scale:

- **P0** — blocks normal use
- **P1** — significant friction / undermines a headline claim, but a workaround exists
- **P2** — moderate friction, non-obvious, workaround exists
- **P3** — future enhancement, no current friction

None of these were fixed as part of Phase 8, per `DECISION_LOG.md` and the plan's bounded-fix
rule: core harness behavior is not changed merely to make a benchmark pass or look better. Each
is a candidate for a scoped Phase 9+ fix.

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

---

## P3 — Broader context-efficiency sweep across more domains/phrasings

**Source:** `benchmarks/CONTEXT_EFFICIENCY.md` verdict section

P1-2 above is evidence from one scenario, not a statistical claim. A genuinely broader sweep
(more fixtures across more domains, more request phrasings per fixture) would strengthen or
narrow the finding, but is out of Phase 8's scope (Phase 8 deliberately avoided building a large
benchmark platform). Reasonable candidate for Phase 9+.
