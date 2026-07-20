# prism-switch

**One provider config, every AI agent.**

A cross-platform CLI to sync and switch API providers across Claude Code, Codex CLI, Gemini CLI and more — from a single declarative YAML file.

## Why

Each AI coding agent has its own config format and location:

| Agent | Config files |
|-------|-------------|
| Claude Code | `~/.claude/settings.json` |
| Codex CLI | `~/.codex/auth.json` + `~/.codex/config.toml` |
| Gemini CLI | `~/.gemini/.env` + `~/.gemini/settings.json` |

Managing API keys and endpoints across all of them is tedious. **prism-switch** lets you define providers once and project them everywhere.

## Quick Start

```bash
# Install
go install github.com/chiga0/prism-switch/cmd/prism@latest

# Create config
mkdir -p ~/.prism
cat > ~/.prism/config.yaml << 'EOF'
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}

agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  codex:
    current: openrouter
    model: o3
  gemini:
    current: anthropic
    model: gemini-2.5-pro
EOF

# Set your keys as env vars (never in the YAML)
export OPENROUTER_API_KEY=sk-or-v1-...
export ANTHROPIC_API_KEY=sk-ant-...

# Sync to all agents
prism sync

# Switch one agent
prism switch claude anthropic

# Switch ALL agents to one provider
prism switch --all openrouter

# Check status + drift detection
prism status
```

## Commands

```
prism sync [agent...]              Project current provider to live config files
prism switch <agent> <provider>    Switch one agent's provider and sync
prism switch --all <provider>      Switch all agents to one provider
prism status [agent...]            Show provider state and drift detection
prism import [agent...]            Backfill from live configs into YAML
prism validate                     Check config structure and env vars
```

Global flag: `-c, --config <path>` to use a custom config file (default: `~/.prism/config.yaml`).

## Config Format

See [docs/config-format.md](docs/config-format.md) for the full specification.

### Key design decisions

- **`${ENV_VAR}` references only** — API keys are never stored in plaintext. The YAML is safe to commit to your dotfiles repo.
- **No database** — the YAML file is the single source of truth.
- **Atomic writes** — all config file writes use tmp + rename to prevent corruption.
- **Preserves existing fields** — syncing Claude's `settings.json` won't wipe your `permissions` block.

## Security

- API keys use `${ENV_VAR}` references, expanded only at sync time
- `prism status` masks keys: `sk-o***3456`
- Config file is written with `0600` permissions
- `prism import` creates `${IMPORTED_<AGENT>_API_KEY}` placeholders — never writes plaintext keys back to YAML
- Error messages never include actual key values

## Drift Detection

`prism status` compares your YAML (desired state) against live config files:

| State | Meaning |
|-------|---------|
| `synced` | Live config matches YAML |
| `drifted` | Live config was modified outside prism (e.g. you changed model in the agent) |
| `missing` | Live config file doesn't exist yet |
| `error` | Can't read config or env var not set |

## Adding a New Agent

Implement the `Projector` interface in one new file:

```go
type Projector interface {
    Name() string                                // "myagent"
    DisplayName() string                         // "My Agent"
    ConfigPaths() []string                       // live config file paths
    Project(p *config.ResolvedProvider) error    // write live config
    ReadLive() (*config.ResolvedProvider, error) // read live config back
}
```

Register it in `internal/cli/root.go`:

```go
agent.Register(agent.NewMyAgentProjector())
```

No changes needed to config, sync, or CLI layers.

## Development

```bash
make build     # Build binary
make test      # Run tests with race detector
make cover     # Generate coverage report
make lint      # Run go vet
```

## Architecture

```
cmd/prism/main.go
  └── internal/cli/          # Cobra commands
        └── internal/sync/   # Sync engine (sync/switch/import/status)
              ├── internal/config/  # YAML parsing + env expansion + validation
              └── internal/agent/   # Per-agent projectors (Claude/Codex/Gemini)
```

See [docs/design.md](docs/design.md) for the full design document.

## License

MIT
