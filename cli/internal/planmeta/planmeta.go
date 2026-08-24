package planmeta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type RetryCounters struct {
	Build           int `yaml:"build"`
	UnitTest        int `yaml:"unit_test"`
	IntegrationTest int `yaml:"integration_test"`
}

type RetryBudget struct {
	Build           int `yaml:"build"`
	UnitTest        int `yaml:"unit_test"`
	IntegrationTest int `yaml:"integration_test"`
}

type PlannedAt struct {
	GitSHA string `yaml:"git_sha"`
}

type Review struct {
	Verdict        string `yaml:"verdict,omitempty"` // PASS | REJECT
	BlockingIssues int    `yaml:"blocking_issues,omitempty"`
	ReviewedAt     string `yaml:"reviewed_at,omitempty"`
}

type Verification struct {
	Verdict    string `yaml:"verdict,omitempty"` // PASS | FAIL
	VerifiedAt string `yaml:"verified_at,omitempty"`
}

type Meta struct {
	Plan        string        `yaml:"plan"`
	RiskLevel   string        `yaml:"risk_level"` // quick-fix | bug | feature | architecture | high-risk
	PlannedAt   PlannedAt     `yaml:"planned_at"`
	Status      string        `yaml:"status,omitempty"` // deprecated (Phase 2) — see State
	State       string        `yaml:"state"`
	WriteScope  []string      `yaml:"write_scope"`
	Retry       RetryCounters `yaml:"retry"`
	RetryBudget RetryBudget   `yaml:"retry_budget"`

	RequiresApproval bool         `yaml:"requires_approval"`
	ApprovedAt       string       `yaml:"approved_at,omitempty"`
	ApprovedBy       string       `yaml:"approved_by,omitempty"`
	Review           Review       `yaml:"review,omitempty"`
	Verification     Verification `yaml:"verification,omitempty"`
}

const FileName = "plan.yaml"
const EventsFileName = "events.jsonl"

// legacyStatusToState migrates Phase 2's write-once `status` field to the
// richer Phase 3 `state` enum for plan.yaml files created before Phase 3.
var legacyStatusToState = map[string]string{
	"planned":   "PLANNED",
	"reviewed":  "REVIEWED",
	"executing": "EXECUTING",
	"verified":  "COMPLETED",
	"failed":    "FAILED",
}

func Load(planDir string) (*Meta, error) {
	data, err := os.ReadFile(filepath.Join(planDir, FileName))
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.State == "" {
		if mapped, ok := legacyStatusToState[m.Status]; ok {
			m.State = mapped
		} else {
			m.State = "NEW"
		}
	}
	return &m, nil
}

func Save(planDir string, m *Meta) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(planDir, FileName), data, 0o644)
}

// DefaultBudget is used when neither plan.yaml nor .agent/project.yaml
// declares a retry_budget.
func DefaultBudget() RetryBudget {
	return RetryBudget{Build: 2, UnitTest: 2, IntegrationTest: 1}
}

type Event struct {
	Type   string `json:"type"`
	At     string `json:"at"`
	Detail string `json:"detail,omitempty"`
}

// AppendEvent records one line to <planDir>/events.jsonl. Append-only by
// design — plan.yaml stays a small, current-state snapshot; the full
// history of every transition lives here instead, per Phase 3's "preserve
// history rather than overwriting evidence" requirement.
func AppendEvent(planDir, eventType, detail string) error {
	f, err := os.OpenFile(filepath.Join(planDir, EventsFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	ev := Event{Type: eventType, At: time.Now().UTC().Format(time.RFC3339), Detail: detail}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}
