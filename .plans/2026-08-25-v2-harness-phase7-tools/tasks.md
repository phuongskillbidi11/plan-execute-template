# Phase 7 Tasks — Tool/MCP Adapter Runtime & Permission Enforcement

Execute in order. Run each task's own build/test check before moving to the next. Every
"Old" snippet is verified against the actual current file content before writing this
document — if a real mismatch is found during execution, stop and treat it as a
contradiction, not something to paper over.

---

## Task 1 — `internal/toolcap`: capability naming and risk model

- [x] **1.1** Create `cli/internal/toolcap/toolcap.go`:

  ```go
  package toolcap

  // Risk classifies how much trust invoking a capability requires — the
  // safety model Phase 7 establishes before any DESTRUCTIVE/HIGH_RISK
  // adapter (PLC write, Modbus write, ...) exists to use it (Requirement 4).
  type Risk string

  const (
  	RiskRead        Risk = "READ"
  	RiskWrite       Risk = "WRITE"
  	RiskDestructive Risk = "DESTRUCTIVE"
  	RiskHighRisk    Risk = "HIGH_RISK"
  )

  // RiskRank gives each tier a total order (READ < WRITE < DESTRUCTIVE <
  // HIGH_RISK) so role ceilings and policy checks stay one integer
  // comparison instead of a growing switch statement. An unknown Risk
  // value ranks above every known tier — fail toward "more restrictive,"
  // never less.
  func RiskRank(r Risk) int {
  	switch r {
  	case RiskRead:
  		return 0
  	case RiskWrite:
  		return 1
  	case RiskDestructive:
  		return 2
  	case RiskHighRisk:
  		return 3
  	default:
  		return 4
  	}
  }

  // Capability is one named, risk-classified operation an Adapter exposes
  // — e.g. {"git.status", RiskRead} or {"git.force_push", RiskDestructive}.
  // Naming convention: "<adapter>.<operation>" (Phase 7 spec.md Decision 5).
  type Capability struct {
  	Name string
  	Risk Risk
  }
  ```

- [x] **1.2** Create `cli/internal/toolcap/toolcap_test.go`:

  ```go
  package toolcap

  import "testing"

  func TestRiskRankOrdering(t *testing.T) {
  	if !(RiskRank(RiskRead) < RiskRank(RiskWrite)) {
  		t.Fatal("expected READ < WRITE")
  	}
  	if !(RiskRank(RiskWrite) < RiskRank(RiskDestructive)) {
  		t.Fatal("expected WRITE < DESTRUCTIVE")
  	}
  	if !(RiskRank(RiskDestructive) < RiskRank(RiskHighRisk)) {
  		t.Fatal("expected DESTRUCTIVE < HIGH_RISK")
  	}
  }

  func TestRiskRankUnknownRanksAboveHighRisk(t *testing.T) {
  	if RiskRank(Risk("bogus")) <= RiskRank(RiskHighRisk) {
  		t.Fatal("expected an unknown risk to rank above HIGH_RISK (fail restrictive)")
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/toolcap/... -v`.

---

## Task 2 — Revise `tooladapter.Adapter`, upgrade `GitAdapter`, fix `toolrouter.Filter`

- [x] **2.1** Replace the full contents of `cli/internal/tooladapter/tooladapter.go`:

  ```go
  package tooladapter

  import (
  	"fmt"
  	"os/exec"
  	"strings"

  	"eng/internal/toolcap"
  )

  // Adapter exposes one external tool/capability to the harness. Distinct
  // from internal/agent.Adapter (which launches/talks to a coding agent) —
  // see Phase 5 spec.md Decision 10 for why these stay separate. This
  // interface is a Phase 7 revision of the Phase 5 foundation (Phase 7
  // spec.md Decision 1) — Phase 5's own DECISION_LOG called it
  // "foundation only," not a frozen contract.
  type Adapter interface {
  	Name() string
  	Provider() string // "local-binary" | "github-cli" | "mcp" | ...
  	Version() string   // best-effort; "" if unknown or unavailable
  	Capabilities() []toolcap.Capability
  	Available() bool
  	Doctor() (string, error)
  	// Invoke runs capability with args in dir, returning its output. dir
  	// is explicit (not the process's own cwd) so invocation is testable
  	// without changing directories globally.
  	Invoke(capability string, args []string, dir string) (string, error)
  }

  // GitAdapter is a local-binary reference implementation — git is already
  // unconditionally required throughout this harness, so Available() is
  // simply whether it's on PATH.
  type GitAdapter struct {
  	available bool
  }

  func NewGitAdapter(available bool) GitAdapter { return GitAdapter{available: available} }

  func (g GitAdapter) Name() string     { return "git" }
  func (g GitAdapter) Provider() string { return "local-binary" }

  func (g GitAdapter) Version() string {
  	out, err := exec.Command("git", "--version").Output()
  	if err != nil {
  		return ""
  	}
  	return strings.TrimSpace(string(out))
  }

  func (g GitAdapter) Available() bool { return g.available }

  func (g GitAdapter) Capabilities() []toolcap.Capability {
  	return []toolcap.Capability{
  		{Name: "git.status", Risk: toolcap.RiskRead},
  		{Name: "git.diff", Risk: toolcap.RiskRead},
  		{Name: "git.log", Risk: toolcap.RiskRead},
  		{Name: "git.commit", Risk: toolcap.RiskWrite},
  		{Name: "git.push", Risk: toolcap.RiskWrite},
  		{Name: "git.force_push", Risk: toolcap.RiskDestructive},
  	}
  }

  func (g GitAdapter) Doctor() (string, error) {
  	if g.available {
  		return "git is on PATH", nil
  	}
  	return "", fmt.Errorf("git not found on PATH")
  }

  func (g GitAdapter) Invoke(capability string, args []string, dir string) (string, error) {
  	if !g.available {
  		return "", fmt.Errorf("git not found on PATH")
  	}
  	var gitArgs []string
  	switch capability {
  	case "git.status":
  		gitArgs = append([]string{"status"}, args...)
  	case "git.diff":
  		gitArgs = append([]string{"diff"}, args...)
  	case "git.log":
  		gitArgs = append([]string{"log"}, args...)
  	case "git.commit":
  		gitArgs = append([]string{"commit"}, args...)
  	case "git.push":
  		gitArgs = append([]string{"push"}, args...)
  	case "git.force_push":
  		gitArgs = append([]string{"push", "--force"}, args...)
  	default:
  		return "", fmt.Errorf("git adapter does not support capability %q", capability)
  	}
  	cmd := exec.Command("git", gitArgs...)
  	cmd.Dir = dir
  	out, err := cmd.CombinedOutput()
  	return string(out), err
  }
  ```

- [x] **2.2** Replace the full contents of `cli/internal/tooladapter/tooladapter_test.go`:

  ```go
  package tooladapter

  import (
  	"os"
  	"os/exec"
  	"path/filepath"
  	"testing"

  	"eng/internal/toolcap"
  )

  func TestGitAdapterImplementsAdapter(t *testing.T) {
  	var _ Adapter = GitAdapter{}
  }

  func TestGitAdapterCapabilitiesIncludeExpectedRisks(t *testing.T) {
  	g := NewGitAdapter(true)
  	byName := map[string]toolcap.Risk{}
  	for _, c := range g.Capabilities() {
  		byName[c.Name] = c.Risk
  	}
  	if byName["git.status"] != toolcap.RiskRead {
  		t.Fatalf("expected git.status to be READ, got %v", byName["git.status"])
  	}
  	if byName["git.push"] != toolcap.RiskWrite {
  		t.Fatalf("expected git.push to be WRITE, got %v", byName["git.push"])
  	}
  	if byName["git.force_push"] != toolcap.RiskDestructive {
  		t.Fatalf("expected git.force_push to be DESTRUCTIVE, got %v", byName["git.force_push"])
  	}
  }

  func TestGitAdapterUnavailableRefuses(t *testing.T) {
  	g := NewGitAdapter(false)
  	if _, err := g.Doctor(); err == nil {
  		t.Fatal("expected an error when unavailable")
  	}
  	if _, err := g.Invoke("git.status", nil, "."); err == nil {
  		t.Fatal("expected Invoke to refuse when unavailable")
  	}
  }

  func TestGitAdapterInvokeUnsupportedCapabilityErrors(t *testing.T) {
  	g := NewGitAdapter(true)
  	if _, err := g.Invoke("git.nonexistent", nil, "."); err == nil {
  		t.Fatal("expected an error for an unsupported capability")
  	}
  }

  func TestGitAdapterInvokeStatusInARealRepo(t *testing.T) {
  	if _, err := exec.LookPath("git"); err != nil {
  		t.Skip("git not found on PATH in this environment")
  	}
  	dir := t.TempDir()
  	run := func(args ...string) {
  		cmd := exec.Command("git", args...)
  		cmd.Dir = dir
  		if out, err := cmd.CombinedOutput(); err != nil {
  			t.Fatalf("git %v failed: %v\n%s", args, err, out)
  		}
  	}
  	run("init", "-q")
  	run("config", "user.email", "t@example.com")
  	run("config", "user.name", "t")
  	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)

  	g := NewGitAdapter(true)
  	out, err := g.Invoke("git.status", nil, dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if out == "" {
  		t.Fatal("expected non-empty git status output")
  	}
  }
  ```

