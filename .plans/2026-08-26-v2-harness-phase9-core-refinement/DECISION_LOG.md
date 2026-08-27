# Phase 9 Decision Log

## 1. Triage stays a keyword-category heuristic, not a scored/weighted model

Considered scoring every request across all named signals (scope, API impact, migration impact,
security impact, hardware impact, verification simplicity) as the instruction's signal list
suggests. Rejected for triage specifically (unlike skill/doc matching, where weighted scoring
*is* the fix) because most of those signals — "estimated change scope," "number of affected
symbols/files" — are not computable before a plan exists; `eng workflow start` triages raw
request text only. Chose instead: keep the existing table-ordered category match (already
risk-descending, already checked first-match-wins), broaden the risk-elevating categories, and
add one new narrowly-scoped quick-fix-positive signal (change-verb + digit) that is only
consulted after every risk category has already had a chance to match. This preserves
"deterministic and explainable" and requires the smallest possible diff, since the
risk-priority ordering that makes this safe already existed in the code before this phase.

## 2. Skill/doc matching gets word-boundary tokenization AND weighted scoring — both, not one

Reproduction found two independent failure classes (see spec.md): a pure substring bug (`"form"`
inside `"WinForms"`) and a genuine-whole-word-but-generic-vocabulary collision (`"protocol"`,
`"framing"`). Word-boundary tokenization alone fixes the first but not the second; a
weight+threshold alone fixes the second but not the first (a substring hit is still a "real"
score-1 event, and two unlucky substring hits could still cross any threshold by accident).
Both fixes are structurally cheap (a tokenize-and-match helper, and a constant weight table),
so there was no reason to ship only one and leave a known-reproduced gap in the other.

## 3. No new config surface for the matching threshold

Considered exposing `min_skill_match_score`/`min_doc_match_score` in `.agent/context.yaml`.
Rejected: nothing in this phase's reproduction showed a real need to tune it per-project, and
the instruction explicitly says "do not overcomplicate." Package-level constants
(`skillmatch.TagTriggerWeight`, `skillmatch.DescriptionWordWeight`, `skillmatch.MinMatchScore`,
and the `docsearch` equivalents) are exported so a future phase can promote them to config if a
real need appears, without another schema migration today.

## 4. `docsearch`/`skillmatch` weight calibration was checked against the existing test suite AND the real reproduction, not chosen blindly

`skillmatch`'s `TestDomainProfileFillAfterStrongMatches` relied on the *old* weak-scoring
behavior (a skill whose `Description` happens to literally contain the request word) to
represent "a strong match." Under the new weighted scheme this is a single description-word hit
(weight 1), which correctly no longer clears the new minimum-score threshold (2) — the fixture
was updated to give that skill a real `Tags` entry instead, which is what an actual "strong
match" should look like under the new design; the test's *intent* (a genuine match alongside an
unrelated domain-profile fill) is unchanged, only its synthetic fixture is. This is called out
explicitly rather than silently editing the assertion, per Phase 8's own "don't optimize a
result to pass" precedent.

## 5. `tasks.md` completion: keep the existing gate unchanged, fix the confusion instead

Considered making `tasksComplete` also require every per-task `**Status:**` marker to be
`[x]`. Rejected: every plan created since Phase 3 has only ever had its bottom checklist
checked (that's the only thing the code has ever gated on) — requiring the per-task markers too
would silently block `eng workflow advance` on every pre-Phase-9 plan whose bottom checklist is
done but whose per-task markers were never individually updated, which is the opposite of
"legacy plans do not break." The bottom checklist remains sole authority (zero logic change);
the fix is entirely in observability (name the specific unchecked line(s) in the advance
output) and in the template's own header text (state plainly which marker is authoritative).

## 6. Skill dedup collapses only a legacy-vs-qualified bare-name collision, never two qualified skills

`automation/modbus` and a hypothetical future `networking/modbus` sharing the bare name
`modbus` is Phase 6's own documented, intentional behavior (`docs/skills.md`) — collapsing that
would be a real regression, not a fix. The new collapse pass in `skills.ResolveWithPrivate` only
triggers when a bare-name group mixes at least one legacy (frontmatter-less, `Domain` `""`/
`"unknown"`) entry with at least one qualified entry; a group made entirely of qualified,
distinct-domain entries is left untouched. This is deliberately narrower than "dedupe anything
with a matching short name," per the instruction's own explicit warning against that.

## 7. Skill taxonomy: two skills per new project class, not one broad one

`software/csharp-dotnet` + `software/winforms` (not one combined skill) so a non-WinForms C#/.NET
project (a console tool, an ASP.NET service) still gets the `.NET`-level guidance without an
unrelated UI-framework skill being forced in — matches the instruction's explicit "do not make
WinForms mandatory for all C# projects." Same reasoning for `embedded/embedded-linux` (concepts)
vs. `embedded/cross-compilation` (the CMake/toolchain mechanics) — a project that cross-compiles
without CMake (a raw Makefile + toolchain) still benefits from the sysroot/ldd/systemd half
without inheriting CMake-specific instructions it doesn't need.

## 8. `software/serial-communications` never mentions Modbus addressing/framing specifics

The whole point of P2-2 is that RS485/serial vocabulary must not imply PLC/Modbus. Duplicating
even a little of `automation/modbus`'s content into the new serial skill would silently
re-couple the two domains through shared prose vocabulary — exactly the mechanism that caused
the original bug. Instead, `software/serial-communications`'s `when_not_to_use` explicitly names
`automation/modbus` as the skill to reach for once an actual Modbus PDU/register scheme is in
play, and the two skills share no overlapping trigger words by design.

## 9. `eng benchmark` CLI: still not built

Reaffirming Phase 8's Decision 1 for this phase too — new scenario/result YAML files under the
existing `benchmarks/` convention, run via documented command sequences in
`benchmarks/README.md`, exactly like every other Phase 8 scenario. Nothing about Phase 9's
defects changes that calculus.

## 11. `embedded/esp32`'s bare "embedded" tag was fixed too, within P2-2's existing scope

While building and dogfooding the new `qt-cmake-embedded-linux` eval scenario (Task 8),
`embedded/esp32` false-matched the request (score 4, via a bare `embedded` tag colliding with
the request's literal word "embedded") and won a budget slot ahead of the genuinely-relevant
`embedded/cross-compilation` (score 2) under the real default `max_skills: 5` budget. This is
the same defect class as P2-2 — a generic word functioning as a strong (weight-3) curated-tag
signal — found via dogfooding this phase's own new content, not hypothesized in advance. Fixed
by removing the tag (`esp32`'s real identifying triggers/tags are untouched); a regression guard
was added directly to the eval scenario (`forbidden_skills: [esp32]`) rather than deferred,
since it directly blocked this phase's own Example B acceptance criterion and the fix is a
single-line, well-justified, in-scope tag removal — not a new unrelated refactor.
`automation/plc`'s own bare `automation` tag is a similar latent risk, observed but **not**
fixed in this phase: nothing in Phase 9's own scenarios exercises it, so per the same
bounded-fix discipline, it is left as a documented risk for a future pass rather than fixed
speculatively.

## 12. `ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION` are set unconditionally when known, never guessed

`cmdStart` already computes `harnessDir()` and the project's own `os.Getwd()` before launching
the agent — these are real, already-resolved values, not something a new mechanism has to infer.
`ENG_VERSION` is read from the existing `~/.engineering-harness/VERSION` file (the same one
`eng doctor` already reads) via a small shared helper, factored out to avoid a second inline
copy of that read — not a new versioning mechanism.
