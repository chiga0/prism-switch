package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync [agent...]",
	Short: "Project current provider config to agent live files",
	Long: `Sync reads the YAML config, resolves ${ENV_VAR} references, and writes
each agent's live configuration file. Omit agent names to sync all agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}
		if err := engine.Sync(args); err != nil {
			return err
		}
		if len(args) == 0 {
			fmt.Println("✓ All agents synced")
		} else {
			for _, a := range args {
				fmt.Printf("✓ %s synced\n", a)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
