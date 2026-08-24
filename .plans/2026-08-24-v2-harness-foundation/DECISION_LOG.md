# Decision Log — V2 Harness Foundation

> Read alongside `spec.md`'s "Design decisions" section, which explains each of these in more
> depth. This file exists so future sprints can find "what's decided" without re-reading the
> full spec.

---

## Decisions

### 2026-08-24 — CLI runtime: Go
**Context:** V2 needs one globally-installable `eng` binary that works natively on this
project's actual dev environment (Windows 10, PowerShell) as well as Mac/Linux, without the
WSL requirement V1's bash scripts currently carry.
**Decision:** Implement `eng` in Go, compiled to a single static binary per platform.
**Reasoning:** User's explicit choice (asked via `AskUserQuestion`, given four options with
Go's cross-platform zero-dependency binary as the key differentiator). No runtime to install;
small dependency surface (`gopkg.in/yaml.v3` only).
**Alternatives rejected:**
- Node.js — adds a Node runtime dependency and an npm publish/version story
- Python — pipx/PATH friction on Windows
- Bash + PowerShell dual maintenance — permanently doubles script maintenance, never yields
  one global install artifact
**Decided by:** User
**Status:** Active

### 2026-08-24 — Harness payload lives in this repository, under `harness/`
**Context:** V2 needs a source location `eng install` can copy from. A brand-new repo would
sever the lineage from the V1 template teams have already adopted.
**Decision:** Add `harness/` (installable payload) and `cli/` (Go source) to this same repo,
alongside the untouched V1 root-level files. `eng install --from <path>` points at a checkout
of this repo.
**Reasoning:** Requirement 38 ("don't pollute a project's git diff with harness content")
applies to *consumer* projects, not this harness-source repo, which is expected to contain
what it distributes.
**Alternatives rejected:** New repo now — premature; splittable later via `git filter-repo`
without blocking this plan.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Three-mode compatibility model (`legacy` / `hybrid` / `modern`), opt-in only
**Context:** Requirement 1 mandates legacy projects keep working with zero required action.
**Decision:** Mode is stored in `.agent/project.yaml` only once a project opts in. No
`.agent/` directory = always `legacy` by convention; no marker file, no imposed opinion.
**Reasoning:** The safest default for an untouched project is that `eng` writes nothing to it
until explicitly asked (`eng init`/`eng migrate`).
**Alternatives rejected:** A required marker file even for legacy projects — forces a write to
every existing project just to be recognized.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — Skill schema: additive YAML frontmatter, legacy heading format still valid
**Context:** Requirement 6 asks for a richer skill schema (domain, tags, triggers,
dependencies, conflicts, when_to_use/when_not_to_use) without breaking existing project
skills, which use a plain `# Skill: name` + `## Purpose` heading with no metadata.
**Decision:** New/global `SKILL.md` files carry YAML frontmatter; the resolver falls back to
the legacy heading-extraction rule (same logic `scripts/update-manifest.sh` already uses)
when frontmatter is absent.
**Reasoning:** Every existing project-local skill keeps resolving forever, unmodified —
matches the precedent already set by treating `CLAUDE.md` as permanently user-owned.
**Alternatives rejected:** A hand-maintained global `manifest.yaml` — a second file to keep in
sync, the exact staleness failure mode `docs/src-map.md`'s own design note warns against.
**Decided by:** Planner
**Status:** Active

### 2026-08-24 — No manifest file for skill resolution; live directory walk
**Context:** V1's `skills/manifest.json` requires an explicit `update-manifest.sh` sync step.
**Decision:** `eng skills list`/`doctor`/`init` walk `SKILL.md` files directly at query time
and merge by name (local overrides global) — no pre-generated manifest required.
**Reasoning:** Skill counts are small; a live walk removes a whole class of stale-manifest
bugs for negligible performance cost. V1's own manifest.json + `update-manifest.sh` remain
untouched for anyone still using `load_skill.sh` directly.
**Alternatives rejected:** Porting `update-manifest.sh` to Go and requiring a sync step.
**Decided by:** Planner
**Status:** Active

---

## Superseded decisions

_None yet._
