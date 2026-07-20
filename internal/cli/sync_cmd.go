package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var dryRun bool
var syncAll bool

var syncCmd = &cobra.Command{
	Use:   "sync [agent...]",
	Short: "Project current provider config to agent live files",
	Long: `Sync reads the YAML config, resolves ${ENV_VAR} references, and writes
each agent's live configuration file.

By default, only agents whose CLI is detected in PATH are synced.
Use --all to sync every agent in the config regardless of installation.
Explicitly naming agents (prism sync claude codex) always syncs them.

Use --dry-run to preview what would be written without touching any files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}

		// Determine which agents to sync
		targets := args
		if len(targets) == 0 && !syncAll {
			// Default: only sync installed agents
			targets = filterInstalled(engine.Config().AgentNames())
			if len(targets) == 0 {
				fmt.Println("No installed agents found in config. Use --all to sync anyway, or name agents explicitly.")
				return nil
			}
			fmt.Printf("Syncing installed agents: %s\n", strings.Join(targets, ", "))
		}

		if dryRun {
			entries, err := engine.DryRun(targets)
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

		if err := engine.Sync(targets); err != nil {
			return err
		}
		if len(targets) == 0 || (len(args) == 0 && syncAll) {
			fmt.Println("✓ All agents synced")
		} else {
			for _, a := range targets {
				fmt.Printf("✓ %s synced\n", a)
			}
		}
		return nil
	},
}

// filterInstalled returns only agent names whose CLI binary is found in PATH.
func filterInstalled(agentNames []string) []string {
	var installed []string
	for _, name := range agentNames {
		bins := agentBinaries[name]
		for _, bin := range bins {
			if _, err := exec.LookPath(bin); err == nil {
				installed = append(installed, name)
				break
			}
		}
	}
	return installed
}

func init() {
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview what would be written without modifying files")
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all agents in config regardless of installation")
	rootCmd.AddCommand(syncCmd)
}
