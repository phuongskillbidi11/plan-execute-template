# Spec — V2 Harness Phase 4 (Context Engineering / Context Manager)

> **Planner note:** Read `.plans/2026-08-24-v2-harness-foundation/`,
> `.plans/2026-08-24-v2-harness-phase2/`, and `.plans/2026-08-24-v2-harness-phase3/`
> (`spec.md` + `DECISION_LOG.md` of each) before this file. Phase 4 is narrower than it first
> sounds: most of what a "context manager" needs already exists from Phases 1–3. This plan
> identifies the small number of genuine gaps and closes exactly those.

---

## Goal

A harness with hundreds of skills, long-running plans, and growing project docs must not hand
every AI role the entire knowledge base on every request — "large knowledge base does not mean
large prompt." Phase 4 makes the harness select what's relevant instead of dumping everything,
using nothing more exotic than keyword matching over metadata and markdown conventions that
already exist in this repository.

**Done looks like:** `eng context skills "<request>"` returns a ranked, capped subset of
resolved skills instead of all of them — finally using the `tags`/`triggers` fields Phase 1's
skill schema defined but no code ever read; `eng context project "<request>"` returns only the
`docs/src-map.md`/`docs/gotchas.md` sections that match the request, not the whole file;
`eng context task <plan-dir>` returns the Executor's current unchecked task block plus a
one-paragraph goal summary, not the entire `tasks.md`/`spec.md`; `eng context bundle <role>
<plan-dir>` composes the above per role and writes a `context-manifest.yaml` recording exactly
what was included and how much was omitted; `eng verify`'s test output is written in full to
`.agent/logs/` while the report/stdout carries a bounded summary with a pointer to the full
log. A project with zero context configuration gets sensible defaults; nothing in Phases 1–3
changes behavior.

---

## Background — what's already sufficient vs. what's missing

**Already sufficient, reused as-is:**
- **Skill resolution** (`internal/skills.Resolve`, Phase 1) — Phase 4 filters its *output*,
  never re-implements discovery or the global/local merge.
- **Fresh context per role** — already true by construction: Planner, Reviewer, Executor, and
  Verifier are already separate AI sessions communicating through `spec.md`/`tasks.md`/
  `tests.md`/`plan.yaml`/`review.md`/`verify-report.md`/`events.jsonl` (Phase 2–3). Phase 4
  does not need to build this; it needs to make sure what gets handed to each fresh session is
  curated, which is the actual gap.
- **Persistent, bounded history** — `plan.yaml` (current-state snapshot) plus `events.jsonl`
  (append-only) already implement exactly the "compact state without carrying full
  conversation history" pattern Requirement 9 asks for. No new persistence artifact is needed.
- **Capability Registry** (`internal/capabilities`, Phase 3) — already the foundation
  Requirement 12 asks Phase 4 to prepare for; a Tool Router is correctly deferred until an MCP
  adapter layer exists to route to, per this phase's own scope constraint.

**Missing, and why each is now in scope:**
- **Skill `tags`/`triggers` are schema-only.** Phase 1's Decision 4 added
  `tags`/`triggers`/`when_to_use` to the SKILL.md frontmatter specifically so a future router
  could use them — no code has ever read those fields. `eng skills list` returns every
  resolved skill unconditionally, every time.
- **No bounded project-doc retrieval.** `docs/src-map.md` and `docs/gotchas.md` are meant to be
  read in full by a human Planner today; at "hundreds of skills, many project documents" scale
  that stops being practical, but nothing suggests which sections actually matter for a given
  request.
- **No task-scoped extraction.** The Executor's methodology already says "find the first
  unchecked `[ ]` subtask," but nothing hands it just that block — an Executor session reads
  the whole `tasks.md`.
- **Unbounded tool output.** `runVerify` (Phase 3) embeds the *entire* test command's output
  into `verify-report.md` and prints it in full — fine for this repo's small test suites,
  a real problem at the scale this phase is designed for.