- [x] **2.3** In `cli/internal/toolrouter/toolrouter.go`, `Filter` matched against the now-
  removed singular `a.Capability()` — update it to match by adapter `Name()` instead (the
  Phase 5 GitAdapter's one capability string and its Name() were always identical, so this
  is behavior-preserving for every existing test — Phase 7 spec.md Decision, Task 2 note):

  Old:
  ```go
  package toolrouter

  import "eng/internal/tooladapter"

  // Filter returns the subset of adapters matching a required capability
  // list and currently available. This is the entire Tool Router for
  // Phase 5 — it exposes nothing to any agent session because no session
  // object exists in this architecture to expose into yet (Requirement 16:
  // foundation only).
  func Filter(required []string, adapters []tooladapter.Adapter) []tooladapter.Adapter {
  	want := map[string]bool{}
  	for _, r := range required {
  		want[r] = true
  	}
  	var out []tooladapter.Adapter
  	for _, a := range adapters {
  		if want[a.Capability()] && a.Available() {
  			out = append(out, a)
  		}
  	}
  	return out
  }
  ```

  New:
  ```go
  package toolrouter

  import "eng/internal/tooladapter"

  // Filter returns the subset of adapters whose Name() is in required and
  // currently available. This is the Phase 5 filtering foundation, kept
  // unchanged in shape — Phase 7's Adapter interface replaced the single
  // Capability() string with plural, risk-tagged Capabilities(), so this
  // now matches by adapter identity instead (Route, added in Task 6 below,
  // is the capability-aware, policy-aware successor).
  func Filter(required []string, adapters []tooladapter.Adapter) []tooladapter.Adapter {
  	want := map[string]bool{}
  	for _, r := range required {
  		want[r] = true
  	}
  	var out []tooladapter.Adapter
  	for _, a := range adapters {
  		if want[a.Name()] && a.Available() {
  			out = append(out, a)
  		}
  	}
  	return out
  }
  ```

  `cli/internal/toolrouter/toolrouter_test.go` needs no change — all three existing tests
  (`TestFilterMatchesRequiredAndAvailable`, `TestFilterExcludesUnavailable`,
  `TestFilterExcludesUnrequested`) already use `"git"`/`"docker"` as both the adapter's name
  and its old single capability, so matching by `Name()` produces byte-identical results.

**Verify:** `cd cli && go build ./... && go test ./internal/tooladapter/... ./internal/toolrouter/... -v`
— all of Phase 5's original `toolrouter` tests must still pass unmodified.

---

## Task 3 — Role permission: risk ceiling alongside the existing toolbox check

- [x] **3.1** Replace the full contents of `cli/internal/agent/permissions.go` (a second
  real defect was found and fixed during final regression verification, after this task was
  first marked done: `internal/capabilities.Known` must use the literal binary name `"gh"`
  for `exec.LookPath`, but `RolePermissions` only listed the adapter's conceptual name
  `"github"` — so `eng capabilities list --role <role> --verbose` silently never showed
  `gh` for any role even though both the binary and the GitHub adapter were available.
  Fixed by listing both `"github"` and `"gh"` in every role's toolbox, with a regression
  test, `TestRoleMayUseGhBinaryNameInEveryRolesToolbox`, added to
  `cli/internal/agent/permissions_test.go`):

  ```go
  package agent

  import "eng/internal/toolcap"

  // RolePermissions is a static, reporting-only table — nothing yet
  // enforces this against a real tool invocation (see Phase 5 spec.md
  // Decision 11); Phase 7's toolpolicy.Decide is the first real consumer.
  var RolePermissions = map[Role][]string{
  	RolePlanner:  {"git", "github"},
  	RoleReviewer: {"git", "github"},
  	RoleExecutor: {"git", "github", "claude", "codex", "docker"},
  	RoleVerifier: {"git", "github", "docker"},
  }

  // RoleMayUse reports whether role is permitted to consider capability.
  func RoleMayUse(role, capability string) bool {
  	for _, c := range RolePermissions[Role(role)] {
  		if c == capability {
  			return true
  		}
  	}
  	return false
  }

  // RoleMaxRisk is the highest capability risk tier a role may invoke
  // without an explicit approval escalation (Phase 7 Requirement 5) — an
  // axis independent from RolePermissions above: an adapter being in a
  // role's toolbox does not by itself grant every risk tier that adapter
  // exposes (Phase 7 spec.md Decision 4).
  var RoleMaxRisk = map[Role]toolcap.Risk{
  	RolePlanner:  toolcap.RiskRead,
  	RoleReviewer: toolcap.RiskRead,
  	RoleExecutor: toolcap.RiskWrite,
  	RoleVerifier: toolcap.RiskRead,
  }

  // RoleMayInvokeRisk reports whether role's risk ceiling covers risk. An
  // unknown role has no ceiling — nothing is permitted.
  func RoleMayInvokeRisk(role string, risk toolcap.Risk) bool {
  	max, ok := RoleMaxRisk[Role(role)]
  	if !ok {
  		return false
  	}
  	return toolcap.RiskRank(risk) <= toolcap.RiskRank(max)
  }
  ```

- [x] **3.2** Append to `cli/internal/agent/permissions_test.go` (add
  `"eng/internal/toolcap"` to its imports):

  ```go
  func TestRoleMayUseGitHubInEveryRolesToolbox(t *testing.T) {
  	for _, role := range []string{"planner", "plan-reviewer", "executor", "verifier"} {
  		if !RoleMayUse(role, "github") {
  			t.Fatalf("expected %s to have github in its toolbox", role)
  		}
  	}
  }

  func TestRoleMaxRiskPlannerReadOnly(t *testing.T) {
  	if !RoleMayInvokeRisk("planner", toolcap.RiskRead) {
  		t.Fatal("expected planner to invoke READ")
  	}
  	if RoleMayInvokeRisk("planner", toolcap.RiskWrite) {
  		t.Fatal("expected planner NOT to invoke WRITE")
  	}
  }

  func TestRoleMaxRiskExecutorReadWrite(t *testing.T) {
  	if !RoleMayInvokeRisk("executor", toolcap.RiskWrite) {
  		t.Fatal("expected executor to invoke WRITE")
  	}
  	if RoleMayInvokeRisk("executor", toolcap.RiskDestructive) {
  		t.Fatal("expected executor NOT to invoke DESTRUCTIVE")
  	}
  }

  func TestRoleMaxRiskUnknownRoleDeniedEverything(t *testing.T) {
  	if RoleMayInvokeRisk("not-a-real-role", toolcap.RiskRead) {
  		t.Fatal("expected an unknown role to be denied even READ")
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/agent/... -v` — the three
pre-existing `RoleMayUse` tests must still pass unmodified.

---

## Task 4 — `internal/toolpolicy`: project policy + hard deny + `Decide`

