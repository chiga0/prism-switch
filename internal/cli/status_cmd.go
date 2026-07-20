package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var statusCmd = &cobra.Command{
	Use:   "status [agent...]",
	Short: "Show current provider state and drift detection",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}
		statuses, err := engine.Status(args)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "AGENT\tPROVIDER\tMODEL\tAPI KEY\tSTATE\tCONFIG FILES")
		for _, s := range statuses {
			model := s.Model
			if model == "" {
				model = "-"
			}
			paths := "-"
			if len(s.ConfigPaths) > 0 {
				paths = strings.Join(s.ConfigPaths, ", ")
			}
			state := s.State
			if s.Detail != "" {
				state = fmt.Sprintf("%s (%s)", s.State, s.Detail)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				s.Agent, s.Provider, model, s.APIKeyMask, state, paths)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
