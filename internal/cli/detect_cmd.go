package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

// agentBinaries maps agent names to possible binary names for detection.
var agentBinaries = map[string][]string{
	"claude":    {"claude"},
	"codex":     {"codex"},
	"gemini":    {"gemini"},
	"opencode":  {"opencode"},
	"qwen-code": {"qwen", "qwen-code"},
	"zcode":     {"zcode"},
}

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect installed agent CLIs and generate a config",
	Long: `Detect scans your PATH for installed AI coding agent CLIs and generates
a starter config containing only the agents found on this machine.

If a config already exists, it prints what would be added without overwriting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var detected []string
		for _, name := range agent.AvailableNamesList() {
			bins := agentBinaries[name]
			for _, bin := range bins {
				if _, err := exec.LookPath(bin); err == nil {
					detected = append(detected, name)
					break
				}
			}
		}

		if len(detected) == 0 {
			fmt.Println("No supported agent CLIs found in PATH.")
			fmt.Println("Install one of: claude, codex, gemini, opencode, qwen, zcode")
			return nil
		}

		fmt.Printf("Detected %d agent(s): ", len(detected))
		for i, name := range detected {
			if i > 0 {
				fmt.Print(", ")
			}
			proj, _ := agent.Get(name)
			fmt.Print(proj.DisplayName())
		}
		fmt.Println()

		path := resolveCfgPath()
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("\nConfig already exists at %s — printing detected agents:\n", path)
			for _, name := range detected {
				fmt.Printf("  %s:\n    current: <your-provider>\n", name)
			}
			return nil
		}

		// Generate config with detected agents only
		cfg := &config.Config{
			Providers: map[string]*config.Provider{
				"my-provider": {
					APIKey: "${MY_API_KEY}",
					BaseURLs: map[config.Protocol]string{
						config.ProtocolOpenAI:    "https://your-endpoint/compatible-mode/v1",
						config.ProtocolAnthropic: "https://your-endpoint/anthropic",
					},
				},
			},
			Agents: make(map[string]*config.AgentConfig),
		}
		for _, name := range detected {
			cfg.Agents[name] = &config.AgentConfig{
				Current: "my-provider",
			}
		}

		if err := config.Save(path, cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Printf("\n✓ Created %s with %d agent(s)\n", path, len(detected))
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the config: set your real provider api_key and base_urls")
		fmt.Println("  2. Set env var:     export MY_API_KEY=sk-...")
		fmt.Println("  3. Sync:            prism sync")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
