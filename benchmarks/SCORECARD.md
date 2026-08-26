# Harness Scorecard — Phase 8

Ten dimensions, each rated from the specific committed evidence named beside it. No rating here
is unsourced. Scale: **Strong** (worked as designed, no material gap found) / **Adequate** (works,
with a documented gap that has a workaround) / **Weak** (material gap, no workaround) — Phase 8
found nothing in the Weak tier.

| # | Dimension | Rating | Evidence |
|---|---|---|---|
| 1 | Setup UX | Strong | `eng init` auto-detected `hybrid` mode against the legacy fixture with zero flags and zero required migration; `eng doctor` reported project mode, stack, skills, tools, and capabilities cleanly. (`legacy-v1-compat.yaml`) |
| 2 | Quick Fix UX | Adequate | Mechanism is correct and cheap once triggered (1 file, TRIAGED→EXECUTING→VERIFYING→COMPLETED, no scaffold overhead) — but auto-triage misclassifies the instruction's own canonical example as `feature`, requiring a manual `--risk quick-fix` override. (`quick-fix-timeout-harness-v2.yaml`, `BACKLOG.md` P1-1) |
| 3 | Feature Planning (Spec-First) | Strong | The one hard, falsifiable claim held: `tasks.md`/`tests.md` were confirmed genuinely placeholder/absent-in-content until `eng plan approve-spec` ran. Drift detection also correctly caught a real out-of-order edit mid-run. (`feature-csv-export-harness-v2.yaml`) |
| 4 | Context Efficiency | Adequate | Skill routing was 0-false-positive on both a cross-domain and a single-domain request; doc/project-context routing was 67% false-positive on the one case tested — a real, asymmetric gap. (`CONTEXT_EFFICIENCY.md`, `BACKLOG.md` P1-2) |
| 5 | Skill Routing | Strong | Exact expected skill set selected for the instruction's own cross-domain example, including correct dependency-expansion (`plc` pulled in only via `siemens-s7`'s `requires:`, not text match) and correct exclusion of every unrelated-domain skill. (`cross-domain-esp32-siemens-modbus-harness-v2.yaml`) |
| 6 | Tool Routing | Strong | READ capability (`git.status`) allowed with an exact, minimal 8-field audit event, no raw output inline; WRITE capability (`git.push`) refused before any subprocess ran, in both the dedicated tool-routing check and the Category 7 safety check. (`tool-routing-git-status-harness-v2.yaml`, `failure-safety-harness-v2.yaml` 10.5) |
| 7 | Safety (Failure Recovery) | Strong | All five failure/safety sub-cases produced the exact required deterministic state transition — reviewer REJECT → `NEEDS_REPLAN`, verifier FAIL → no `COMPLETED`, drift → `NEEDS_REPLAN`, missing approval → stuck at `NEEDS_APPROVAL` across repeated advance calls, denied tool capability → refused pre-invocation. Zero regressions found. (`failure-safety-harness-v2.yaml`) |
| 8 | Legacy Compatibility | Strong | V1's own scripts (`load_skill.sh`, `plan-executor.sh`) behaved identically to an unharnessed run against the legacy fixture; `eng init`/`eng doctor`/`eng workflow start` worked against the same fixture with zero required migration steps and zero legacy files touched. (`legacy-v1-compat.yaml`) |
| 9 | Cross-Domain Usability | Strong | The one cross-domain scenario run (ESP32 + Siemens S7 + Modbus TCP) reproduced Phase 6's own committed router eval exactly, as a durable Phase 8 artifact. Benchmark categories overall span Software, Embedded, Automation, and a documented IT/Networking-adjacent request — not Embedded-only. (`cross-domain-esp32-siemens-modbus-harness-v2.yaml`) |
| 10 | Maintainability | Adequate | One real template-engine gap found (dual, unsynchronized `tasks.md` completion conventions) — internally consistent but non-obvious, and it recurred independently in a second, unrelated scenario without any workaround baked into the tool's own messaging. (`BACKLOG.md` P2-1) |

## Summary

- **Strong:** 7 of 10 dimensions (Setup UX, Feature Planning, Skill Routing, Tool Routing, Safety,
  Legacy Compatibility, Cross-Domain Usability)
- **Adequate:** 3 of 10 dimensions (Quick Fix UX, Context Efficiency, Maintainability) — each has
  a real, documented, workaround-available gap tracked in `BACKLOG.md` (P1-1, P1-2, P2-1)
- **Weak:** 0 of 10 dimensions
- **P0 backlog items (blocking normal use):** 0

No safety, legacy-compatibility, or tool-routing regression was found anywhere in this benchmark
run. The three "Adequate" ratings are real, reproducible gaps — not smoothed over — but none of
them blocks normal use; each has a working manual path around it today.

## Decision

**READY_TO_EXPAND**

Reasoning: every P0-severity bar (blocks normal use) came back clear across all eight benchmark
categories, including the five-way failure/safety sweep and the full legacy-compatibility check.
The three gaps found (Quick Fix triage misclassification, doc-context over-matching, dual
tasks.md conventions) are real and should be prioritized early in whatever comes next — they are
recorded as P1/P1/P2 in `benchmarks/BACKLOG.md`, not waved away — but none of them independently
or together rises to "core refinement required before further use." The harness is measurably
doing what Phases 1–7 designed it to do on the categories this benchmark actually exercised.