- [x] **4.1** Create `cli/internal/toolpolicy/toolpolicy.go`:

  ```go
  package toolpolicy

  import (
  	"eng/internal/agent"
  	"eng/internal/toolcap"
  )

  // Policy is the project-level tool policy — .agent/project.yaml's
  // `tools:` block (Phase 7 spec.md Decision 2: deliberately new/nested,
  // not a reuse of the pre-existing, undocumented, unread
  // Config.RequireApproval field).
  type Policy struct {
  	Allow           []string `yaml:"allow,omitempty"`
  	RequireApproval []string `yaml:"require_approval,omitempty"`
  	Deny            []string `yaml:"deny,omitempty"`
  }

  // HardDeny is the built-in, project-config-immune deny list (Requirement
  // 21) — no override mechanism exists in Phase 7, on purpose.
  var HardDeny = map[string]bool{
  	"git.force_push": true,
  }

  type Verdict string

  const (
  	Allowed       Verdict = "ALLOWED"
  	NeedsApproval Verdict = "NEEDS_APPROVAL"
  	Denied        Verdict = "DENIED"
  )

  type Decision struct {
  	Verdict Verdict
  	Reason  string
  }

  func contains(list []string, s string) bool {
  	for _, v := range list {
  		if v == s {
  			return true
  		}
  	}
  	return false
  }

  // Decide applies the fixed precedence in Phase 7 spec.md Decision 7:
  // hard deny -> project deny -> role toolbox -> role risk ceiling ->
  // project require_approval -> project allow -> default (READ open,
  // WRITE+ needs approval). adapterName is the owning adapter's Name()
  // (e.g. "git"), used for the coarse role-toolbox check; approved is
  // whether this plan's execution approval (plan.yaml's approved_at) has
  // already been granted.
  func Decide(capability string, risk toolcap.Risk, adapterName, role string, policy Policy, approved bool) Decision {
  	if HardDeny[capability] {
  		return Decision{Denied, "hard deny — never invocable regardless of policy"}
  	}
  	if contains(policy.Deny, capability) {
  		return Decision{Denied, "denied by project tools.deny"}
  	}
  	if !agent.RoleMayUse(role, adapterName) {
  		return Decision{Denied, "role's toolbox does not include adapter " + adapterName}
  	}
  	if !agent.RoleMayInvokeRisk(role, risk) {
  		return Decision{Denied, "role may not invoke " + string(risk) + "-risk capabilities"}
  	}
  	if contains(policy.RequireApproval, capability) {
  		if approved {
  			return Decision{Allowed, "requires approval — plan is approved"}
  		}
  		return Decision{NeedsApproval, "listed in project tools.require_approval — plan not yet approved"}
  	}
  	if contains(policy.Allow, capability) {
  		return Decision{Allowed, "allowed by project tools.allow"}
  	}
  	if risk == toolcap.RiskRead {
  		return Decision{Allowed, "read capability, no explicit policy — default-open for READ"}
  	}
  	if approved {
  		return Decision{Allowed, string(risk) + "-risk capability, plan is approved"}
  	}
  	return Decision{NeedsApproval, string(risk) + "-risk capability not explicitly listed — requires plan approval before invocation"}
  }
  ```

- [x] **4.2** Create `cli/internal/toolpolicy/toolpolicy_test.go` (defect found during
  implementation: the drafted `TestDecideDestructiveAlwaysNeedsApprovalEvenIfNotHardDenied`
  test asserted `NEEDS_APPROVAL`, but the actual — correct — behavior is `DENIED`, since no
  role's `RoleMaxRisk` ceiling reaches DESTRUCTIVE, so the role-ceiling check denies it
  before the default-needs-approval branch is ever reached; this is the intended safety
  property from Requirement 4, not a bug in `Decide`. Renamed to
  `TestDecideDestructiveDeniedByRoleCeilingWithNoElevatedRole` and fixed to assert `Denied`):

  ```go
  package toolpolicy

  import (
  	"testing"

  	"eng/internal/toolcap"
  )

  func TestDecideHardDenyWinsOverProjectAllow(t *testing.T) {
  	p := Policy{Allow: []string{"git.force_push"}}
  	d := Decide("git.force_push", toolcap.RiskDestructive, "git", "executor", p, true)
  	if d.Verdict != Denied {
  		t.Fatalf("expected hard deny to win even when explicitly allowed, got %+v", d)
  	}
  }

  func TestDecideProjectDeny(t *testing.T) {
  	p := Policy{Deny: []string{"git.push"}}
  	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", p, true)
  	if d.Verdict != Denied {
  		t.Fatalf("expected DENIED, got %+v", d)
  	}
  }

  func TestDecideRoleToolboxDenied(t *testing.T) {
  	d := Decide("docker.inspect", toolcap.RiskRead, "docker", "planner", Policy{}, false)
  	if d.Verdict != Denied {
  		t.Fatalf("expected planner's toolbox to exclude docker, got %+v", d)
  	}
  }

  func TestDecideRoleRiskCeilingDenied(t *testing.T) {
  	d := Decide("git.push", toolcap.RiskWrite, "git", "planner", Policy{}, true)
  	if d.Verdict != Denied {
  		t.Fatalf("expected planner's READ ceiling to deny WRITE, got %+v", d)
  	}
  }

  func TestDecideRequireApprovalNotYetApproved(t *testing.T) {
  	p := Policy{RequireApproval: []string{"github.issue.comment"}}
  	d := Decide("github.issue.comment", toolcap.RiskWrite, "github", "executor", p, false)
  	if d.Verdict != NeedsApproval {
  		t.Fatalf("expected NEEDS_APPROVAL, got %+v", d)
  	}
  }

  func TestDecideRequireApprovalApproved(t *testing.T) {
  	p := Policy{RequireApproval: []string{"github.issue.comment"}}
  	d := Decide("github.issue.comment", toolcap.RiskWrite, "github", "executor", p, true)
  	if d.Verdict != Allowed {
  		t.Fatalf("expected ALLOWED once approved, got %+v", d)
  	}
  }

  func TestDecideProjectAllowList(t *testing.T) {
  	p := Policy{Allow: []string{"git.push"}}
  	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", p, false)
  	if d.Verdict != Allowed {
  		t.Fatalf("expected ALLOWED via tools.allow even without plan approval, got %+v", d)
  	}
  }

  func TestDecideDefaultReadOpen(t *testing.T) {
  	d := Decide("git.status", toolcap.RiskRead, "git", "executor", Policy{}, false)
  	if d.Verdict != Allowed {
  		t.Fatalf("expected READ to default-allow with no policy, got %+v", d)
  	}
  }

  func TestDecideDefaultWriteNeedsApproval(t *testing.T) {
  	d := Decide("git.push", toolcap.RiskWrite, "git", "executor", Policy{}, false)
  	if d.Verdict != NeedsApproval {
  		t.Fatalf("expected unlisted WRITE to require approval by default, got %+v", d)
  	}
  }

  func TestDecideDestructiveAlwaysNeedsApprovalEvenIfNotHardDenied(t *testing.T) {
  	d := Decide("some.destructive_op", toolcap.RiskDestructive, "git", "executor", Policy{}, false)
  	if d.Verdict != NeedsApproval {
  		t.Fatalf("expected DESTRUCTIVE with no policy entry to require approval, got %+v", d)
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/toolpolicy/... -v`.

---

## Task 5 — Project config: `Tools` policy field

- [x] **5.1** In `cli/internal/project/project.go`, add the import and extend `Config`:

  Old:
  ```go
  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"

  	"eng/internal/executil"
  )
  ```

  New:
  ```go
  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"gopkg.in/yaml.v3"

  	"eng/internal/executil"
  	"eng/internal/toolpolicy"
  )
  ```

  Old:
  ```go
  	// PrivateSkillsPath, if set, is resolved relative to the project root
  	// (or used as-is if absolute) as an extra skill root between global and
  	// local precedence. Empty (the default for every existing
  	// project.yaml) skips the private tier entirely — see Phase 6 spec.md
  	// Decision 8.
  	PrivateSkillsPath string `yaml:"private_skills_path,omitempty"`
  }
  ```

  New:
  ```go
  	// PrivateSkillsPath, if set, is resolved relative to the project root
  	// (or used as-is if absolute) as an extra skill root between global and
  	// local precedence. Empty (the default for every existing
  	// project.yaml) skips the private tier entirely — see Phase 6 spec.md
  	// Decision 8.
  	PrivateSkillsPath string `yaml:"private_skills_path,omitempty"`

  	// Tools is the Phase 7 project-level tool policy (allow/require_approval/
  	// deny, by capability name). Deliberately not a reuse of the
  	// pre-existing, unread RequireApproval field above — see Phase 7
  	// spec.md Decision 2.
  	Tools toolpolicy.Policy `yaml:"tools,omitempty"`
  }
  ```

- [x] **5.2** Append to `cli/internal/project/project_test.go` (add
  `"eng/internal/toolpolicy"` to its imports):

  ```go
  func TestToolsPolicyDefaultsToEmpty(t *testing.T) {
  	dir := t.TempDir()
  	cfg := &Config{ProjectName: "x", Mode: "modern"}
  	if err := Save(dir, cfg); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(got.Tools.Allow) != 0 || len(got.Tools.Deny) != 0 || len(got.Tools.RequireApproval) != 0 {
  		t.Fatalf("expected empty tool policy by default, got %+v", got.Tools)
  	}
  }

  func TestToolsPolicyRoundTrip(t *testing.T) {
  	dir := t.TempDir()
  	cfg := &Config{ProjectName: "x", Mode: "modern", Tools: toolpolicy.Policy{
  		Allow:           []string{"git.status"},
  		RequireApproval: []string{"github.issue.comment"},
  		Deny:            []string{"git.force_push"},
  	}}
  	if err := Save(dir, cfg); err != nil {
  		t.Fatal(err)
  	}
  	got, err := Load(dir)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(got.Tools.Allow) != 1 || got.Tools.Allow[0] != "git.status" ||
  		len(got.Tools.RequireApproval) != 1 || len(got.Tools.Deny) != 1 {
  		t.Fatalf("round-trip mismatch: %+v", got.Tools)
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/project/... -v` — every
pre-Phase-7 `project` test must still pass unmodified.

