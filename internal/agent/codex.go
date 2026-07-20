package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chiga0/prism-switch/internal/config"
	"github.com/pelletier/go-toml/v2"
)

// CodexProjector manages ~/.codex/auth.json and ~/.codex/config.toml.
type CodexProjector struct {
	baseDir string
}

func NewCodexProjector() *CodexProjector {
	home, _ := os.UserHomeDir()
	return &CodexProjector{baseDir: filepath.Join(home, ".codex")}
}

func NewCodexProjectorWithBase(dir string) *CodexProjector {
	return &CodexProjector{baseDir: dir}
}

func (c *CodexProjector) Name() string        { return "codex" }
func (c *CodexProjector) DisplayName() string  { return "Codex CLI" }
func (c *CodexProjector) ConfigPaths() []string {
	return []string{
		filepath.Join(c.baseDir, "auth.json"),
		filepath.Join(c.baseDir, "config.toml"),
	}
}

// Project writes the provider into Codex's auth.json and config.toml.
func (c *CodexProjector) Project(p *config.ResolvedProvider) error {
	auth := map[string]string{"OPENAI_API_KEY": p.APIKey}
	if err := atomicWriteJSON(filepath.Join(c.baseDir, "auth.json"), auth); err != nil {
		return fmt.Errorf("write codex auth: %w", err)
	}

	tomlPath := filepath.Join(c.baseDir, "config.toml")
	tomlData := make(map[string]interface{})
	if data, err := os.ReadFile(tomlPath); err == nil {
		_ = toml.Unmarshal(data, &tomlData)
	}
	if p.Model != "" {
		tomlData["model"] = p.Model
	}
	out, err := toml.Marshal(tomlData)
	if err != nil {
		return fmt.Errorf("marshal codex config: %w", err)
	}
	return atomicWrite(tomlPath, out, 0o644)
}

// ReadLive reads the current Codex auth.json and config.toml.
func (c *CodexProjector) ReadLive() (*config.ResolvedProvider, error) {
	p := &config.ResolvedProvider{Name: "live"}

	authPath := filepath.Join(c.baseDir, "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, fmt.Errorf("read codex auth: %w", err)
	}
	var auth map[string]string
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("parse codex auth: %w", err)
	}
	p.APIKey = auth["OPENAI_API_KEY"]

	tomlPath := filepath.Join(c.baseDir, "config.toml")
	if tomlData, err := os.ReadFile(tomlPath); err == nil {
		var parsed map[string]interface{}
		if toml.Unmarshal(tomlData, &parsed) == nil {
			if m, ok := parsed["model"].(string); ok {
				p.Model = m
			}
		}
	}

	if p.APIKey == "" {
		return nil, fmt.Errorf("no codex api key found")
	}

	return p, nil
}
