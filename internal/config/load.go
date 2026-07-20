package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var envVarRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// DefaultConfigPath returns ~/.prism/config.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".prism", "config.yaml")
	}
	return filepath.Join(home, ".prism", "config.yaml")
}

// Dir returns the directory portion of a config path.
func Dir(path string) string {
	return filepath.Dir(path)
}

// Load reads and parses the YAML config file.
// Environment variable references (${VAR}) are NOT expanded at load time.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]*Provider)
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]*AgentConfig)
	}
	return &cfg, nil
}

// Save writes the config to disk with 0600 permissions, preserving ${VAR} references.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ExpandEnv expands ${VAR} references in a string.
// Returns an error listing all unset variables.
func ExpandEnv(s string) (string, error) {
	var missing []string
	result := envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		name := envVarRe.FindStringSubmatch(match)[1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("environment variables not set: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// Resolve expands environment variables in a provider and combines with agent model.
func Resolve(name string, p *Provider, agentCfg *AgentConfig) (*ResolvedProvider, error) {
	apiKey, err := ExpandEnv(p.APIKey)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", name, err)
	}
	baseURL := ""
	if p.BaseURL != "" {
		baseURL, err = ExpandEnv(p.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", name, err)
		}
	}
	model := ""
	if agentCfg != nil {
		model = agentCfg.Model
	}
	return &ResolvedProvider{
		Name:    name,
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}, nil
}
