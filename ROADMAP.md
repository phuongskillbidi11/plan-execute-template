# Roadmap

Phases 1–7 built the Global Engineering Harness (global install, plan lifecycle state machine,
context routing, natural-language runtime, multi-domain skill ecosystem, audited tool/MCP
runtime). Phase 8 validated it with a real benchmark suite instead of assuming it worked. Phase
9 fixed everything Phase 8 found, plus two more real defects found via subsequent dogfooding,
and added real-world skill coverage (C#/.NET desktop, Qt/CMake, serial/protocol engineering,
embedded Linux, reverse engineering). Phase 10 turned the Planner/Plan Reviewer/Executor/
Verifier lifecycle from a documented convention into a runtime-enforced one — role activation
is now validated and recorded, and the state machine can no longer retroactively legitimize
work that happened outside it — and added a minimal, read-only Codex delegation adapter. See
[`README.md`](README.md) for what's implemented today and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
for how it fits together. The detailed design record for each phase lives in its own
`.plans/YYYY-MM-DD-v2-harness-*/spec.md` and `DECISION_LOG.md` — this file stays a short,
current-reality summary, not a phase-by-phase archive.

## Where things stand

**Internal dogfooding / team-usable beta.** Phase 8's scorecard: 7/10 dimensions Strong, 3/10
Adequate, 0/10 Weak, 0 P0 (blocking) backlog items — decision `READY_TO_EXPAND` (see
[`benchmarks/SCORECARD.md`](benchmarks/SCORECARD.md)). Phase 9 closed all three Phase 8
refinement items (Quick Fix triage misclassification, doc-context over-matching, the dual
`tasks.md` completion-convention confusion) plus two more found during real dogfooding
(RS485-implies-PLC skill-routing false positive, global/local duplicate skill resolution).
Phase 10 closed a P1 architectural gap found during real dogfooding on an actual investigation
workflow: the workflow state machine could be advanced after real work had already happened
outside it — see [`benchmarks/results/investigation-bypass-blocked.yaml`](benchmarks/results/investigation-bypass-blocked.yaml)
for the real reproduced-and-fixed proof. See [`benchmarks/BACKLOG.md`](benchmarks/BACKLOG.md)
for the full evidence trail on every closed item.

## Now: continued real-world dogfooding

Use the harness — including its now-enforced role lifecycle and the new Codex delegation path —
on real, non-benchmark work across the newly-covered project types (C#/.NET desktop, Qt/CMake,
embedded Linux) and any others that come up, before expanding scope further. Feed anything that
surfaces — a misrouted skill, an approval-friction pattern, a context bundle that's too large in
practice, a role-activation edge case — back into `benchmarks/BACKLOG.md` the same way
Phases 8–10 did, with evidence, not assertion. Known, low-priority latent risks already tracked
there rather than fixed speculatively: `automation/plc`'s bare `automation` tag (same class of
bug as the RS485 fix, but nothing currently exercises it), the inherent budget-vs-breadth
tension when a single request genuinely concerns more skills than the default budget allows,
and the lack of fresh-session isolation for the Verifier role (recommended in docs, not
enforced).

## Later: MCP / skill distribution expansion

Only after real-world dogfooding above has run for a while with no new P0/P1-class findings:

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
