# Core Method: Triage

Domain-agnostic request classification, run before any plan is written.

## Levels

| Level | Examples | Workflow |
|---|---|---|
| quick-fix | typo, rename, comment, formatting | Single-file plan — skip spec/tasks/tests split |
| bug | reproducible defect, broken behavior | Reproduce → fix → regression test |
| feature | new capability, no architecture change | Full spec.md + tasks.md + tests.md |
| architecture | crosses module boundaries, changes a decision | Research first, consult ADRs, full plan + Plan Reviewer required |
| high-risk | production deploy, data migration, firmware flash, PLC write, destructive operation | Full plan + Plan Reviewer + explicit human approval before Executor touches anything real |

## Role

Triage is a classification step, not a gate — it decides which workflow a request follows,
not whether the request is allowed. `eng triage "<text>"` provides a keyword-based hint;
the Planner makes the actual determination using full context the heuristic doesn't have
(the project's own conventions, `docs/gotchas.md`, prior `DECISION_LOG.md` entries).

## Before writing spec.md

1. Determine the level using the table above.
2. Record it as `plan.yaml`'s `risk_level` (via `eng plan new --risk <level>`).
3. If the level is `architecture` or `high-risk`, the Plan Reviewer step is mandatory even if
   `.agent/project.yaml`'s `workflow.plan_review` is otherwise optional for this project.
4. If the level is `high-risk`, cross-check `.agent/project.yaml`'s `require_approval` list —
   if the request matches a listed category, say so explicitly in `spec.md` and flag every
   task that touches it with `**Requires approval:**`.
