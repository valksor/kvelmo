# mehr agents

Manage AI agents including built-in agents and user-defined aliases.

> **Note:** Agent listing is CLI-only. Agents are configured in the [Web UI Settings](../web-ui/settings.md) under the Agent section.

## Synopsis

```bash
mehr agents <subcommand> [flags]
```

## Description

The `agents` command group provides utilities for managing AI agents. Agents are the AI backends that perform planning, implementation, and review operations.

## Subcommands

### mehr agents list

List all registered agents.

```bash
mehr agents list
```

**Output:**

```
NAME          TYPE      EXTENDS       AVAILABLE  DESCRIPTION
claude        built-in  -             yes        -
work-account  alias     claude        yes        Claude with work API key
work-fast     alias     work-account  yes        Work account with lower token limit
```

**Columns:**
| Column | Description |
|--------|-------------|
| NAME | Agent identifier (use with `--agent` flag) |
| TYPE | `built-in` or `alias` |
| EXTENDS | Base agent for aliases |
| AVAILABLE | Whether the agent is available (CLI installed, etc.) |
| DESCRIPTION | Human-readable description |

## Configuring Agent Aliases

Aliases are defined in `.mehrhof/config.yaml`:

```yaml
agents:
  work-account:
    extends: claude # Required: base agent to wrap
    description: "Claude with work API key" # Optional: shown in list
    env: # Optional: environment variables
      ANTHROPIC_API_KEY: "${WORK_API_KEY}" # ${VAR} references system env vars

  work-fast:
    extends: work-account # Aliases can extend other aliases
    description: "Work account with lower tokens"
    env:
      MAX_TOKENS: "2048"
```

### Configuration Fields

| Field         | Required | Description                                            |
|---------------|----------|--------------------------------------------------------|
| `extends`     | Yes      | Name of the agent to wrap (built-in or another alias)  |
| `description` | No       | Human-readable description shown in `mehr agents list` |
| `env`         | No       | Environment variables passed to the agent              |

### Environment Variable Expansion

Values in `env` support shell variable expansion:

```yaml
env:
  ANTHROPIC_API_KEY: "${WORK_API_KEY}" # Expands $WORK_API_KEY
  MAX_TOKENS: "${MAX_TOKENS:-4096}" # With default (shell syntax)
  LITERAL_VALUE: "sk-ant-123" # Literal value (no expansion)
```

### mehr agents explain

Explain agent configuration priority and resolution order.

```bash
mehr agents explain
```

Displays a detailed explanation of how agents are selected based on configuration sources. This helps you understand which agent will be used when multiple configuration options conflict.

**Priority (highest to lowest):**

| Priority | Source                                                      | Scope           | Example                         |
|----------|-------------------------------------------------------------|-----------------|---------------------------------|
| 1        | `--agent-plan`, `--agent-implement`, `--agent-review` flags | Single step     | `mehr plan --agent-plan opus`   |
| 2        | `--agent` flag                                              | Entire workflow | `mehr start --agent sonnet ...` |
| 3        | Task frontmatter step-specific                              | Single step     | `agent_steps.planning.agent`    |
| 4        | Task frontmatter default                                    | Entire workflow | `agent: sonnet`                 |
| 5        | Workspace config step-specific                              | Single step     | `agent.steps.planning.name`     |
| 6        | Workspace config default                                    | Entire workflow | `agent.default: claude`         |
| 7        | Auto-detection                                              | Fallback        | First available agent           |

**Use when:** You want to understand why a specific agent is being used, or troubleshoot agent selection issues.

## Using Aliases

### At Task Start

```bash
mehr start --agent work-account file:task.md
```

### In Auto Mode

```bash
mehr auto --agent work-account file:task.md
```

### Change Default Agent

Set in workspace config:

```yaml
# .mehrhof/config.yaml
agent:
  default: work-account # Use alias as default
```

## Retry Configuration

Agents automatically retry on transient failures (network errors, temporary API outages). Configure retry behavior in `.mehrhof/config.yaml`:

```yaml
agent:
  retry_count: 3    # Number of attempts before giving up (default: 3)
  retry_delay: 5s   # Delay between retry attempts (default: 5s)
```

| Setting | Default | Description |
|---------|---------|-------------|
| `retry_count` | `3` | Total attempts per agent invocation. Set to `1` to disable retries |
| `retry_delay` | `5s` | Wait time between retries. Applies between each attempt |

Retries are context-aware: if the user cancels an operation (Ctrl+C), retries stop immediately.

## Common Patterns

### Multiple API Keys

Use aliases for different API keys or accounts:

```yaml
agents:
  work-claude:
    extends: claude
    description: "Work account"
    env:
      ANTHROPIC_API_KEY: "${WORK_CLAUDE_KEY}"

  personal-claude:
    extends: claude
    description: "Personal account"
    env:
      ANTHROPIC_API_KEY: "${PERSONAL_CLAUDE_KEY}"
```

### Different Configurations

Configure different model parameters:

```yaml
agents:
  claude-fast:
    extends: claude
    description: "Fast responses"
    env:
      MAX_TOKENS: "2048"

  claude-thorough:
    extends: claude
    description: "Detailed responses"
    env:
      MAX_TOKENS: "16000"
```

### Chained Aliases

Build on top of other aliases:

```yaml
agents:
  work-account:
    extends: claude
    env:
      ANTHROPIC_API_KEY: "${WORK_API_KEY}"

  work-fast:
    extends: work-account # Inherits work account's API key
    env:
      MAX_TOKENS: "2048"

  work-thorough:
    extends: work-account # Inherits work account's API key
    env:
      MAX_TOKENS: "16000"
```

## Troubleshooting

### "alias extends unknown agent"

The `extends` field references an agent that doesn't exist:

```
Error: alias "foo" extends unknown agent "bar"
```

Fix: Ensure the base agent name is correct (check with `mehr agents list`).

### "circular alias dependency"

Aliases cannot extend themselves (directly or indirectly):

```
Error: circular alias dependency detected: a
```

Fix: Review your alias chain and break the cycle.

### Alias Not Available

If an alias shows `AVAILABLE: no`, check that:

1. The base agent is available (`claude` CLI installed)
2. Any required environment variables are set

```bash
claude --version
```

## Web UI

Prefer a visual interface? See the Agent Settings section in [Settings](../web-ui/settings.md).

## See Also

- [AI Agents](../agents/index.md) - How agents work, aliases, and per-step configuration
- [Configuration Guide](../configuration/index.md) - Config file reference
