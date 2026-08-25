# Phase 7 Tests — Tool/MCP Adapter Runtime & Permission Enforcement

## Per-task unit/build gates

Run after each corresponding task in `tasks.md`.

- **T1** `cd cli && go test ./internal/toolcap/... -v` — 2 tests pass.
- **T2** `cd cli && go test ./internal/tooladapter/... ./internal/toolrouter/... -v` — all
  of Phase 5's original 3 `toolrouter` tests pass unmodified, plus 4 new `tooladapter`
  tests.
- **T3** `cd cli && go test ./internal/agent/... -v` — the 3 pre-existing `RoleMayUse` tests
  pass unmodified, plus 4 new risk-ceiling tests.
- **T4** `cd cli && go test ./internal/toolpolicy/... -v` — 10 tests pass, covering every
  precedence step in spec.md Decision 7.
- **T5** `cd cli && go test ./internal/project/... -v` — every pre-Phase-7 test passes
  unmodified, plus 2 new `Tools` policy tests.
- **T6** `cd cli && go test ./internal/toolrouter/... -v` — 6 new `Route` tests pass,
  including deterministic alphabetical provider precedence with two synthetic adapters.
- **T7** `cd cli && go test ./internal/capabilities/... -v` — unmodified, `gh` now in
  `Known`.
- **T8** `cd cli && go test ./internal/tooladapter/... -v` — `GitHubAdapter` tests pass;
  the live-`gh` test skips cleanly if `gh` is absent or unauthenticated.
- **T9** `cd cli && go test ./internal/mcpregistry/... -v` — 3 tests pass, including
  loading the real `harness/mcp/servers.yaml`.
- **T10** `cd cli && go test ./internal/tooladapter/... -v` — `ReferenceMCPAdapter` tests
  pass, including a real search hit and a real no-match case.
- **T11** `cd cli && go build ./... && echo BUILD_OK`.
- **T12** `cd cli && go build ./... && echo BUILD_OK`.
- **T13** `cd cli && go build ./... && echo BUILD_OK`.
- **T14** `cd cli && go build ./... && go vet ./... && echo OK`.
- **T15** `cd cli && go vet ./... && go build -o eng . && ./eng` — usage lists `tools
  invoke` and the updated `capabilities` lines.
- **T16** `cd cli && go build -o eng . && ./eng skills validate` — `0 error(s)`.
- **T17/T18** manual read.
- **T19** `cat harness/VERSION` — `0.7.0-phase7-tools`.

---

## End-to-end walkthroughs

Run from a fresh test project, `eng install --from ..` already applied so the global
harness has Phase 7's `harness/mcp/servers.yaml` and the (unchanged by this phase) skill
tree.

### E2E-READONLY — read-only tool routing succeeds without approval

```bash
REPO="$(git rev-parse --show-toplevel)"
rm -rf /tmp/eng-test-p7-tools && mkdir -p /tmp/eng-test-p7-tools && cd /tmp/eng-test-p7-tools
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
"$REPO/cli/eng" init
"$REPO/cli/eng" plan new demo --risk feature
PLAN=$(ls -d .plans/*-demo)

"$REPO/cli/eng" tools invoke executor git.status "$PLAN"; echo "exit=$?"
```

**Pass:** exit `0`; real `git status` output printed; `"$PLAN/events.jsonl"` gained one
`tool_invocation` line with `"result":"ALLOWED"`, `"adapter":"git"`,
`"capability":"git.status"` — no plan approval was needed.
**Fail:** non-zero exit, or no audit event recorded.

### E2E-PERMISSION-DENIAL — role toolbox/risk ceiling refuses before invocation

```bash
"$REPO/cli/eng" tools invoke planner git.push "$PLAN"; echo "exit=$?"
grep '"result":"DENIED"' "$PLAN/events.jsonl"
```

**Pass:** non-zero exit; `REFUSED (DENIED): role may not invoke WRITE-risk capabilities`
(or the toolbox-check message, whichever fires first) printed; `git push` is **never
actually run** (no network call, no repository mutation); an audit event with
`"result":"DENIED"` is still recorded.
**Fail:** the command exits `0`, or `git push` is actually invoked, or no audit event is
written for a refusal.

### E2E-APPROVAL-REQUIRED — a write capability blocks, then unblocks after approval

```bash
"$REPO/cli/eng" tools invoke executor git.push "$PLAN"; echo "before-approval-exit=$?"
grep '"result":"NEEDS_APPROVAL"' "$PLAN/events.jsonl"

"$REPO/cli/eng" plan approve "$PLAN" --by reviewer
"$REPO/cli/eng" tools invoke executor git.push "$PLAN"; echo "after-approval-exit=$?"
```