- **No context budget config, and no observability into what was selected/omitted.**

---

## Design decisions

### Decision 1 — New commands only; zero existing command changes behavior (except one, called out explicitly)
**Chosen:** `eng context {skills,project,task,bundle}` are brand-new subcommands. `eng skills
list`, `eng doctor`, `eng scan`, and every Phase 1–3 command keep their exact current output
shape. The one exception is `eng verify`'s printed test-output block, which becomes a bounded
summary with a full-log pointer (Decision 6) — its `Verdict:`/exit-code contract, and the
`verification.verdict` field in `plan.yaml` that `eng workflow advance` actually gates on, are
unchanged.
**Why:** The safest possible backward-compatibility posture, consistent with every prior
phase — a project with zero Phase 4 awareness sees zero behavior change from the commands it
already uses.
**Rejected:** Changing `eng skills list`'s default behavior to be selective — that command's
existing contract ("show me everything resolved") is still useful (it's what `eng doctor` and
this very debugging workflow rely on); selectivity belongs in a new command, not a breaking
change to an old one.

### Decision 2 — Skill selection: keyword score over `tags`/`triggers`/`description`, with `enabled_skills` always guaranteed
**Chosen:** `internal/skillmatch.Select` scores each resolved skill by counting how many of its
`tags`, `triggers`, and description words appear as substrings of the (lowercased) request
text, ranks by score, and returns the top `max_skills`. Any skill named in `.agent/
project.yaml`'s `enabled_skills` is **always** included, even beyond `max_skills` — the cap
only limits additional discovered-but-not-required skills.
**Why:** `enabled_skills` is a Phase 1 concept meaning "this project wants this skill, full
stop" — a new filtering layer must never silently drop it. Keyword scoring over already-typed
metadata (`tags`/`triggers`) is the "simple filesystem/symbol/metadata retrieval" this phase is
explicitly asked to check first, before reaching for anything heavier.
**Rejected:** Embeddings/semantic similarity — explicitly out of scope; keyword matching over
structured metadata that already exists is sufficient at the scale this phase targets (tens to
low hundreds of skills), and adding a vector store here would be exactly the premature
complexity Karpathy's Simplicity First principle warns against.

### Decision 3 — Project-doc retrieval parses existing markdown conventions; no new doc format
**Chosen:** `internal/docsearch.ParseSections` splits `docs/src-map.md` and `docs/gotchas.md`
on `### ` headers — the section-per-module/per-gotcha convention both files already use — and
`Match` keyword-scores each section against the request, capped at `max_docs`.
**Why:** Both files were designed from the start (Foundation-era `docs/src-map.md`/
`gotchas.md`) with "one entry per section, scannable independently" as an explicit goal — this
decision just automates the scanning a human Planner already does by eye. No new file, no new
authoring convention.
**Rejected:** A structured (YAML/JSON) rewrite of `src-map.md`/`gotchas.md` — would break every
existing entry and the human-authoring workflow those files were designed around, for a benefit
(slightly more reliable parsing) the existing `### ` convention already delivers.

### Decision 4 — Task-scoped context reuses `tasks.md`'s existing `## Task N` convention; no new task metadata format
**Chosen:** `internal/taskscope.CurrentTask` splits `tasks.md` on `## Task \d+` headers (the
literal convention used in every plan since the Foundation `_template`) and returns the first
block still containing an unchecked `- [ ]` line — the same signal `scripts/plan-executor.sh`
(V1) and `workflow_cmd.go`'s `tasksComplete` (Phase 3) already trust.
**Why:** Reusing an existing, already-battle-tested parsing signal (unchecked-checkbox
detection) is lower-risk than inventing a new per-task ID/metadata scheme that every future
plan would need to adopt.
**Rejected:** Requiring tasks.md to declare structured `depends_on`/task IDs now — that's
Requirement 18's task dependency graph from the original harness brief, explicitly deferred
past this phase; current-task extraction doesn't need it.

