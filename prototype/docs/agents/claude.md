# Claude Agent

> **⚠️ Third-Party Integration**: This integration depends on external APIs that may change. Not fully tested beyond unit tests. Behavior may vary depending on the third-party service. Manual validation recommended before production use.


Claude is the default AI agent. Mehrhof calls the `claude` CLI command.

## Prerequisites

- Claude CLI installed and configured
- Your Claude settings (API keys, model preferences) already set up

```bash
claude --version
```

## Configuration

Use as default:

```yaml
# .mehrhof/config.yaml
agent:
  default: claude
```

Or specify via CLI:

```bash
mehr start --agent claude file:task.md
```

## Workflow Behavior

Mehrhof automatically configures Claude's permission mode based on the workflow step:

| Step            | Permission Mode | Description                               |
|-----------------|-----------------|-------------------------------------------|
| `planning`      | `plan`          | Read-only analysis, no file modifications |
| `implementing`  | `acceptEdits`   | Claude can write/modify files             |
| `reviewing`     | `acceptEdits`   | Claude can apply review fixes             |
| `checkpointing` | Default         | Summary generation only                   |

Permission modes are set explicitly to ensure consistent behavior regardless of user's default Claude settings. This prevents:
- Planning step from modifying files (if user defaults to `acceptEdits`)
- Implementation step from being read-only (if user defaults to `plan`)

> **Note**: Agent aliases that extend Claude (like `work-account` or `opus`) inherit this behavior automatically.
