# Tests: V2 Harness Foundation

Run each test after completing the corresponding task. Stop and report on first failure.
T0 is a blocking prerequisite gate — do not proceed to any other task until it passes.

---

## T0 — Go toolchain available (before Task 1)

```bash
go version
```

**Pass:** Output shows `go1.22` or newer (e.g. `go version go1.22.0 windows/amd64`).
**Fail:** `command not found` / `go: command not found` — Go was confirmed absent on this
machine on 2026-08-24. Install Go from https://go.dev/dl/ before continuing, then re-run this
test. Do not attempt any other task until this passes.

---

## T1 — Module scaffold present (after Task 1)

```bash
cat cli/go.mod
test -f cli/main.go && echo "main.go exists"
```

**Pass:** `go.mod` prints `module eng` and a `go 1.22` line; `main.go exists` is printed.
**Fail:** Either file missing or `go.mod` module name is not `eng` — internal package imports
in later tasks (`"eng/internal/..."`) will not resolve.

---

## T2 — Stack detection unit tests (after Task 2)

```bash
cd cli && go test ./internal/detect/...
```

**Pass:** `ok` for `eng/internal/detect`, both `TestDetectGo` and `TestDetectUnknown` pass.
**Fail:** Any `FAIL` line — paste the full test output.

---

## T3 — Project config unit tests (after Task 3)

```bash
cd cli && go test ./internal/project/...
```

**Pass:** `ok` for `eng/internal/project`, all three tests
(`TestDetectModeLegacy`, `TestDetectModeNone`, `TestSaveLoadRoundTrip`) pass.
**Fail:** Any `FAIL` line — paste the full test output.

---

## T4 — Skill resolution unit tests (after Task 4)

```bash
cd cli && go test ./internal/skills/...
```

**Pass:** `ok` for `eng/internal/skills`, all four tests pass
(`TestParseFrontmatter`, `TestParseLegacyHeading`, `TestResolveLocalOverridesGlobal`,
`TestResolveMissingRoots`).
**Fail:** Any `FAIL` line — paste the full test output. If `TestParseLegacyHeading` fails,
the legacy-skill fallback is broken — this is a backward-compatibility regression, treat it
as high priority.

---

## T5 — `harness/` tree complete (after Task 5)

```bash
test -f harness/VERSION && \
test -f harness/core/planner/METHOD.md && \
test -f harness/core/executor/METHOD.md && \
test -f harness/core/principles/karpathy.md && \
test -f harness/skills/engineering/karpathy-guidelines/SKILL.md && \
test -f harness/profiles/software.yaml && \
test -f harness/templates/plan/spec.md && \
test -f harness/templates/.agentignore && \
echo "ALL PRESENT"
grep -q "^name: karpathy-guidelines" harness/skills/engineering/karpathy-guidelines/SKILL.md && echo "FRONTMATTER OK"
```

**Pass:** Both `ALL PRESENT` and `FRONTMATTER OK` print.
**Fail:** Any `test -f` fails (missing file) or grep finds no `name:` frontmatter line —
report which file is missing or malformed.

---

## T6 — Full build succeeds (after Task 10.2)

```bash
cd cli && go mod tidy && go build -o eng .
```

**Pass:** Exits 0. A binary named `eng` (or `eng.exe` on Windows) appears in `cli/`.
**Fail:** Any compile error — paste the full `go build` output, including the offending file
and line.

---

## T7 — `eng install` populates the global harness dir

```bash
cd cli && ./eng install --from ..
ls "$HOME/.engineering-harness/core/planner/METHOD.md"
ls "$HOME/.engineering-harness/skills/engineering/karpathy-guidelines/SKILL.md"
```

**Pass:** `Installed harness to ...` is printed, and both `ls` commands find their file
(no "No such file or directory").
**Fail:** Install command errors, or either file is missing from
`$HOME/.engineering-harness/` after install — report the exact error or missing path.

---

## T8 — `eng init` on a fresh (non-legacy) directory

```bash
mkdir -p /tmp/eng-test-modern && cd /tmp/eng-test-modern
touch go.mod   # give it a detectable stack
/path/to/cli/eng init
cat .agent/project.yaml
```

**Pass:** `Created .agent/project.yaml — mode: modern, stack: go` is printed, and
`.agent/project.yaml` contains `mode: modern` and `stack:` with `type: go`.
**Fail:** Wrong mode reported, missing file, or any other project-root file was created —
`eng init` in a directory with no `CLAUDE.md`/`.plans/` must create exactly one file:
`.agent/project.yaml`.

