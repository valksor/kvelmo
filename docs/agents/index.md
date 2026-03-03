# Agents

Agents are AI models that execute kvelmo's workflow phases. kvelmo orchestrates agents to handle planning, implementation, and review.

## Supported Agents

| Agent | Description |
|-------|-------------|
| [Claude](/agents/claude.md) | Anthropic's Claude via CLI |
| [Codex](/agents/codex.md) | OpenAI's Codex via CLI |
| [Custom](/agents/custom.md) | Your own agent implementation |

## How Agents Work

kvelmo doesn't call AI APIs directly. Instead, it orchestrates existing CLI tools:

```
kvelmo → Agent CLI → AI Model → Response
```

This means:
- You use your existing CLI subscription
- No API keys to configure in kvelmo
- kvelmo adds workflow structure on top

## Agent Selection

kvelmo selects agents in this priority order:

1. Command flag: `--agent claude`
2. Task configuration
3. Project configuration
4. Global configuration
5. Auto-detect (finds installed CLI)

## Configuring Agents

### Global Configuration

Set the default agent in `~/.valksor/kvelmo/kvelmo.yaml`:

```json
{
  "default_agent": "claude"
}
```

Or via CLI:
```bash
kvelmo config set default_agent claude
```

### Per-Phase Agents

Use different agents for different phases:

```json
{
  "agent_steps": {
    "planning": "claude",
    "implementing": "codex",
    "reviewing": "claude"
  }
}
```

## Agent Events

During execution, agents emit events:

| Event | Description |
|-------|-------------|
| `token` | Output token streamed |
| `tool_call` | Agent called a tool |
| `tool_result` | Tool returned a result |
| `permission` | Permission requested |
| `completion` | Agent finished |

These events are streamed to the Web UI and CLI.

## Agent Permissions

Agents request permissions for actions:

- Read-only tools are auto-approved
- Write tools require explicit approval
- Some tools can be pre-approved in settings

## Adding a New Agent

See [Custom Agents](/agents/custom.md) for implementing your own agent.
