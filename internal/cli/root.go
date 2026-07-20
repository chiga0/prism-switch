package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "One provider config, every AI agent",
	Long: `Prism-Switch syncs a single declarative YAML config to all your
AI coding agents (Claude Code, Codex CLI, Gemini CLI) in one command.

Define providers once with ${ENV_VAR} references — never plaintext keys.
Then sync, switch, and audit across every agent from one place.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path (default: ~/.prism/config.yaml)")

	// Register built-in projectors
	agent.Register(agent.NewClaudeProjector())
	agent.Register(agent.NewCodexProjector())
	agent.Register(agent.NewGeminiProjector())
}

func resolveCfgPath() string {
	if cfgPath != "" {
		return cfgPath
	}
	return config.DefaultConfigPath()
}
