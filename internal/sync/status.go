package sync

import (
	"fmt"

	"github.com/chiga0/prism-switch/internal/agent"
	"github.com/chiga0/prism-switch/internal/config"
)

// AgentStatus reports the state of one agent's live config vs desired config.
type AgentStatus struct {
	Agent       string
	Provider    string
	Model       string
	APIKeyMask  string
	State       string // synced, drifted, missing, error
	Detail      string
	ConfigPaths []string
}

// MaskKey masks an API key for safe display: "sk-or-v1-abc...xyz" → "sk-o...xyz".
func MaskKey(key string) string {
	if key == "" {
		return "(empty)"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

// Status checks all agents (or the given subset) and returns their states.
func (e *Engine) Status(agentNames []string) ([]AgentStatus, error) {
	if len(agentNames) == 0 {
		agentNames = e.agentNames()
	}

	var statuses []AgentStatus
	for _, name := range agentNames {
		statuses = append(statuses, e.statusOne(name))
	}
	return statuses, nil
}

func (e *Engine) statusOne(agentName string) AgentStatus {
	s := AgentStatus{Agent: agentName, State: "error"}

	agentCfg, ok := e.cfg.Agents[agentName]
	if !ok {
		s.Detail = "not in config"
		return s
	}
	s.Provider = agentCfg.Current
	s.Model = agentCfg.Model

	provider, ok := e.cfg.Providers[agentCfg.Current]
	if !ok {
		s.Detail = fmt.Sprintf("provider %q not found", agentCfg.Current)
		return s
	}

	proj, err := agent.Get(agentName)
	if err != nil {
		s.Detail = err.Error()
		return s
	}
	s.ConfigPaths = proj.ConfigPaths()

	resolved, err := config.Resolve(agentCfg.Current, provider, agentCfg, proj.Protocol())
	if err != nil {
		s.Detail = err.Error()
		return s
	}
	s.APIKeyMask = MaskKey(resolved.APIKey)

	live, err := proj.ReadLive()
	if err != nil {
		s.State = "missing"
		s.Detail = "live config not found"
		return s
	}

	if live.APIKey == resolved.APIKey && live.Model == resolved.Model && live.BaseURL == resolved.BaseURL {
		s.State = "synced"
	} else {
		s.State = "drifted"
		s.Detail = "live config differs from desired"
	}

	return s
}
