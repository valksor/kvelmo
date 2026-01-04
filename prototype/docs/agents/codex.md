# Codex Agent

> **⚠️ Third-Party Integration**: This integration depends on external APIs that may change. Not fully tested beyond unit tests. Behavior may vary depending on the third-party service. Manual validation recommended before production use.


Codex is an alternative AI agent. Mehrhof calls the `codex` CLI command.

## Prerequisites

- Codex CLI installed and configured
- Your Codex settings (API keys, model preferences) already set up

```bash
codex --version
```

## Configuration

```bash
mehr start --agent codex file:task.md
```

## Workflow Behavior

Mehrhof automatically configures Codex's approval mode based on the workflow step:

| Step | Flag | Description |
|------|------|-------------|
| `planning` | Default | Codex analyzes without modifications |
| `implementing` | `--full-auto` | Workspace-write sandbox + on-request approval |
| `reviewing` | `--full-auto` | Can apply review fixes |
| `checkpointing` | Default | Summary generation only |

The `--full-auto` flag is a shortcut for `--sandbox workspace-write --ask-for-approval on-request`, allowing Codex to write files within the workspace without prompting for each operation.

> **Note**: Agent aliases that extend Codex inherit this behavior automatically.
