# kvelmo list

List all tasks in the workspace.

## Usage

```bash
kvelmo list
```

## Options

| Flag     | Description    |
|----------|----------------|
| `--json` | Output as JSON |

## Output

```
Tasks:
  #1 [implemented] Add user auth (github:owner/repo#123)
  #2 [planned] Fix login bug (file:task.md)
  #3 [submitted] Update docs (github:owner/repo#456)
```

## Examples

```bash
# List all tasks
kvelmo list

# JSON output
kvelmo list --json
```

## Related

- [status](/cli/status.md) — Current task details
- [projects](/cli/projects.md) — Project registry
