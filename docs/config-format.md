# 配置格式规范

## 文件位置

默认：`~/.prism/config.yaml`

覆盖：`prism --config /path/to/config.yaml <command>`

## 结构

```yaml
providers:
  <name>:
    api_key: <string>     # 必填。必须使用 ${ENV_VAR} 引用。
    base_url: <string>    # 可选。自定义 API 端点。

agents:
  <agent-name>:
    current: <string>     # 必填。引用一个 provider 名称。
    model: <string>       # 可选。模型标识符。
```

## Providers

Provider 定义共享的 API 凭据。定义一次，所有 Agent 复用。

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    # 不填 base_url → 使用 Agent 默认端点
```

### 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `api_key` | string | ✅ | API Key。**必须**使用 `${ENV_VAR}` 语法，运行时展开。 |
| `base_url` | string | ❌ | 自定义 API 端点。不填则使用 Agent 默认端点。 |

### 环境变量语法

- 格式：`${VAR_NAME}`，在 `prism sync` 时通过 `os.LookupEnv` 展开
- 变量名必须匹配 `[A-Za-z_][A-Za-z0-9_]*`
- 引用的变量未设置时，`prism sync` 报错并列出所有缺失变量
- 运行 `prism validate` 可预检所有引用

## Agents

每个 Agent 条目引用一个 Provider，并可选设置模型。

```yaml
agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
  qwen-code:
    current: anthropic
    model: qwen3-coder
```

### 支持的 Agent

| Agent 名称 | 显示名 | 配置文件 |
|------------|--------|---------|
| `claude` | Claude Code | `~/.claude/settings.json` |
| `codex` | Codex CLI | `~/.codex/auth.json` + `~/.codex/config.toml` |
| `gemini` | Gemini CLI | `~/.gemini/.env` + `~/.gemini/settings.json` |
| `opencode` | OpenCode | `~/.config/opencode/opencode.json` |
| `qwen-code` | Qwen Code | `~/.qwen/settings.json` |

### 投影映射

| 通用字段 | Claude Code | Codex CLI | Gemini CLI | OpenCode | Qwen Code |
|---------|-------------|-----------|------------|----------|-----------|
| `api_key` | `env.ANTHROPIC_AUTH_TOKEN` | `OPENAI_API_KEY` (auth.json) | `GEMINI_API_KEY` (.env) | `provider.prism.options.apiKey` | `env.QWEN_API_KEY` |
| `base_url` | `env.ANTHROPIC_BASE_URL` | `api_base_url` (config.toml) | `GOOGLE_GEMINI_BASE_URL` (.env) | `provider.prism.options.baseURL` | `env.QWEN_BASE_URL` |
| `model` | `env.ANTHROPIC_MODEL` | `model` (config.toml) | `model` (settings.json) | `model` (顶层) | `model` (顶层) |

## 完整示例

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
    base_url: https://ai-proxy.company.com/v1

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
  opencode:
    current: internal
    model: deepseek/deepseek-v4-pro
  qwen-code:
    current: internal
    model: qwen3-coder
```

## Import 行为

`prism import` 从 Agent 的 live 配置反向读取，创建 Provider 条目：

```yaml
# 执行 prism import claude 后：
providers:
  claude-imported:
    api_key: ${IMPORTED_CLAUDE_API_KEY}   # 你需要自己设置这个环境变量
    base_url: https://...                  # 从 live 配置复制

agents:
  claude:
    current: claude-imported
    model: claude-sonnet-4                 # 从 live 配置复制
```

明文 API Key **永远不会**写入 YAML。

## 写入行为

- **保留已有字段**：同步 Claude 的 `settings.json` 不会清除 `permissions`、`theme` 等
- **原子写入**：tmp 文件 + rename，写入中断不会损坏配置
- **损坏保护**：已有文件解析失败时，自动备份为 `*.prism-backup-<timestamp>` 并警告
- **文件权限**：config.yaml 为 `0600`，Agent 配置文件为 `0644`（.env 为 `0600`）
