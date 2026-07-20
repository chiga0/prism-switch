package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues with your prism setup",
	Long: `Doctor checks:
  - Config file exists and is valid
  - Referenced environment variables are set
  - Agent CLI tools are installed
  - Config file permissions are correct
  - Agent config directories are writable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		issues := 0
		path := resolveCfgPath()

		// 1. Config file exists
		fmt.Print("Config file... ")
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("✗ not found at %s (run 'prism init')\n", path)
			issues++
		} else {
			fmt.Printf("✓ %s\n", path)
			// Check permissions
			fmt.Print("Config permissions... ")
			if info.Mode().Perm() == 0o600 {
				fmt.Println("✓ 0600")
			} else {
				fmt.Printf("⚠ %o (should be 0600, run: chmod 600 %s)\n", info.Mode().Perm(), path)
				issues++
			}
		}

		// 2. Load and validate
		cfg, err := config.Load(path)
		if err != nil {
			fmt.Printf("Config parse... ✗ %v\n", err)
			issues++
			return summary(issues)
		}
		fmt.Print("Config parse... ✓\n")

		structErrs := config.Validate(cfg)
		fmt.Print("Config structure... ")
		if len(structErrs) == 0 {
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ %d error(s)\n", len(structErrs))
			for _, e := range structErrs {
				fmt.Printf("    • %v\n", e)
			}
			issues += len(structErrs)
		}

		// 3. Environment variables
		envErrs := config.CheckEnvVars(cfg)
		fmt.Print("Environment variables... ")
		if len(envErrs) == 0 {
			fmt.Println("✓ all set")
		} else {
			fmt.Printf("✗ %d missing\n", len(envErrs))
			for _, e := range envErrs {
				fmt.Printf("    • %v\n", e)
			}
			issues += len(envErrs)
		}

		// 4. Agent CLIs installed
		fmt.Println("Agent CLIs:")
		cliNames := map[string][]string{
			"claude":    {"claude"},
			"codex":     {"codex"},
			"gemini":    {"gemini"},
			"opencode":  {"opencode"},
			"qwen-code": {"qwen", "qwen-code"},
			"zcode":     {"zcode"},
		}
		for _, name := range agent.AvailableNamesList() {
			if _, inCfg := cfg.Agents[name]; !inCfg {
				continue
			}
			candidates := cliNames[name]
			found := ""
			for _, bin := range candidates {
				if p, err := exec.LookPath(bin); err == nil {
					found = p
					break
				}
			}
			if found != "" {
				fmt.Printf("    ✓ %-10s → %s\n", name, found)
			} else {
				fmt.Printf("    ⚠ %-10s not installed (tried: %s)\n", name, strings.Join(candidates, ", "))
			}
		}

		// 5. Agent config dirs writable
		fmt.Println("Agent config dirs:")
		for _, name := range agent.AvailableNamesList() {
			if _, inCfg := cfg.Agents[name]; !inCfg {
				continue
			}
			proj, err := agent.Get(name)
			if err != nil {
				continue
			}
			paths := proj.ConfigPaths()
			if len(paths) == 0 {
				continue
			}
			dir := paths[0]
			// Check parent dir
			parent := dir[:strings.LastIndex(dir, "/")]
			if info, err := os.Stat(parent); err == nil && info.IsDir() {
				fmt.Printf("    ✓ %-10s %s\n", name, parent)
			} else {
				fmt.Printf("    ⚠ %-10s %s (will be created on sync)\n", name, parent)
			}
		}

		return summary(issues)
	},
}

func summary(issues int) error {
	fmt.Println()
	if issues == 0 {
		fmt.Println("✓ All checks passed. Run 'prism sync' to project your config.")
	} else {
		fmt.Printf("✗ %d issue(s) found. Fix them and re-run 'prism doctor'.\n", issues)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
