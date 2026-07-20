package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
	"github.com/pelletier/go-toml/v2"
)

func TestCodexProject(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	if p.Name() != "codex" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "Codex CLI" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}
	if len(p.ConfigPaths()) != 2 {
		t.Errorf("ConfigPaths() len = %d, want 2", len(p.ConfigPaths()))
	}

	provider := &config.ResolvedProvider{
		Name:  "test",
		APIKey: "sk-codex-key",
		Model: "o3",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	// Verify auth.json
	authData, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var auth map[string]string
	json.Unmarshal(authData, &auth)
	if auth["OPENAI_API_KEY"] != "sk-codex-key" {
		t.Errorf("OPENAI_API_KEY = %q", auth["OPENAI_API_KEY"])
	}

	// Verify config.toml
	tomlData, err := os.ReadFile(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var tomlCfg map[string]interface{}
	toml.Unmarshal(tomlData, &tomlCfg)
	if tomlCfg["model"] != "o3" {
		t.Errorf("model = %v", tomlCfg["model"])
	}
}

func TestCodexProjectPreservesToml(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	// Write existing config.toml with extra fields
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("model = \"old\"\napproval_mode = \"full-auto\"\n"), 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "k", Model: "new-model"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	tomlData, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	var tomlCfg map[string]interface{}
	toml.Unmarshal(tomlData, &tomlCfg)
	if tomlCfg["model"] != "new-model" {
		t.Errorf("model not updated: %v", tomlCfg["model"])
	}
	if tomlCfg["approval_mode"] != "full-auto" {
		t.Errorf("approval_mode lost: %v", tomlCfg["approval_mode"])
	}
}

func TestCodexProjectNoModel(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "k"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	tomlData, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	var tomlCfg map[string]interface{}
	toml.Unmarshal(tomlData, &tomlCfg)
	if _, exists := tomlCfg["model"]; exists {
		t.Error("model should not be set when empty")
	}
}

func TestCodexReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"OPENAI_API_KEY":"sk-live"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("model = \"o4-mini\"\n"), 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "sk-live" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
	if live.Model != "o4-mini" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestCodexReadLiveMissingAuth(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing auth.json")
	}
}

func TestCodexReadLiveEmptyKey(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"OTHER":"val"}`), 0o644)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for empty api key")
	}
}

func TestCodexReadLiveInvalidAuth(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{bad`), 0o644)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for invalid auth.json")
	}
}

func TestCodexRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	original := &config.ResolvedProvider{Name: "rt", APIKey: "sk-rt", Model: "o3"}
	if err := p.Project(original); err != nil {
		t.Fatal(err)
	}
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != original.APIKey || live.Model != original.Model {
		t.Errorf("round-trip mismatch: got %+v", live)
	}
}

func TestCodexProjectWithBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	provider := &config.ResolvedProvider{
		Name:    "custom",
		APIKey:  "sk-custom",
		BaseURL: "https://custom.endpoint.com/v1",
		Model:   "gpt-4o",
	}
	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.BaseURL != "https://custom.endpoint.com/v1" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
	if live.Model != "gpt-4o" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestCodexProjectRemovesBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	// First project with BaseURL
	p.Project(&config.ResolvedProvider{Name: "a", APIKey: "k1", BaseURL: "https://x.com", Model: "m1"})
	// Then project without BaseURL
	p.Project(&config.ResolvedProvider{Name: "b", APIKey: "k2", Model: "m2"})

	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.BaseURL != "" {
		t.Errorf("BaseURL should be empty after removal, got %q", live.BaseURL)
	}
}

func TestCodexProjectCorruptToml(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	// Write corrupt TOML
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("{{{invalid toml"), 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "sk-k", Model: "o3"}
	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() should succeed despite corrupt TOML: %v", err)
	}

	// Verify backup was created
	entries, _ := os.ReadDir(dir)
	backupFound := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".prism-backup-") {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("backup should be created for corrupt TOML")
	}

	// Verify new config is valid
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.Model != "o3" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestCodexReadLiveWithBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := NewCodexProjectorWithBase(dir)

	os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"OPENAI_API_KEY":"sk-live"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("model = \"o4-mini\"\napi_base_url = \"https://proxy.example.com\"\n"), 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.BaseURL != "https://proxy.example.com" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
}

func TestNewCodexProjectorDefault(t *testing.T) {
	p, err := NewCodexProjector()
	if err != nil {
		t.Fatalf("NewCodexProjector() error: %v", err)
	}
	if p.Name() != "codex" {
		t.Errorf("Name() = %q", p.Name())
	}
}
