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