### Decision 5 — Context budget lives in a new optional `.agent/context.yaml`, mirroring `.agent/hooks.yaml`'s pattern
**Chosen:** `internal/contextcfg.Config{Strategy, MaxSkills, MaxDocs, MaxLogLines,
IncludeCompletedTasks, SummarizeToolOutput}`, defaulted by `Default()` (`strategy: selective,
max_skills: 5, max_docs: 8, max_log_lines: 300, summarize_tool_output: true`), loadable from
a project's `.agent/context.yaml` (full override) or `harness/context/default.yaml`, falling
back to `Default()` if neither exists.
**Why:** Matches the exact schema shape requested and the precedent Phase 2's `hooks.yaml`
already set (optional project file, full-replace, sensible global default). `strategy: full`
is the explicit escape hatch back to "select everything" for anyone who wants it.
**Rejected:** Folding these fields into `.agent/project.yaml` — context strategy is a
cross-cutting orchestration concern, not project identity; keeping it a separate optional file
matches how `hooks.yaml` was already split out for the same reason.

### Decision 6 — Tool output compaction: full log to `.agent/logs/`, bounded head+tail summary inline
**Chosen:** `runVerify` writes the complete test command output to
`.agent/logs/verify-<timestamp>.log` and, when `summarize_tool_output` is true (the default),
embeds only the first and last `max_log_lines/2` lines in `verify-report.md`/stdout, with an
explicit "Full output: `<path>`" pointer. The `Verdict:` line and `plan.yaml`'s
`verification.verdict` field — the only two things `eng workflow advance` actually reads — are
completely unaffected.
**Why:** Line-count bounding (not token counting) is model-agnostic, per Requirement 7's
explicit instruction not to hard-code token counts to one model. Head+tail (not just head) is
deliberate — build/test failures are conventionally reported at the *end* of output.
**Rejected:** Always truncating to head-only — would silently hide the actual failure lines
most test runners print last.

### Decision 7 — `eng context bundle` composes the other three commands per role; it does not replace `eng adapter prompt`
**Chosen:** `eng context bundle <role> <plan-dir>` calls the same selection logic as `eng
context skills/project/task`, assembles a role-appropriate subset (Planner: project context +
skills; Plan Reviewer: plan facts + project context; Executor: task scope + skills; Verifier:
plan's `write_scope`/verification rules), and writes `context-manifest.yaml` recording what was
selected. `eng adapter prompt` (Phase 3) is untouched — it still prints the role's
`METHOD.md` plus plan file paths.
**Why:** Two separate, composable commands are safer to land than merging them immediately:
`eng adapter prompt`'s contract (used by `eng workflow advance`'s printed next-action commands)
stays stable while `eng context bundle` proves itself. Wiring them together is a natural next
step once the bundle format has been used for a while — not a decision to make blind.
**Rejected:** Making `eng adapter prompt` internally call `eng context bundle` now — would
change an already-shipped Phase 3 command's output shape for every existing caller, including
the exact strings `workflow_cmd.go` prints as "next action" instructions.

---

## Responsibilities (extending Phase 2/3's table)

| Role | Phase 4 addition |
|---|---|
| **Context Manager (new)** | Not an AI role — a mechanical selection step (`eng context bundle`) any role's session can run first. Read-only; writes only `context-manifest.yaml`. Documented fail-safe: if a role's request text is empty/too generic to score anything, it must say so and request more specific input rather than guessing a selection (see `harness/core/context-manager/METHOD.md`). |
| **Planner** | Runs `eng context bundle planner <plan-dir> "<request>"` before `eng plan new`, in addition to Phase 3's triage step |
| **Plan Reviewer** | Runs `eng context bundle plan-reviewer <plan-dir>` for plan facts + matching project context |
| **Executor** | Runs `eng context bundle executor <plan-dir>` for the current task block instead of reading all of `tasks.md` |
| **Verifier** | Reads `.agent/logs/verify-*.log` only when the bounded summary in `verify-report.md` isn't enough to diagnose a FAIL |

