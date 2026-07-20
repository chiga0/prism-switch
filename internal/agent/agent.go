package agent

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chiga0/prism-switch/internal/config"
)

// Projector writes and reads agent-specific configuration files.
type Projector interface {
	// Name returns the agent identifier used in config.yaml (e.g. "claude").
	Name() string
	// DisplayName returns the human-readable name (e.g. "Claude Code").
	DisplayName() string
	// Protocol returns the API wire format this agent speaks.
	Protocol() config.Protocol
	// ConfigPaths returns the live config file paths this agent manages.
	ConfigPaths() []string
	// Project writes the resolved provider to the agent's live config files.
	Project(p *config.ResolvedProvider) error
	// ReadLive reads the current live configuration back as a ResolvedProvider.
	ReadLive() (*config.ResolvedProvider, error)
}

var registry = map[string]Projector{}

// Register adds a projector to the global registry.
func Register(p Projector) {
	registry[p.Name()] = p
}

// Get returns a projector by agent name.
func Get(name string) (Projector, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %q (available: %s)", name, AvailableNames())
	}
	return p, nil
}

// All returns all registered projectors sorted by name.
func All() []Projector {
	result := make([]Projector, 0, len(registry))
	for _, p := range registry {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// AvailableNames returns a comma-separated list of registered agent names.
func AvailableNames() string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// ResetRegistry clears all registered projectors (for testing).
func ResetRegistry() {
	registry = map[string]Projector{}
}
