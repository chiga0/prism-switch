package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var dryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync [agent...]",
	Short: "Project current provider config to agent live files",
	Long: `Sync reads the YAML config, resolves ${ENV_VAR} references, and writes
each agent's live configuration file. Omit agent names to sync all agents.

Use --dry-run to preview what would be written without touching any files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}

		if dryRun {
			entries, err := engine.DryRun(args)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "AGENT\tPROVIDER\tMODEL\tAPI KEY\tBASE URL\tCONFIG FILES")
			for _, e := range entries {
				baseURL := e.BaseURL
				if baseURL == "" {
					baseURL = "(default)"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					e.Agent, e.Provider, e.Model, e.APIKeyMask, baseURL,
					strings.Join(e.ConfigPaths, ", "))
			}
			w.Flush()
			fmt.Println("\n(dry run — no files were modified)")
			return nil
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
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview what would be written without modifying files")
	rootCmd.AddCommand(syncCmd)
}
