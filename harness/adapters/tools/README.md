# Tool/MCP Adapters (placeholder)

This directory is the intended home for future external-tool adapters — the
`internal/tooladapter.Adapter` implementations beyond the one reference implementation
(`GitAdapter`) that ships with Phase 5.

Nothing here is implemented yet. Per Phase 5's explicit scope constraint, no docker/ssh/
github/database adapter, and certainly no live PLC/Modbus/OPC UA/industrial-control adapter,
is built in this phase — this is architectural foundation only.

See `cli/internal/tooladapter/tooladapter.go` for the interface and
`.plans/2026-08-24-v2-harness-phase5-runtime/spec.md` Decision 10 for why Tool Adapters are
kept structurally separate from Agent Adapters (`harness/adapters/agents/`).
