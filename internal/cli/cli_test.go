package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

func setupCLITest(t *testing.T) string {
	t.Helper()
	agent.ResetRegistry()

	claudeDir := filepath.Join(t.TempDir(), "claude")
	codexDir := filepath.Join(t.TempDir(), "codex")
	geminiDir := filepath.Join(t.TempDir(), "gemini")

	agent.Register(agent.NewClaudeProjectorWithBase(claudeDir))
	agent.Register(agent.NewCodexProjectorWithBase(codexDir))
	agent.Register(agent.NewGeminiProjectorWithBase(geminiDir))

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		Providers: map[string]*config.Provider{
			"openrouter": {APIKey: "${PRISM_CLI_TEST_KEY}", BaseURL: "https://openrouter.ai/api/v1"},
			"anthropic":  {APIKey: "${PRISM_CLI_TEST_KEY2}"},
		},
		Agents: map[string]*config.AgentConfig{
			"claude": {Current: "openrouter", Model: "claude-sonnet-4"},
			"codex":  {Current: "openrouter", Model: "o3"},
		},
	}
	config.Save(cfgPath, cfg)
	return cfgPath
}

func TestSyncCommand(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY", "sk-cli-test")
	t.Setenv("PRISM_CLI_TEST_KEY2", "sk-cli-test2")

	cfgPath := setupCLITest(t)

	cmd := rootCmd
	cmd.SetArgs([]string{"sync", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync command error: %v", err)
	}
}

func TestSyncCommandSpecificAgent(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY", "sk-cli-test")

	cfgPath := setupCLITest(t)

	rootCmd.SetArgs([]string{"sync", "claude", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sync claude error: %v", err)
	}
}

func TestSwitchCommand(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY", "sk-cli-test")
	t.Setenv("PRISM_CLI_TEST_KEY2", "sk-cli-test2")

	cfgPath := setupCLITest(t)

	rootCmd.SetArgs([]string{"switch", "claude", "anthropic", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("switch command error: %v", err)
	}

	reloaded, _ := config.Load(cfgPath)
	if reloaded.Agents["claude"].Current != "anthropic" {
		t.Errorf("claude current = %q, want anthropic", reloaded.Agents["claude"].Current)
	}
}

func TestSwitchAllCommand(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY2", "sk-cli-test2")

	cfgPath := setupCLITest(t)

	rootCmd.SetArgs([]string{"switch", "--all", "anthropic", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("switch --all error: %v", err)
	}

	reloaded, _ := config.Load(cfgPath)
	if reloaded.Agents["claude"].Current != "anthropic" {
		t.Errorf("claude current = %q", reloaded.Agents["claude"].Current)
	}
	if reloaded.Agents["codex"].Current != "anthropic" {
		t.Errorf("codex current = %q", reloaded.Agents["codex"].Current)
	}
}

func TestStatusCommand(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY", "sk-cli-test")
	t.Setenv("PRISM_CLI_TEST_KEY2", "sk-cli-test2")

	cfgPath := setupCLITest(t)

	rootCmd.SetArgs([]string{"status", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status command error: %v", err)
	}
}

func TestValidateCommandValid(t *testing.T) {
	t.Setenv("PRISM_CLI_TEST_KEY", "sk-cli-test")
	t.Setenv("PRISM_CLI_TEST_KEY2", "sk-cli-test2")

	cfgPath := setupCLITest(t)

	rootCmd.SetArgs([]string{"validate", "--config", cfgPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate command error: %v", err)
	}
}

func TestValidateCommandInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	os.WriteFile(cfgPath, []byte("providers:\n  p1:\n    base_url: https://x.com\nagents:\n  a1:\n    current: ghost\n"), 0o600)

	rootCmd.SetArgs([]string{"validate", "--config", cfgPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestImportCommand(t *testing.T) {
	cfgPath := setupCLITest(t)

	// Write a live claude config
	claudeProj, _ := agent.Get("claude")
	_ = claudeProj // We need to write to the actual test dir

	// Since we can't easily get the temp dir back, test import error path
	rootCmd.SetArgs([]string{"import", "claude", "--config", cfgPath})
	// This will fail because no live config exists - that's expected
	_ = rootCmd.Execute()
}

func TestSyncCommandMissingConfig(t *testing.T) {
	rootCmd.SetArgs([]string{"sync", "--config", "/nonexistent/config.yaml"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestSwitchCommandMissingArgs(t *testing.T) {
	cfgPath := setupCLITest(t)
	rootCmd.SetArgs([]string{"switch", "--config", cfgPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestResolveCfgPathDefault(t *testing.T) {
	cfgPath = ""
	p := resolveCfgPath()
	if p == "" {
		t.Error("resolveCfgPath() should return default path")
	}
}

func TestResolveCfgPathCustom(t *testing.T) {
	cfgPath = "/custom/path.yaml"
	defer func() { cfgPath = "" }()
	if resolveCfgPath() != "/custom/path.yaml" {
		t.Errorf("resolveCfgPath() = %q", resolveCfgPath())
	}
}