---

## Scope

### In scope (Phase 4 MVP)
- `cli/internal/contextcfg/` — `.agent/context.yaml` schema, `Default()`, pointer-based
  override parsing (distinguishing "unset" from "explicitly false" for the two bool fields)
- `cli/internal/skillmatch/` — `Score`/`Select` over `internal/skills.Skill`
- `cli/internal/docsearch/` — `### `-header section parsing + keyword `Match`
- `cli/internal/taskscope/` — `CurrentTask`/`GoalSummary` extraction from `tasks.md`/`spec.md`
- `eng context skills "<text>"` / `eng context project "<text>"` / `eng context task <dir>` /
  `eng context bundle <role> <dir> ["<text>"]`
- `harness/context/default.yaml` — the global default budget
- `harness/core/context-manager/METHOD.md` — fail-safe methodology
- Updates to `harness/core/{planner,plan-reviewer,executor,verifier}/METHOD.md` referencing
  the new context-bundle step
- `runVerify` (`cli/verify_cmd.go`) log compaction per Decision 6
- `eng init` also ensures `.agent/logs/` is listed in the project's `.gitignore` (creating one
  with just that line if none exists — never overwriting an existing `.gitignore`'s other
  content)

### Out of scope (explicitly excluded, with reasons)
- **Semantic/vector search** — Decision 2; keyword-over-metadata is sufficient at this phase's
  target scale, per the phase's own "first inspect whether simple retrieval is sufficient"
  instruction.
- **Symbol/AST-level source search (tree-sitter/ctags/LSP)** — same category as the original
  harness brief's Requirement 29, already deferred there; nothing in this phase's actual gaps
  (skills, docs, tasks) needs source-symbol indexing.
- **A public/structured ADR format or `docs/adr/` tooling** — this repository has no ADRs yet;
  `eng context project` would include `docs/adr/*.md` sections the same way it includes
  `src-map.md`/`gotchas.md` *if that directory existed*, but creating the ADR convention itself
  is unrelated scope.
- **Wiring `eng context bundle` into `eng adapter prompt`** — Decision 7; kept separate until
  proven.
- **Full MCP Tool Router** — Requirement 12 explicitly says not to build this yet; the
  Capability Registry it would sit on already exists from Phase 3.
- **Token-count-based budgets** — Requirement 7 explicitly rules this out; budgets are
  structural (counts of skills/docs/lines), not token estimates for a specific model.
- **A persistent "working notes" artifact** — Requirement 10 explicitly warns against
  duplicating `plan.yaml`/`events.jsonl`/plan documents; no gap was found during this review
  that those don't already cover, so nothing new is added here.

### Later improvements (explicitly deferred, not designed here)
- Wiring `eng context bundle`'s output into `eng adapter prompt` once the bundle format has
  been used in practice
- Applying the same log-compaction pattern to `eng hooks run`'s captured output
- A richer `context-manifest.yaml` schema (per-category scores, not just names) if the current
  one proves insufficient for debugging bad selections
- Symbol-level source search, once/if a real need for it surfaces (not assumed here)
- A Tool Router once an MCP adapter layer exists to route to

---

## Affected files

