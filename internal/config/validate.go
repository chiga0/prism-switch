package config

import (
	"fmt"
	"os"
)

// Validate checks the config for structural errors.
func Validate(cfg *Config) []error {
	var errs []error

	if len(cfg.Providers) == 0 {
		errs = append(errs, fmt.Errorf("no providers defined"))
	}

	for name, p := range cfg.Providers {
		if p.APIKey == "" {
			errs = append(errs, fmt.Errorf("provider %q: api_key is required", name))
		}
	}

	for agentName, ac := range cfg.Agents {
		if ac.Current == "" {
			errs = append(errs, fmt.Errorf("agent %q: current provider is required", agentName))
			continue
		}
		if _, ok := cfg.Providers[ac.Current]; !ok {
			errs = append(errs, fmt.Errorf("agent %q: references unknown provider %q", agentName, ac.Current))
		}
	}

	return errs
}

// CheckEnvVars verifies that all referenced environment variables are set.
func CheckEnvVars(cfg *Config) []error {
	var errs []error
	seen := make(map[string]bool)

	for name, p := range cfg.Providers {
		for _, field := range []string{p.APIKey, p.BaseURL} {
			matches := envVarRe.FindAllStringSubmatch(field, -1)
			for _, m := range matches {
				varName := m[1]
				if seen[varName] {
					continue
				}
				seen[varName] = true
				if _, ok := os.LookupEnv(varName); !ok {
					errs = append(errs, fmt.Errorf("provider %q: environment variable %s is not set", name, varName))
				}
			}
		}
	}

	return errs
}
