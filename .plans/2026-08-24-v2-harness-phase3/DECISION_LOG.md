# Decision Log — V2 Harness Phase 3

> Read alongside `spec.md`'s "Design decisions" section for full depth.

---

## Decisions

### 2026-08-24 — `plan.yaml`'s `status` superseded by a full `state` enum, with legacy migration
**Context:** Phase 2's `status` field was write-once (set at `eng plan new`, never updated
again) — effectively unused scaffolding for the real lifecycle Phase 3 needs.
**Decision:** Add `state` carrying the full 14-value lifecycle enum (including `NEEDS_FIX`,
the Verifier-FAIL-with-budget-remaining state goal #8 names explicitly). `status` stays in the
struct for backward parsing; `planmeta.Load` infers `state` from a legacy `status` value when
`state` is empty, defaulting to `NEW` if neither is present.
**Reasoning:** `status` was never load-bearing, so there's no real behavior to preserve beyond
"old files must still parse and get a sensible starting state" — same additive-migration
pattern as Phase 2's own `config_version` fix on `.agent/project.yaml`.
**Alternatives rejected:** A separate `workflow-state.yaml` — contradicts the explicit
instruction to reuse `plan.yaml`.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Reviewer/Verifier verdicts live in `plan.yaml`, not parsed from `review.md`
**Context:** The brief explicitly forbids relying on natural-language parsing to gate
execution. `review.md` is prose with checkboxes; `eng verify` already computes a verdict as
code but never persisted it.
**Decision:** `plan.yaml` gains `review: {verdict, blocking_issues, reviewed_at}` (written by a
new `eng plan review` command, the Reviewer's explicit last step) and
`verification: {verdict, verified_at}` (written automatically by `eng verify`, which already
knows the answer).
**Reasoning:** A one-line command is no heavier for the Reviewer than `eng plan retry` already
is for the Executor. `review.md` remains the human-readable "why"; `plan.yaml` is the only
thing code ever reads.
**Alternatives rejected:** YAML frontmatter on `review.md` (the `SKILL.md` pattern) — that
pattern fits static metadata, not a state transition, which is what `plan.yaml` already models.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Orchestrator automates mechanical stages only; every human/AI stage is a hard stop
**Context:** The brief asks for a lightweight orchestrator while explicitly warning against a
general-purpose workflow engine and against unattended dangerous execution.
**Decision:** `eng workflow advance` performs drift checks, the tasks.md-completion check, and
`eng verify` automatically. It never writes plan content and never invokes an agent
unattended — every AI-driven stage ends with a printed next command and a stop.
**Reasoning:** Matches "preserving human control at important gates" applied to every AI stage,
not only the named approval ones. Keeps the orchestrator's own new failure surface at zero:
everything automatic is either read-only or already-safe (per Phase 2's own verifier
guarantee).
**Alternatives rejected:** Non-interactive `claude -p` auto-driving Planner/Executor — real
automation, but a materially larger, unreviewed-code-change risk; deferred explicitly.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Approval enforcement is a persisted flag + explicit command, not inference
**Context:** Phase 2 only documented approval as a convention. Phase 3's brief explicitly asks
for real enforcement "where possible."
**Decision:** `plan.yaml` gains `requires_approval` (default `true` for `high-risk`, settable
otherwise) and `approved_at`/`approved_by`, populated only by `eng plan approve`.
`eng workflow advance` refuses REVIEWED→APPROVED (and everything after) while
`requires_approval` is true and unapproved, reporting `NEEDS_APPROVAL`.
**Reasoning:** This is the one place Phase 3 can deliver real enforcement at the layer that
exists today (the harness's own bookkeeping) without needing the tool/device policy layer a
real firmware-flash/PLC-write gate would require — that remains correctly out of scope.
**Alternatives rejected:** Inferring approval need from `spec.md` prose — the same
natural-language-inference problem the Reviewer-verdict decision already ruled out.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — One `Adapter` interface; Claude Code is the only implementation; assemble-and-launch, never auto-drive
**Context:** The brief asks to prioritize Claude Code, design for future adapters, and
explicitly avoid deep integration of every agent in this phase.
**Decision:** `internal/agent.Adapter{Name, Available, RolePrompt}`. `ClaudeCodeAdapter`
detects the `claude` binary via the capability registry and assembles a role prompt from
`core/<role>/METHOD.md` plus plan file paths. `eng start` optionally execs into an
*interactive*, terminal-attached `claude` session — the human still drives once it's open.
**Reasoning:** Assembling the right context is mechanical and safe. Launching an interactive
session the human then drives is categorically different from unattended auto-driving
(rejected above) — nothing acts without the human present and typing.
**Alternatives rejected:** A `CollectResult` responsibility on the interface now — meaningless
without non-interactive invocation, which is deferred; speculative surface area otherwise.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Failure routing is a static lookup table
**Context:** "Do not let the agent improvise the next step" is explicit in the brief.
**Decision:** `internal/workflow` encodes every transition (14 states, ~14 rows) as data;
`eng workflow advance` applies exactly one matching row per invocation and stops.
**Reasoning:** A table too small to justify a generic state-machine dependency, and small
enough that an Executor/Planner session cannot argue its way around it.
**Alternatives rejected:** A pluggable/generic FSM library — no benefit at this table size.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Structured command execution is additive; `sh -c` stays the default
**Context:** Phase 2 documented `sh -c` as a Windows-native-execution gap.
**Decision:** `internal/executil.Run` supports a plain-string `Shell` field (existing
behavior, unchanged) and a structured `{Command, Args}` form (no shell). Both
`.agent/project.yaml` and `hooks.yaml` may use either form per key.
**Reasoning:** 100% backward compatible — every existing plain-string command keeps working
exactly as before. The structured form is what new/hand-written configs can opt into.
**Alternatives rejected:** Migrating `detect.go` to emit structured commands now — unrelated
scope creep for this plan; tracked as a later improvement.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — `eng install` gains opt-in PATH setup; default stays print-only
**Context:** Phase 2's own `docs/gotchas.md` entry: hooks assume `eng` is on PATH, and nothing
installs it there.
**Decision:** `eng install` always copies the running binary to
`~/.engineering-harness/bin/` and prints the platform PATH line. `--add-to-path` additionally
applies it (`setx` on Windows; profile-file append on macOS/Linux).
**Reasoning:** `eng install` is re-run often (every harness change); mutating PATH/profile
files unconditionally on every run would be a real surprise. An explicit flag matches this
repo's existing bias toward non-silent, reversible-but-deliberate changes.
**Alternatives rejected:** Unconditional PATH mutation — rejected for the reason above.
**Decided by:** Planner
**Status:** Active

---

## Superseded decisions

_None yet._
