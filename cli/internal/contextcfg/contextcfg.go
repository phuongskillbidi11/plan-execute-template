package contextcfg

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the effective, fully-resolved context budget — never has nil
// fields once returned from Load or Default.
type Config struct {
	Strategy              string // full | selective
	MaxSkills             int
	MaxDocs               int
	MaxLogLines           int
	IncludeCompletedTasks bool
	SummarizeToolOutput   bool

	MaxLogFiles   int // .agent/logs/ retention — see internal/logprune
	MaxLogAgeDays int
	MaxLogTotalMB int
}

// override mirrors Config but with pointer fields, so YAML unmarshal can
// distinguish "this key was absent" (nil) from "this key was explicitly
// false/zero" — plain bool/int fields can't make that distinction, which
// would otherwise silently reset an unspecified field to its zero value
// every time any override file is loaded.
type override struct {
	Strategy              *string `yaml:"strategy"`
	MaxSkills             *int    `yaml:"max_skills"`
	MaxDocs               *int    `yaml:"max_docs"`
	MaxLogLines           *int    `yaml:"max_log_lines"`
	IncludeCompletedTasks *bool   `yaml:"include_completed_tasks"`
	SummarizeToolOutput   *bool   `yaml:"summarize_tool_output"`

	MaxLogFiles   *int `yaml:"max_log_files"`
	MaxLogAgeDays *int `yaml:"max_log_age_days"`
	MaxLogTotalMB *int `yaml:"max_log_total_mb"`
}

func Default() Config {
	return Config{
		Strategy:              "selective",
		MaxSkills:             5,
		MaxDocs:               8,
		MaxLogLines:           300,
		IncludeCompletedTasks: false,
		SummarizeToolOutput:   true,
		MaxLogFiles:           100,
		MaxLogAgeDays:         30,
		MaxLogTotalMB:         250,
	}
}

// Load reads .agent/context.yaml if present, else globalDefaultPath, else
// returns Default() unchanged — a project with no context config at all
// (the common case, per Requirement 15) works with zero extra files.
func Load(projectDir, globalDefaultPath string) (Config, error) {
	cfg := Default()

	path := globalDefaultPath
	local := filepath.Join(projectDir, ".agent", "context.yaml")
	if _, err := os.Stat(local); err == nil {
		path = local
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var o override
	if err := yaml.Unmarshal(data, &o); err != nil {
		return cfg, err
	}
	if o.Strategy != nil {
		cfg.Strategy = *o.Strategy
	}
	if o.MaxSkills != nil {
		cfg.MaxSkills = *o.MaxSkills
	}
	if o.MaxDocs != nil {
		cfg.MaxDocs = *o.MaxDocs
	}
	if o.MaxLogLines != nil {
		cfg.MaxLogLines = *o.MaxLogLines
	}
	if o.IncludeCompletedTasks != nil {
		cfg.IncludeCompletedTasks = *o.IncludeCompletedTasks
	}
	if o.SummarizeToolOutput != nil {
		cfg.SummarizeToolOutput = *o.SummarizeToolOutput
	}
	if o.MaxLogFiles != nil {
		cfg.MaxLogFiles = *o.MaxLogFiles
	}
	if o.MaxLogAgeDays != nil {
		cfg.MaxLogAgeDays = *o.MaxLogAgeDays
	}
	if o.MaxLogTotalMB != nil {
		cfg.MaxLogTotalMB = *o.MaxLogTotalMB
	}
	return cfg, nil
}
