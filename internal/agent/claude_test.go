package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

func TestClaudeProject(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	if p.Name() != "claude" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "Claude Code" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}
	if len(p.ConfigPaths()) != 1 {
		t.Errorf("ConfigPaths() len = %d", len(p.ConfigPaths()))
	}

	provider := &config.ResolvedProvider{
		Name:    "test",
		APIKey:  "sk-test-key",
		BaseURL: "https://proxy.example.com",
		Model:   "claude-sonnet-4",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	// Verify written file
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	env := settings["env"].(map[string]interface{})
	if env["ANTHROPIC_AUTH_TOKEN"] != "sk-test-key" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v", env["ANTHROPIC_AUTH_TOKEN"])
	}
	if env["ANTHROPIC_BASE_URL"] != "https://proxy.example.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %v", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_MODEL"] != "claude-sonnet-4" {
		t.Errorf("ANTHROPIC_MODEL = %v", env["ANTHROPIC_MODEL"])
	}
}

func TestClaudeProjectPreservesExistingFields(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	// Write initial settings with permissions
	initial := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN": "old-key",
		},
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(*)"},
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644)

	// Project new provider
	provider := &config.ResolvedProvider{
		Name:   "new",
		APIKey: "new-key",
		Model:  "claude-opus-4",
	}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	// Verify permissions preserved
	result, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(result, &settings)

	perms, ok := settings["permissions"]
	if !ok {
		t.Fatal("permissions field was lost")
	}
	permMap := perms.(map[string]interface{})
	if permMap["allow"] == nil {
		t.Error("permissions.allow was lost")
	}

	env := settings["env"].(map[string]interface{})
	if env["ANTHROPIC_AUTH_TOKEN"] != "new-key" {
		t.Errorf("key not updated: %v", env["ANTHROPIC_AUTH_TOKEN"])
	}
	// BaseURL should be removed since provider has no BaseURL
	if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
		t.Error("ANTHROPIC_BASE_URL should be removed when provider has no BaseURL")
	}
}

func TestClaudeProjectNoBaseURLRemovesField(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	// First project with BaseURL
	p.Project(&config.ResolvedProvider{Name: "a", APIKey: "k1", BaseURL: "https://x.com", Model: "m1"})
	// Then project without BaseURL
	p.Project(&config.ResolvedProvider{Name: "b", APIKey: "k2", Model: "m2"})

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	env := settings["env"].(map[string]interface{})
	if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
		t.Error("ANTHROPIC_BASE_URL should be removed")
	}
}

func TestClaudeReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	settings := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN": "sk-live-key",
			"ANTHROPIC_BASE_URL":   "https://live.example.com",
			"ANTHROPIC_MODEL":      "claude-live",
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(dir, "settings.json"), data, 0o644)

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
	if live.Model != "claude-live" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestClaudeReadLiveMissing(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing settings.json")
	}
}

func TestClaudeReadLiveNoEnv(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"permissions":{}}`), 0o644)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing env section")
	}
}

func TestClaudeReadLiveInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{invalid`), 0o644)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestClaudeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	original := &config.ResolvedProvider{
		Name:    "rt",
		APIKey:  "sk-roundtrip",
		BaseURL: "https://rt.example.com",
		Model:   "claude-rt",
	}
	if err := p.Project(original); err != nil {
		t.Fatal(err)
	}
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != original.APIKey || live.BaseURL != original.BaseURL || live.Model != original.Model {
		t.Errorf("round-trip mismatch: got %+v, want %+v", live, original)
	}
}

func TestNewClaudeProjectorDefault(t *testing.T) {
	p, err := NewClaudeProjector()
	if err != nil {
		t.Fatalf("NewClaudeProjector() error: %v", err)
	}
	if p.Name() != "claude" {
		t.Errorf("Name() = %q", p.Name())
	}
}

func TestClaudeProjectCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProjectorWithBase(dir)

	// Write corrupt JSON
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{invalid`), 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "sk-k", Model: "m"}
	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() should succeed despite corrupt JSON: %v", err)
	}

	// Verify backup was created
	entries, _ := os.ReadDir(dir)
	backupFound := false
	for _, e := range entries {
		if len(e.Name()) > len("settings.json") && e.Name()[:13] == "settings.json" {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("backup should be created for corrupt JSON")
	}

	// Verify new config is valid
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != "sk-k" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
}
