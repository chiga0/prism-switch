package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chiga0/prism-switch/internal/config"
)

// QwenCodeProjector manages ~/.qwen/settings.json.
// Qwen Code stores API keys in the "env" section and model providers in "modelProviders".
type QwenCodeProjector struct {
	baseDir string
}

func NewQwenCodeProjector() (*QwenCodeProjector, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir for qwen-code: %w", err)
	}
	return &QwenCodeProjector{baseDir: filepath.Join(home, ".qwen")}, nil
}

func NewQwenCodeProjectorWithBase(dir string) *QwenCodeProjector {
	return &QwenCodeProjector{baseDir: dir}
}

func (q *QwenCodeProjector) Name() string        { return "qwen-code" }
func (q *QwenCodeProjector) DisplayName() string  { return "Qwen Code" }
func (q *QwenCodeProjector) Protocol() config.Protocol { return config.ProtocolOpenAI }
func (q *QwenCodeProjector) ConfigPaths() []string {
	return []string{filepath.Join(q.baseDir, "settings.json")}
}

// Project writes the provider into Qwen Code's settings.json.
// Qwen Code uses env vars for API keys (similar to Claude Code) and modelProviders for model config.
func (q *QwenCodeProjector) Project(p *config.ResolvedProvider) error {
	settingsPath := filepath.Join(q.baseDir, "settings.json")
	settings := readJSONOrWarn(settingsPath)

	// Set env vars
	env, _ := settings["env"].(map[string]interface{})
	if env == nil {
		env = make(map[string]interface{})
	}
	env["QWEN_API_KEY"] = p.APIKey
	if p.BaseURL != "" {
		env["QWEN_BASE_URL"] = p.BaseURL
	} else {
		delete(env, "QWEN_BASE_URL")
	}
	settings["env"] = env

	// Set model in modelProviders if model is specified
	if p.Model != "" {
		modelProviders, _ := settings["modelProviders"].(map[string]interface{})
		if modelProviders == nil {
			modelProviders = make(map[string]interface{})
		}
		// Store active model selection
		settings["model"] = p.Model
		settings["modelProviders"] = modelProviders
	}

	return atomicWriteJSON(settingsPath, settings)
}

// ReadLive reads the current Qwen Code settings.json.
func (q *QwenCodeProjector) ReadLive() (*config.ResolvedProvider, error) {
	settingsPath := filepath.Join(q.baseDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("read qwen-code settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse qwen-code settings: %w", err)
	}

	p := &config.ResolvedProvider{Name: "live"}

	env, _ := settings["env"].(map[string]interface{})
	if env != nil {
		if v, ok := env["QWEN_API_KEY"].(string); ok {
			p.APIKey = v
		}
		if v, ok := env["QWEN_BASE_URL"].(string); ok {
			p.BaseURL = v
		}
	}

	if m, ok := settings["model"].(string); ok {
		p.Model = m
	}

	if p.APIKey == "" {
		return nil, fmt.Errorf("no qwen-code api key found")
	}

	return p, nil
}
