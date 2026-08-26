# Context Efficiency — Findings

Phase 4's design principle under test: **large knowledge base ≠ large prompt** — a project can
carry many skills/docs while a single request only pulls in what's actually relevant. This
document is grounded in exactly two committed results: Category 4 (`large-context-auth-validation-
harness-v2.yaml`) and Category 5 (`cross-domain-esp32-siemens-modbus-harness-v2.yaml`). No number
here was recomputed or extrapolated beyond what those two files record.

## Category 5 — Skill routing (tight)

Request: "ESP32 reads Siemens S7-1200 over Modbus TCP" (the exact example named in the Phase 8
instruction, and the same request already covered by Phase 6's own committed router eval,
`harness/evals/embedded/esp32-siemens-modbus.yaml`).

- Skills selected: 6 out of the repo's full shipped skill set — `karpathy-guidelines` (always
  explicit), `siemens-s7`, `modbus`, `esp32`, `tcp-ip` (all matched request text), and `plc`
  (pulled in only via `siemens-s7`'s hard `requires:` dependency, not a text match).
- No unrelated-domain skill (`software/cpp`, `devops/docker`, `it/linux`, etc.) was selected.
- This proves both halves of Phase 6's design working together on real input: keyword/trigger
  matching *and* dependency expansion — not just a keyword list.

## Category 4 — Docs/project context routing (loose)

Request: "add input validation to the auth token check so an empty token is rejected," against a
6-package fixture (`api/`, `auth/`, `cache/`, `cli/`, `db/`, `utils/`) with one `docs/src-map.md`
section per package.

- `eng context skills` on this request: only the always-explicit `karpathy-guidelines` matched —
  0 domain skills, 0 false positives (correctly, since no shipped skill concerns generic
  auth/token validation).
- `eng context project` (`internal/docsearch.Match`) on the identical request: **4 of 6** doc
  sections matched (`auth/`, `cli/`, `api/`, `utils/`) — only `auth/` is actually relevant. That
  is a 67% false-positive rate against this fixture's own section count.
- `db/` and `cache/` were correctly excluded in both cases, so the doc router isn't simply
  "match everything" — it is measurably looser than the skill router on identical input, not
  indiscriminate.

## Ratio summary

| | Total available | Selected | Relevant | False positives |
|---|---|---|---|---|
| Skills (Category 5, cross-domain) | full shipped set | 6 | 6 (5 direct matches + 1 required dependency) | 0 |
| Skills (Category 4, single-domain) | full shipped set | 1 (always-explicit only) | 1 | 0 |
| Docs (Category 4) | 6 sections | 4 | 1 | 3 (75% of the selected set) |

## Verdict

Phase 4's "large knowledge base ≠ large prompt" principle **holds well for skill routing**
(`internal/skillmatch.Score`'s weighted tag/trigger/description-word model) across both a
cross-domain and a single-domain request in this benchmark. It **holds measurably less well for
doc/project-context routing** (`internal/docsearch.Match`) on the one case tested here — a real,
reproducible gap, not a hypothetical one. See `BACKLOG.md` P1-2 for the classified backlog entry;
this was not fixed as part of Phase 8 per the plan's bounded-fix rule (core harness behavior is
not changed merely to make a benchmark look better).

This is evidence from **one context-efficiency scenario each**, not a statistical claim across
all possible requests — a genuinely broader sweep (more fixtures, more request phrasings) is a
reasonable Phase 9+ follow-up, not something Phase 8 itself attempts.
