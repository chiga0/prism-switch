package config

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantCount int
	}{
		{
			name:      "empty config",
			cfg:       &Config{Providers: map[string]*Provider{}, Agents: map[string]*AgentConfig{}},
			wantCount: 1, // no providers
		},
		{
			name: "valid config",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${KEY}"},
				},
				Agents: map[string]*AgentConfig{
					"claude": {Current: "p1", Model: "m"},
				},
			},
			wantCount: 0,
		},
		{
			name: "provider missing api_key",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {BaseURL: "https://x.com"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 1,
		},
		{
			name: "agent missing current",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${KEY}"},
				},
				Agents: map[string]*AgentConfig{
					"claude": {Model: "m"},
				},
			},
			wantCount: 1,
		},
		{
			name: "agent references unknown provider",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${KEY}"},
				},
				Agents: map[string]*AgentConfig{
					"claude": {Current: "nonexistent"},
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple errors",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {BaseURL: "https://x.com"}, // missing api_key
				},
				Agents: map[string]*AgentConfig{
					"a1": {Current: "unknown"},  // unknown provider
					"a2": {},                    // missing current
				},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.cfg)
			if len(errs) != tt.wantCount {
				t.Errorf("Validate() returned %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
		})
	}
}

func TestCheckEnvVars(t *testing.T) {
	t.Setenv("PRISM_CHK_EXISTS", "value")

	tests := []struct {
		name      string
		cfg       *Config
		wantCount int
	}{
		{
			name: "all set",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${PRISM_CHK_EXISTS}"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 0,
		},
		{
			name: "missing var",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${PRISM_CHK_MISSING_XYZ}"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 1,
		},
		{
			name: "missing base_url var",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${PRISM_CHK_EXISTS}", BaseURL: "${PRISM_CHK_URL_MISSING}"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 1,
		},
		{
			name: "duplicate var counted once",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "${PRISM_CHK_MISSING_XYZ}"},
					"p2": {APIKey: "${PRISM_CHK_MISSING_XYZ}"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 1,
		},
		{
			name: "no env refs",
			cfg: &Config{
				Providers: map[string]*Provider{
					"p1": {APIKey: "plain-key"},
				},
				Agents: map[string]*AgentConfig{},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := CheckEnvVars(tt.cfg)
			if len(errs) != tt.wantCount {
				t.Errorf("CheckEnvVars() returned %d errors, want %d: %v", len(errs), tt.wantCount, errs)
			}
		})
	}
}
