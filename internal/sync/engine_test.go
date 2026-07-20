package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

func setupTestEnv(t *testing.T) (string, string) {
	t.Helper()
	agent.ResetRegistry()

	claudeDir := filepath.Join(t.TempDir(), "claude")
	codexDir := filepath.Join(t.TempDir(), "codex")
	geminiDir := filepath.Join(t.TempDir(), "gemini")

	agent.Register(agent.NewClaudeProjectorWithBase(claudeDir))
	agent.Register(agent.NewCodexProjectorWithBase(codexDir))
	agent.Register(agent.NewGeminiProjectorWithBase(geminiDir))

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	return cfgPath, claudeDir
}

func testConfig() *config.Config {
	return &config.Config{
		Providers: map[string]*config.Provider{
			"openrouter": {APIKey: "${PRISM_TEST_OR_KEY}", BaseURL: "https://openrouter.ai/api/v1"},
			"anthropic":  {APIKey: "${PRISM_TEST_ANT_KEY}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "openrouter", Model: "claude-sonnet-4"},
			"codex":  {Current: "openrouter", Model: "o3"},
		},
	}
}

func TestSyncAll(t *testing.T) {
	t.Setenv("PRISM_TEST_OR_KEY", "sk-or-test")
	t.Setenv("PRISM_TEST_ANT_KEY", "sk-ant-test")

	cfgPath, claudeDir := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	if err := engine.Sync(nil); err != nil {
		t.Fatalf("Sync() error: %v", err)
	}

	// Verify claude got the config
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("claude settings.json is empty")
	}
}

