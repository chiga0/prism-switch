# Prism-Switch 设计文档

## 定位

一个跨平台 CLI 工具，用一套声明式 YAML 配置统一管理多个 AI Coding Agent（Claude Code、Codex、Gemini CLI 等）的 Provider（API Key / Endpoint / Model），一键投影到各 Agent 的原生配置文件。

## 核心差异

cc-switch 是**每个 Agent 各管各的 Provider**；prism-switch 是**一套 Provider 定义，一键投影到所有 Agent**。

## 配置格式

```yaml
# ~/.prism/config.yaml

# 共享 Provider 池 —— 凭据只定义一次
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    base_url: https://openrouter.ai/api/v1
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
  google:
    api_key: ${GEMINI_API_KEY}

# 每个 Agent 的配置 —— 引用 Provider + Agent 特有的 model
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

### 设计约束

- **密钥不落盘**：配置文件中只允许 `${ENV_VAR}` 引用，运行时展开，绝不存储明文
- **Git 友好**：config.yaml 可安全提交到 dotfiles 仓库
- **无数据库**：不依赖 SQLite，YAML 文件即 SSOT

## 投影映射

每个 Agent Projector 将通用字段映射到 Agent 原生格式：

| 通用字段 | Claude Code | Codex | Gemini CLI |
|----------|-------------|-------|------------|
| api_key | `env.ANTHROPIC_AUTH_TOKEN` → `~/.claude/settings.json` | `OPENAI_API_KEY` → `~/.codex/auth.json` | `GEMINI_API_KEY` → `~/.gemini/.env` |
| base_url | `env.ANTHROPIC_BASE_URL` → `~/.claude/settings.json` | — | `GOOGLE_GEMINI_BASE_URL` → `~/.gemini/.env` |
| model | `env.ANTHROPIC_MODEL` → `~/.claude/settings.json` | `model` → `~/.codex/config.toml` | `model` → `~/.gemini/settings.json` |

### 写入策略

- **Claude Code**：读取现有 `settings.json` → 合并 `env` 字段 → 保留 `permissions` 等其他字段 → 原子写回
- **Codex**：写 `auth.json`（JSON）+ 更新 `config.toml` 中的 `model` 字段
- **Gemini CLI**：写 `.env`（KEY=VALUE）+ 更新 `settings.json` 中的 `model` 字段
- **所有写入**：tmp 文件 + rename 原子替换，防止写入中断导致配置损坏

## 命令设计

```bash
prism sync [agent...]              # 投影当前 Provider 到 live 配置（省略 agent = 全部）
prism switch <agent> <provider>    # 切换某个 Agent 的当前 Provider + 投影
prism switch --all <provider>      # 一键切换所有 Agent 到同一 Provider
prism status                       # 显示各 Agent 当前 Provider + Drift 检测
prism import [agent...]            # 反向回读：从 live 配置回写到 YAML（backfill）
prism validate                     # 校验配置格式 + 检查环境变量是否存在
```

## 架构

```
cmd/prism/main.go
  └── internal/cli/          # Cobra 命令层
        └── internal/sync/   # 同步引擎
              ├── internal/config/  # YAML 解析 + 环境变量展开 + 校验
              └── internal/agent/   # 各 Agent 投影器
```

### 模块职责

| 模块 | 职责 |
|------|------|
| `config` | 加载/解析 YAML、展开 `${ENV_VAR}`、校验配置完整性和环境变量存在性 |
| `agent` | `Projector` 接口 + 各 Agent 实现（读写 live 配置文件） |
| `sync` | 编排引擎：sync / switch / import / status / drift 检测 |
| `cli` | Cobra 命令定义、参数解析、输出格式化 |

### Projector 接口

```go
type Projector interface {
    Name() string                              // "claude"
    DisplayName() string                       // "Claude Code"
    ConfigPaths() []string                     // live 配置文件路径列表
    Project(p *config.ResolvedProvider) error  // 写入 live 配置
    ReadLive() (*config.ResolvedProvider, error) // 从 live 回读
}
```

## 安全设计

1. **密钥不落盘**：config.yaml 中只允许 `${ENV_VAR}` 引用
2. **运行时展开**：环境变量仅在 `Project()` 调用时展开，不缓存
3. **输出脱敏**：`prism status` 中 API Key 显示为 `sk-***...***`
4. **文件权限**：config.yaml 写入时设置 `0600`
5. **错误信息**：错误消息中不包含环境变量的实际值

## Drift 检测

`prism status` 对比 YAML 中的期望配置与 live 文件的实际内容：

- **synced**：live 配置与 YAML 一致
- **drifted**：live 配置被手动修改（如用户在 Agent 内改了 model）
- **missing**：live 配置文件不存在
- **error**：读取失败

## 扩展性

新增 Agent 只需：
1. 实现 `Projector` 接口（一个新文件）
2. 在 `agent.Register()` 中注册
3. 无需修改 config / sync / cli 层

## 不做的事（v1 范围外）

- MCP 服务器管理
- Proxy 接管
- Skill 分发
- GUI / TUI
- 自动更新
- 多设备同步（通过 git 天然解决）
