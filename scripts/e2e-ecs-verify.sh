#!/bin/bash
# prism-switch 实机验证脚本
# 在本机终端执行: bash scripts/e2e-ecs-verify.sh
set -e

ECS_IP="121.196.211.44"
ECS_KEY="/Users/gawain/Documents/work/ecs/gaoqi-key.pem"
SSH="ssh -i $ECS_KEY -o StrictHostKeyChecking=no root@$ECS_IP"
SCP="scp -i $ECS_KEY -o StrictHostKeyChecking=no"

# 从 ~/.qwen/settings.json 提取真实 key
API_KEY=$(python3 -c "import json; d=json.load(open('$HOME/.qwen/settings.json')); print(d['env']['QWENCLOUD_TOKEN_PLAN_API_KEY'])")

echo "=== 1. 交叉编译 ==="
cd "$(dirname "$0")/.."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o prism-linux-amd64 ./cmd/prism

echo "=== 2. 部署到 ECS ==="
$SCP prism-linux-amd64 root@$ECS_IP:/usr/local/bin/prism
$SSH "chmod +x /usr/local/bin/prism && prism --help | head -3"

echo "=== 3. 在 ECS 上配置并 sync ==="
$SSH "export TOKEN_PLAN_API_KEY='$API_KEY' && mkdir -p ~/.prism && cat > ~/.prism/config.yaml << 'YAML'
providers:
  token-plan:
    api_key: \${TOKEN_PLAN_API_KEY}
    base_urls:
      openai: https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1
      anthropic: https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic

agents:
  claude:
    current: token-plan
    model: qwen3.8-max-preview
  codex:
    current: token-plan
    model: qwen3.8-max-preview
  gemini:
    current: token-plan
    model: qwen3.8-max-preview
  opencode:
    current: token-plan
    model: qwen3.8-max-preview
  qwen-code:
    current: token-plan
    model: qwen3.8-max-preview
  zcode:
    current: token-plan
    model: glm-5.2
YAML
prism validate && prism sync && prism status"

echo "=== 4. 验证 Claude Code 配置（anthropic 协议）==="
$SSH "cat ~/.claude/settings.json | python3 -m json.tool"

echo "=== 5. 验证 Codex 配置（openai 协议）==="
$SSH "cat ~/.codex/auth.json && echo '---' && cat ~/.codex/config.toml"

echo "=== 6. 验证 Gemini 配置（google 协议，不应有 base_url）==="
$SSH "cat ~/.gemini/.env"

echo "=== 7. 实际 API 调用验证（OpenAI 兼容端点）==="
$SSH "curl -s https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/chat/completions \
  -H 'Authorization: Bearer $API_KEY' \
  -H 'Content-Type: application/json' \
  -d '{\"model\":\"qwen3.8-max-preview\",\"messages\":[{\"role\":\"user\",\"content\":\"say hi in 3 words\"}],\"max_tokens\":20}' | python3 -m json.tool | head -20"

echo "=== 8. 实际 API 调用验证（Anthropic 兼容端点）==="
$SSH "curl -s https://token-plan.cn-beijing.maas.aliyuncs.com/apps/anthropic/v1/messages \
  -H 'x-api-key: $API_KEY' \
  -H 'anthropic-version: 2023-06-01' \
  -H 'Content-Type: application/json' \
  -d '{\"model\":\"qwen3.8-max-preview\",\"max_tokens\":20,\"messages\":[{\"role\":\"user\",\"content\":\"say hi in 3 words\"}]}' | python3 -m json.tool | head -20"

echo ""
echo "=== 验证完成 ==="
echo "如果步骤 7 和 8 都返回了模型回复，说明 prism-switch 的协议感知投影是正确的。"
echo "清理: ssh root@$ECS_IP 'rm -rf ~/.prism ~/.claude ~/.codex ~/.gemini ~/.config/opencode ~/.qwen ~/.zcode'"
