# prism-switch

**One provider config, every AI agent.**

一个 YAML 文件管理所有 AI Coding Agent 的 API Provider。定义一次，一键同步到 Claude Code、Codex CLI、Gemini CLI、OpenCode、Qwen Code。

## 安装

```bash
# macOS (Homebrew)
brew install chiga0/tap/prism

# Go install
go install github.com/chiga0/prism-switch/cmd/prism@latest

# 或从 GitHub Releases 下载二进制
# https://github.com/chiga0/prism-switch/releases
```

## 30 秒上手

```bash
# 1. 生成配置文件
prism init

# 2. 设置你的 API Key（环境变量，不写入文件）
export OPENROUTER_API_KEY=sk-or-v1-...

# 3. 一键同步到所有 Agent
prism sync
```

完事。Claude Code、Codex、Gemini CLI、OpenCode、Qwen Code 全部配好了。

## 日常使用

```bash
# 切换某个 Agent 的 Provider
prism switch claude anthropic

# 一键切换所有 Agent 到同一个 Provider
prism switch --all openrouter

# 查看当前状态（谁用了什么 Provider，是否 drift）
prism status

# 预览同步会写什么（不实际修改文件）
prism sync --dry-run

# 从 Agent 现有配置反向导入到 YAML
prism import claude
```

## 配置文件

位置：`~/.prism/config.yaml`（用 `--config` 指定其他路径）

### 最简配置

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1

agents:
  claude:
    current: openrouter
    model: anthropic/claude-sonnet-4
```

### 多 Provider + 多 Agent

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

### 配置字段说明

| 字段 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `api_key` | providers.\<name\> | ✅ | API Key，**必须**用 `${ENV_VAR}` 引用 |
| `base_url` | providers.\<name\> | ❌ | 自定义端点，不填则用 Agent 默认 |
| `current` | agents.\<name\> | ✅ | 当前使用的 Provider 名称 |
| `model` | agents.\<name\> | ❌ | 模型标识符 |

## 支持的 Agent

| Agent | 配置文件 | 投影方式 |
|-------|---------|---------|
| Claude Code | `~/.claude/settings.json` | `env.ANTHROPIC_AUTH_TOKEN` / `ANTHROPIC_BASE_URL` / `ANTHROPIC_MODEL` |
| Codex CLI | `~/.codex/auth.json` + `config.toml` | `OPENAI_API_KEY` + `model` + `api_base_url` |
| Gemini CLI | `~/.gemini/.env` + `settings.json` | `GEMINI_API_KEY` / `GOOGLE_GEMINI_BASE_URL` + `model` |
| OpenCode | `~/.config/opencode/opencode.json` | `provider.prism.options.{apiKey,baseURL}` + `model` |
| Qwen Code | `~/.qwen/settings.json` | `env.QWEN_API_KEY` / `QWEN_BASE_URL` + `model` |

## 命令参考

```
prism init                     创建 starter 配置文件
prism sync [agent...]          同步到所有（或指定）Agent
prism sync --dry-run           预览，不修改文件
prism switch <agent> <provider>  切换某个 Agent
prism switch --all <provider>  切换所有 Agent
prism status [agent...]        查看状态 + drift 检测
prism import [agent...]        从 live 配置反向导入
prism validate                 校验配置 + 检查环境变量
```

全局 flag：`-c, --config <path>` 指定配置文件路径。

## 安全设计

- **密钥不落盘**：config.yaml 里只有 `${ENV_VAR}` 引用，可安全提交到 git
- **输出脱敏**：`prism status` 显示 `sk-o***9999`，不暴露完整 key
- **文件权限**：config.yaml 写入时 `0600`，仅本人可读
- **import 安全**：反向导入时生成 `${IMPORTED_<AGENT>_API_KEY}` 占位符，不写明文
- **原子写入**：所有配置文件用 tmp + rename，写入中断不会损坏
- **损坏保护**：已有配置文件损坏时自动备份为 `*.prism-backup-<timestamp>` 再覆盖

## Drift 检测

`prism status` 对比 YAML（期望状态）和 live 配置文件（实际状态）：

| 状态 | 含义 |
|------|------|
| `synced` | 一致 |
| `drifted` | live 被手动改过（如在 Agent 内切了 model） |
| `missing` | live 文件不存在 |
| `error` | 环境变量未设置或读取失败 |

## 高阶配置

### Shell 补全

```bash
# Bash
prism completion bash > ~/.bash_completion.d/prism

# Zsh
prism completion zsh > "${fpath[1]}/_prism"

# Fish
prism completion fish > ~/.config/fish/completions/prism.fish
```

### 多设备同步

config.yaml 是纯文本，用 git/dotfiles 工具同步即可：

```bash
# chezmoi
chezmoi add ~/.prism/config.yaml

# 或 symlink 到 dotfiles 仓库
ln -s ~/dotfiles/prism/config.yaml ~/.prism/config.yaml
```

每台设备设置自己的环境变量（key 不在文件里），`prism sync` 即可。

### 自定义 Agent 配置路径

如果 Agent 配置不在默认位置，可以用 `--config` 配合不同的 YAML 文件管理不同环境：

```bash
prism --config ~/.prism/work.yaml sync      # 工作用 Provider
prism --config ~/.prism/personal.yaml sync  # 个人用 Provider
```

### CI/CD 集成

```bash
# 在 CI 中验证配置合法性
prism validate --config .prism/config.yaml

# 预览同步结果（JSON 输出，方便脚本解析）
prism status --json
```

## 开发

```bash
make build     # 构建
make test      # 测试（race detector）
make cover     # 覆盖率报告
make lint      # go vet
```

架构详见 [docs/design.md](docs/design.md)，配置格式详见 [docs/config-format.md](docs/config-format.md)。

## License

MIT
