package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

func TestZCodeProject(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)

	if p.Name() != "zcode" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "ZCode" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}

	provider := &config.ResolvedProvider{
		Name:    "test",
		APIKey:  "sk-zcode-key",
		BaseURL: "https://open.bigmodel.cn/api/anthropic",
		Model:   "GLM-5.2",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	providerSection := settings["provider"].(map[string]interface{})
	prism := providerSection["prism"].(map[string]interface{})
	if prism["enabled"] != true {
		t.Error("prism provider should be enabled")
	}
	opts := prism["options"].(map[string]interface{})
	if opts["apiKey"] != "sk-zcode-key" {
		t.Errorf("apiKey = %v", opts["apiKey"])
	}
	if opts["baseURL"] != "https://open.bigmodel.cn/api/anthropic" {
		t.Errorf("baseURL = %v", opts["baseURL"])
	}
	models := prism["models"].(map[string]interface{})
	if _, ok := models["GLM-5.2"]; !ok {
		t.Error("GLM-5.2 model not set")
	}
}

func TestZCodeProjectPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)

	existing := map[string]interface{}{
		"provider": map[string]interface{}{
			"zai": map[string]interface{}{
				"enabled": true,
				"options": map[string]interface{}{
					"apiKey":  "existing-jwt",
					"baseURL": "https://api.z.ai/api/anthropic",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "new-key", Model: "GLM-5-Turbo"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	result, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	var settings map[string]interface{}
	json.Unmarshal(result, &settings)

	providerSection := settings["provider"].(map[string]interface{})
	if _, ok := providerSection["zai"]; !ok {
		t.Error("existing provider 'zai' was lost")
	}
	if _, ok := providerSection["prism"]; !ok {
		t.Error("prism provider not created")
	}
}

func TestZCodeReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)

	cfg := map[string]interface{}{
		"provider": map[string]interface{}{
			"prism": map[string]interface{}{
				"enabled": true,
				"options": map[string]interface{}{
					"apiKey":  "sk-live-zcode",
					"baseURL": "https://api.z.ai/api/anthropic",
				},
				"models": map[string]interface{}{
					"GLM-5.2": map[string]interface{}{},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-live-zcode" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
	if live.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
	if live.Model != "GLM-5.2" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestZCodeReadLiveFallback(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)

	cfg := map[string]interface{}{
		"provider": map[string]interface{}{
			"custom": map[string]interface{}{
				"options": map[string]interface{}{
					"apiKey": "sk-fallback-zcode",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-fallback-zcode" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
}

func TestZCodeReadLiveMissing(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestZCodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewZCodeProjectorWithBase(dir)

	original := &config.ResolvedProvider{
		Name:    "rt",
		APIKey:  "sk-rt-zcode",
		BaseURL: "https://rt.z.ai",
		Model:   "GLM-5.2",
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

func TestNewZCodeProjectorDefault(t *testing.T) {
	p, err := NewZCodeProjector()
	if err != nil {
		t.Fatalf("NewZCodeProjector() error: %v", err)
	}
	if p.Name() != "zcode" {
		t.Errorf("Name() = %q", p.Name())
	}
}
