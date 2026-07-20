package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chiga0/prism-switch/internal/config"
)

// ZCodeProjector manages ~/.zcode/v2/config.json.
// ZCode (Zhipu Z.AI) stores providers under "provider" → "<id>" → "options" → {apiKey, baseURL}.
type ZCodeProjector struct {
	baseDir string
}

func NewZCodeProjector() (*ZCodeProjector, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir for zcode: %w", err)
	}
	return &ZCodeProjector{baseDir: filepath.Join(home, ".zcode", "v2")}, nil
}

func NewZCodeProjectorWithBase(dir string) *ZCodeProjector {
	return &ZCodeProjector{baseDir: dir}
}

func (z *ZCodeProjector) Name() string        { return "zcode" }
func (z *ZCodeProjector) DisplayName() string  { return "ZCode" }
func (z *ZCodeProjector) ConfigPaths() []string {
	return []string{filepath.Join(z.baseDir, "config.json")}
}

// Project writes the provider into ZCode's config.json.
func (z *ZCodeProjector) Project(p *config.ResolvedProvider) error {
	configPath := filepath.Join(z.baseDir, "config.json")
	settings := readJSONOrWarn(configPath)

	providerSection, _ := settings["provider"].(map[string]interface{})
	if providerSection == nil {
		providerSection = make(map[string]interface{})
	}

	// Use "prism" as the managed provider id
	prismProvider, _ := providerSection["prism"].(map[string]interface{})
	if prismProvider == nil {
		prismProvider = map[string]interface{}{
			"enabled": true,
		}
	}
	prismProvider["enabled"] = true

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

	// Set model
	if p.Model != "" {
		models, _ := prismProvider["models"].(map[string]interface{})
		if models == nil {
			models = make(map[string]interface{})
		}
		models[p.Model] = map[string]interface{}{}
		prismProvider["models"] = models
	}

	providerSection["prism"] = prismProvider
	settings["provider"] = providerSection

	return atomicWriteJSON(configPath, settings)
}

// ReadLive reads the current ZCode config.
func (z *ZCodeProjector) ReadLive() (*config.ResolvedProvider, error) {
	configPath := filepath.Join(z.baseDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read zcode config: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse zcode config: %w", err)
	}

	p := &config.ResolvedProvider{Name: "live"}

	providerSection, _ := settings["provider"].(map[string]interface{})
	if providerSection != nil {
		// Look for "prism" provider first, then fall back to first enabled
		if prism, ok := providerSection["prism"].(map[string]interface{}); ok {
			if opts, ok := prism["options"].(map[string]interface{}); ok {
				if k, ok := opts["apiKey"].(string); ok {
					p.APIKey = k
				}
				if u, ok := opts["baseURL"].(string); ok {
					p.BaseURL = u
				}
			}
			if models, ok := prism["models"].(map[string]interface{}); ok {
				for name := range models {
					p.Model = name
					break
				}
			}
		} else {
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

	if p.APIKey == "" {
		return nil, fmt.Errorf("no zcode api key found")
	}

	return p, nil
}
