# kvelmo start

Start kvelmo sockets for the current directory.

## Usage

```bash
kvelmo start
```

## Options

| Flag              | Description                        |
|-------------------|------------------------------------|
| `--from`          | Load task from provider (optional) |
| `--verbose`, `-v` | Show socket paths                  |

## Provider Formats

When using `--from`:

| Provider | Format                        | Example                     |
|----------|-------------------------------|-----------------------------|
| File     | `file:<path>`                 | `file:task.md`              |
| GitHub   | `github:<owner>/<repo>#<num>` | `github:valksor/kvelmo#123` |
| GitLab   | `gitlab:<project>#<num>`      | `gitlab:group/project#456`  |
| Wrike    | `wrike:<id>`                  | `wrike:abc123`              |

## Examples

```bash
# Start sockets
kvelmo start

# Show socket paths
kvelmo start --verbose

# Start and load a task from a file
kvelmo start --from file:task.md

# Start and load from GitHub issue
kvelmo start --from github:valksor/kvelmo#123
```

## What Happens

1. Global socket starts at `~/.valksor/kvelmo/global.sock` (if not already running)
2. Worktree socket starts for the current directory
3. If `--from` is provided, the task is loaded and state transitions to `loaded`

Also in Web UI: [Creating Tasks](/web-ui/creating-tasks.md).

## Related

- [plan](/cli/plan.md) — Next step after loading a task
- [Providers](/providers/index.md) — Task sources