---

## Task 6 — `internal/toolrouter`: `Route`, the authoritative selection path

- [x] **6.1** Append to `cli/internal/toolrouter/toolrouter.go` (add
  `"sort"`, `"eng/internal/toolcap"`, and `"eng/internal/toolpolicy"` to its imports):

  ```go
  type Selection struct {
  	Adapter    string
  	Capability string
  	Reason     string
  }

  type Blocked struct {
  	Adapter    string
  	Capability string
  	Reason     string
  }

  type Result struct {
  	Allowed       []Selection
  	NeedsApproval []Blocked
  	Blocked       []Blocked
  }

  // Route is the Phase 7 authoritative tool-selection path — deterministic
  // provider precedence (alphabetical among available candidates; see
  // Phase 7 spec.md Decision 5 for why no config field overrides this
  // yet), then one toolpolicy.Decide call per required capability,
  // bucketed into Allowed/NeedsApproval/Blocked with an explanation each.
  func Route(required []string, adapters []tooladapter.Adapter, role string, policy toolpolicy.Policy, approved bool) Result {
  	var result Result
  	for _, capName := range required {
  		var candidates []tooladapter.Adapter
  		for _, a := range adapters {
  			if !a.Available() {
  				continue
  			}
  			for _, c := range a.Capabilities() {
  				if c.Name == capName {
  					candidates = append(candidates, a)
  					break
  				}
  			}
  		}
  		if len(candidates) == 0 {
  			result.Blocked = append(result.Blocked, Blocked{Capability: capName, Reason: "no available adapter provides this capability"})
  			continue
  		}
  		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Name() < candidates[j].Name() })
  		owner := candidates[0]

  		var risk toolcap.Risk
  		for _, c := range owner.Capabilities() {
  			if c.Name == capName {
  				risk = c.Risk
  			}
  		}

  		decision := toolpolicy.Decide(capName, risk, owner.Name(), role, policy, approved)
  		switch decision.Verdict {
  		case toolpolicy.Allowed:
  			result.Allowed = append(result.Allowed, Selection{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
  		case toolpolicy.NeedsApproval:
  			result.NeedsApproval = append(result.NeedsApproval, Blocked{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
  		default:
  			result.Blocked = append(result.Blocked, Blocked{Adapter: owner.Name(), Capability: capName, Reason: decision.Reason})
  		}
  	}
  	return result
  }
  ```

- [x] **6.2** Create `cli/internal/toolrouter/route_test.go` (defect found during
  implementation: the drafted `TestRouteDeterministicProviderPrecedenceAlphabetical` used
  synthetic adapter names `"a-provider"`/`"b-provider"` not present in
  `agent.RolePermissions`' executor toolbox, so `Route`'s role-toolbox check correctly
  denied both before precedence logic ran at all — this was a defect in the test's fixture,
  not in `Route`. Fixed to use `"docker"`/`"git"`, both real toolbox entries):

  ```go
  package toolrouter

  import (
  	"testing"

  	"eng/internal/toolcap"
  	"eng/internal/toolpolicy"
  )

  type fakeAdapter struct {
  	name      string
  	available bool
  	caps      []toolcap.Capability
  }

  func (f fakeAdapter) Name() string                    { return f.name }
  func (f fakeAdapter) Provider() string                { return "fake" }
  func (f fakeAdapter) Version() string                 { return "" }
  func (f fakeAdapter) Available() bool                 { return f.available }
  func (f fakeAdapter) Capabilities() []toolcap.Capability { return f.caps }
  func (f fakeAdapter) Doctor() (string, error)         { return "", nil }
  func (f fakeAdapter) Invoke(string, []string, string) (string, error) { return "", nil }

  func TestRouteAllowsReadByDefault(t *testing.T) {
  	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
  	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
  	if len(result.Allowed) != 1 {
  		t.Fatalf("expected 1 allowed, got %+v", result)
  	}
  }

  func TestRouteBlocksMissingAdapter(t *testing.T) {
  	result := Route([]string{"git.status"}, nil, "executor", toolpolicy.Policy{}, false)
  	if len(result.Blocked) != 1 {
  		t.Fatalf("expected 1 blocked (no adapter), got %+v", result)
  	}
  }

  func TestRouteBlocksUnavailableAdapter(t *testing.T) {
  	a := fakeAdapter{name: "git", available: false, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
  	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
  	if len(result.Blocked) != 1 {
  		t.Fatalf("expected 1 blocked (unavailable), got %+v", result)
  	}
  }

  func TestRouteNeedsApprovalForUnlistedWrite(t *testing.T) {
  	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.push", Risk: toolcap.RiskWrite}}}
  	result := Route([]string{"git.push"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
  	if len(result.NeedsApproval) != 1 {
  		t.Fatalf("expected 1 needs-approval, got %+v", result)
  	}
  }

  func TestRouteDeterministicProviderPrecedenceAlphabetical(t *testing.T) {
  	b := fakeAdapter{name: "b-provider", available: true, caps: []toolcap.Capability{{Name: "shared.read", Risk: toolcap.RiskRead}}}
  	a := fakeAdapter{name: "a-provider", available: true, caps: []toolcap.Capability{{Name: "shared.read", Risk: toolcap.RiskRead}}}
  	result := Route([]string{"shared.read"}, []tooladapter.Adapter{b, a}, "executor", toolpolicy.Policy{}, false)
  	if len(result.Allowed) != 1 || result.Allowed[0].Adapter != "a-provider" {
  		t.Fatalf("expected a-provider to win alphabetical precedence, got %+v", result)
  	}
  }

  func TestRouteExplanationReasonsNonEmpty(t *testing.T) {
  	a := fakeAdapter{name: "git", available: true, caps: []toolcap.Capability{{Name: "git.status", Risk: toolcap.RiskRead}}}
  	result := Route([]string{"git.status"}, []tooladapter.Adapter{a}, "executor", toolpolicy.Policy{}, false)
  	if len(result.Allowed) != 1 || result.Allowed[0].Reason == "" {
  		t.Fatalf("expected a non-empty reason, got %+v", result)
  	}
  }
  ```

  (`fakeAdapter` needs `"eng/internal/tooladapter"` imported too, for the `[]tooladapter.Adapter`
  slice literals — add it alongside `toolcap`/`toolpolicy`.)

**Verify:** `cd cli && go build ./... && go test ./internal/toolrouter/... -v` — all of
Task 2's Filter tests plus these six new Route tests must pass.

---

## Task 7 — Add `gh` to the binary-detection capability list

- [x] **7.1** In `cli/internal/capabilities/capabilities.go`:

  Old:
  ```go
  var Known = []string{"git", "claude", "codex", "docker"}
  ```

  New:
  ```go
  var Known = []string{"git", "claude", "codex", "docker", "gh"}
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/capabilities/... -v` — existing
tests iterate `Known` generically and require no other change.

---

## Task 8 — `GitHubAdapter`: the external reference adapter

- [x] **8.1** Create `cli/internal/tooladapter/github.go`:

  ```go
  package tooladapter

  import (
  	"fmt"
  	"os/exec"
  	"strings"

  	"eng/internal/toolcap"
  )

  // GitHubAdapter is the Phase 7 external reference adapter — read-only,
  // delegating entirely to the `gh` CLI's own authentication (a token
  // stored in the OS keyring by `gh auth login`; the harness never reads,
  // stores, or displays it — see Phase 7 spec.md Decision 10).
  type GitHubAdapter struct {
  	available bool
  }

  func NewGitHubAdapter(available bool) GitHubAdapter { return GitHubAdapter{available: available} }

  func (g GitHubAdapter) Name() string     { return "github" }
  func (g GitHubAdapter) Provider() string { return "github-cli" }

  func (g GitHubAdapter) Version() string {
  	out, err := exec.Command("gh", "--version").Output()
  	if err != nil {
  		return ""
  	}
  	lines := strings.SplitN(string(out), "\n", 2)
  	return strings.TrimSpace(lines[0])
  }

  func (g GitHubAdapter) Available() bool { return g.available }

  func (g GitHubAdapter) Capabilities() []toolcap.Capability {
  	return []toolcap.Capability{
  		{Name: "github.repo.read", Risk: toolcap.RiskRead},
  		{Name: "github.pr.read", Risk: toolcap.RiskRead},
  		{Name: "github.issue.read", Risk: toolcap.RiskRead},
  	}
  }

  func (g GitHubAdapter) Doctor() (string, error) {
  	if !g.available {
  		return "", fmt.Errorf("gh not found on PATH")
  	}
  	if _, err := exec.Command("gh", "auth", "status").CombinedOutput(); err != nil {
  		return "", fmt.Errorf("gh is installed but not authenticated: %w", err)
  	}
  	return "gh is on PATH and authenticated", nil
  }

  func (g GitHubAdapter) Invoke(capability string, args []string, dir string) (string, error) {
  	if !g.available {
  		return "", fmt.Errorf("gh not found on PATH")
  	}
  	var ghArgs []string
  	switch capability {
  	case "github.repo.read":
  		ghArgs = append([]string{"repo", "view"}, args...)
  	case "github.pr.read":
  		ghArgs = append([]string{"pr", "list"}, args...)
  	case "github.issue.read":
  		ghArgs = append([]string{"issue", "list"}, args...)
  	default:
  		return "", fmt.Errorf("github adapter does not support capability %q", capability)
  	}
  	cmd := exec.Command("gh", ghArgs...)
  	cmd.Dir = dir
  	out, err := cmd.CombinedOutput()
  	return string(out), err
  }
  ```

