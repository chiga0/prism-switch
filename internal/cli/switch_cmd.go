package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var switchAll bool

var switchCmd = &cobra.Command{
	Use:   "switch <agent> <provider>",
	Short: "Switch an agent's current provider and sync",
	Long: `Switch updates the agent's "current" field in the YAML config,
saves the file, and immediately projects the new provider to the agent's
live config files.

Use --all to switch every agent to the same provider at once.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}
		if switchAll {
			if len(args) != 1 {
				return fmt.Errorf("--all requires exactly 1 argument: <provider>")
			}
			providerName := args[0]
			if err := engine.SwitchAll(providerName); err != nil {
				return err
			}
			fmt.Printf("✓ All agents switched to %q\n", providerName)
		} else {
			if len(args) != 2 {
				return fmt.Errorf("requires 2 arguments: <agent> <provider>")
			}
			agentName, providerName := args[0], args[1]
			if err := engine.Switch(agentName, providerName); err != nil {
				return err
			}
			fmt.Printf("✓ %s switched to %q\n", agentName, providerName)
		}
		return nil
	},
}

func init() {
	switchCmd.Flags().BoolVar(&switchAll, "all", false, "switch all agents to the given provider")
	rootCmd.AddCommand(switchCmd)
}
