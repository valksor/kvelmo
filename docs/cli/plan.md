# kvelmo plan

Generate an implementation specification for the current task.

## Usage

```bash
kvelmo plan
```

## Options

| Flag      | Description                    |
|-----------|--------------------------------|
| `--force` | Re-run even if already planned |

## Prerequisites

- Task must be in `loaded` state
- Run `kvelmo start` first

## Examples

```bash
# Generate specification
kvelmo plan

# Force re-plan
kvelmo plan --force
```

## What Happens

1. Agent analyzes the task requirements
2. Agent explores the codebase for context
3. A specification is generated in `.kvelmo/specifications/`
4. A git checkpoint is created
5. State transitions to `planned`

## Specification Output

The specification is saved to:
```
.kvelmo/specifications/specification.md
```

Review it before implementing:
```bash
cat .kvelmo/specifications/specification.md
```

## If Planning Fails

- Check your task description is clear
- Add more context to the task
- Use `kvelmo reset` to recover

Also in Web UI: [Planning Phase](/web-ui/planning.md).

## Related

- [start](/cli/start.md) — Start a task first
- [implement](/cli/implement.md) — Execute the specification