func TestSyncSpecificAgent(t *testing.T) {
	t.Setenv("PRISM_TEST_OR_KEY", "sk-or-test")

	cfgPath, claudeDir := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	if err := engine.Sync([]string{"claude"}); err != nil {
		t.Fatalf("Sync([claude]) error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(claudeDir, "settings.json")); err != nil {
		t.Error("claude settings.json should exist after sync")
	}
}

func TestSyncUnknownAgent(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Sync([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestSyncMissingProvider(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "ghost"},
		},
	}
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Sync(nil)
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestSyncMissingEnvVar(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"p": {APIKey: "${PRISM_TOTALLY_MISSING_SYNC_VAR}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "p"},
		},
	}
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Sync(nil)
	if err == nil {
		t.Error("expected error for missing env var")
	}
}

func TestSwitch(t *testing.T) {
	t.Setenv("PRISM_TEST_OR_KEY", "sk-or-test")
	t.Setenv("PRISM_TEST_ANT_KEY", "sk-ant-test")

	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	if err := engine.Switch("claude", "anthropic"); err != nil {
		t.Fatalf("Switch() error: %v", err)
	}

	// Verify config was saved with new current
	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Agents["claude"].Current != "anthropic" {
		t.Errorf("claude current = %q, want anthropic", reloaded.Agents["claude"].Current)
	}
	// codex should be unchanged
	if reloaded.Agents["codex"].Current != "openrouter" {
		t.Errorf("codex current changed unexpectedly: %q", reloaded.Agents["codex"].Current)
	}
}

func TestSwitchUnknownAgent(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Switch("nonexistent", "openrouter")
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestSwitchUnknownProvider(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Switch("claude", "nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestSwitchAll(t *testing.T) {
	t.Setenv("PRISM_TEST_ANT_KEY", "sk-ant-test")

	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	if err := engine.SwitchAll("anthropic"); err != nil {
		t.Fatalf("SwitchAll() error: %v", err)
	}

	reloaded, _ := config.Load(cfgPath)
	if reloaded.Agents["claude"].Current != "anthropic" {
		t.Errorf("claude current = %q", reloaded.Agents["claude"].Current)
	}
	if reloaded.Agents["codex"].Current != "anthropic" {
		t.Errorf("codex current = %q", reloaded.Agents["codex"].Current)
	}
}

func TestSwitchAllUnknownProvider(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.SwitchAll("nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestImport(t *testing.T) {
	cfgPath, claudeDir := setupTestEnv(t)

	// Write a live claude config to import from
	os.MkdirAll(claudeDir, 0o755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"sk-imported","ANTHROPIC_MODEL":"claude-imported"}}`), 0o644)

	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents:    map[string]*config.AgentConfig{},
	}
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	results, err := engine.Import([]string{"claude"})
	if err != nil {
		t.Fatalf("Import() error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Agent != "claude" {
		t.Errorf("Agent = %q", r.Agent)
	}
	if r.Provider != "claude-imported" {
		t.Errorf("Provider = %q", r.Provider)
	}
	if r.EnvVar != "IMPORTED_CLAUDE_API_KEY" {
		t.Errorf("EnvVar = %q", r.EnvVar)
	}

	// Verify the YAML has env-var placeholder, NOT plaintext key
	reloaded, _ := config.Load(cfgPath)
	p := reloaded.Providers["claude-imported"]
	if p == nil {
		t.Fatal("imported provider not found in config")
	}
	if p.APIKey != "${IMPORTED_CLAUDE_API_KEY}" {
		t.Errorf("APIKey should be env-var placeholder, got %q", p.APIKey)
	}
	if reloaded.Agents["claude"].Current != "claude-imported" {
		t.Errorf("claude current = %q", reloaded.Agents["claude"].Current)
	}
	if reloaded.Agents["claude"].Model != "claude-imported" {
		t.Errorf("claude model = %q", reloaded.Agents["claude"].Model)
	}
}

func TestImportUnknownAgent(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents:    map[string]*config.AgentConfig{},
	}
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	_, err := engine.Import([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestImportNoLiveConfig(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := &config.Config{
		Providers: map[string]*config.Provider{},
		Agents:    map[string]*config.AgentConfig{},
	}
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	_, err := engine.Import([]string{"claude"})
	if err == nil {
		t.Error("expected error when no live config exists")
	}
}

func TestNewEngine(t *testing.T) {
	t.Setenv("PRISM_TEST_OR_KEY", "sk-test")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine, err := NewEngine(cfgPath)
	if err != nil {
		t.Fatalf("NewEngine() error: %v", err)
	}
	if engine.Config() == nil {
		t.Error("Config() should not be nil")
	}
}

func TestNewEngineMissingFile(t *testing.T) {
	_, err := NewEngine("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestConfigAccessor(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	engine := NewEngineWithConfig(cfg, cfgPath)
	if engine.Config() != cfg {
		t.Error("Config() should return the same config pointer")
	}
}

func TestSwitchRollbackOnSyncFailure(t *testing.T) {
	// PRISM_TEST_ANT_KEY is NOT set, so switching to anthropic will fail at Resolve
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	err := engine.Switch("claude", "anthropic")
	if err == nil {
		t.Fatal("expected error when env var is missing")
	}

	// YAML should NOT have been updated — current should still be openrouter
	reloaded, _ := config.Load(cfgPath)
	if reloaded.Agents["claude"].Current != "openrouter" {
		t.Errorf("claude current = %q, want openrouter (rollback failed)", reloaded.Agents["claude"].Current)
	}
}

func TestSwitchAllRollbackOnSyncFailure(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	// anthropic env var not set → sync will fail
	err := engine.SwitchAll("anthropic")
	if err == nil {
		t.Fatal("expected error when env var is missing")
	}

	reloaded, _ := config.Load(cfgPath)
	if reloaded.Agents["claude"].Current != "openrouter" {
		t.Errorf("claude current = %q, want openrouter (rollback failed)", reloaded.Agents["claude"].Current)
	}
	if reloaded.Agents["codex"].Current != "openrouter" {
		t.Errorf("codex current = %q, want openrouter (rollback failed)", reloaded.Agents["codex"].Current)
	}
}

func TestDryRun(t *testing.T) {
	t.Setenv("PRISM_TEST_OR_KEY", "sk-or-dryrun")

	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	entries, err := engine.DryRun(nil)
	if err != nil {
		t.Fatalf("DryRun() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify no files were written
	for _, e := range entries {
		for _, p := range e.ConfigPaths {
			if _, err := os.Stat(p); err == nil {
				t.Errorf("DryRun should not create files, but %s exists", p)
			}
		}
	}

	// Verify content
	found := false
	for _, e := range entries {
		if e.Agent == "claude" {
			found = true
			if e.Provider != "openrouter" {
				t.Errorf("claude provider = %q", e.Provider)
			}
			if e.APIKeyMask != "sk-o***yrun" {
				t.Errorf("claude mask = %q", e.APIKeyMask)
			}
			if len(e.ConfigPaths) == 0 {
				t.Error("claude should have config paths")
			}
		}
	}
	if !found {
		t.Error("claude not found in dry run entries")
	}
}

func TestDryRunUnknownAgent(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	_, err := engine.DryRun([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestDryRunMissingEnvVar(t *testing.T) {
	cfgPath, _ := setupTestEnv(t)
	cfg := testConfig()
	config.Save(cfgPath, cfg)

	engine := NewEngineWithConfig(cfg, cfgPath)
	_, err := engine.DryRun(nil)
	if err == nil {
		t.Error("expected error for missing env var")
	}
}
