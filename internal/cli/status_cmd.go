package cli

import (
	"fmt"
	"os"
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
		fmt.Fprintln(w, "AGENT\tPROVIDER\tMODEL\tAPI KEY\tSTATE\tDETAIL")
		for _, s := range statuses {
			detail := s.Detail
			if detail == "" {
				detail = "-"
			}
			model := s.Model
			if model == "" {
				model = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				s.Agent, s.Provider, model, s.APIKeyMask, s.State, detail)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
