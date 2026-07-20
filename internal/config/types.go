package config

// Config is the root configuration structure loaded from ~/.prism/config.yaml.
type Config struct {
	Providers map[string]*Provider    `yaml:"providers"`
	Agents    map[string]*AgentConfig `yaml:"agents"`
}

// Provider defines a shared API provider. Credentials use ${ENV_VAR} references.
type Provider struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url,omitempty"`
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