| File | Change type | Reason |
|---|---|---|
| `cli/internal/contextcfg/contextcfg.go` | Create | `.agent/context.yaml` schema + defaults |
| `cli/internal/contextcfg/contextcfg_test.go` | Create | Default/override/unset-vs-false tests |
| `cli/internal/skillmatch/skillmatch.go` | Create | Scoring + required-skill-guaranteed selection |
| `cli/internal/skillmatch/skillmatch_test.go` | Create | Scoring and cap tests |
| `cli/internal/docsearch/docsearch.go` | Create | Section parsing + keyword match |
| `cli/internal/docsearch/docsearch_test.go` | Create | Parsing and matching tests |
| `cli/internal/taskscope/taskscope.go` | Create | Current-task and goal-summary extraction |
| `cli/internal/taskscope/taskscope_test.go` | Create | Extraction tests |
| `cli/context_cmd.go` | Create | `eng context skills/project/task/bundle` |
| `cli/main.go` | Modify | Dispatch `eng context ...` |
| `cli/verify_cmd.go` | Modify | Log compaction per Decision 6 |
| `cli/init_cmd.go` | Modify | Ensure `.agent/logs/` is gitignored |
| `harness/context/default.yaml` | Create | Global default context budget |
| `harness/core/context-manager/METHOD.md` | Create | Fail-safe context-selection methodology |
| `harness/core/planner/METHOD.md` | Modify | Reference `eng context bundle planner` |
| `harness/core/plan-reviewer/METHOD.md` | Modify | Reference `eng context bundle plan-reviewer` |
| `harness/core/executor/METHOD.md` | Modify | Reference `eng context bundle executor` |
| `harness/core/verifier/METHOD.md` | Modify | Reference `.agent/logs/` for full test output |
| `harness/VERSION` | Modify | Bump `0.3.0-phase3` → `0.4.0-phase4-context` |
| `docs/src-map.md` | Modify | Add Phase 4 module entries (last task) |
| `README.md` | Modify | Additive Phase 4 section |
| `ROADMAP.md` | Modify | Note Phase 4 plan link |

---

## Risks and unknowns

- **Keyword scoring is crude by design.** A request using none of a highly-relevant skill's
  exact tag/trigger words will score it 0 and it won't be selected unless it's in
  `enabled_skills`. Acceptable for MVP (the fallback is always "run `eng skills list` for the
  full, unfiltered set," which never goes away) — noted as the honest limitation of "simple
  retrieval," not hidden.
- **`### ` header parsing assumes both `src-map.md` and `gotchas.md` keep using that exact
  convention.** True for every entry in this repository today (verified by reading both files
  during planning); a future entry that doesn't follow the convention simply won't be
  discovered as its own section — the fix is the same discipline already required to add an
  entry correctly, not new parsing complexity.
- **`.agent/logs/` growing unbounded over time.** This phase writes logs but does not add
  rotation/cleanup — acceptable for MVP since `.gitignore`-ing the directory (Task in scope)
  keeps it out of the repository at least; log rotation is a reasonable future addition, not
  blocking here.

## Open questions

- [ ] Should `eng context bundle`'s manifest be consumed by anything automated, or is it purely
  for human debugging? **This plan treats it as human/debugging-only for now** — nothing in
  `eng workflow advance` reads `context-manifest.yaml`; it exists purely to answer "why did the
  bundle include/exclude X" after the fact, per Requirement 13. Automated consumption is a
  later-improvement question, not one this plan needs to answer.

---

## Self-evaluation (plan-quality.md rubric)

| Principle | Criterion | Score | Notes |
|---|---|---|---|
| Think Before Planning | spec.md written first; user-observable goal stated | 9/10 | Goal is CLI-observable (`eng context skills` output), not "context is managed" |
| Simplicity First | ≥3 out-of-scope items with reasoning | 10/10 | 7 items excluded, each reasoned, plus 5 later improvements |
| Surgical Changes | Every file listed with change type and exact reason | 9/10 | New-package creation is coarser than single-symbol edits, same unavoidable caveat as every prior phase |
| Goal-Driven Execution | Each scope item traces to an acceptance criterion | 9/10 | Every new command and the log-compaction change gets a test in `tests.md` |

**Total: 37/40 → 9/10**

---

**User confirmation received:** [ ] Yes
**Confirmed on:** _pending_