- [x] **8.2** Create `cli/internal/tooladapter/github_test.go`:

  ```go
  package tooladapter

  import (
  	"os/exec"
  	"testing"

  	"eng/internal/toolcap"
  )

  func TestGitHubAdapterImplementsAdapter(t *testing.T) {
  	var _ Adapter = GitHubAdapter{}
  }

  func TestGitHubAdapterCapabilitiesAllRead(t *testing.T) {
  	g := NewGitHubAdapter(true)
  	for _, c := range g.Capabilities() {
  		if c.Risk != toolcap.RiskRead {
  			t.Fatalf("expected every GitHubAdapter capability to be READ, got %+v", c)
  		}
  	}
  }

  func TestGitHubAdapterUnavailableRefuses(t *testing.T) {
  	g := NewGitHubAdapter(false)
  	if _, err := g.Doctor(); err == nil {
  		t.Fatal("expected an error when unavailable")
  	}
  	if _, err := g.Invoke("github.repo.read", nil, "."); err == nil {
  		t.Fatal("expected Invoke to refuse when unavailable")
  	}
  }

  func TestGitHubAdapterInvokeUnsupportedCapabilityErrors(t *testing.T) {
  	g := NewGitHubAdapter(true)
  	if _, err := g.Invoke("github.nonexistent", nil, "."); err == nil {
  		t.Fatal("expected an error for an unsupported capability")
  	}
  }

  func TestGitHubAdapterLiveDoctorIfInstalled(t *testing.T) {
  	if _, err := exec.LookPath("gh"); err != nil {
  		t.Skip("gh not found on PATH in this environment")
  	}
  	g := NewGitHubAdapter(true)
  	msg, err := g.Doctor()
  	if err != nil {
  		t.Skip("gh installed but not authenticated in this environment:", err)
  	}
  	if msg == "" {
  		t.Fatal("expected a non-empty status message")
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/tooladapter/... -v`.

---

## Task 9 — `internal/mcpregistry`: static MCP discovery foundation

- [x] **9.1** Create `cli/internal/mcpregistry/mcpregistry.go`:

  ```go
  package mcpregistry

  import (
  	"os"

  	"gopkg.in/yaml.v3"
  )

  // Server is one declared MCP-style server entry — deliberately
  // credential-free by construction: no field in this struct can hold a
  // secret (Phase 7 spec.md Decision 9/10). Real transport/auth wiring is
  // out of scope for Phase 7; this is a static, local discovery/inspection
  // registry only.
  type Server struct {
  	Name                string   `yaml:"name"`
  	Transport           string   `yaml:"transport"` // "mock" today; "stdio"/"http" for a future real server
  	Capabilities        []string `yaml:"capabilities"`
  	PermissionCategory  string   `yaml:"permission_category"`
  	Description         string   `yaml:"description,omitempty"`
  }

  type registryFile struct {
  	Servers []Server `yaml:"servers"`
  }

  // Load reads path (typically harness/mcp/servers.yaml). A missing file
  // is not an error — it returns an empty slice, so a harness install
  // that predates this registry still works.
  func Load(path string) ([]Server, error) {
  	data, err := os.ReadFile(path)
  	if err != nil {
  		if os.IsNotExist(err) {
  			return nil, nil
  		}
  		return nil, err
  	}
  	var f registryFile
  	if err := yaml.Unmarshal(data, &f); err != nil {
  		return nil, err
  	}
  	return f.Servers, nil
  }
  ```

- [x] **9.2** Create `harness/mcp/servers.yaml`:

  ```yaml
  servers:
    - name: docs-search
      transport: mock
      capabilities: [docs.search]
      permission_category: read
      description: >
        Deterministic local reference adapter — searches this repo's docs/*.md files.
        Proves the MCP-style adapter lifecycle (discovery, health, capability declaration,
        routing, permission, invocation, audit) without a live MCP server or network
        dependency. See docs/tools.md.
  ```

- [x] **9.3** Create `cli/internal/mcpregistry/mcpregistry_test.go`:

  ```go
  package mcpregistry

  import (
  	"os"
  	"path/filepath"
  	"testing"
  )

  func TestLoadParsesServers(t *testing.T) {
  	dir := t.TempDir()
  	path := filepath.Join(dir, "servers.yaml")
  	os.WriteFile(path, []byte("servers:\n  - name: docs-search\n    transport: mock\n    capabilities: [docs.search]\n    permission_category: read\n"), 0o644)
  	servers, err := Load(path)
  	if err != nil {
  		t.Fatal(err)
  	}
  	if len(servers) != 1 || servers[0].Name != "docs-search" || servers[0].Transport != "mock" {
  		t.Fatalf("got %+v", servers)
  	}
  }

  func TestLoadMissingFileIsNotError(t *testing.T) {
  	servers, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
  	if err != nil || len(servers) != 0 {
  		t.Fatalf("expected empty, no error; got %+v, %v", servers, err)
  	}
  }

  func TestLoadRealHarnessRegistry(t *testing.T) {
  	servers, err := Load(filepath.Join("..", "..", "..", "harness", "mcp", "servers.yaml"))
  	if err != nil {
  		t.Fatal(err)
  	}
  	found := false
  	for _, s := range servers {
  		if s.Name == "docs-search" {
  			found = true
  		}
  	}
  	if !found {
  		t.Fatal("expected the real harness/mcp/servers.yaml to declare docs-search")
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/mcpregistry/... -v`.

---

## Task 10 — `ReferenceMCPAdapter`: deterministic mock MCP-style adapter

- [x] **10.1** Create `cli/internal/tooladapter/reference_mcp.go`:

  ```go
  package tooladapter

  import (
  	"fmt"
  	"io/fs"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/toolcap"
  )

  // ReferenceMCPAdapter is the Phase 7 deterministic mock/reference
  // MCP-style adapter (Phase 7 spec.md Decision 9) — no network, no
  // external process, always available when docsRoot exists. It proves
  // the adapter lifecycle for an MCP-style integration without
  // implementing real MCP transport. Its one capability, docs.search,
  // greps docsRoot's *.md files for a query and returns bounded, matching
  // file paths.
  type ReferenceMCPAdapter struct {
  	docsRoot string
  }

  func NewReferenceMCPAdapter(docsRoot string) ReferenceMCPAdapter {
  	return ReferenceMCPAdapter{docsRoot: docsRoot}
  }

  func (a ReferenceMCPAdapter) Name() string     { return "mcp-docs" }
  func (a ReferenceMCPAdapter) Provider() string { return "mcp" }
  func (a ReferenceMCPAdapter) Version() string  { return "1.0.0" }

  func (a ReferenceMCPAdapter) Available() bool {
  	info, err := os.Stat(a.docsRoot)
  	return err == nil && info.IsDir()
  }

  func (a ReferenceMCPAdapter) Capabilities() []toolcap.Capability {
  	return []toolcap.Capability{{Name: "docs.search", Risk: toolcap.RiskRead}}
  }

  func (a ReferenceMCPAdapter) Doctor() (string, error) {
  	if !a.Available() {
  		return "", fmt.Errorf("docs root %s not found", a.docsRoot)
  	}
  	return "mock MCP server — docs root found at " + a.docsRoot, nil
  }

  func (a ReferenceMCPAdapter) Invoke(capability string, args []string, dir string) (string, error) {
  	if capability != "docs.search" {
  		return "", fmt.Errorf("mcp-docs adapter does not support capability %q", capability)
  	}
  	if len(args) == 0 {
  		return "", fmt.Errorf("docs.search requires a query argument")
  	}
  	if !a.Available() {
  		return "", fmt.Errorf("docs root %s not found", a.docsRoot)
  	}
  	query := strings.ToLower(strings.Join(args, " "))

  	var matches []string
  	filepath.WalkDir(a.docsRoot, func(path string, d fs.DirEntry, err error) error {
  		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
  			return nil
  		}
  		data, err := os.ReadFile(path)
  		if err != nil {
  			return nil
  		}
  		if strings.Contains(strings.ToLower(string(data)), query) {
  			matches = append(matches, path)
  		}
  		return nil
  	})

  	if len(matches) == 0 {
  		return fmt.Sprintf("no matches for %q under %s", query, a.docsRoot), nil
  	}
  	const maxMatches = 10
  	if len(matches) > maxMatches {
  		matches = matches[:maxMatches]
  	}
  	return "matches for " + query + ":\n- " + strings.Join(matches, "\n- "), nil
  }
  ```

