package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chiga0/prism-switch/internal/config"
)

// GeminiProjector manages ~/.gemini/.env and ~/.gemini/settings.json.
type GeminiProjector struct {
	baseDir string
}

func NewGeminiProjector() (*GeminiProjector, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir for gemini: %w", err)
	}
	return &GeminiProjector{baseDir: filepath.Join(home, ".gemini")}, nil
}

func NewGeminiProjectorWithBase(dir string) *GeminiProjector {
	return &GeminiProjector{baseDir: dir}
}

func (g *GeminiProjector) Name() string        { return "gemini" }
func (g *GeminiProjector) DisplayName() string  { return "Gemini CLI" }
func (g *GeminiProjector) ConfigPaths() []string {
	return []string{
		filepath.Join(g.baseDir, ".env"),
		filepath.Join(g.baseDir, "settings.json"),
	}
}

// Project writes the provider into Gemini's .env and settings.json.
func (g *GeminiProjector) Project(p *config.ResolvedProvider) error {
	var envLines []string
	envLines = append(envLines, fmt.Sprintf("GEMINI_API_KEY=%s", p.APIKey))
	if p.BaseURL != "" {
		envLines = append(envLines, fmt.Sprintf("GOOGLE_GEMINI_BASE_URL=%s", p.BaseURL))
	}
	envContent := strings.Join(envLines, "\n") + "\n"
	if err := atomicWrite(filepath.Join(g.baseDir, ".env"), []byte(envContent), 0o600); err != nil {
		return fmt.Errorf("write gemini .env: %w", err)
	}

	settingsPath := filepath.Join(g.baseDir, "settings.json")
	settings := readJSONOrWarn(settingsPath)
	if p.Model != "" {
		settings["model"] = p.Model
	}
	return atomicWriteJSON(settingsPath, settings)
}

// ReadLive reads the current Gemini .env and settings.json.
func (g *GeminiProjector) ReadLive() (*config.ResolvedProvider, error) {
	p := &config.ResolvedProvider{Name: "live"}

	envPath := filepath.Join(g.baseDir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return nil, fmt.Errorf("read gemini .env: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "GEMINI_API_KEY":
			p.APIKey = parts[1]
		case "GOOGLE_GEMINI_BASE_URL":
			p.BaseURL = parts[1]
		}
	}

	settingsPath := filepath.Join(g.baseDir, "settings.json")
	if settingsData, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		if json.Unmarshal(settingsData, &settings) == nil {
			if m, ok := settings["model"].(string); ok {
				p.Model = m
			}
		}
	}

	if p.APIKey == "" {
		return nil, fmt.Errorf("no gemini api key found")
	}

	return p, nil
}
