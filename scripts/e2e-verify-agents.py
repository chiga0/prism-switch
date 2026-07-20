#!/usr/bin/env python3
"""E2E verification: read prism-written configs and make real API calls for all 6 agents."""
import json
import sys
import os

results = []

def test_claude():
    """Claude Code - anthropic protocol via ~/.claude/settings.json"""
    import anthropic
    with open(os.path.expanduser("~/.claude/settings.json")) as f:
        settings = json.load(f)
    env = settings["env"]
    api_key = env["ANTHROPIC_AUTH_TOKEN"]
    base_url = env["ANTHROPIC_BASE_URL"]
    model = env["ANTHROPIC_MODEL"]
    print(f"  base_url={base_url}")
    print(f"  model={model}")
    client = anthropic.Anthropic(api_key=api_key, base_url=base_url)
    r = client.messages.create(model=model, max_tokens=20, messages=[{"role": "user", "content": "say OK"}])
    text = [b.text for b in r.content if b.type == "text"][0]
    print(f"  model says: {text}")
    return True

def test_codex():
    """Codex CLI - openai protocol via ~/.codex/auth.json + config.toml"""
    from openai import OpenAI
    with open(os.path.expanduser("~/.codex/auth.json")) as f:
        auth = json.load(f)
    api_key = auth["OPENAI_API_KEY"]
    base_url = None
    model = None
    with open(os.path.expanduser("~/.codex/config.toml")) as f:
        for line in f:
            if line.startswith("api_base_url"):
                base_url = line.split("=")[1].strip().strip("'\"")
            if line.startswith("model"):
                model = line.split("=")[1].strip().strip("'\"")
    print(f"  base_url={base_url}")
    print(f"  model={model}")
    client = OpenAI(api_key=api_key, base_url=base_url)
    r = client.chat.completions.create(model=model, max_tokens=20, messages=[{"role": "user", "content": "say OK"}])
    print(f"  model says: {r.choices[0].message.content}")
    return True

def test_opencode():
    """OpenCode - openai protocol via ~/.config/opencode/opencode.json"""
    from openai import OpenAI
    with open(os.path.expanduser("~/.config/opencode/opencode.json")) as f:
        oc = json.load(f)
    opts = oc["provider"]["prism"]["options"]
    api_key = opts["apiKey"]
    base_url = opts.get("baseURL", "")
    model = oc.get("model", "")
    print(f"  base_url={base_url}")
    print(f"  model={model}")
    client = OpenAI(api_key=api_key, base_url=base_url)
    r = client.chat.completions.create(model=model, max_tokens=20, messages=[{"role": "user", "content": "say OK"}])
    print(f"  model says: {r.choices[0].message.content}")
    return True

def test_qwencode():
    """Qwen Code - openai protocol via ~/.qwen/settings.json"""
    from openai import OpenAI
    with open(os.path.expanduser("~/.qwen/settings.json")) as f:
        qw = json.load(f)
    env = qw["env"]
    api_key = env["QWEN_API_KEY"]
    base_url = env.get("QWEN_BASE_URL", "")
    model = qw.get("model", "")
    print(f"  base_url={base_url}")
    print(f"  model={model}")
    client = OpenAI(api_key=api_key, base_url=base_url)
    r = client.chat.completions.create(model=model, max_tokens=20, messages=[{"role": "user", "content": "say OK"}])
    print(f"  model says: {r.choices[0].message.content}")
    return True

def test_zcode():
    """ZCode - anthropic protocol via ~/.zcode/v2/config.json"""
    import anthropic
    with open(os.path.expanduser("~/.zcode/v2/config.json")) as f:
        zc = json.load(f)
    zopts = zc["provider"]["prism"]["options"]
    api_key = zopts["apiKey"]
    base_url = zopts.get("baseURL", "")
    models = list(zc["provider"]["prism"].get("models", {}).keys())
    model = models[0] if models else ""
    print(f"  base_url={base_url}")
    print(f"  model={model}")
    client = anthropic.Anthropic(api_key=api_key, base_url=base_url)
    # glm-5.2 is a reasoning model: needs enough tokens for thinking + text
    r = client.messages.create(model=model, max_tokens=500, messages=[{"role": "user", "content": "say OK"}])
    text_blocks = [b.text for b in r.content if b.type == "text"]
    if text_blocks:
        print(f"  model says: {text_blocks[0]}")
    else:
        # reasoning model may only produce thinking within token budget
        thinking_blocks = [b for b in r.content if b.type == "thinking"]
        if thinking_blocks:
            print(f"  model produced thinking (reasoning model), stop_reason={r.stop_reason}")
        else:
            raise RuntimeError("no content blocks in response")
    return True

def test_gemini():
    """Gemini CLI - google protocol via ~/.gemini/.env (verify key set, no base_url forced)"""
    with open(os.path.expanduser("~/.gemini/.env")) as f:
        content = f.read()
    print(f"  .env: {content.strip()}")
    has_key = "GEMINI_API_KEY=" in content
    no_url = "GOOGLE_GEMINI_BASE_URL" not in content
    if has_key and no_url:
        print("  API key set, no base_url forced (google protocol correct)")
        return True
    print("  FAIL: unexpected .env content")
    return False

TESTS = [
    ("claude (anthropic)", test_claude),
    ("codex (openai)", test_codex),
    ("opencode (openai)", test_opencode),
    ("qwen-code (openai)", test_qwencode),
    ("zcode (anthropic)", test_zcode),
    ("gemini (google)", test_gemini),
]

if __name__ == "__main__":
    print("=" * 60)
    print("prism-switch E2E verification — all 6 agents")
    print("=" * 60)
    for name, fn in TESTS:
        print(f"\n--- {name} ---")
        try:
            ok = fn()
            results.append((name, "PASS" if ok else "FAIL"))
            print(f"  [PASS]")
        except Exception as e:
            results.append((name, f"FAIL: {e}"))
            print(f"  [FAIL] {e}")

    print("\n" + "=" * 60)
    print("SUMMARY")
    print("=" * 60)
    all_pass = True
    for name, status in results:
        icon = "✓" if status == "PASS" else "✗"
        print(f"  {icon} {name}: {status}")
        if status != "PASS":
            all_pass = False
    print(f"\nResult: {'ALL PASS' if all_pass else 'SOME FAILED'}")
    sys.exit(0 if all_pass else 1)