- [x] **10.2** Create `cli/internal/tooladapter/reference_mcp_test.go`:

  ```go
  package tooladapter

  import (
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"

  	"eng/internal/toolcap"
  )

  func TestReferenceMCPAdapterImplementsAdapter(t *testing.T) {
  	var _ Adapter = ReferenceMCPAdapter{}
  }

  func TestReferenceMCPAdapterAvailableWhenDocsRootExists(t *testing.T) {
  	a := NewReferenceMCPAdapter(t.TempDir())
  	if !a.Available() {
  		t.Fatal("expected available when docs root exists")
  	}
  }

  func TestReferenceMCPAdapterUnavailableWhenDocsRootMissing(t *testing.T) {
  	a := NewReferenceMCPAdapter(filepath.Join(t.TempDir(), "nope"))
  	if a.Available() {
  		t.Fatal("expected unavailable when docs root is missing")
  	}
  	if _, err := a.Doctor(); err == nil {
  		t.Fatal("expected Doctor to error when unavailable")
  	}
  }

  func TestReferenceMCPAdapterCapabilityIsRead(t *testing.T) {
  	a := NewReferenceMCPAdapter(t.TempDir())
  	caps := a.Capabilities()
  	if len(caps) != 1 || caps[0].Name != "docs.search" || caps[0].Risk != toolcap.RiskRead {
  		t.Fatalf("got %+v", caps)
  	}
  }

  func TestReferenceMCPAdapterSearchFindsMatch(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, "example.md"), []byte("# Modbus addressing\nPDU address is not the same as 40001.\n"), 0o644)
  	a := NewReferenceMCPAdapter(dir)
  	out, err := a.Invoke("docs.search", []string{"modbus"}, ".")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(out, "example.md") {
  		t.Fatalf("expected match to reference example.md, got %q", out)
  	}
  }

  func TestReferenceMCPAdapterSearchNoMatch(t *testing.T) {
  	dir := t.TempDir()
  	os.WriteFile(filepath.Join(dir, "example.md"), []byte("# Something else\n"), 0o644)
  	a := NewReferenceMCPAdapter(dir)
  	out, err := a.Invoke("docs.search", []string{"nonexistent-term-xyz"}, ".")
  	if err != nil {
  		t.Fatal(err)
  	}
  	if !strings.Contains(out, "no matches") {
  		t.Fatalf("expected a no-matches message, got %q", out)
  	}
  }

  func TestReferenceMCPAdapterInvokeRequiresQuery(t *testing.T) {
  	a := NewReferenceMCPAdapter(t.TempDir())
  	if _, err := a.Invoke("docs.search", nil, "."); err == nil {
  		t.Fatal("expected an error when no query is given")
  	}
  }
  ```

**Verify:** `cd cli && go build ./... && go test ./internal/tooladapter/... -v`.

---

## Task 11 — `eng tools invoke`: the invocation boundary and audit trail

- [x] **11.1** Create `cli/tools_cmd.go`:

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"

  	"eng/internal/capabilities"
  	"eng/internal/mcpregistry"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/tooladapter"
  	"eng/internal/toolcap"
  	"eng/internal/toolpolicy"
  )

  func cmdTools(args []string) {
  	if len(args) < 1 || args[0] != "invoke" {
  		fmt.Println("Usage: eng tools invoke <role> <capability> <plan-dir> [args...]")
  		os.Exit(1)
  	}
  	toolsInvoke(args[1:])
  }

  // registeredAdapters returns every adapter the harness knows about,
  // evaluated fresh each call so Available()/Doctor() reflect live state.
  // Shared by `eng tools invoke`, `eng capabilities explain`, `eng
  // doctor`, and buildContextBundle's ## Tools section — the one place
  // adapters are listed (Phase 7 Requirement 1: this is harness wiring,
  // not a responsibility any adapter or command owns itself).
  func registeredAdapters(repoRoot string) []tooladapter.Adapter {
  	adapters := []tooladapter.Adapter{
  		tooladapter.NewGitAdapter(capabilities.Detect("git")),
  		tooladapter.NewGitHubAdapter(capabilities.Detect("gh")),
  	}

  	servers, _ := mcpregistry.Load(filepath.Join(harnessDir(), "mcp", "servers.yaml"))
  	for _, s := range servers {
  		if s.Name == "docs-search" && s.Transport == "mock" {
  			adapters = append(adapters, tooladapter.NewReferenceMCPAdapter(filepath.Join(repoRoot, "docs")))
  		}
  	}
  	return adapters
  }

  func toolsInvoke(args []string) {
  	if len(args) < 3 {
  		fmt.Println("Usage: eng tools invoke <role> <capability> <plan-dir> [args...]")
  		os.Exit(1)
  	}
  	role := args[0]
  	capability := args[1]
  	planDir, err := filepath.Abs(args[2])
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	invokeArgs := args[3:]

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}
  	approved := meta.ApprovedAt != ""

  	var policy toolpolicy.Policy
  	if cfg, err := project.Load(repoRoot); err == nil {
  		policy = cfg.Tools
  	}

  	adapters := registeredAdapters(repoRoot)
  	var owner tooladapter.Adapter
  	var risk toolcap.Risk
  	found := false
  	for _, a := range adapters {
  		for _, c := range a.Capabilities() {
  			if c.Name == capability {
  				owner = a
  				risk = c.Risk
  				found = true
  			}
  		}
  	}
  	if !found {
  		fmt.Printf("no adapter declares capability %q\n", capability)
  		os.Exit(1)
  	}
  	if !owner.Available() {
  		fmt.Printf("adapter %q is not available\n", owner.Name())
  		os.Exit(1)
  	}

  	decision := toolpolicy.Decide(capability, risk, owner.Name(), role, policy, approved)
  	audit := map[string]interface{}{
  		"adapter":    owner.Name(),
  		"capability": capability,
  		"role":       role,
  		"result":     string(decision.Verdict),
  		"reason":     decision.Reason,
  	}

  	if decision.Verdict != toolpolicy.Allowed {
  		planmeta.AppendStructuredEvent(planDir, "tool_invocation", audit)
  		fmt.Printf("REFUSED (%s): %s\n", decision.Verdict, decision.Reason)
  		os.Exit(1)
  	}

  	out, invokeErr := owner.Invoke(capability, invokeArgs, repoRoot)

  	ctxCfg := loadContextConfig(repoRoot)
  	logPath, logErr := writeFullLog(repoRoot, "tool-"+owner.Name(), out)
  	display := out
  	if ctxCfg.SummarizeToolOutput {
  		display = summarizeOutput(out, ctxCfg.MaxLogLines)
  	}
  	if logErr == nil {
  		audit["log_path"] = logPath
  	}
  	if invokeErr != nil {
  		audit["result"] = "ERROR"
  		audit["error"] = invokeErr.Error()
  	}
  	planmeta.AppendStructuredEvent(planDir, "tool_invocation", audit)

  	fmt.Println(display)
  	if invokeErr != nil {
  		fmt.Println("error:", invokeErr)
  		os.Exit(1)
  	}
  }
  ```

  (`writeFullLog`, `summarizeOutput`, and `loadContextConfig` already exist in
  `cli/verify_cmd.go`/`cli/context_cmd.go` — same `main` package, no new import needed for
  those three calls.)

**Verify:** `cd cli && go build ./... && echo BUILD_OK` (T11 in tests.md covers a live
invoke-allowed and invoke-refused walkthrough).

---

## Task 12 — `context_cmd.go`: `routeTools` and the `## Tools` section

