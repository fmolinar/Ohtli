// Package devices holds the controller's device registry: which targets
// exist, how to reach them, and how to authenticate.
package devices

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Target describes one gNMI-reachable device.
type Target struct {
	Name    string    `yaml:"name"`
	Address string    `yaml:"address"` // host:port
	TLS     TargetTLS `yaml:"tls"`
	Paths   []string  `yaml:"paths,omitempty"` // subscription paths; falls back to the registry default
}

// TargetTLS mirrors gnmi.TLSConfig in YAML form so it can be loaded from a
// registry file without internal/devices depending on internal/gnmi.
type TargetTLS struct {
	Insecure           bool   `yaml:"insecure"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CAFile             string `yaml:"ca_file"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
}

// Registry is the loaded set of targets plus any registry-wide defaults.
type Registry struct {
	DefaultPaths []string `yaml:"default_paths"`
	Targets      []Target `yaml:"targets"`
}

// Load reads a YAML device registry from path.
func Load(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("devices: reading registry %s: %w", path, err)
	}

	var reg Registry
	if err := yaml.Unmarshal(raw, &reg); err != nil {
		return nil, fmt.Errorf("devices: parsing registry %s: %w", path, err)
	}

	for i, t := range reg.Targets {
		if t.Name == "" {
			return nil, fmt.Errorf("devices: target at index %d is missing a name", i)
		}
		if t.Address == "" {
			return nil, fmt.Errorf("devices: target %q is missing an address", t.Name)
		}
		if len(t.Paths) == 0 {
			reg.Targets[i].Paths = reg.DefaultPaths
		}
	}

	return &reg, nil
}
