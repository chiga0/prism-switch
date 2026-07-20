package sync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

// ImportResult reports what happened during an import operation.
type ImportResult struct {
	Agent    string
	Provider string
	EnvVar   string
}

// Engine orchestrates sync, switch, import and status operations.
type Engine struct {
	cfg     *config.Config
	cfgPath string
}

// NewEngine loads config from path and returns an Engine.
func NewEngine(cfgPath string) (*Engine, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	return &Engine{cfg: cfg, cfgPath: cfgPath}, nil
}

// NewEngineWithConfig creates an Engine with an already-loaded config (for testing).
func NewEngineWithConfig(cfg *config.Config, cfgPath string) *Engine {
	return &Engine{cfg: cfg, cfgPath: cfgPath}
}

// Config returns the current config.
func (e *Engine) Config() *config.Config {
	return e.cfg
}

// Sync projects the current provider to live config files.
// If agentNames is empty, syncs all agents in config.
func (e *Engine) Sync(agentNames []string) error {
	if len(agentNames) == 0 {
		agentNames = e.agentNames()
	}
	for _, name := range agentNames {
		if err := e.syncOne(name); err != nil {
			return fmt.Errorf("sync %s: %w", name, err)
		}
	}
	return nil
}

func (e *Engine) syncOne(agentName string) error {
	agentCfg, ok := e.cfg.Agents[agentName]
	if !ok {
		return fmt.Errorf("agent %q not found in config", agentName)
	}
	provider, ok := e.cfg.Providers[agentCfg.Current]
	if !ok {
		return fmt.Errorf("provider %q not found for agent %q", agentCfg.Current, agentName)
	}
	proj, err := agent.Get(agentName)
	if err != nil {
		return err
	}
	resolved, err := config.Resolve(agentCfg.Current, provider, agentCfg, proj.Protocol())
	if err != nil {
		return err
	}
	return proj.Project(resolved)
}

// Switch changes one agent's current provider, projects to live config, then saves YAML.
// Projection happens before save so a failed sync leaves the YAML unchanged.
func (e *Engine) Switch(agentName, providerName string) error {
	agentCfg, ok := e.cfg.Agents[agentName]
	if !ok {
		return fmt.Errorf("agent %q not found in config", agentName)
	}
	if _, ok := e.cfg.Providers[providerName]; !ok {
		return fmt.Errorf("provider %q not found in config", providerName)
	}

	prev := agentCfg.Current
	agentCfg.Current = providerName
	if err := e.syncOne(agentName); err != nil {
		agentCfg.Current = prev
		return err
	}
	if err := config.Save(e.cfgPath, e.cfg); err != nil {
		agentCfg.Current = prev
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// SwitchAll changes all agents' current provider, projects all, then saves YAML.
func (e *Engine) SwitchAll(providerName string) error {
	if _, ok := e.cfg.Providers[providerName]; !ok {
		return fmt.Errorf("provider %q not found in config", providerName)
	}

	prev := make(map[string]string, len(e.cfg.Agents))
	for name, agentCfg := range e.cfg.Agents {
		prev[name] = agentCfg.Current
		agentCfg.Current = providerName
	}

	if err := e.Sync(nil); err != nil {
		for name, agentCfg := range e.cfg.Agents {
			agentCfg.Current = prev[name]
		}
		return err
	}
	if err := config.Save(e.cfgPath, e.cfg); err != nil {
		for name, agentCfg := range e.cfg.Agents {
			agentCfg.Current = prev[name]
		}
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// Import reads live configs and creates provider entries with env-var placeholders.
// It never stores plaintext keys in the YAML.
func (e *Engine) Import(agentNames []string) ([]ImportResult, error) {
	if len(agentNames) == 0 {
		agentNames = e.agentNames()
	}
	var results []ImportResult
	for _, name := range agentNames {
		r, err := e.importOne(name)
		if err != nil {
			return results, fmt.Errorf("import %s: %w", name, err)
		}
		results = append(results, *r)
	}
	if err := config.Save(e.cfgPath, e.cfg); err != nil {
		return results, fmt.Errorf("save config: %w", err)
	}
	return results, nil
}

func (e *Engine) importOne(agentName string) (*ImportResult, error) {
	proj, err := agent.Get(agentName)
	if err != nil {
		return nil, err
	}
	live, err := proj.ReadLive()
	if err != nil {
		return nil, fmt.Errorf("read live config: %w", err)
	}

	providerName := fmt.Sprintf("%s-imported", agentName)
	envVarName := fmt.Sprintf("IMPORTED_%s_API_KEY", strings.ToUpper(strings.ReplaceAll(agentName, "-", "_")))

	e.cfg.Providers[providerName] = &config.Provider{
		APIKey:  fmt.Sprintf("${%s}", envVarName),
		BaseURL: live.BaseURL,
	}

	agentCfg, ok := e.cfg.Agents[agentName]
	if !ok {
		agentCfg = &config.AgentConfig{}
		e.cfg.Agents[agentName] = agentCfg
	}
	agentCfg.Current = providerName
	if live.Model != "" {
		agentCfg.Model = live.Model
	}

	return &ImportResult{
		Agent:    agentName,
		Provider: providerName,
		EnvVar:   envVarName,
	}, nil
}

func (e *Engine) agentNames() []string {
	names := make([]string, 0, len(e.cfg.Agents))
	for name := range e.cfg.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DryRunEntry describes what a sync would write for one agent.
type DryRunEntry struct {
	Agent       string
	Provider    string
	Model       string
	APIKeyMask  string
	BaseURL     string
	ConfigPaths []string
}

// DryRun resolves providers and returns what would be written without touching any files.
func (e *Engine) DryRun(agentNames []string) ([]DryRunEntry, error) {
	if len(agentNames) == 0 {
		agentNames = e.agentNames()
	}
	var entries []DryRunEntry
	for _, name := range agentNames {
		agentCfg, ok := e.cfg.Agents[name]
		if !ok {
			return nil, fmt.Errorf("agent %q not found in config", name)
		}
		provider, ok := e.cfg.Providers[agentCfg.Current]
		if !ok {
			return nil, fmt.Errorf("provider %q not found for agent %q", agentCfg.Current, name)
		}
		proj, err := agent.Get(name)
		if err != nil {
			return nil, err
		}
		resolved, err := config.Resolve(agentCfg.Current, provider, agentCfg, proj.Protocol())
		if err != nil {
			return nil, err
		}
		entries = append(entries, DryRunEntry{
			Agent:       name,
			Provider:    agentCfg.Current,
			Model:       resolved.Model,
			APIKeyMask:  MaskKey(resolved.APIKey),
			BaseURL:     resolved.BaseURL,
			ConfigPaths: proj.ConfigPaths(),
		})
	}
	return entries, nil
}