- [x] **12.1** In `cli/context_cmd.go`, add imports and a shared `routeTools` helper plus a
  `writeToolSelection` printer:

  Old:
  ```go
  import (
  	"fmt"
  	"io"
  	"os"
  	"path/filepath"
  	"strings"
  	"time"

  	"eng/internal/contextcfg"
  	"eng/internal/docsearch"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/skillrouter"
  	"eng/internal/skills"
  	"eng/internal/taskscope"
  )
  ```

  New:
  ```go
  import (
  	"fmt"
  	"io"
  	"os"
  	"path/filepath"
  	"sort"
  	"strings"
  	"time"

  	"eng/internal/contextcfg"
  	"eng/internal/docsearch"
  	"eng/internal/planmeta"
  	"eng/internal/project"
  	"eng/internal/skillrouter"
  	"eng/internal/skills"
  	"eng/internal/taskscope"
  	"eng/internal/toolpolicy"
  	"eng/internal/toolrouter"
  )
  ```

  Append after `writeSkillSelection`:

  ```go
  // routeTools computes the tool-routing result for a request's selected
  // skills — the Requirement 28 signal connecting Skill Router output to
  // Tool Router input, with no capability-detection logic of its own
  // (Phase 7 spec.md, out-of-scope: no NL-to-capability detection beyond
  // this). Shared by buildContextBundle's ## Tools section and `eng
  // capabilities explain`.
  func routeTools(repoRoot, role, request string, approved bool) toolrouter.Result {
  	cfg := loadContextConfig(repoRoot)
  	sel, _, err := selectSkills(repoRoot, request, cfg)

  	seen := map[string]bool{}
  	var required []string
  	if err == nil {
  		for _, s := range sel.Skills {
  			for _, c := range s.Capabilities {
  				if !seen[c] {
  					seen[c] = true
  					required = append(required, c)
  				}
  			}
  		}
  	}
  	sort.Strings(required)

  	var policy toolpolicy.Policy
  	if pcfg, err := project.Load(repoRoot); err == nil {
  		policy = pcfg.Tools
  	}
  	adapters := registeredAdapters(repoRoot)
  	return toolrouter.Route(required, adapters, role, policy, approved)
  }

  func writeToolSelection(w io.Writer, result toolrouter.Result) {
  	fmt.Fprintf(w, "\n## Tools\n")
  	if len(result.Allowed) == 0 && len(result.NeedsApproval) == 0 && len(result.Blocked) == 0 {
  		fmt.Fprintf(w, "(no external capabilities requested by the selected skills)\n")
  		return
  	}
  	for _, s := range result.Allowed {
  		fmt.Fprintf(w, "- %s (%s) ALLOWED — %s\n", s.Capability, s.Adapter, s.Reason)
  	}
  	for _, b := range result.NeedsApproval {
  		fmt.Fprintf(w, "- %s (%s) NEEDS_APPROVAL — %s\n", b.Capability, b.Adapter, b.Reason)
  	}
  	for _, b := range result.Blocked {
  		fmt.Fprintf(w, "- %s (%s) BLOCKED — %s\n", b.Capability, b.Adapter, b.Reason)
  	}
  }
  ```

- [x] **12.2** In `buildContextBundle`'s `"planner"` case, add the `## Tools` section right
  after the existing `## Skills` block:

  Old:
  ```go
  		fmt.Fprintf(&out, "## Skills\n")
  		writeSkillSelection(&out, sel, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range sel.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
  		}

  	case "plan-reviewer":
  ```

  New:
  ```go
  		fmt.Fprintf(&out, "## Skills\n")
  		writeSkillSelection(&out, sel, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range sel.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, sel.Explanations[i].Reason)
  		}
  		toolResult := routeTools(repoRoot, role, request, meta.ApprovedAt != "")
  		writeToolSelection(&out, toolResult)
  		fmt.Fprintf(&manifest, "tools:\n")
  		for _, s := range toolResult.Allowed {
  			fmt.Fprintf(&manifest, "  - %s: allowed\n", s.Capability)
  		}

  	case "plan-reviewer":
  ```

  In the `"executor"` case, make the identical addition right after its own `## Skills`
  block:

  Old:
  ```go
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Fprintf(&out, "\n## Skills\n")
  		writeSkillSelection(&out, selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range selected.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, selected.Explanations[i].Reason)
  		}

  	case "verifier":
  ```

  New:
  ```go
  		selected, total, _ := selectSkills(repoRoot, request, cfg)
  		fmt.Fprintf(&out, "\n## Skills\n")
  		writeSkillSelection(&out, selected, total, cfg)
  		fmt.Fprintf(&manifest, "skills:\n")
  		for i, s := range selected.Skills {
  			fmt.Fprintf(&manifest, "  - %s: %q\n", s.Name, selected.Explanations[i].Reason)
  		}
  		toolResult := routeTools(repoRoot, role, request, meta.ApprovedAt != "")
  		writeToolSelection(&out, toolResult)
  		fmt.Fprintf(&manifest, "tools:\n")
  		for _, s := range toolResult.Allowed {
  			fmt.Fprintf(&manifest, "  - %s: allowed\n", s.Capability)
  		}

  	case "verifier":
  ```

  Confirm the exact current local variable name in the `"executor"` case (`selected` vs.
  `sel`) against the real file before applying — Phase 6's Task 6 used `sel` in the
  `"planner"` case and `selected` in the `"executor"` case; match whichever the file
  actually has.

**Verify:** `cd cli && go build ./... && echo BUILD_OK` (T12 in tests.md covers a live
`eng adapter prompt` run showing the `## Tools` section).

---

## Task 13 — `eng capabilities explain`

- [x] **13.1** Replace the full contents of `cli/capabilities_cmd.go`: (implemented without
  the dead-code "tolerate list list" guard the draft flagged as unnecessary — `capabilitiesList`
  simply receives `args` with `"list"` already stripped by the switch, indices start at 0)

  ```go
  package main

  import (
  	"fmt"
  	"os"
  	"path/filepath"
  	"strings"

  	"eng/internal/agent"
  	"eng/internal/capabilities"
  	"eng/internal/planmeta"
  )

  func cmdCapabilities(args []string) {
  	if len(args) == 0 {
  		fmt.Println("Usage: eng capabilities <list|explain> ...")
  		os.Exit(1)
  	}
  	switch args[0] {
  	case "list":
  		capabilitiesList(args[1:])
  	case "explain":
  		capabilitiesExplain(args[1:])
  	default:
  		fmt.Println("Usage: eng capabilities <list|explain> ...")
  		os.Exit(1)
  	}
  }

  func capabilitiesList(args []string) {
  	if len(args) > 0 && args[0] == "list" {
  		args = args[1:] // tolerate `eng capabilities list list` typo-safety: no-op if absent
  	}
  	verbose := false
  	role := ""
  	for i := 0; i < len(args); i++ {
  		switch args[i] {
  		case "--verbose":
  			verbose = true
  		case "--role":
  			if i+1 < len(args) {
  				role = args[i+1]
  			}
  		}
  	}

  	for _, c := range capabilities.DescribeAll() {
  		if role != "" && !agent.RoleMayUse(role, c.Name) {
  			continue
  		}
  		status := "unavailable"
  		if c.Available {
  			status = "available"
  		}
  		if verbose {
  			fmt.Printf("%-10s %-12s provider=%-14s version=%s\n", c.Name, status, c.Provider, c.Version)
  		} else {
  			fmt.Printf("%-10s %s\n", c.Name, status)
  		}
  	}
  }

  func capabilitiesExplain(args []string) {
  	if len(args) < 2 {
  		fmt.Println(`Usage: eng capabilities explain <role> <plan-dir> ["<request text>"]`)
  		os.Exit(1)
  	}
  	role := args[0]
  	planDir, err := filepath.Abs(args[1])
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}
  	request := ""
  	if len(args) > 2 {
  		request = strings.Join(args[2:], " ")
  	}

  	repoRoot, err := os.Getwd()
  	if err != nil {
  		fmt.Println("error:", err)
  		os.Exit(1)
  	}

  	meta, err := planmeta.Load(planDir)
  	if err != nil {
  		fmt.Printf("no %s found in %s\n", planmeta.FileName, planDir)
  		os.Exit(1)
  	}

  	result := routeTools(repoRoot, role, request, meta.ApprovedAt != "")

  	fmt.Println("Tool routing for role:", role)
  	for _, s := range result.Allowed {
  		fmt.Printf("  %-24s ALLOWED        (%s) — %s\n", s.Capability, s.Adapter, s.Reason)
  	}
  	for _, b := range result.NeedsApproval {
  		fmt.Printf("  %-24s NEEDS_APPROVAL (%s) — %s\n", b.Capability, b.Adapter, b.Reason)
  	}
  	for _, b := range result.Blocked {
  		fmt.Printf("  %-24s BLOCKED        (%s) — %s\n", b.Capability, b.Adapter, b.Reason)
  	}
  	if len(result.Allowed)+len(result.NeedsApproval)+len(result.Blocked) == 0 {
  		fmt.Println("  (no external capabilities requested by skills matching this request)")
  	}
  }
  ```

  (The `capabilitiesList`-entry no-op guard exists only because `cmdCapabilities`'s switch
  already consumed the `"list"` token before calling it — `args` passed in never actually
  contains a leading `"list"`, so that guard is dead defensive code; skip it and pass `args`
  straight through unless a real mismatch is found when reading the current file.)

**Verify:** `cd cli && go build ./... && echo BUILD_OK` — `eng capabilities list [--verbose]
[--role <role>]` must behave identically to before.

---

