# mehr reset

Reset the workflow state to idle without losing work.

## Usage

```bash
mehr reset [flags]
```

## Description

The `reset` command resets the workflow state to idle when an agent hangs or crashes. This preserves all your work (specifications, notes, code changes) but allows you to retry the current step.

Use this when:
- An agent hangs and you had to kill it
- State is stuck in planning/implementing/reviewing
- You want to retry a step without abandoning the task

## Flags

| Flag    | Short | Description              |
|---------|-------|--------------------------|
| `--yes` | `-y`  | Skip confirmation prompt |

## Examples

### Reset with confirmation

```bash
mehr reset
```

Output:
```
About to reset workflow state from 'planning' to 'idle'
  Task: Implement user authentication

  This preserves all specifications, notes, and code changes.

Continue? [y/N] y
State reset to idle

Retry the step:
  mehr plan - Create specifications
  mehr implement - Implement specifications
  mehr review - Review changes
```

### Reset without confirmation

```bash
mehr reset --yes
```

## Alternative: --force flag

You can also use the `--force` flag on step commands to reset and retry in one command:

```bash
mehr plan --force      # Reset state and start planning
mehr implement --force # Reset state and start implementing
mehr review --force    # Reset state and start reviewing
```

## When to use

| Scenario                       | Command                                  |
|--------------------------------|------------------------------------------|
| Agent hung during planning     | `mehr reset --yes` then `mehr plan`      |
| Agent hung during implementing | `mehr reset --yes` then `mehr implement` |
| One-step recovery              | `mehr plan --force` (reset + plan)       |

## Web UI

Prefer a visual interface? See [Web UI: Reset](/web-ui/reset.md).

## See Also

- [mehr undo](undo.md) - Revert to previous checkpoint
- [mehr abandon](abandon.md) - Discard task completely
- [mehr status](status.md) - View current task state
