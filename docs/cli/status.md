# kvelmo status

Show current task state and information.

## Usage

```bash
kvelmo status
```

## Options

| Flag     | Description    |
|----------|----------------|
| `--json` | Output as JSON |

## Examples

```bash
# Show status
kvelmo status

# JSON output
kvelmo status --json
```

## Output

```
Task: Add user authentication
State: implemented
Branch: feature/add-user-auth
Provider: github:valksor/kvelmo#123

Checkpoints:
  - plan: abc1234
  - implement: def5678
```

## States

| State          | Description                |
|----------------|----------------------------|
| `none`         | No active task             |
| `loaded`       | Task loaded                |
| `planning`     | Planning in progress       |
| `planned`      | Ready to implement         |
| `implementing` | Implementation in progress |
| `implemented`  | Ready to review            |
| `reviewing`    | Review in progress         |
| `submitted`    | Task complete              |

Also in Web UI: [Dashboard](/web-ui/dashboard.md).

## Related

- [State Machine](/concepts/state-machine.md) — All states
- [list](/cli/list.md) — List all tasks
