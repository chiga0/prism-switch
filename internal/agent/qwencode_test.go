package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

func TestQwenCodeProject(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)

	if p.Name() != "qwen-code" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "Qwen Code" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}

	provider := &config.ResolvedProvider{
		Name:    "test",
		APIKey:  "sk-qwen-key",
		BaseURL: "https://qwen-proxy.example.com/v1",
		Model:   "qwen3-max",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	env := settings["env"].(map[string]interface{})
	if env["QWEN_API_KEY"] != "sk-qwen-key" {
		t.Errorf("QWEN_API_KEY = %v", env["QWEN_API_KEY"])
	}
	if env["QWEN_BASE_URL"] != "https://qwen-proxy.example.com/v1" {
		t.Errorf("QWEN_BASE_URL = %v", env["QWEN_BASE_URL"])
	}
	if settings["model"] != "qwen3-max" {
		t.Errorf("model = %v", settings["model"])
	}
}

func TestQwenCodeProjectPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)

	existing := map[string]interface{}{
		"env": map[string]interface{}{
			"OTHER_KEY": "other-value",
		},
		"channels": map[string]interface{}{
			"my-channel": map[string]interface{}{"type": "dingtalk"},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "new-key", Model: "m"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	result, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(result, &settings)

	env := settings["env"].(map[string]interface{})
	if env["OTHER_KEY"] != "other-value" {
		t.Error("existing env var was lost")
	}
	if env["QWEN_API_KEY"] != "new-key" {
		t.Errorf("QWEN_API_KEY = %v", env["QWEN_API_KEY"])
	}
	if _, ok := settings["channels"]; !ok {
		t.Error("channels section was lost")
	}
}

func TestQwenCodeProjectNoBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)

	// First with BaseURL
	p.Project(&config.ResolvedProvider{Name: "a", APIKey: "k1", BaseURL: "https://x.com"})
	// Then without
	p.Project(&config.ResolvedProvider{Name: "b", APIKey: "k2"})

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	env := settings["env"].(map[string]interface{})
	if _, exists := env["QWEN_BASE_URL"]; exists {
		t.Error("QWEN_BASE_URL should be removed when empty")
	}
}

func TestQwenCodeReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)

	settings := map[string]interface{}{
		"env": map[string]interface{}{
			"QWEN_API_KEY":  "sk-live-qwen",
			"QWEN_BASE_URL": "https://live.qwen.com",
		},
		"model": "qwen-live-model",
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-live-qwen" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
	if live.BaseURL != "https://live.qwen.com" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
	if live.Model != "qwen-live-model" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestQwenCodeReadLiveMissing(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing settings")
	}
}

func TestQwenCodeReadLiveNoKey(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"env":{"OTHER":"val"}}`), 0o644)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing api key")
	}
}

func TestQwenCodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewQwenCodeProjectorWithBase(dir)

	original := &config.ResolvedProvider{
		Name:    "rt",
		APIKey:  "sk-rt-qwen",
		BaseURL: "https://rt.qwen.com",
		Model:   "qwen-rt",
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

func TestNewQwenCodeProjectorDefault(t *testing.T) {
	p, err := NewQwenCodeProjector()
	if err != nil {
		t.Fatalf("NewQwenCodeProjector() error: %v", err)
	}
	if p.Name() != "qwen-code" {
		t.Errorf("Name() = %q", p.Name())
	}
}
