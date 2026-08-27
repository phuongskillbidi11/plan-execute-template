# Phase 9 — Core Refinement & Real-World Skill Coverage

## Goal

Move the harness from "internal dogfood / alpha" toward "team-usable beta" by fixing the
concrete P1/P2 defects Phase 8's benchmark and real dogfooding surfaced, and by adding a
focused set of real-world skills (C#/.NET desktop, Qt/CMake, serial/protocol engineering,
embedded Linux, reverse engineering) that the harness is now actually being used against. This
is refinement, not expansion: no new architectural layer, no marketplace, no MCP transport, no
new subsystem.

## Architecture / gap analysis (read before the fix strategy below)

Every defect below was inspected directly in the current `cli/` source and, where practical,
**reproduced empirically against the real binary** before any fix was designed — not assumed
from the bug report text.

### P1-1 — Quick Fix triage misclassification

`cli/triage_cmd.go`'s `keywordLevel` iterates `triageKeywords` (`high-risk` → `architecture` →
`bug` → `quick-fix`, in that literal table order) and returns the **first** category whose word
list matches, falling through to `"feature"` if nothing matches. Two real gaps:

1. The `quick-fix` word list (`typo`, `rename`, `comment`, `formatting`, `"small change"`) has
   no entry for numeric/parameter-tweak phrasing — confirmed via `eng triage "Increase the
   reconnect timeout from 1000 ms to 1500 ms."` returning `feature` (already documented in
   `benchmarks/BACKLOG.md` P1-1).
2. The `high-risk`/`architecture` word lists don't cover security/auth, hardware-control, or
   schema/public-API-contract phrasing at all — so a request combining a risky signal with
   size-hinting wording has nothing to elevate it, once the fix in (1) is added.

**Load-bearing fact for the fix:** the table is already checked in risk-descending order before
falling through to the default — a risk-elevating match today already wins over anything later
in the table for free. The fix does not need to restructure this priority order; it only needs
(a) broader risk-elevating word lists and (b) a new, narrowly-scoped quick-fix-positive signal
that runs *after* the existing table and therefore automatically loses to any risk match.

### P1-2 / P2-2 — weak, unweighted, unbounded substring scoring (docsearch + skillmatch)

Both `docsearch.Match` (project docs) and `skillmatch.Score` (skills) share the same structural
weakness: every signal (title word, body word, tag, trigger, description word) is worth exactly
1 point via `strings.Contains`, and a caller includes anything with `score > 0`. Two distinct
failure modes were found, reproduced against the real binary:

1. **Naive substring, no word boundary.** `skillmatch.Score`'s description-word check does
   `strings.Contains(requestText, descriptionWord)` — a substring test, not a token test. The
   word `"form"` (from `engineering/debugging`'s description "...form a hypothesis...") is a
   substring of the request word `"WinForms"`, scoring a match neither word actually shares.
   Confirmed via `eng context skills "Maintain a C# WinForms UHF RFID configuration tool using
   RS485 and USB-HID binary protocol framing."` selecting `engineering/debugging`.
2. **Generic-vocabulary collision, no minimum strength.** `automation/siemens-s7`'s description
   contains the word `"protocol"` (from "protocol split"); `automation/modbus`'s description
   contains `"framing"` (from "framing pitfalls"/"framing differs"). The same RS485 request
   above contains both `"protocol"` and `"framing"` as genuine whole words — a real, non-bug
   substring match, but on generic engineering vocabulary that has nothing to do with either
   skill's actual subject matter. `automation/plc` then gets force-included via
   `automation/siemens-s7`'s `requires:` edge, even though `plc` itself never matched anything —
   `skillrouter.Route`'s final forced-dependency pass (Tier A-continuation) applies to the whole
   final selection, not just Tier A/explicit skills, so one weak Tier B match cascades its
   `requires:` closure regardless of budget. Reproduced end to end: `eng context skills` on the
   RS485 request above selects `modbus`, `siemens-s7`, `plc`, and `tcp-ip` (via `modbus`'s
   `recommends:`) — none of which have any real bearing on a C#/RS485/USB-HID project.

`docsearch.Match` has the identical shape (word length `> 2`, uniform weight, `score > 0`
inclusion, no title/body distinction) and was already shown in Phase 8 to over-match 4 of 6 doc
sections for a request that should concern only 1 (`benchmarks/results/
large-context-auth-validation-harness-v2.yaml`).

**Fix direction confirmed by reproduction, not guessed:** both functions need (a) word-boundary
(tokenized) matching instead of raw substring containment, to kill class-1 false positives
outright, and (b) a weighted score with a minimum-strength inclusion threshold, to kill class-2
false positives — a single curated signal (a skill's `tags:`/`triggers:`, a doc section's
*title*) should be enough on its own, but a single generic prose-word match should not. This
matches the instruction's own suggested signals (exact phrase match, heading relevance,
skill-specific triggers, minimum relevance threshold) without adding embeddings or an LLM.