**Pass:** the first call exits non-zero with `NEEDS_APPROVAL` and its own audit event
(no push happened, no remote exists to push to anyway in this local-only test repo — the
command should still fail cleanly at the policy gate, not at git's own error); the plan
remains unapproved. After `eng plan approve`, the second call is `ALLOWED` by
`toolpolicy.Decide` (git itself may still fail since this test repo has no remote — that's
an adapter-level error, not a policy refusal, and is a legitimate, different exit path;
what matters is the audit event's `"result"` field reads `"ALLOWED"` before the underlying
`git push` attempt, not `"NEEDS_APPROVAL"`).
**Fail:** the second call still reports `NEEDS_APPROVAL` after approval.

### E2E-AUDIT — audit events stay compact regardless of output size

```bash
"$REPO/cli/eng" tools invoke executor git.log "$PLAN" --all
tail -n 1 "$PLAN/events.jsonl"
```

**Pass:** the audit line is small (adapter/capability/role/result/reason/log_path fields
only) even though `git log` output could be large — the full output went to
`.agent/logs/tool-git-*.log` via the reused `writeFullLog`, and the event references it by
path rather than embedding it.
**Fail:** the raw command output appears inline in `events.jsonl`.

### E2E-ADAPTER-HEALTH — `eng doctor`'s `Tools:` section is bounded and secret-free

```bash
"$REPO/cli/eng" doctor
```

**Pass:** a `Tools:` section appears with one line per adapter (`git`, `github`,
`mcp-docs`), each showing `available`/`unavailable` and a capability count — no token, no
credential, no raw `gh auth status` output printed anywhere in the doctor report.
**Fail:** more than a handful of lines per adapter, or any credential-looking string in the
output.

### E2E-LEGACY — a project with no `tools:` block behaves exactly as before this phase

```bash
rm -rf /tmp/eng-test-p7-legacy && mkdir -p /tmp/eng-test-p7-legacy && cd /tmp/eng-test-p7-legacy
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init
mkdir -p .agent
cat > .agent/project.yaml <<'EOF'
project_name: eng-test-p7-legacy
mode: modern
stack:
    type: go
    build_cmd: go build ./...
    test_cmd: echo ok
EOF
# No `tools:` block at all.

"$REPO/cli/eng" plan new legacy-check --risk feature
PLAN=$(ls -d .plans/*-legacy-check)
"$REPO/cli/eng" tools invoke executor git.status "$PLAN"; echo "read-exit=$?"
"$REPO/cli/eng" tools invoke executor git.push "$PLAN"; echo "write-exit=$?"
```

**Pass:** `read-exit=0` (READ defaults open with no policy configured — identical to every
project before this phase, which had no enforcement at all); `write-exit` is non-zero with
`NEEDS_APPROVAL` (the new safe default for an unlisted WRITE, not a crash or a parse
error over the missing `tools:` block).
**Fail:** either command errors on the missing `tools:` block itself, rather than applying
the documented default.

---

## Regression gates — Phase 1 through 6 and V1

```bash
cd "$REPO/cli" && go build ./... && go vet ./... && go test ./... 2>&1
```

**Pass:** builds clean, `go vet` clean, every package's tests pass — including every
Phase 1–6 package untouched by this phase (`internal/workflow`, `internal/skillrouter`,
`internal/skillgraph`, `internal/skillvalidate`, `internal/contextcfg`, `internal/logprune`,
`internal/planmeta`, `internal/hooks`, ...) alongside the new Phase 7 packages.

```bash
rm -rf /tmp/eng-test-p7-regress && mkdir -p /tmp/eng-test-p7-regress && cd /tmp/eng-test-p7-regress
git init -q && git config user.email t@example.com && git config user.name t
touch go.mod && git add go.mod && git commit -q -m init

"$REPO/cli/eng" init
"$REPO/cli/eng" doctor
"$REPO/cli/eng" scan
"$REPO/cli/eng" skills list
"$REPO/cli/eng" skills validate
"$REPO/cli/eng" capabilities list
"$REPO/cli/eng" capabilities list --verbose --role executor
"$REPO/cli/eng" context skills "add a feature"
"$REPO/cli/eng" context project "add a feature"
"$REPO/cli/eng" plan new regress --risk feature
PLAN=$(ls -d .plans/*-regress)
"$REPO/cli/eng" plan drift "$PLAN"
"$REPO/cli/eng" plan retry "$PLAN" unit_test
"$REPO/cli/eng" triage "fix a bug"
"$REPO/cli/eng" hooks run before_execute "$PLAN"
"$REPO/cli/eng" logs prune --dry-run
"$REPO/cli/eng" capabilities explain executor "$PLAN" "add a feature"
```

**Pass:** every command runs and behaves as documented in its own phase's plan — Phase 7
only ever adds subcommands/fields (`capabilities explain`, `tools invoke`, `tools:` policy
block, `eng hooks run`'s already-Phase-6 optional plan-dir argument), never changes an
existing one's meaning.

```bash
cd "$REPO" && \
./scripts/load_skill.sh list && \
./scripts/plan-executor.sh new smoke-test-phase7-regression && \
./scripts/plan-executor.sh list | grep smoke-test-phase7-regression && \
rm -rf .plans/*-smoke-test-phase7-regression && \
echo "V1 REGRESSION CHECK OK"
```

**Pass:** identical output to every prior phase's run of this same check.

Re-run Phase 5's Quick Fix/Spec-First walkthroughs and Phase 6's router eval integration
test explicitly, to confirm nothing in this phase's `context_cmd.go` edit (the `## Tools`
addition) regressed the `## Skills` section or the context-manifest shape those depend on:

```bash
cd "$REPO/cli" && go test ./... -run TestRouterEvalScenarios -v
```

**Pass:** all three Phase 6 router eval scenarios still pass unchanged.

---

## Cleanup

```bash
rm -rf /tmp/eng-test-p7-tools /tmp/eng-test-p7-legacy /tmp/eng-test-p7-regress
```
