package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	p := DefaultConfigPath()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}
	want := filepath.Join(home, ".prism", "config.yaml")
	if p != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", p, want)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}

agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  codex:
    current: anthropic
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(cfg.Providers))
	}
	if cfg.Providers["openrouter"].APIKey != "${OPENROUTER_API_KEY}" {
		t.Errorf("api_key not preserved: %q", cfg.Providers["openrouter"].APIKey)
	}
	if cfg.Providers["openrouter"].BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("base_url wrong: %q", cfg.Providers["openrouter"].BaseURL)
	}
	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}
	if cfg.Agents["claude"].Current != "openrouter" {
		t.Errorf("claude current = %q, want openrouter", cfg.Agents["claude"].Current)
	}
	if cfg.Agents["claude"].Model != "anthropic/claude-sonnet-4" {
		t.Errorf("claude model = %q", cfg.Agents["claude"].Model)
	}
	if cfg.Agents["codex"].Model != "" {
		t.Errorf("codex model should be empty, got %q", cfg.Agents["codex"].Model)
	}
}

func TestLoadNilMaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Providers == nil {
		t.Error("Providers map should be initialized")
	}
	if cfg.Agents == nil {
		t.Error("Agents map should be initialized")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.yaml")

	cfg := &Config{
		Providers: map[string]*Provider{
			"test": {APIKey: "${TEST_KEY}", BaseURL: "https://example.com"},
		},
		Agents: map[string]*AgentConfig{
			"claude": {Current: "test", Model: "claude-sonnet-4"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file perm = %o, want 600", info.Mode().Perm())
	}

	// Reload and verify
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save error: %v", err)
	}
	if loaded.Providers["test"].APIKey != "${TEST_KEY}" {
		t.Errorf("api_key not preserved after round-trip: %q", loaded.Providers["test"].APIKey)
	}
	if loaded.Agents["claude"].Model != "claude-sonnet-4" {
		t.Errorf("model not preserved: %q", loaded.Agents["claude"].Model)
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("PRISM_TEST_KEY", "sk-test-123")
	t.Setenv("PRISM_TEST_URL", "https://example.com")

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"single var", "${PRISM_TEST_KEY}", "sk-test-123", false},
		{"var in string", "key=${PRISM_TEST_KEY}", "key=sk-test-123", false},
		{"multiple vars", "${PRISM_TEST_KEY}:${PRISM_TEST_URL}", "sk-test-123:https://example.com", false},
		{"no vars", "plain-text", "plain-text", false},
		{"empty string", "", "", false},
		{"missing var", "${PRISM_NONEXISTENT_VAR_XYZ}", "", true},
		{"mixed missing", "${PRISM_TEST_KEY}:${PRISM_NONEXISTENT_VAR_XYZ}", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandEnv(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandEnv(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExpandEnv(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Setenv("PRISM_RESOLVE_KEY", "sk-resolved")
	t.Setenv("PRISM_RESOLVE_URL", "https://resolved.example.com")

	t.Run("full resolve", func(t *testing.T) {
		p := &Provider{APIKey: "${PRISM_RESOLVE_KEY}", BaseURL: "${PRISM_RESOLVE_URL}"}
		ac := &AgentConfig{Current: "test", Model: "gpt-4"}
		r, err := Resolve("test", p, ac, ProtocolOpenAI)
		if err != nil {
			t.Fatal(err)
		}
		if r.Name != "test" {
			t.Errorf("Name = %q", r.Name)
		}
		if r.APIKey != "sk-resolved" {
			t.Errorf("APIKey = %q", r.APIKey)
		}
		if r.BaseURL != "https://resolved.example.com" {
			t.Errorf("BaseURL = %q", r.BaseURL)
		}
		if r.Model != "gpt-4" {
			t.Errorf("Model = %q", r.Model)
		}
	})

	t.Run("no base_url", func(t *testing.T) {
		p := &Provider{APIKey: "${PRISM_RESOLVE_KEY}"}
		r, err := Resolve("test", p, nil, ProtocolOpenAI)
		if err != nil {
			t.Fatal(err)
		}
		if r.BaseURL != "" {
			t.Errorf("BaseURL should be empty, got %q", r.BaseURL)
		}
		if r.Model != "" {
			t.Errorf("Model should be empty with nil AgentConfig, got %q", r.Model)
		}
	})

	t.Run("missing env var", func(t *testing.T) {
		p := &Provider{APIKey: "${PRISM_TOTALLY_MISSING_VAR}"}
		_, err := Resolve("test", p, nil, ProtocolOpenAI)
		if err == nil {
			t.Error("expected error for missing env var")
		}
	})

	t.Run("missing base_url env var", func(t *testing.T) {
		p := &Provider{APIKey: "${PRISM_RESOLVE_KEY}", BaseURL: "${PRISM_TOTALLY_MISSING_VAR}"}
		_, err := Resolve("test", p, nil, ProtocolOpenAI)
		if err == nil {
			t.Error("expected error for missing base_url env var")
		}
	})

	t.Run("per-protocol base_urls", func(t *testing.T) {
		p := &Provider{
			APIKey: "${PRISM_RESOLVE_KEY}",
			BaseURLs: map[Protocol]string{
				ProtocolOpenAI:    "https://openai-compat.example.com/v1",
				ProtocolAnthropic: "https://anthropic-compat.example.com/v1",
			},
		}
		rOpenAI, err := Resolve("test", p, nil, ProtocolOpenAI)
		if err != nil {
			t.Fatal(err)
		}
		if rOpenAI.BaseURL != "https://openai-compat.example.com/v1" {
			t.Errorf("OpenAI BaseURL = %q", rOpenAI.BaseURL)
		}
		rAnthropic, err := Resolve("test", p, nil, ProtocolAnthropic)
		if err != nil {
			t.Fatal(err)
		}
		if rAnthropic.BaseURL != "https://anthropic-compat.example.com/v1" {
			t.Errorf("Anthropic BaseURL = %q", rAnthropic.BaseURL)
		}
		// Google protocol not in base_urls → falls back to BaseURL (empty)
		rGoogle, err := Resolve("test", p, nil, ProtocolGoogle)
		if err != nil {
			t.Fatal(err)
		}
		if rGoogle.BaseURL != "" {
			t.Errorf("Google BaseURL should be empty, got %q", rGoogle.BaseURL)
		}
	})
}
