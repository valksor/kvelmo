# FAQ

Frequently asked questions about kvelmo.

## General

### What is kvelmo?

kvelmo is a structured development workflow that orchestrates AI agents. It manages the lifecycle of development tasks from requirements through implementation to PR submission.

### Is kvelmo an AI?

No. kvelmo is an **orchestrator**. It coordinates existing AI tools (like Claude CLI) through a structured workflow with human oversight.

### Is kvelmo free?

Yes. kvelmo is BSD-3 licensed and completely free. You use your existing AI CLI subscriptions.

### Does kvelmo require API keys?

Not for kvelmo itself. Your agent CLI (Claude, Codex) handles authentication. If your CLI works, kvelmo works.

## Installation

### How do I install kvelmo?

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash
```

See [INSTALL.md](/INSTALL.md) for detailed instructions.

### What are the requirements?

- An AI agent CLI (Claude, Codex)
- Git
- Unix-like environment (macOS, Linux, WSL2)

### Does kvelmo work on Windows?

Yes, through WSL2. See [Windows WSL Setup](/guides/windows-wsl.md).

## Usage

### How do I start a task?

```bash
kvelmo start --from file:task.md
```

Or from GitHub:
```bash
kvelmo start --from github:owner/repo#123
```

### What's the workflow?

1. `start` — Load task
2. `plan` — Generate specification
3. `implement` — Execute specification
4. `review` — Review changes
5. `submit` — Create PR

### Can I undo changes?

Yes. Every phase creates a git checkpoint:
```bash
kvelmo undo
```

### How do I use the Web UI?

```bash
kvelmo serve
```

Then open http://localhost:6337.

## Troubleshooting

### "command not found: kvelmo"

kvelmo isn't in your PATH. Reinstall:
```bash
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash
```

### "no agent found"

Install an agent CLI:
```bash
claude --version  # Check Claude
codex --version   # Check Codex
```

### Task is stuck

Reset the state:
```bash
kvelmo reset
```

### Server not responding

Restart the server:
```bash
# Stop (Ctrl+C) then
rm ~/.valksor/kvelmo/global.sock
kvelmo serve
```

## Configuration

### Where is the config file?

Global: `~/.valksor/kvelmo/config.json`
Project: `.kvelmo/config.json`

### How do I change settings?

```bash
kvelmo config set <key> <value>
```

Or edit the JSON file directly.

### How do I set up GitHub integration?

Set your token:
```bash
export GITHUB_TOKEN=ghp_xxxx
```

## Agents

### Which agents are supported?

- Claude (recommended)
- Codex
- Custom agents

### Can I use different agents for different phases?

Yes. Configure in settings:
```json
{
  "agent_steps": {
    "planning": "claude",
    "implementing": "codex"
  }
}
```

## Getting Help

### Where can I report bugs?

[GitHub Issues](https://github.com/valksor/kvelmo/issues)

### Where can I ask questions?

[GitHub Discussions](https://github.com/valksor/kvelmo/discussions)

### Is there documentation?

Yes. [Documentation](https://valksor.com/docs/kvelmo/nightly)
