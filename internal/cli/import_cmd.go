package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var importCmd = &cobra.Command{
	Use:   "import [agent...]",
	Short: "Backfill: read live agent configs into YAML",
	Long: `Import reads each agent's live configuration file and creates a provider
entry in the YAML config. API keys are stored as ${IMPORTED_<AGENT>_API_KEY}
env-var placeholders — you must set the env var yourself. Plaintext keys are
never written to the YAML.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}
		results, err := engine.Import(args)
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Printf("✓ %s → provider %q\n", r.Agent, r.Provider)
			fmt.Printf("  Set the env var: export %s=<your-api-key>\n", r.EnvVar)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