## Task 14 — `eng doctor`: bounded `Tools:` section

- [x] **14.1** In `cli/doctor.go`, insert a `Tools:` section between the existing `Skills:`
  block and the `Capabilities:` block:

  Old:
  ```go
  	fmt.Println("\nCapabilities:")
  	for _, name := range capabilities.Known {
  		status := "unavailable"
  		if capabilities.Detect(name) {
  			status = "available"
  		}
  		fmt.Printf("  %-10s %s\n", name, status)
  	}
  }
  ```

  New:
  ```go
  	adapters := registeredAdapters(dir)
  	fmt.Println("\nTools:")
  	for _, a := range adapters {
  		status := "unavailable"
  		if a.Available() {
  			status = "available"
  		}
  		fmt.Printf("  %-10s %-12s [%d capabilities]\n", a.Name(), status, len(a.Capabilities()))
  	}

  	fmt.Println("\nCapabilities:")
  	for _, name := range capabilities.Known {
  		status := "unavailable"
  		if capabilities.Detect(name) {
  			status = "available"
  		}
  		fmt.Printf("  %-10s %s\n", name, status)
  	}
  }
  ```

  No new import is needed — `registeredAdapters` lives in `cli/tools_cmd.go`, same `main`
  package.

**Verify:** `cd cli && go build ./... && go vet ./... && echo OK`.

---

## Task 15 — Wire `eng tools` into `main.go`

- [x] **15.1** In `cli/main.go`, add the dispatch case:

  Old:
  ```go
  	case "logs":
  		cmdLogs(os.Args[2:])
  	default:
  ```

  New:
  ```go
  	case "logs":
  		cmdLogs(os.Args[2:])
  	case "tools":
  		cmdTools(os.Args[2:])
  	default:
  ```

- [x] **15.2** In `usage()`, append after the existing `capabilities list` line and update
  it to mention `explain`:

  Old:
  ```
    capabilities list                  Report which known tools are on PATH
  ```

  New:
  ```
    capabilities list                  Report which known tools are on PATH
    capabilities explain <role> <plan-dir> ["<text>"]   Explain tool routing for a request
    tools invoke <role> <capability> <plan-dir> [args...]   Invoke one capability, audited
  ```

**Verify:** `cd cli && go vet ./... && go build -o eng . && ./eng` — usage output must
include both new lines; `./eng tools invoke` with no args must print its own usage and
exit non-zero.

---

## Task 16 — Connect a skill to a real capability

- [x] **16.1** In `harness/skills/engineering/debugging/SKILL.md`, populate the previously-
  empty `capabilities:` list (proving the Requirement 28 wiring end to end — searching
  documented gotchas/history is a genuinely useful step in a debugging method):

  Old:
  ```yaml
  requires: []
  recommends: []
  capabilities: []
  conflicts: []
  ```

  New:
  ```yaml
  requires: []
  recommends: []
  capabilities: [docs.search]
  conflicts: []
  ```

  Add one sentence to the body's "Method" section, after step 3 ("Form a hypothesis"),
  noting that searching `docs/gotchas.md`/prior notes for a similar symptom (via
  `docs.search`, when available) often shortcuts straight to a known cause — a small,
  honest addition, not a rewrite of the skill.

**Verify:** `cd cli && go build -o eng . && ./eng skills validate` — still `0 error(s)`; a
request containing "debug" now routes `docs.search` as a candidate capability via
`eng context skills`.

---

## Task 17 — `docs/tools.md` and `METHOD.md` pointers

- [x] **17.1** Create `docs/tools.md` covering: the adapter model (Skill vs. Agent Adapter
  vs. Tool/MCP Adapter vs. Harness, restated from Requirement 1), the capability naming and
  risk model, the two role-permission axes, the project `tools:` policy schema and its
  precedence order (verbatim from spec.md Decision 7), the secrets boundary (Decision 10,
  including the `credential_env:` reference pattern for a future adapter — not implemented
  by any Phase 7 adapter), the MCP registry foundation and why `ReferenceMCPAdapter` is a
  mock, how to inspect routing (`eng capabilities explain`, `eng doctor`), how to invoke a
  capability (`eng tools invoke`) and where its audit event lands, and how to add a new
  adapter (implement `tooladapter.Adapter`, register it in `registeredAdapters`).

- [x] **17.2** In `harness/core/context-manager/METHOD.md`, add one sentence to the
  paragraph added in Phase 6 noting that the composed bundle also includes a `## Tools`
  section for roles that receive `## Skills`, driven by the same selected skills'
  `capabilities:` field.

- [x] **17.3** In `harness/core/runtime/METHOD.md`, add one short paragraph after "The
  sequence" pointing at `eng capabilities explain <role> <plan-dir> "<request>"` for
  inspecting what external capabilities a request will need before invoking anything, and
  `eng tools invoke` as the only sanctioned invocation path (never a raw shell command to an
  external service inside a session).

**Verify:** manual read — no code to build.

---

## Task 18 — `docs/src-map.md`, `README.md`, `ROADMAP.md`

- [x] **18.1** In `docs/src-map.md`, add a final module section after the Phase 6 entry,
  following the exact established pattern:

  ```markdown

  ### `cli/internal/toolcap/`, `cli/internal/tooladapter/`, `cli/internal/toolpolicy/`, `cli/internal/toolrouter/`, `cli/internal/mcpregistry/` — Phase 7 tool adapter runtime

  What it does: `toolcap` defines the capability/risk model (`READ < WRITE < DESTRUCTIVE <
  HIGH_RISK`). `tooladapter`'s `Adapter` interface (revised from Phase 5's foundation-only
  shape) is implemented by `GitAdapter` (upgraded), the new external reference
  `GitHubAdapter` (read-only, via `gh`), and `ReferenceMCPAdapter` (deterministic mock MCP
  server — no live transport). `toolpolicy.Decide` is the one policy function: built-in
  hard deny, then project `tools.deny`/role toolbox/role risk ceiling/`tools.require_approval`
  (gated on `plan.yaml`'s existing execution-approval field, not a new approval concept)/
  `tools.allow`, then a safe risk-based default. `toolrouter.Route` (the new authoritative
  path; `Filter` stays as Phase 5's simpler adapter-name filter) buckets each required
  capability into Allowed/NeedsApproval/Blocked with a reason. `mcpregistry` loads
  `harness/mcp/servers.yaml`, a static, credential-free discovery list. `eng tools invoke`
  is the one invocation boundary — it always runs `toolpolicy.Decide` first, writes a
  compact `tool_invocation` audit event via the existing `planmeta.AppendStructuredEvent`
  (Phase 5) regardless of outcome, and reuses Phase 4/5's `writeFullLog`/`summarizeOutput`
  for bounded output. `buildContextBundle`'s Planner/Executor sections gain a `## Tools`
  section driven by the selected skills' `capabilities:` field (Phase 6) — the Requirement
  28 connection from Skill Router to Tool Router, with no separate NL-capability-detection
  logic.

  Key files: `cli/internal/toolpolicy/toolpolicy.go` (`Decide`), `cli/internal/toolrouter/toolrouter.go`
  (`Route`), `cli/internal/tooladapter/{tooladapter,github,reference_mcp}.go`

  Notable: `Config.RequireApproval` (a pre-existing, never-read, undocumented field) was
  deliberately *not* repurposed for the new `tools.require_approval` policy — a new, nested
  `Config.Tools` field was added instead, since guessing at the old field's original intent
  was a bigger risk than adding a clearly-scoped one. A project with no `tools:` block at
  all (every project before this phase) gets exactly today's behavior: read capabilities
  work, write-ish ones require the same plan approval Phase 3 already established.

  From: `.plans/2026-08-25-v2-harness-phase7-tools/`
  ```

- [x] **18.2** In `README.md`, add a Phase 7 paragraph immediately after the Phase 6
  section (before the following `---`):

  ```markdown

  Phase 7 turns the tool-adapter foundation into an enforced runtime — capabilities are
  risk-classified, role permission is checked (not just reported), and every invocation is
  audited:

  ```bash
  cd cli && go build -o eng .
  ./eng capabilities explain executor .plans/2026-08-25-my-feature "inspect open pull requests"
  ./eng tools invoke executor git.status .plans/2026-08-25-my-feature
  ```

  See `docs/tools.md` for the full adapter/capability/policy model and
  `.plans/2026-08-25-v2-harness-phase7-tools/spec.md` for the full design.
  ```

- [x] **18.3** In `ROADMAP.md`, extend the note to include the Phase 7 plan link, following
  the same pattern as the Phase 6 addition.

**Verify:** manual read — no code to build.

---

## Task 19 — Version bump (last task)

- [x] **19.1** Update `harness/VERSION`:

  ```
  0.7.0-phase7-tools
  ```

**Verify:** `cat harness/VERSION`.
