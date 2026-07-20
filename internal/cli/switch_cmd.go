package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	psync "github.com/chiga0/prism-switch/internal/sync"
)

var switchAll bool

// fuzzyMatch resolves an abbreviated name against a list of candidates.
// Priority: exact > prefix > substring > subsequence (chars in order).
// Returns error if ambiguous or no match.
func fuzzyMatch(input string, candidates []string) (string, error) {
	// Exact match
	for _, c := range candidates {
		if c == input {
			return c, nil
		}
	}
	// Prefix match
	var prefixMatches []string
	for _, c := range candidates {
		if strings.HasPrefix(c, input) {
			prefixMatches = append(prefixMatches, c)
		}
	}
	if len(prefixMatches) == 1 {
		return prefixMatches[0], nil
	}
	if len(prefixMatches) > 1 {
		return "", fmt.Errorf("ambiguous %q: matches %s", input, strings.Join(prefixMatches, ", "))
	}
	// Substring match
	var subMatches []string
	for _, c := range candidates {
		if strings.Contains(c, input) {
			subMatches = append(subMatches, c)
		}
	}
	if len(subMatches) == 1 {
		return subMatches[0], nil
	}
	if len(subMatches) > 1 {
		return "", fmt.Errorf("ambiguous %q: matches %s", input, strings.Join(subMatches, ", "))
	}
	// Subsequence match (chars appear in order, e.g. "or" → "openrouter")
	var seqMatches []string
	for _, c := range candidates {
		if isSubsequence(input, c) {
			seqMatches = append(seqMatches, c)
		}
	}
	if len(seqMatches) == 1 {
		return seqMatches[0], nil
	}
	if len(seqMatches) > 1 {
		return "", fmt.Errorf("ambiguous %q: matches %s", input, strings.Join(seqMatches, ", "))
	}
	return "", fmt.Errorf("no match for %q (available: %s)", input, strings.Join(candidates, ", "))
}

// isSubsequence checks if all chars of needle appear in haystack in order.
func isSubsequence(needle, haystack string) bool {
	ni := 0
	for hi := 0; hi < len(haystack) && ni < len(needle); hi++ {
		if haystack[hi] == needle[ni] {
			ni++
		}
	}
	return ni == len(needle)
}

var switchCmd = &cobra.Command{
	Use:   "switch <agent> <provider>",
	Short: "Switch an agent's current provider and sync",
	Long: `Switch updates the agent's "current" field in the YAML config,
saves the file, and immediately projects the new provider to the agent's
live config files.

Agent and provider names support fuzzy matching:
  prism switch cl or       → claude + openrouter
  prism switch gem google  → gemini + google

Use --all to switch every agent to the same provider at once.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := psync.NewEngine(resolveCfgPath())
		if err != nil {
			return err
		}
		cfg := engine.Config()

		// Build candidate lists
		agentNames := make([]string, 0, len(cfg.Agents))
		for name := range cfg.Agents {
			agentNames = append(agentNames, name)
		}
		providerNames := make([]string, 0, len(cfg.Providers))
		for name := range cfg.Providers {
			providerNames = append(providerNames, name)
		}

		if switchAll {
			if len(args) != 1 {
				return fmt.Errorf("--all requires exactly 1 argument: <provider>")
			}
			providerName, err := fuzzyMatch(args[0], providerNames)
			if err != nil {
				return err
			}
			if err := engine.SwitchAll(providerName); err != nil {
				return err
			}
			fmt.Printf("✓ All agents switched to %q\n", providerName)
		} else {
			if len(args) != 2 {
				return fmt.Errorf("requires 2 arguments: <agent> <provider>")
			}
			agentName, err := fuzzyMatch(args[0], agentNames)
			if err != nil {
				return err
			}
			providerName, err := fuzzyMatch(args[1], providerNames)
			if err != nil {
				return err
			}
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