---

## T9 — `eng init` on a legacy project does not touch existing files (backward compatibility)

```bash
mkdir -p /tmp/eng-test-legacy && cd /tmp/eng-test-legacy
mkdir -p .plans skills
echo "# My Project" > CLAUDE.md
cp -r /path/to/repo/scripts .
md5sum CLAUDE.md scripts/*.sh > /tmp/before.md5
/path/to/cli/eng init
md5sum CLAUDE.md scripts/*.sh > /tmp/after.md5
diff /tmp/before.md5 /tmp/after.md5 && echo "UNCHANGED"
cat .agent/project.yaml
```

**Pass:** `diff` prints nothing and `UNCHANGED` is printed (CLAUDE.md and every script are
byte-for-byte unchanged); `.agent/project.yaml` exists and contains `mode: hybrid`; the
`eng init` output includes the line "Existing CLAUDE.md / .plans/ / skills/ were left
untouched."
**Fail:** `diff` shows any change to an existing file, or mode is not `hybrid` — this is the
single most important test in this plan; a failure here means the tool violates
Requirement 1 (legacy projects must keep working unmodified).

---

## T10 — `eng doctor` reports correctly in all three modes

```bash
# In /tmp/eng-test-legacy (from T9, now hybrid after eng init) and
# in /tmp/eng-test-modern (from T8):
cd /tmp/eng-test-legacy && /path/to/cli/eng doctor
cd /tmp/eng-test-modern && /path/to/cli/eng doctor

# In a brand-new empty directory:
mkdir -p /tmp/eng-test-none && cd /tmp/eng-test-none && /path/to/cli/eng doctor
```

**Pass:** The legacy-turned-hybrid dir reports `Project mode:      hybrid`; the modern dir
reports `Project mode:      modern`; the empty dir reports
`Project mode:      none — not yet initialized`. All three also print
`Harness install:   found at ...` (from T7) and a `Skills resolved:` count ≥ 1
(`engineering/karpathy-guidelines` from the global install).
**Fail:** Any mode misreported, or `Skills resolved: 0` after T7's install succeeded — report
which directory and which line was wrong.

---

## T11 — `eng scan` respects `.agentignore`

```bash
mkdir -p /tmp/eng-test-scan/node_modules /tmp/eng-test-scan/src
echo "console.log(1)" > /tmp/eng-test-scan/node_modules/junk.js
echo "console.log(1)" > /tmp/eng-test-scan/src/app.js
cd /tmp/eng-test-scan && /path/to/cli/eng scan
```

**Pass:** Output's "File counts by extension" shows `.js  1` (only `src/app.js` counted —
`node_modules/` excluded by the default ignore list, since no `.agentignore` file is present
in this test dir).
**Fail:** Count shows `.js  2` (node_modules was walked) — the default ignore fallback in
`loadAgentIgnore` is broken.

---

## T12 — `eng skills list` merges global and local, local wins

```bash
cd /tmp/eng-test-modern
mkdir -p skills/karpathy-guidelines
cat > skills/karpathy-guidelines/SKILL.md <<'EOF'
---
name: karpathy-guidelines
domain: engineering
description: PROJECT-LOCAL OVERRIDE
---
EOF
/path/to/cli/eng skills list
```

**Pass:** Output contains exactly one `karpathy-guidelines` line, showing
`[local  ]` and description `PROJECT-LOCAL OVERRIDE` — not the global description.
**Fail:** Two lines printed for `karpathy-guidelines` (dedup failed), or the global
description is shown instead of the local override.

---

## T13 — V1 behavior is provably untouched (regression gate)

```bash
cd /path/to/repo   # this repo, unmodified root-level files
./scripts/load_skill.sh list
./scripts/plan-executor.sh new smoke-test-v2-regression
./scripts/plan-executor.sh list | grep smoke-test-v2-regression
rm -rf .plans/*-smoke-test-v2-regression
```

**Pass:** `load_skill.sh list` prints the same three skills as before this plan
(`karpathy-guidelines`, `example`, `planner-executor-setup`); `plan-executor.sh new` and
`list` behave exactly as documented in `README.md`.
**Fail:** Any V1 script errors, behaves differently, or a skill is missing — this plan added
files under `cli/` and `harness/` only and must not have altered any V1 script's behavior.
