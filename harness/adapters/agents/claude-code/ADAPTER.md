# Adapter: Claude Code

Reference implementation of the `internal/agent.Adapter` interface.

## Capability

Detected via `claude` on PATH (see `eng capabilities list`).

## Responsibilities implemented

- **Detect** — `Available()` checks the capability registry.
- **Provide role instructions** — `RolePrompt(role, planDir)` reads
  `core/<role>/METHOD.md` and prepends it to a short block naming the plan directory and
  which files to read first.

## Responsibilities NOT implemented (by design — see Phase 3 DECISION_LOG)

- **Launch agent (non-interactive)** — `eng start` launches an *interactive* `claude` session
  attached to the current terminal; it does not pipe a prompt in and run unattended.
- **Collect result** — meaningless without non-interactive invocation; deferred.

## Adding a new adapter

Implement `internal/agent.Adapter` (`Name`, `Available`, `RolePrompt`) and register it where
`eng adapter`/`eng start` select an adapter. No other file in this repository should need to
change — this is exactly the boundary the interface exists to draw.
