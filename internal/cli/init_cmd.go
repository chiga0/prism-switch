package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chiga0/prism-switch/internal/config"
)

const defaultConfigTemplate = `# prism-switch configuration
# Define providers once, sync to all agents.
# API keys use ${ENV_VAR} references — never plaintext.
#
# base_urls maps protocol → endpoint. Each agent speaks one protocol:
#   claude, zcode       → anthropic
#   codex, opencode, qwen-code → openai
#   gemini              → google (usually no base_url needed)

providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_urls:
      openai: https://openrouter.ai/api/v1
      anthropic: https://openrouter.ai/api/v1
  # token-plan:
  #   api_key: ${TOKEN_PLAN_API_KEY}
  #   base_urls:
  #     openai: https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1
  #     anthropic: https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic
  # google:
  #   api_key: ${GEMINI_API_KEY}

agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  codex:
    current: openrouter
    model: o3
  gemini:
    current: openrouter
    model: gemini-2.5-pro
  opencode:
    current: openrouter
    model: anthropic/claude-sonnet-4
  qwen-code:
    current: openrouter
    model: qwen3-coder
  zcode:
    current: openrouter
    model: GLM-5.2
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter config file",
	Long: `Init creates a starter config at ~/.prism/config.yaml (or --config path)
with example providers and agent entries. It will not overwrite an existing file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := resolveCfgPath()

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists at %s (use --config to specify a different path)", path)
		}

		if err := os.MkdirAll(config.Dir(path), 0o700); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultConfigTemplate), 0o600); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf("✓ Created %s\n", path)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Set your API key:  export OPENROUTER_API_KEY=sk-or-v1-...")
		fmt.Println("  2. Preview the sync:  prism sync --dry-run")
		fmt.Println("  3. Sync all agents:   prism sync")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
