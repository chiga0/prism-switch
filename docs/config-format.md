# Configuration Format

Prism-switch uses a single YAML file at `~/.prism/config.yaml` (override with `--config`).

## Structure

```yaml
providers:
  <name>:
    api_key: <string>     # Required. Use ${ENV_VAR} references.
    base_url: <string>    # Optional. Custom API endpoint.

agents:
  <agent-name>:
    current: <string>     # Required. Must reference a provider name.
    model: <string>       # Optional. Model identifier for this agent.
```

## Providers

Providers define shared API credentials. Define once, use across all agents.

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    # no base_url → uses the agent's default endpoint
  google:
    api_key: ${GEMINI_API_KEY}
```

### Field reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_key` | string | ✅ | API key. **Must** use `${ENV_VAR}` syntax. |
| `base_url` | string | ❌ | Custom API endpoint. Omit to use the agent's default. |

### Environment variable syntax

- Format: `${VAR_NAME}` — expanded at sync time via `os.LookupEnv`
- Variable names must match `[A-Za-z_][A-Za-z0-9_]*`
- If a referenced variable is not set, `prism sync` fails with an error listing all missing variables
- Run `prism validate` to check all references before syncing

## Agents

Each agent entry references a provider and optionally sets a model.

```yaml
agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  codex:
    current: openrouter
    model: o3
  gemini:
    current: google
    model: gemini-2.5-pro
```

### Supported agents

| Agent name | Display name | Live config files |
|------------|-------------|-------------------|
| `claude` | Claude Code | `~/.claude/settings.json` |
| `codex` | Codex CLI | `~/.codex/auth.json`, `~/.codex/config.toml` |
| `gemini` | Gemini CLI | `~/.gemini/.env`, `~/.gemini/settings.json` |

### Projection mapping

| Generic field | Claude Code | Codex CLI | Gemini CLI |
|--------------|-------------|-----------|------------|
| `api_key` | `env.ANTHROPIC_AUTH_TOKEN` | `OPENAI_API_KEY` in auth.json | `GEMINI_API_KEY` in .env |
| `base_url` | `env.ANTHROPIC_BASE_URL` | — | `GOOGLE_GEMINI_BASE_URL` in .env |
| `model` | `env.ANTHROPIC_MODEL` | `model` in config.toml | `model` in settings.json |

## Complete example

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
  google:
    api_key: ${GEMINI_API_KEY}
  internal:
    api_key: ${INTERNAL_PROXY_KEY}
    base_url: https://ai-proxy.internal.com/v1

agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  codex:
    current: openrouter
    model: o3
  gemini:
    current: google
    model: gemini-2.5-pro
```

## Import behavior

`prism import` reads live agent configs and creates provider entries with env-var placeholders:

```yaml
# After: prism import claude
providers:
  claude-imported:
    api_key: ${IMPORTED_CLAUDE_API_KEY}   # You must set this env var
    base_url: https://...                  # Copied from live config

agents:
  claude:
    current: claude-imported
    model: claude-sonnet-4                 # Copied from live config
```

Plaintext API keys are **never** written to the YAML.
