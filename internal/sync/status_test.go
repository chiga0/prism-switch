package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "(empty)"},
		{"short", "***"},
		{"12345678", "***"},
		{"sk-or-v1-abcdef123456", "sk-o***3456"},
		{"sk-ant-api03-xyz", "sk-a***-xyz"},
		{"AIzaSyD1234567890", "AIza***7890"},
	}
	for _, tt := range tests {
		got := MaskKey(tt.input)
		if got != tt.want {
			t.Errorf("MaskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStatusSynced(t *testing.T) {
	t.Setenv("PRISM_ST_KEY", "sk-status-test")

	agent.ResetRegistry()
	claudeDir := filepath.Join(t.TempDir(), "claude")
	agent.Register(agent.NewClaudeProjectorWithBase(claudeDir))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p1": {APIKey: "${PRISM_ST_KEY}", BaseURL: "https://x.com"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p1", Model: "claude-test"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)

	// First sync to create live config
	if err := engine.Sync(nil); err != nil {
		t.Fatal(err)
	}

	statuses, err := engine.Status(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	s := statuses[0]
	if s.Agent != "claude" {
		t.Errorf("Agent = %q", s.Agent)
	}
	if s.Provider != "p1" {
		t.Errorf("Provider = %q", s.Provider)
	}
	if s.State != "synced" {
		t.Errorf("State = %q, want synced (detail: %s)", s.State, s.Detail)
	}
	if s.APIKeyMask != "sk-s***test" {
		t.Errorf("APIKeyMask = %q", s.APIKeyMask)
	}
}

func TestStatusDrifted(t *testing.T) {
	t.Setenv("PRISM_ST_KEY2", "sk-desired")

	agent.ResetRegistry()
	claudeDir := filepath.Join(t.TempDir(), "claude")
	agent.Register(agent.NewClaudeProjectorWithBase(claudeDir))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p1": {APIKey: "${PRISM_ST_KEY2}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p1", Model: "desired-model"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	// Write a DIFFERENT live config manually
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-manual","ANTHROPIC_MODEL":"manual-model"}}`), 0o644)

	statuses, _ := engine.Status(nil)
	if statuses[0].State != "drifted" {
		t.Errorf("State = %q, want drifted", statuses[0].State)
	}
}

func TestStatusMissing(t *testing.T) {
	t.Setenv("PRISM_ST_KEY3", "sk-missing")

	agent.ResetRegistry()
	claudeDir := filepath.Join(t.TempDir(), "claude")
	agent.Register(agent.NewClaudeProjectorWithBase(claudeDir))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p1": {APIKey: "${PRISM_ST_KEY3}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p1"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status(nil)
	if statuses[0].State != "missing" {
		t.Errorf("State = %q, want missing", statuses[0].State)
	}
}

func TestStatusUnknownAgent(t *testing.T) {
	agent.ResetRegistry()

	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents: map[string]*config.AgentConfig{
			"unknown_agent": {Current: "p1"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status([]string{"unknown_agent"})
	if statuses[0].State != "error" {
		t.Errorf("State = %q, want error", statuses[0].State)
	}
}

func TestStatusMissingProvider(t *testing.T) {
	agent.ResetRegistry()
	agent.Register(agent.NewClaudeProjectorWithBase(t.TempDir()))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "ghost"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status(nil)
	if statuses[0].State != "error" {
		t.Errorf("State = %q, want error", statuses[0].State)
	}
	if statuses[0].Detail == "" {
		t.Error("Detail should explain the missing provider")
	}
}

func TestStatusMissingEnvVar(t *testing.T) {
	agent.ResetRegistry()
	agent.Register(agent.NewClaudeProjectorWithBase(t.TempDir()))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p1": {APIKey: "${PRISM_STATUS_TOTALLY_MISSING}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p1"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status(nil)
	if statuses[0].State != "error" {
		t.Errorf("State = %q, want error", statuses[0].State)
	}
}

func TestStatusSpecificAgent(t *testing.T) {
	t.Setenv("PRISM_ST_KEY4", "sk-specific")

	agent.ResetRegistry()
	agent.Register(agent.NewClaudeProjectorWithBase(t.TempDir()))
	agent.Register(agent.NewCodexProjectorWithBase(t.TempDir()))

	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p1": {APIKey: "${PRISM_ST_KEY4}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p1"},
			"codex":  {Current: "p1"},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status([]string{"claude"})
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Agent != "claude" {
		t.Errorf("Agent = %q", statuses[0].Agent)
	}
}

func TestStatusNotInConfig(t *testing.T) {
	agent.ResetRegistry()

	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents:    map[string]*config.AgentConfig{},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	config.Save(cfgPath, cfg)
	engine := NewEngineWithConfig(cfg, cfgPath)

	statuses, _ := engine.Status([]string{"claude"})
	if statuses[0].State != "error" {
		t.Errorf("State = %q, want error", statuses[0].State)
	}
	if statuses[0].Detail != "not in config" {
		t.Errorf("Detail = %q", statuses[0].Detail)
	}
}
