# Codex Agent

> **⚠️ EXPERIMENTAL - Untested Implementation**
>
> This Codex agent implementation is based on CLI documentation only.
> The JSON output format and tool use behavior have NOT been validated
> against an actual Codex CLI. File operations and output parsing may
> not work correctly until tested and adjusted.
>
> **Before using**: Verify you have `codex` CLI installed and run:
> ```bash
> codex exec --json "echo test"
> ```
> To see the actual JSON output format.

Codex is an alternative AI agent. Mehrhof calls the `codex exec --json` CLI command.

## Prerequisites

- Codex CLI installed and configured
- Your Codex settings (API keys, model preferences) already set up

```bash
codex --version
```

## Configuration

Use as default:

```yaml
# .mehrhof/config.yaml
agent:
  default: codex
```

Or specify via CLI:

```bash
mehr start --agent codex file:task.md
```

## Workflow Behavior

Mehrhof automatically configures Codex based on the workflow step:

| Step            | Args                  | Description                                 |
|-----------------|-----------------------|---------------------------------------------|
| `planning`      | `--sandbox read-only` | Read-only analysis, no file modifications   |
| `implementing`  | `--full-auto`         | Auto-execution with workspace-write sandbox |
| `reviewing`     | `--full-auto`         | Auto-execution with workspace-write sandbox |
| `checkpointing` | (none)                | Summary generation only                     |

**Note**: `--full-auto` is equivalent to `--sandbox workspace-write --ask-for-approval on-failure`.

## Differences from Claude

| Feature            | Claude                               | Codex                                     |
|--------------------|--------------------------------------|-------------------------------------------|
| Permission control | `--permission-mode plan/acceptEdits` | `--sandbox` + `--ask-for-approval`        |
| JSON output        | `--output-format stream-json`        | `--json`                                  |
| Non-interactive    | `--print`                            | `codex exec` (non-interactive by default) |
| Subcommand         | None (runs directly)                 | Requires `exec` subcommand                |

## Sandbox Modes

Codex uses sandbox policies to control what the agent can do:

| Mode                 | Description                                                            |
|----------------------|------------------------------------------------------------------------|
| `read-only`          | Cannot modify any files (used for planning)                            |
| `workspace-write`    | Can write files within the workspace (used for implementing/reviewing) |
| `danger-full-access` | Can write anywhere (not recommended)                                   |

## Agent Aliases

You can create Codex aliases with custom settings:

```yaml
# .mehrhof/config.yaml
agents:
  codex-gpt5:
    extends: codex
    description: "Codex with GPT-5 model"
    args: ["--model", "gpt-5-codex"]

  codex-work:
    extends: codex
    description: "Codex with work API key"
    env:
      OPENAI_API_KEY: "${WORK_API_KEY}"
```

## Per-Step Configuration

Different workflow steps can use different Codex configurations:

```yaml
# .mehrhof/config.yaml
agent:
  default: codex
  steps:
    planning:
      name: codex
    implementing:
      name: codex-gpt5  # Use GPT-5 for implementation
    reviewing:
      name: codex
```

## Known Limitations

1. **Untested Output Parsing**: The JSON output format from `codex exec --json` may differ from what's expected. File operations (` ```yaml:file` blocks) may not be detected without parser updates.

2. **Tool Use Format**: Unknown if Codex outputs tool calls in the same format as Claude (Read, Write, Edit, etc.).

3. **Event Structure**: The event types and structure may differ from Claude's `stream-json` format.

## Testing the Implementation

Once you have a Codex subscription, test the implementation:

```bash
# 1. Verify Codex CLI works
codex exec --json "create a hello world file" | head -20

# 2. Test with mehrhof in a temporary directory
mkdir -p /tmp/codex-test
cd /tmp/codex-test
git init
echo "# Test Task" > task.md

mehr start --agent codex file:task.md
mehr plan
mehr implement
```

If output parsing fails, capture the raw output for debugging:

```bash
codex exec --json "your task" > /tmp/codex-output.jsonl
# Use this to create a proper parser if JSONLineParser doesn't work
```

## See Also

- [Claude Agent](claude.md) - Primary supported agent
- [Codex CLI Reference](https://developers.openai.com/codex/cli/reference/) - Official documentation
- [Agent Configuration](../configuration/index.md) - Full config options
