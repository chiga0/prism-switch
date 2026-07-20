package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chiga0/prism-switch/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config structure and check env vars",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(resolveCfgPath())
		if err != nil {
			return err
		}

		structErrs := config.Validate(cfg)
		envErrs := config.CheckEnvVars(cfg)

		if len(structErrs) == 0 && len(envErrs) == 0 {
			fmt.Println("✓ Config is valid, all env vars set")
			return nil
		}

		for _, e := range structErrs {
			fmt.Printf("✗ %v\n", e)
		}
		for _, e := range envErrs {
			fmt.Printf("✗ %v\n", e)
		}
		return fmt.Errorf("validation failed with %d error(s)", len(structErrs)+len(envErrs))
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
