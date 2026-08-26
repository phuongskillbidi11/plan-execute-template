# Roadmap

Phases 1–7 built the Global Engineering Harness (global install, plan lifecycle state machine,
context routing, natural-language runtime, multi-domain skill ecosystem, audited tool/MCP
runtime). Phase 8 validated it with a real benchmark suite instead of assuming it worked. See
[`README.md`](README.md) for what's implemented today and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
for how it fits together. The detailed design record for each phase lives in its own
`.plans/YYYY-MM-DD-v2-harness-*/spec.md` and `DECISION_LOG.md` — this file stays a short,
current-reality summary, not a phase-by-phase archive.

## Where things stand

**Internal dogfooding / early team use.** Phase 8's scorecard: 7/10 dimensions Strong, 3/10
Adequate, 0/10 Weak, 0 P0 (blocking) backlog items. Decision: `READY_TO_EXPAND` — see
[`benchmarks/SCORECARD.md`](benchmarks/SCORECARD.md).

## Now: Core Refinement

Address the real, benchmark-confirmed gaps in [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md)
before adding new surface area:

- **P1 — Quick Fix triage misclassification.** Broaden the triage keyword list (or add a
  size-based heuristic) so plausible numeric/parameter-tweak requests route to Quick Fix
  without a manual `--risk quick-fix` override.
- **P1 — Doc-context over-matching.** Bring `internal/docsearch.Match`'s doc-section routing
  closer to `internal/skillmatch.Score`'s weighted model instead of its current apparent
  simple word-overlap approach.
- **P2 — Dual `tasks.md` completion conventions.** Either drop the per-task `Status:` marker in
  favor of the bottom checklist alone, or make `tasksComplete()` recognize both — and make the
  "unchecked items" message name the specific blocking line(s).

Each of these needs its own scoped plan under `.plans/` with a regression test proving the fix,
followed by a re-run of the relevant benchmark scenario to confirm it actually closed the gap —
not just a code change asserted to fix it.

## Next: continued real-world dogfooding

Use the harness on real, non-benchmark work across more than one domain before expanding scope
further. Feed anything that surfaces — a misrouted skill, an approval-friction pattern, a
context bundle that's too large in practice — back into `benchmarks/BACKLOG.md` the same way
Phase 8 did, with evidence, not assertion.

## Later: MCP / skill distribution expansion

Only after the refinement backlog above is addressed:

- A real MCP JSON-RPC transport (today's one MCP-style adapter, `docs-search`, is a
  deterministic local mock — see [`docs/tools.md`](docs/tools.md)).
- Broader skill packaging/distribution beyond directory-tree + precedence-tier resolution (a
  version-constrained skill package manager is explicitly not built yet).
- Additional real tool adapters, added the same way as today's (`tooladapter.Adapter` +
  registration in `cli/tools_cmd.go`), each with an explicit risk-tier and approval-policy
  decision — never silently defaulted to `ALLOWED`.

## Explicitly not planned right now

These would meaningfully increase scope/risk without a demonstrated need yet — revisit only if
real usage surfaces one:

- A tool/skill marketplace or public registry
- Live industrial-control write adapters (PLC/Modbus/OPC UA) — Phase 7's risk tiers exist so a
  future one has a safety model to plug into, not because one is scheduled
- Cloud telemetry, a benchmark SaaS, or a distributed-runner benchmark platform — Phase 8
  deliberately stayed local/deterministic; see [`benchmarks/README.md`](benchmarks/README.md)
- Multi-executor parallel/autonomous execution across independent task groups

## V1 template

The original per-project Planner/Executor template (`scripts/`, project-local `skills/`,
`.plans/`) is not on this roadmap — it is a completed, stable surface the harness is required
to keep working unmodified (see [Backward compatibility promise](README.md#backward-compatibility-promise)
in the README). Changes to it happen only to fix an actual regression, never as a feature
enhancement.
