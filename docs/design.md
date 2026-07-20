# Prism-Switch 设计文档

## 定位

跨平台 CLI 工具，用一套声明式 YAML 配置统一管理多个 AI Coding Agent 的 Provider（API Key / Endpoint / Model），一键投影到各 Agent 的原生配置文件。

## 核心差异

cc-switch 是**每个 Agent 各管各的 Provider**（桌面 GUI + SQLite）；prism-switch 是**一套 Provider 定义，一键投影到所有 Agent**（CLI + 纯 YAML）。

## 架构

```
cmd/prism/main.go
  └── internal/cli/          # Cobra 命令层
        └── internal/sync/   # 同步引擎（sync/switch/import/status/dry-run）
              ├── internal/config/  # YAML 解析 + 环境变量展开 + 校验
              └── internal/agent/   # 各 Agent 投影器
```

### 模块职责

| 模块 | 职责 |
|------|------|
| `config` | 加载/解析 YAML、展开 `${ENV_VAR}`、校验配置完整性和环境变量存在性 |
| `agent` | `Projector` 接口 + 各 Agent 实现（读写 live 配置文件） |
| `sync` | 编排引擎：sync / switch / import / status / dry-run / drift 检测 |
| `cli` | Cobra 命令定义、参数解析、输出格式化 |

### Projector 接口

```go
type Projector interface {
    Name() string                                // "claude"
    DisplayName() string                         // "Claude Code"
    ConfigPaths() []string                       // live 配置文件路径列表
    Project(p *config.ResolvedProvider) error    // 写入 live 配置
    ReadLive() (*config.ResolvedProvider, error) // 从 live 回读
}
```

新增 Agent 只需：实现接口（一个新文件）+ 在 `root.go` 注册一行。

## 支持的 Agent

| Agent | 配置路径 | 格式 |
|-------|---------|------|
| Claude Code | `~/.claude/settings.json` | JSON（env 字段） |
| Codex CLI | `~/.codex/auth.json` + `config.toml` | JSON + TOML |
| Gemini CLI | `~/.gemini/.env` + `settings.json` | ENV + JSON |
| OpenCode | `~/.config/opencode/opencode.json` | JSON（provider 字段） |
| Qwen Code | `~/.qwen/settings.json` | JSON（env 字段） |

## 数据流

```
~/.prism/config.yaml (SSOT)
  │
  ├─ prism sync ──→ 展开 ${ENV_VAR} ──→ 按 Agent 格式投影 ──→ live 配置文件
  │
  ├─ prism switch ──→ 更新 YAML current ──→ 投影 ──→ 保存 YAML（投影成功后）
  │
  ├─ prism import ──→ 读取 live 配置 ──→ 生成 ${IMPORTED_*} 占位 ──→ 保存 YAML
  │
  └─ prism status ──→ 对比 YAML vs live ──→ synced / drifted / missing
```

## 安全设计

1. **密钥不落盘**：config.yaml 中只允许 `${ENV_VAR}` 引用
2. **运行时展开**：环境变量仅在 `Project()` 调用时展开，不缓存
3. **输出脱敏**：`prism status` 中 API Key 显示为 `sk-o***9999`
4. **文件权限**：config.yaml `0600`，.env `0600`
5. **错误信息**：错误消息中不包含环境变量的实际值
6. **import 安全**：反向导入生成占位符，不写明文 key

## 写入策略

- **原子写入**：所有配置文件用 tmp + rename，防止写入中断损坏
- **保留已有字段**：读取 → 合并 → 写回，不清除 Agent 内其他配置
- **损坏保护**：解析失败时自动备份 + 警告到 stderr
- **Switch 原子性**：先投影成功，再保存 YAML；投影失败则回滚 current 字段

## 跨平台

- Go 交叉编译：darwin/linux/windows × amd64/arm64
- 路径基于 `os.UserHomeDir()`，三平台一致
- OpenCode 使用 `~/.config/opencode/`（XDG 风格，macOS/Linux 一致）
- Goreleaser 自动构建 + Homebrew tap 分发

## 不做的事（v1 范围外）

- MCP 服务器管理
- Proxy 接管
- Skill 分发
- GUI / TUI
- 自动更新
- 多设备同步（通过 git 天然解决）
- macOS Keychain / Linux secret-service 集成（后续考虑）
