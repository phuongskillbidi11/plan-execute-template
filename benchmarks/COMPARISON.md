# V1 vs V2 (vs illustrative baseline) — Comparison Tables

Every cell below is sourced from a committed `benchmarks/results/*.yaml` file — none of these
numbers were invented for this document. Mode A (`baseline`) rows are labeled
**single-run/illustrative** throughout: they come from one `general-purpose` agent dispatch per
scenario, not a statistical sample, per `.plans/2026-08-26-v2-harness-phase8-benchmark/DECISION_LOG.md`
Decision 3. Treat Mode A columns as "what one reasonable unharnessed run looked like," not as a
baseline with error bars.

## Category 1 — Quick Fix (`quick-fix-timeout`)

| Metric | Mode A — Baseline | Mode B — V1 | Mode C — Harness V2 |
|---|---|---|---|
| Files scaffolded before editing | 0 | 3 (`spec.md`, `tasks.md`, `tests.md`, always) | 1 (`plan.yaml` + template, only after explicit `--risk quick-fix`) |
| Files changed | 1 | 1 | 1 |
| Human interventions | 0 | 1 (hand-authored spec/tasks/tests as Planner) | 1 (explicit `--risk quick-fix` override — see finding below) |
| Manual/agent steps | 5 | 4 | N/A (state-machine driven once triaged) |
| Verification | self-run, PASS | manual (`go build`/`go run`; V1's own `test` hit a local `python3` PATH quirk, not a V1 defect) | `eng verify`, PASS |
| Skills/docs routed | n/a (no concept) | n/a (no concept) | 2 skills (`karpathy-guidelines`, `tcp-ip`), 0 docs |
| Context bundle size | n/a | n/a | 79 lines (`eng adapter prompt executor`) |

**Finding:** `eng workflow start "Increase the reconnect timeout from 1000 ms to 1500 ms."`
auto-triages to risk level `feature`, not `quick-fix` — the exact request text the Phase 8
instruction itself uses as its Quick Fix example. The underlying Quick Fix state machine works
correctly (`mechanism_proof` in `quick-fix-timeout-harness-v2.yaml`: TRIAGED → EXECUTING →
VERIFYING → COMPLETED, 1 file, PASS, once explicitly risk-tagged) — the gap is specifically in
natural-language triage classification, not the workflow itself. See `BACKLOG.md` P1-1.

**Reading this table:** V1 always pays the 3-file-scaffold cost regardless of change size; V2's
mechanism is cheaper *only when triage classifies correctly*, which it currently does not for
this phrasing without a manual override. Mode A pays no planning-artifact cost at all, at the
cost of no audit trail.

---

## Category 2 — Feature / Spec-First (`feature-csv-export`)

| Metric | Mode A — Baseline | Mode B — V1 | Mode C — Harness V2 |
|---|---|---|---|
| Planning artifact before code | none | `spec.md`/`tasks.md`/`tests.md`, written but not gated | `spec.md` gated behind `NEEDS_SPEC_APPROVAL`; `tasks.md`/`tests.md` confirmed absent (still placeholder) until spec approved |
| Spec/requirements approval gate | none (cannot meaningfully pause mid-dispatch) | none (structural — Planner/Executor separation is a convention, not enforced) | enforced (`eng plan approve-spec`) |
| Files changed | 1 | 1 | 1 |
| Test added | no (no existing suite to extend) | no | no |
| Human interventions | 0 | 1 | 3 (spec approval, review verdict, drift recovery from a self-inflicted ordering mistake) |
| Clarifying question needed | yes (CSV header row) — resolved with a defensible default | n/a (Planner decided directly) | n/a |
| Verification | self-run, PASS | manual, PASS | `eng verify`, PASS |

**Confirmed core claim:** `tasks_tests_absent_before_spec_approval: true` — the one hard,
falsifiable success criterion for this category held.

**Two findings surfaced (both documented, neither silently fixed):**
1. *Positive:* drift detection correctly caught a real self-inflicted ordering mistake (editing
   `main.go` while still `APPROVED`, before `EXECUTING`) and forced `NEEDS_REPLAN` rather than
   silently continuing — evidence the Category 7 safety mechanism works outside its own synthetic
   test case too.
2. *Defect:* the plan template's `tasks.md` has two independent, unsynchronized
   completion-tracking conventions (per-task `**Status:**` markers vs. a separate bottom
   "Completion checklist"); only the bottom checklist gates `eng workflow advance`. See
   `BACKLOG.md` P2-1.

---

## Category 8 — Legacy (`legacy-v1-compat`)

| Metric | Mode B — V1 | Mode C — Harness V2 |
|---|---|---|
| `load_skill.sh list` behavior | unchanged: "No manifest found," instructs `update-manifest.sh` | not applicable — this command isn't part of the V2 path |
| `plan-executor.sh new` behavior | unchanged: scaffolds `.plans/<date>-<name>/` normally | not applicable |
| Mode detected | n/a (V1 has no mode concept) | `hybrid`, auto-detected with no flag |
| Existing legacy files touched by `eng init` | n/a | 0 (`CLAUDE.md`/`.plans/`/`skills/` left untouched) |
| Required migration steps | n/a | 0 |
| `eng doctor` / `eng workflow start` behavior | n/a | identical to a harness-native project — no error/warning tied to the legacy shape |

**Reading this table:** V1's own scripts are provably unaffected by every prior phase of harness
work (same fixture, same commands, same output as a pre-Phase-1 run would produce). The harness
auto-detects this exact legacy shape as `hybrid` mode and requires zero migration before normal
`eng` commands work against it.

---

## Categories not run in more than one mode

Categories 3 (Bug Fix), 4 (Large Context), 5 (Cross-Domain), 6 (Tool-Routing), and 7
(Failure/Safety) ran Mode C only (or, for 7, are internal-to-harness safety checks with no V1/
baseline analogue) per `DECISION_LOG.md` Decision 4 — a fresh skill-routing or tool-routing
concept has no V1 equivalent to tabulate against, so no comparison table is produced for them.
Their findings are discussed directly in `CONTEXT_EFFICIENCY.md` and the individual result files.
