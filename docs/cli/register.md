# mehr serve register / mehr serve unregister

Manage the global project registry.

> **Note**: These commands have been moved under `mehr serve`. The old `mehr register` and `mehr unregister` commands are deprecated.

## Synopsis

```bash
mehr serve register [flags]
mehr serve unregister [project-id]
```

## Description

The `register` and `unregister` subcommands manage a registry of projects for global mode. This is useful for:

- Adding projects to the global mode dashboard (`mehr serve --global`)
- Organizing frequently used projects
- Managing multi-project workflows from a single server

## Commands

### mehr serve register

Registers the current project in the global registry.

```bash
# Register current project
mehr serve register

# List all registered projects
mehr serve register --list
```

#### Flags

| Flag           | Type | Description                                         |
|----------------|------|-----------------------------------------------------|
| `--list`, `-l` | bool | List all registered projects instead of registering |

### mehr serve unregister

Removes a project from the registry.

```bash
# Unregister current project
mehr serve unregister

# Unregister by project ID
mehr serve unregister github.com-user-repo
```

## Project Registry

The registry is stored at `~/.valksor/mehrhof/projects.yaml` with the following structure:

```yaml
version: "1"
projects:
  github.com-user-repo:
    id: github.com-user-repo
    path: /home/user/projects/repo
    remote_url: https://github.com/user/repo
    name: repo
    registered_at: 2024-01-15T10:30:00Z
    last_access: 2024-01-15T14:20:00Z
```

## Examples

### Register and List Projects

```bash
# Navigate to a project
cd ~/projects/my-app

# Register it
mehr serve register
# Output: Registered project: github.com-user-my-app

# List all registered projects
mehr serve register --list
# Output:
# Registered Projects:
#   github.com-user-my-app
#     Path: /home/user/projects/my-app
#     Remote: https://github.com/user/my-app
#     Registered: 2024-01-15 10:30:00
```

### Unregister Projects

```bash
# Unregister current project
cd ~/projects/my-app
mehr serve unregister
# Output: Unregistered project: github.com-user-my-app

# Or unregister by ID
mehr serve unregister github.com-user-old-project
```

## Project ID Generation

Project IDs are automatically generated based on:

1. **Remote URL** (if available): Parsed to `github.com-user-repo` format
2. **Local path** (fallback): Hash of the absolute path for local-only repos

## Use Cases

### Setting Up Global Mode

```bash
# Register projects for the global dashboard
cd ~/projects/important-app
mehr serve register

cd ~/projects/client-work
mehr serve register

# Start server in global mode
mehr serve --global
```

### Managing Project List

```bash
# See what's registered
mehr serve register --list

# Clean up old projects
mehr serve unregister old-project-id
```

## Notes

- Registered projects appear in global mode (`mehr serve --global`)
- The registry persists across sessions at `~/.valksor/mehrhof/`

## See Also

- [serve](serve.md) - Start web UI server (parent command)
- [init](init.md) - Initialize workspace
- [status](status.md) - Check project status
