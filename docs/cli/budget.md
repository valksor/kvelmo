# mehr budget

Manage cost and token budgets.

## Synopsis

```bash
mehr budget <command> [flags]
```

## Description

The `budget` command lets you set workspace-wide defaults and per-task overrides for
token usage and cost limits. Budgets can warn, pause, or stop a task when limits are reached.

Paused tasks must be explicitly resumed with confirmation.

## Commands

| Subcommand | Purpose                                         |
|------------|-------------------------------------------------|
| `status`   | Show budgets for active task and monthly totals |
| `set`      | Set workspace default budgets                   |
| `task set` | Set a budget override for a specific task       |
| `resume`   | Resume a task paused due to budget limits       |
| `reset`    | Reset monthly budget tracking                   |

## Examples

### Show Current Budget Status

```bash
mehr budget status
```

### Set Workspace Defaults

```bash
mehr budget set \
  --task-max-cost 10 \
  --task-warning-at 0.8 \
  --task-on-limit pause \
  --monthly-max-cost 100 \
  --monthly-warning-at 0.8
```

### Set a Task Budget

```bash
mehr budget task set --max-cost 5 --on-limit pause
```

### Resume a Paused Task

```bash
mehr budget resume --confirm
```

### Reset Monthly Tracking

```bash
mehr budget reset --month
```

## Flags

### budget set

| Flag                   | Description                               |
|------------------------|-------------------------------------------|
| `--task-max-cost`      | Default max cost per task (USD)           |
| `--task-max-tokens`    | Default max tokens per task               |
| `--task-on-limit`      | Default behavior: `warn`, `pause`, `stop` |
| `--task-warning-at`    | Default warning threshold (0-1)           |
| `--task-currency`      | Default task currency (e.g., `USD`)       |
| `--monthly-max-cost`   | Monthly max cost (USD)                    |
| `--monthly-warning-at` | Monthly warning threshold (0-1)           |
| `--monthly-currency`   | Monthly currency (e.g., `USD`)            |

### budget task set

| Flag           | Description                       |
|----------------|-----------------------------------|
| `--task`       | Task ID (defaults to active task) |
| `--max-cost`   | Max cost for the task             |
| `--max-tokens` | Max tokens for the task           |
| `--on-limit`   | Behavior: `warn`, `pause`, `stop` |
| `--warning-at` | Warning threshold (0-1)           |
| `--currency`   | Currency code                     |

### budget resume

| Flag        | Description                     |
|-------------|---------------------------------|
| `--confirm` | Required confirmation to resume |

### budget reset

| Flag      | Description                   |
|-----------|-------------------------------|
| `--month` | Reset monthly budget tracking |

## Web UI

Prefer a visual interface? See the Budget display in [Dashboard](../web-ui/dashboard.md).

## See Also

- [cost](cost.md) - View token usage and costs
- [Configuration Guide](../configuration/index.md) - Budget settings in `config.yaml`
