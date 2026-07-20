package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

func TestGeminiProject(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	if p.Name() != "gemini" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.DisplayName() != "Gemini CLI" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}
	if len(p.ConfigPaths()) != 2 {
		t.Errorf("ConfigPaths() len = %d, want 2", len(p.ConfigPaths()))
	}

	provider := &config.ResolvedProvider{
		Name:    "test",
		APIKey:  "AIza-test-key",
		BaseURL: "https://gemini-proxy.example.com",
		Model:   "gemini-2.5-pro",
	}

	if err := p.Project(provider); err != nil {
		t.Fatalf("Project() error: %v", err)
	}

	// Verify .env
	envData, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	envStr := string(envData)
	if !strings.Contains(envStr, "GEMINI_API_KEY=AIza-test-key") {
		t.Errorf(".env missing GEMINI_API_KEY: %s", envStr)
	}
	if !strings.Contains(envStr, "GOOGLE_GEMINI_BASE_URL=https://gemini-proxy.example.com") {
		t.Errorf(".env missing GOOGLE_GEMINI_BASE_URL: %s", envStr)
	}

	// Verify settings.json
	settingsData, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]interface{}
	json.Unmarshal(settingsData, &settings)
	if settings["model"] != "gemini-2.5-pro" {
		t.Errorf("model = %v", settings["model"])
	}
}

func TestGeminiProjectNoBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "AIza-k", Model: "gemini-2.5-flash"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	envData, _ := os.ReadFile(filepath.Join(dir, ".env"))
	if strings.Contains(string(envData), "GOOGLE_GEMINI_BASE_URL") {
		t.Error("GOOGLE_GEMINI_BASE_URL should not be written when empty")
	}
}

func TestGeminiProjectPreservesSettings(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	// Write existing settings with extra fields
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"model":"old","theme":"dark"}`), 0o644)

	provider := &config.ResolvedProvider{Name: "t", APIKey: "k", Model: "new-model"}
	if err := p.Project(provider); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	if settings["model"] != "new-model" {
		t.Errorf("model not updated: %v", settings["model"])
	}
	if settings["theme"] != "dark" {
		t.Errorf("theme lost: %v", settings["theme"])
	}
}

func TestGeminiReadLive(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	os.WriteFile(filepath.Join(dir, ".env"), []byte("GEMINI_API_KEY=AIza-live\nGOOGLE_GEMINI_BASE_URL=https://live.example.com\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"model":"gemini-live"}`), 0o644)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatalf("ReadLive() error: %v", err)
	}
	if live.APIKey != "AIza-live" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
	if live.BaseURL != "https://live.example.com" {
		t.Errorf("BaseURL = %q", live.BaseURL)
	}
	if live.Model != "gemini-live" {
		t.Errorf("Model = %q", live.Model)
	}
}

func TestGeminiReadLiveWithComments(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	os.WriteFile(filepath.Join(dir, ".env"), []byte("# comment\nGEMINI_API_KEY=AIza-k\n\n# another\n"), 0o600)

	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != "AIza-k" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
}

func TestGeminiReadLiveMissing(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for missing .env")
	}
}

func TestGeminiReadLiveEmptyKey(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("OTHER=val\n"), 0o600)
	_, err := p.ReadLive()
	if err == nil {
		t.Error("expected error for empty api key")
	}
}

func TestGeminiReadLiveMalformedLine(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("GEMINI_API_KEY=AIza-ok\nMALFORMED_LINE\n"), 0o600)
	live, err := p.ReadLive()
	if err != nil {
		t.Fatal(err)
	}
	if live.APIKey != "AIza-ok" {
		t.Errorf("APIKey = %q", live.APIKey)
	}
}

func TestGeminiRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProjectorWithBase(dir)

	original := &config.ResolvedProvider{
		Name:    "rt",
		APIKey:  "AIza-rt",
		BaseURL: "https://rt.example.com",
		Model:   "gemini-rt",
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
