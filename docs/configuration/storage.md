# Storage Configuration

Controls where task data and caches are stored.

## Settings

```yaml
storage:
  work_dir: work  # Relative to workspace data directory

cache:
  enabled: true  # Enable/disable caching globally

github:
  cache:
    disabled: false  # Provider-specific override
```

## Storage Structure

```
project/
├── .mehrhof/
│   ├── config.yaml    # Workspace configuration (safe to commit)
│   └── .env           # Project-specific secrets (gitignored)

~/.valksor/mehrhof/workspaces/<project-id>/
├── .active_task       # Current task state
└── work/              # Task work directories
    ├── abc123/
    │   ├── work.yaml
    │   ├── notes.md
    │   ├── source/
    │   ├── specifications/
    │   └── sessions/
    └── def456/
        └── ...
```

## Project ID

The `<project-id>` is automatically derived from your git remote:

| Git Remote URL | Project ID |
|----------------|------------|
| `https://github.com/user/repo` | `github.com-user-repo` |
| `git@github.com:user/project.git` | `github.com-user-project` |
| `https://gitlab.com/group/subgroup/project` | `gitlab.com-group-subgroup-project` |
| No remote (local) | `local-<hash>` |

## See Also

- [Storage Structure Reference](../reference/storage.md) - Complete storage documentation
- [Configuration Overview](index.md) - All configuration options
