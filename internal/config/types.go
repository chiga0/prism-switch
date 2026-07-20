package config

import "sort"

// Protocol identifies the API wire format an agent speaks.
type Protocol string

const (
	ProtocolOpenAI    Protocol = "openai"
	ProtocolAnthropic Protocol = "anthropic"
	ProtocolGoogle    Protocol = "google"
)

// Config is the root configuration structure loaded from ~/.prism/config.yaml.
type Config struct {
	Providers map[string]*Provider    `yaml:"providers"`
	Agents    map[string]*AgentConfig `yaml:"agents"`
}

// AgentNames returns a sorted list of agent names in the config.
func (c *Config) AgentNames() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Provider defines a shared API provider. Credentials use ${ENV_VAR} references.
// BaseURLs maps protocol → endpoint. A single base_url (legacy) applies to all protocols.
type Provider struct {
	APIKey   string            `yaml:"api_key"`
	BaseURL  string            `yaml:"base_url,omitempty"`            // legacy: applies to all protocols
	BaseURLs map[Protocol]string `yaml:"base_urls,omitempty"`         // per-protocol endpoints
}

// BaseURLFor returns the endpoint for a given protocol.
// Priority: base_urls[protocol] > base_url (legacy) > "".
func (p *Provider) BaseURLFor(proto Protocol) string {
	if p.BaseURLs != nil {
		if u, ok := p.BaseURLs[proto]; ok {
			return u
		}
	}
	return p.BaseURL
}

// AgentConfig defines per-agent configuration referencing a provider.
type AgentConfig struct {
	Current string `yaml:"current"`
	Model   string `yaml:"model,omitempty"`
}

// ResolvedProvider is a provider with environment variables expanded at runtime.
type ResolvedProvider struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}
