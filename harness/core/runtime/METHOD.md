# Core Method: Runtime Router

The missing piece between "the harness's tools exist" and "a user just describes what they
want." Not an AI role with its own responsibilities — a documented command sequence any
Claude Code session follows automatically when a human describes an engineering requirement
in plain language, instead of typing low-level `eng` commands themselves.

## When this applies

The human describes a requirement in natural language (not a raw `eng` command). Low-level
commands (`eng plan new`, `eng workflow advance`, `eng context bundle`, ...) remain available
for debugging, CI, and advanced manual use — they are never removed, only no longer the
expected default typing surface for a normal request.

## The sequence

1. Run `eng doctor` once per session to confirm harness/project state. If `eng` isn't found on
   PATH in this shell, `eng start` (Phase 9 onward) sets `ENG_HOME` before launching this
   session — fall back to `"$ENG_HOME/bin/eng"` (`%ENG_HOME%\bin\eng.exe` on Windows) rather
   than concluding the harness isn't installed.
2. Run `eng workflow start "<the exact requirement text>"`. This triages the request (see
   `core/triage/METHOD.md`), scaffolds a plan via `eng plan new`, and reports the initial
   state and risk level.
3. Read the reported risk level.
   - `quick-fix` → follow **Quick Fix path**, below.
   - anything else → follow **Spec-First path**, below (the default `planning_mode` for any
     project initialized under Phase 5 onward; a project whose `.agent/project.yaml` predates
     Phase 5, or was never given `planning_mode: spec_first`, instead follows the single-step
     `TRIAGED → PLANNED` path documented in `core/planner/METHOD.md` — check
     `eng workflow status <plan-dir>`'s reported `Profile:` line if unsure).
4. At every state, use `eng workflow advance <plan-dir>` for the mechanical transition — never
   decide a transition by judgment. Before invoking a role, run
   `eng adapter prompt <role> <plan-dir> "<request text>"`, which now folds in that role's
   `eng context bundle` output automatically. This is not a printed suggestion — it is the
   **activation boundary** (Phase 10): it validates the role is compatible with the current
   workflow state and refuses (no prompt printed, non-zero exit) if not, and its success is
   what `eng workflow advance`'s `APPROVED → EXECUTING` gate and `eng tools invoke`'s role check
   actually consult. A refusal here means the workflow genuinely isn't ready for that role yet —
   stop and re-check state (`eng workflow status <plan-dir>`), don't retry blindly or assume the
   role anyway.
5. Never skip a printed gate (`NEEDS_SPEC_APPROVAL`, `NEEDS_APPROVAL`) — stop and ask the
   human explicitly, in the conversation, before proceeding past one.
6. Before a request that plausibly needs an external capability (reading a PR, searching
   docs, inspecting a container, ...), run `eng capabilities explain <role> <plan-dir>
   "<request text>"` to see what would route and at what verdict (`ALLOWED`/
   `NEEDS_APPROVAL`/`BLOCKED`) before doing anything. The only sanctioned way to actually
   invoke one is `eng tools invoke <role> <capability> <plan-dir> [args...]` — never a raw
   shell command to an external service inside a session. A `NEEDS_APPROVAL` result means
   stop and ask the human, exactly like any other approval gate in this document.

## Quick Fix path

1. If, while working, the change turns out to be broader than it looked (touches more than
   the one localized area, needs a schema/API change, needs review), **do not continue as a
   quick fix** — run `eng plan escalate <plan-dir> --to bug|feature|architecture --reason
   "..."`, then flesh out `spec.md`/`tasks.md`/`tests.md` into the full format and resume via
   the Spec-First or auto_plan path from `TRIAGED`.
2. Otherwise, edit `tasks.md`'s one task block, make the localized change, and run
   `eng verify <plan-dir>`. On `PASS`, a compact `quick_fix` event is recorded automatically —
   no further documentation is needed for a genuine quick fix.

## Spec-First path (the Phase 5 default for new projects)

1. After `eng workflow start`, the state is `TRIAGED`. Write **only** `spec.md` — do not
   write `tasks.md`/`tests.md` yet.
2. Run `eng workflow advance <plan-dir>` — moves to `NEEDS_SPEC_APPROVAL`.
3. Stop. Show the human `spec.md`'s Goal in the conversation and ask for explicit approval or
   revision — the same way this repository's Planner has always asked for spec confirmation.
4. Once approved, run `eng plan approve-spec <plan-dir>` — this is a **requirements**
   approval, distinct from the execution-risk approval gate later in the lifecycle.
5. Run `eng workflow advance <plan-dir>` — moves to `SPEC_APPROVED`.
6. Now write `tasks.md` and `tests.md`.
7. Run `eng workflow advance <plan-dir>` — moves to `PLANNED`, and the rest of the lifecycle
   continues exactly as documented in `core/plan-reviewer/METHOD.md`,
   `core/executor/METHOD.md`, and `core/verifier/METHOD.md`.

## Constraint

This document never authorizes skipping a state, inventing a transition, or treating a
heuristic (Triage's keyword match) as authoritative. Every state change still flows through
`eng workflow advance`'s deterministic `Facts → Decide → Decision` — see
`core/context-manager/METHOD.md`'s fail-safe rule for the same principle applied to context
selection.
