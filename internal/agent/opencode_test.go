package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

func TestOpenCodeProject(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)

	if p.Name() != "opencode" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "OpenCode" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}

	provider := &config.ResolvedProvider{
		Name:    "test",
		APIKey:  "sk-opencode-key",
		BaseURL: "https://proxy.example.com/v1",
		Model:   "deepseek/deepseek-v4-pro",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	if settings["model"] != "deepseek/deepseek-v4-pro" {
		t.Errorf("model = %v", settings["model"])
	}
	providerSection := settings["provider"].(map[string]interface{})
	prism := providerSection["prism"].(map[string]interface{})
	opts := prism["options"].(map[string]interface{})
	if opts["apiKey"] != "sk-opencode-key" {
		t.Errorf("apiKey = %v", opts["apiKey"])
	}
	if opts["baseURL"] != "https://proxy.example.com/v1" {
		t.Errorf("baseURL = %v", opts["baseURL"])
	}
}

func TestOpenCodeProjectPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)

	// Write existing config with another provider
	existing := map[string]interface{}{
		"model": "old-model",
		"provider": map[string]interface{}{
			"idealab": map[string]interface{}{
				"name": "IdeaLab",
				"options": map[string]interface{}{
					"apiKey":  "existing-key",
					"baseURL": "https://idealab.example.com",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "new-key", Model: "new-model"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	result, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	var settings map[string]interface{}
	json.Unmarshal(result, &settings)

	// Existing provider should be preserved
	providerSection := settings["provider"].(map[string]interface{})
	if _, ok := providerSection["idealab"]; !ok {
		t.Error("existing provider 'idealab' was lost")
	}
	// New provider should exist
	if _, ok := providerSection["prism"]; !ok {
		t.Error("prism provider not created")
	}
}

func TestOpenCodeReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)

	cfg := map[string]interface{}{
		"model": "test-model",
		"provider": map[string]interface{}{
			"prism": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey":  "sk-live-key",
					"baseURL": "https://live.example.com",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-live-key" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
	if live.BaseURL != "https://live.example.com" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
	if live.Model != "test-model" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestOpenCodeReadLiveFallback(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)

	// No "prism" provider, should fall back to first available
	cfg := map[string]interface{}{
		"model": "fallback-model",
		"provider": map[string]interface{}{
			"custom": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "sk-fallback",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-fallback" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
}

func TestOpenCodeReadLiveMissing(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestOpenCodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewOpenCodeProjectorWithBase(dir)

	original := &config.ResolvedProvider{
		Name:    "rt",
		APIKey:  "sk-rt",
		BaseURL: "https://rt.example.com",
		Model:   "rt-model",
	}
	if err := p.Project(original); err != nil {
		t.Fatal(err)
	}
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != original.APIKey || live.BaseURL != original.BaseURL || live.Model != original.Model {
		t.Errorf("round-trip mismatch: got %+v", live)
	}
}

func TestNewOpenCodeProjectorDefault(t *testing.T) {
	p, err := NewOpenCodeProjector()
	if err != nil {
		t.Fatalf("NewOpenCodeProjector() error: %v", err)
	}
	if p.Name() != "opencode" {
		t.Errorf("Name() = %q", p.Name())
	}
}
