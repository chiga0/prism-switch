package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chiga0/prism-switch/internal/config"
)

// OpenCodeProjector manages ~/.config/opencode/opencode.json.
type OpenCodeProjector struct {
	baseDir string
}

func NewOpenCodeProjector() (*OpenCodeProjector, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir for opencode: %w", err)
	}
	return &OpenCodeProjector{baseDir: filepath.Join(home, ".config", "opencode")}, nil
}

func NewOpenCodeProjectorWithBase(dir string) *OpenCodeProjector {
	return &OpenCodeProjector{baseDir: dir}
}

func (o *OpenCodeProjector) Name() string        { return "opencode" }
func (o *OpenCodeProjector) DisplayName() string  { return "OpenCode" }
func (o *OpenCodeProjector) ConfigPaths() []string {
	return []string{filepath.Join(o.baseDir, "opencode.json")}
}

// Project writes the provider into OpenCode's opencode.json.
// OpenCode stores providers under "provider" → "<provider-name>" → "options" → {apiKey, baseURL}.
// The active model is set at the top-level "model" field.
func (o *OpenCodeProjector) Project(p *config.ResolvedProvider) error {
	configPath := filepath.Join(o.baseDir, "opencode.json")
	settings := readJSONOrWarn(configPath)

	// Ensure provider section exists
	providerSection, _ := settings["provider"].(map[string]interface{})
	if providerSection == nil {
		providerSection = make(map[string]interface{})
	}

	// Use "prism" as the provider name we manage
	prismProvider, _ := providerSection["prism"].(map[string]interface{})
	if prismProvider == nil {
		prismProvider = map[string]interface{}{
			"name": "Prism Managed",
			"npm":  "@ai-sdk/openai-compatible",
		}
	}

	options, _ := prismProvider["options"].(map[string]interface{})
	if options == nil {
		options = make(map[string]interface{})
	}
	options["apiKey"] = p.APIKey
	if p.BaseURL != "" {
		options["baseURL"] = p.BaseURL
	} else {
		delete(options, "baseURL")
	}
	prismProvider["options"] = options
	providerSection["prism"] = prismProvider
	settings["provider"] = providerSection

	// Set model
	if p.Model != "" {
		settings["model"] = p.Model
	}

	return atomicWriteJSON(configPath, settings)
}

// ReadLive reads the current OpenCode config.
func (o *OpenCodeProjector) ReadLive() (*config.ResolvedProvider, error) {
	configPath := filepath.Join(o.baseDir, "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read opencode config: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse opencode config: %w", err)
	}

	p := &config.ResolvedProvider{Name: "live"}

	// Try to find our managed provider first, then fall back to any provider
	providerSection, _ := settings["provider"].(map[string]interface{})
	if providerSection != nil {
		// Look for "prism" provider first
		if prism, ok := providerSection["prism"].(map[string]interface{}); ok {
			if opts, ok := prism["options"].(map[string]interface{}); ok {
				if k, ok := opts["apiKey"].(string); ok {
					p.APIKey = k
				}
				if u, ok := opts["baseURL"].(string); ok {
					p.BaseURL = u
				}
			}
		} else {
			// Fall back to first provider with options
			for _, v := range providerSection {
				if prov, ok := v.(map[string]interface{}); ok {
					if opts, ok := prov["options"].(map[string]interface{}); ok {
						if k, ok := opts["apiKey"].(string); ok {
							p.APIKey = k
						}
						if u, ok := opts["baseURL"].(string); ok {
							p.BaseURL = u
						}
						break
					}
				}
			}
		}
	}

	if m, ok := settings["model"].(string); ok {
		p.Model = m
	}

	if p.APIKey == "" {
		return nil, fmt.Errorf("no opencode api key found")
	}

	return p, nil
}
