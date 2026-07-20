package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chiga0/prism-switch/internal/config"
)

// ClaudeProjector manages ~/.claude/settings.json.
type ClaudeProjector struct {
	baseDir string
}

func NewClaudeProjector() *ClaudeProjector {
	home, _ := os.UserHomeDir()
	return &ClaudeProjector{baseDir: filepath.Join(home, ".claude")}
}

func NewClaudeProjectorWithBase(dir string) *ClaudeProjector {
	return &ClaudeProjector{baseDir: dir}
}

func (c *ClaudeProjector) Name() string        { return "claude" }
func (c *ClaudeProjector) DisplayName() string  { return "Claude Code" }
func (c *ClaudeProjector) ConfigPaths() []string {
	return []string{filepath.Join(c.baseDir, "settings.json")}
}

// Project writes the provider into Claude's settings.json, preserving existing fields.
func (c *ClaudeProjector) Project(p *config.ResolvedProvider) error {
	settingsPath := filepath.Join(c.baseDir, "settings.json")

	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(data, &settings)
	}

	env, _ := settings["env"].(map[string]interface{})
	if env == nil {
		env = make(map[string]interface{})
	}
	env["ANTHROPIC_AUTH_TOKEN"] = p.APIKey
	if p.BaseURL != "" {
		env["ANTHROPIC_BASE_URL"] = p.BaseURL
	} else {
		delete(env, "ANTHROPIC_BASE_URL")
	}
	if p.Model != "" {
		env["ANTHROPIC_MODEL"] = p.Model
	} else {
		delete(env, "ANTHROPIC_MODEL")
	}
	settings["env"] = env

	return atomicWriteJSON(settingsPath, settings)
}

// ReadLive reads the current Claude settings.json.
func (c *ClaudeProjector) ReadLive() (*config.ResolvedProvider, error) {
	settingsPath := filepath.Join(c.baseDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("read claude settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse claude settings: %w", err)
	}

	env, _ := settings["env"].(map[string]interface{})
	if env == nil {
		return nil, fmt.Errorf("no env section in claude settings")
	}

	p := &config.ResolvedProvider{Name: "live"}
	if v, ok := env["ANTHROPIC_AUTH_TOKEN"].(string); ok {
		p.APIKey = v
	}
	if v, ok := env["ANTHROPIC_BASE_URL"].(string); ok {
		p.BaseURL = v
	}
	if v, ok := env["ANTHROPIC_MODEL"].(string); ok {
		p.Model = v
	}

	return p, nil
}