### P1-3 — `eng start` child environment / PATH

`cli/start_cmd.go`'s `cmdStart` calls `exec.Command("claude")` with `c.Env` left `nil`, which
Go documents as "inherit the current process's environment" — so the mechanism is not "the
child doesn't inherit PATH," it's "the *parent* eng.exe process's own environment never had the
harness `bin/` directory on PATH in the first place." This matches the reported repro exactly:
the user invoked `eng` via its **full path**
(`C:\Users\Admin\.engineering-harness\bin\eng.exe start`), which only works when PATH lookup is
bypassed entirely — meaning that shell session's PATH was never actually updated (consistent
with `eng install --add-to-path`'s own documented Windows caveat: `setx` only affects *new*
sessions). `claude`, and any shell it itself later spawns (Claude Code's own Bash tool), inherit
that same stale environment.

**Fix direction:** `cmdStart` already knows two reliable locations for a working `eng` binary at
the moment it runs — its own executable's directory (`os.Executable()`) and the installed
`binDir()` — regardless of what the inherited PATH says. Explicitly prepending both (deduped) to
the child's `PATH`, and setting `ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION`, closes the gap
without depending on the parent shell's PATH being correct and without destroying whatever PATH
already exists.

### P1-4 — `workflow: {triage,plan_review,verifier}: false` vs. unset

Confirmed directly in `cli/internal/project/project.go`: `Workflow.enabled()` returns
`w.Triage || w.PlanReview || w.Verifier`, and `Config.EffectiveWorkflow()` falls back to an
all-`true` `Workflow{...}` whenever `enabled()` is `false` — which is indistinguishable between
"the block was never written" and "every field was explicitly set to `false`." Only one call
site actually consumes this today: `workflow_cmd.go`'s `gatherFacts` reads
`cfg.EffectiveWorkflow().PlanReview`. `Triage`/`Verifier` are not read anywhere in code yet
(reporting-only fields, same class of latent gap as `RequireApproval` before Phase 7 — see
`docs/tools.md`) but must still round-trip literally, since `eng doctor` is asked to report them
and a future consumer should not silently inherit today's bug.

**Fix direction:** the codebase already solved this exact ambiguity once, for
`RequireSpecApproval` (a `*bool`, `nil` = unset = default `true`, non-nil = literal). Apply the
identical pattern to `Triage`/`PlanReview`/`Verifier`. Per-field pointer defaulting makes the
group-level `enabled()` heuristic entirely unnecessary — each field now defaults independently,
which is simpler than what it replaces, not more complex.

### P2-1 — dual `tasks.md` completion conventions

Confirmed in `harness/templates/plan/tasks.md`: every task carries its own
`**Status:** \`[ ]\`` marker, and a separate bottom `## Completion checklist` section carries
`- [ ] All tasks marked \`[x]\`` plus five more items. `cli/workflow_cmd.go`'s `tasksComplete`
only scans for lines with the literal prefix `"- [ ]"` — i.e. only the bottom checklist gates
`eng workflow advance`; the per-task `**Status:**` field is decorative today. This was already
directly observed causing real confusion during Phase 8 (`benchmarks/results/
feature-csv-export-harness-v2.yaml`'s `finding_2_defect`, recurred in
`benchmarks/results/bug-lastn-off-by-one-harness-v2.yaml`).

**Fix direction:** keep the bottom checklist as the sole machine-authoritative source (zero
change to `tasksComplete`'s logic — the smallest possible, fully backward-compatible choice: no
existing plan's completion state changes). Fix the actual confusion at its two real sources
instead: (a) the template's own header text currently doesn't say which marker is authoritative
— make it say so explicitly; (b) `eng workflow advance`'s "tasks.md still has unchecked items"
message gives no indication of *which* line is blocking — surface the actual unchecked line(s).

### P2-3 — global/local duplicate skill resolution

Confirmed in `cli/internal/skills/skills.go`: `ResolveWithPrivate` merges by `QualifiedName()`,
and `QualifiedName()` returns the bare `Name` whenever `Domain` is `""`/`"unknown"` (the legacy,
frontmatter-less parse path) — so a global `engineering/karpathy-guidelines` (qualified name
`"engineering/karpathy-guidelines"`) and a project-local legacy `karpathy-guidelines` (qualified
name `"karpathy-guidelines"`, since its `Domain` is `"unknown"`) never collide in the
merge-by-key map and are **both** returned as distinct resolved skills, even though they are the
same conceptual skill. `docs/skills.md` already documents the *intended* precedent for two
skills sharing a bare name in **different, real domains** (`automation/modbus` vs. a
hypothetical `networking/modbus`) — that case must keep working; only the
legacy-shadow-of-a-qualified-skill case is the bug.

**Fix direction:** after the existing per-key merge, add one bare-name collision pass: group the
merged set by bare name (the part after the last `/`, or the whole name if unqualified); a group
is only collapsed when it mixes at least one legacy (`Domain` `""`/`"unknown"`) entry with at
least one qualified entry — a group made entirely of qualified (real, distinct-domain) entries is
left untouched. Within a collapsed group, keep the highest-precedence tier (`local` > `private` >
`global`, the existing `Source` field already carries this). This is metadata/qualified-identity
based, not content hashing, per the instruction's own guidance.

## What's out of reproduction reach at triage time (documented, not solved differently)

Several of the suggested triage signals — "number of affected files/symbols," "estimated change
scope" — cannot be computed before a plan exists: `eng workflow start` runs triage on raw
request text alone, before any spec/tasks/diff exists. The fix therefore stays a text-based
heuristic, the same category the existing triage already is, just with broader and
bidirectional (elevate-or-permit) keyword coverage. This is recorded as a real, known limit of
the approach, not silently worked around.

## Fix strategy (implementation-level, one paragraph per defect)

- **P1-1:** extend `triageKeywords`'s `high-risk`/`architecture` word lists with security/auth,
  hardware-control, and schema/public-API terms (still checked first, so they keep priority for
  free); add a new, narrowly-scoped `looksLikeParameterTweak` check (a change-verb word +
  presence of a digit) consulted only after the existing table finds nothing, so it can never
  override a risk-elevating match.
- **P1-2 / P2-2:** rewrite `skillmatch.Score` and `docsearch.Match` to tokenize both the request
  and each candidate signal (letters/digits only, lowercased) and match whole tokens/phrase
  sequences instead of raw substrings; weight curated signals (tags/triggers; doc section
  titles) higher than prose signals (description words; doc body words); require a minimum
  combined score before a text-match tier includes anything. Constants, not new config surface —
  "keep the classifier deterministic and explainable," matching the existing style of this
  codebase (no embeddings, no LLM).
- **P1-3:** add a small, pure `buildChildEnv` helper in `cli/start_cmd.go` that prepends the
  running binary's own directory and the installed `bin/` directory (deduped) to a copied
  `PATH`, and sets `ENG_HOME`/`ENG_PROJECT_ROOT`/`ENG_VERSION`; wire it into `cmdStart`'s
  `exec.Command("claude")` via `c.Env`.
- **P1-4:** change `project.Workflow`'s `Triage`/`PlanReview`/`Verifier` from `bool` to `*bool`
  (matching `RequireSpecApproval`'s existing pattern exactly), add three
  `TriageEnabled()`/`PlanReviewEnabled()`/`VerifierEnabled()` accessors (nil → default `true`,
  non-nil → literal), delete the now-unnecessary `enabled()`/`EffectiveWorkflow()` group
  fallback, and update the one real call site (`workflow_cmd.go`'s `gatherFacts`).
- **P2-1:** edit `harness/templates/plan/tasks.md`'s header note to state the bottom checklist is
  the only thing that gates `eng workflow advance`; add a small helper in `workflow_cmd.go` that
  reports the actual unchecked checklist line(s) alongside the existing "tasks.md still has
  unchecked items" message.
- **P2-3:** add a bare-name collision-collapse pass to `skills.ResolveWithPrivate`, and extend
  `skillvalidate.Validate` to report (as an informational/warning `Issue`) which source actually
  won whenever a raw discovered skill file was shadowed — by ordinary tier precedence or by the
  new collapse — so `eng skills validate` stays the observability surface for "which source won."

## Skill taxonomy additions (decided before any skill file is written)

| Skill | Level | Domain | Requires | Recommends | Purpose |
|---|---|---|---|---|---|
| `software/csharp-dotnet` | technology | software | — | — | .NET Framework vs. modern .NET awareness, target-framework inspection, P/Invoke, x86/x64, managed/unmanaged boundary |
| `software/winforms` | technology | software | `software/csharp-dotnet` | — | WinForms lifecycle, UI-thread rules (`Invoke`/`BeginInvoke`), designer-generated code caution |
| `software/qt` | technology | software | `software/cpp` | — | QObject ownership, signals/slots, event loop, QThread/thread affinity, AUTOMOC/AUTOUIC/AUTORCC |
| `software/cmake` | technology | software | — | — | targets, include/link propagation, toolchain files, out-of-source builds, cross compilation |
| `engineering/protocol-engineering` | engineering | engineering | — | — | domain-agnostic method: framing, buffering, timeouts, CRC/checksum, endianness, retries, transport-vs-protocol separation, capture/replay |
| `software/serial-communications` | technology | software | — | `engineering/protocol-engineering` | RS232/RS485/USB-HID transport specifics — deliberately **not** Modbus-specific; explicit `when_not_to_use` pointing at `automation/modbus` for that |
| `embedded/embedded-linux` | domain | embedded | — | — | cross-toolchain/sysroot concepts, target filesystem, shared-library dependency inspection (`ldd`/`readelf`), runtime loader paths, resource constraints |
| `embedded/cross-compilation` | technology | embedded | — | — | CMake toolchain files, sysroot, cross-toolchain triples — the mechanical half of embedded-linux, kept separate so a non-CMake cross-compile story can still use it |
| `engineering/reverse-engineering` | engineering | engineering | — | — | preserve verified behavior, evidence-based hypotheses, binary/source boundary awareness, small reversible changes, explicitly non-malware-analysis scope |

No board-specific skill is added (explicitly out of scope). `software/qt` reuses `software/cpp`
via `requires:` rather than repeating generic C++ guidance. `RS485`/serial triggers live only on
`software/serial-communications`, never on any `automation/*` skill, and that skill's own
`when_not_to_use` explicitly names Modbus as a distinct, separately-triggered thing — this is
the structural fix behind "serial != Modbus," not a one-off keyword exclusion.

## Domain/profile routing — expected outcomes after this phase

- **Example A** — `"Maintain a C# WinForms UHF RFID configuration tool using RS485 and
  USB-HID."` → `software/csharp-dotnet`, `software/winforms`, `software/serial-communications`
  (+ its `engineering/protocol-engineering` recommend if budget allows), plus whatever of
  `engineering/debugging`/`engineering/testing` genuinely scores on the request text. **Must
  not** select `automation/plc` or `automation/siemens-s7`.
- **Example B** — `"Build this Qt application for an ARM embedded Linux target using CMake."` →
  `software/qt` (+ `software/cpp` via `requires:`), `software/cmake`, `embedded/embedded-linux`,
  `embedded/cross-compilation`, `it/linux`.
- **Example C** — `"ESP32 reads Siemens S7-1200 over Modbus TCP."` → unchanged from Phase 6:
  `esp32`, `siemens-s7`, `plc`, `modbus`, `tcp-ip` — re-verified, not re-derived, since none of
  this phase's changes touch that scenario's own signals.

## Benchmark acceptance criteria (fixed before implementation, per Phase 8's own convention)

Reusing Phase 8's infrastructure — no new benchmark platform, no `eng benchmark` CLI. New/updated
`benchmarks/scenarios/*.yaml` + `benchmarks/results/*.yaml`, run the same way (real commands
against real fixtures, structural proxies only, never invented token counts):

1. `eng triage "Increase the reconnect timeout from 1000 ms to 1500 ms."` → `quick-fix`.
2. `eng triage` on a request combining a size-hint verb with a security/hardware keyword → stays
   at or above its pre-existing risk category, never downgraded to `quick-fix`.
3. `eng context project` on the Phase 8 large-context fixture's auth-validation request →
   materially fewer than the previously-measured 4/6 sections (target: 1/6, the correct answer;
   accept 1–2/6 as "materially reduced" if a legitimate secondary section also scores highly).
4. A pure-function test proves `eng start`'s child environment carries a working `PATH` entry
   for `eng` and the three `ENG_*` variables, without needing a real `claude` binary or a real
   child process spawn.
5. `Workflow{}` (nil pointers) resolves identically to today's default; `{true,true,true}`,
   `{true,false,true}`, and `{false,false,false}` each round-trip and resolve literally —
   `{false,false,false}` must **not** resolve to enabled.
6. `eng workflow advance` on a plan with every per-task `**Status:**` marked `[x]` but the
   bottom checklist still `- [ ]` reports which checklist line(s) are blocking, not just a
   generic message.
7. `eng skills list`/`eng context skills` against a fixture with both a global
   `engineering/karpathy-guidelines` and a legacy project-local `karpathy-guidelines` (identical
   content) resolves to exactly one entry, sourced from the higher-precedence tier.
8. `eng context skills` on the RS485/WinForms request (Example A) selects the new C#/serial
   skills and does **not** select `automation/plc` or `automation/siemens-s7`.
9. `eng context skills` on the Qt/CMake/embedded-Linux request (Example B) selects
   `software/qt`, `software/cmake`, `embedded/embedded-linux`, `embedded/cross-compilation`,
   `it/linux`.
10. `eng context skills "ESP32 reads Siemens S7-1200 over Modbus TCP"` still selects exactly
    Phase 6's expected set — unchanged.

## Out of scope (unchanged from the instruction, restated for the plan's own record)

Real MCP JSON-RPC transport, plugin/skill marketplace, cloud registry, full package manager,
live Modbus write, PLC output control, production deploy automation, arbitrary SSH execution,
distributed agents, vector DB, semantic embeddings, autonomous swarm, board-specific skill
packs, a new `eng benchmark` CLI, any redesign of the CLI → Runtime → Triage → Context Manager →
Skill Router → Workflow → Roles → Capability/Tool Router → Adapters layering.

## Fixture strategy

Reuse `benchmarks/fixtures/` conventions from Phase 8. Two new small fixtures represent the two
real project shapes named in the instruction, without copying any private/large repository:

- `benchmarks/fixtures/csharp-winforms-rfid/` — a minimal C# project shape (`.csproj`, a
  `Program.cs`/form stub) whose only purpose is to exist as a `eng init`-able directory for the
  Example-A routing scenario; the routing scenarios themselves only need `eng context skills`
  against a request string, not a real build.
- `benchmarks/fixtures/qt-cmake-embedded-linux/` — likewise minimal: a `CMakeLists.txt` shape
  representative enough for stack detection, no real Qt/toolchain dependency.

A third fixture, `benchmarks/fixtures/legacy-duplicate-skill/`, reproduces the exact P2-3 shape:
a project-local `skills/karpathy-guidelines/SKILL.md` (legacy heading convention, no
frontmatter) alongside the real global `engineering/karpathy-guidelines`.
