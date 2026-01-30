# Storage Configuration

Controls where task data and caches are stored.

## Settings

```yaml
storage:
  work_dir: work  # Relative to workspace data directory

specification:
  save_in_project: false          # Save specifications to project directory (for version control)
  project_dir: ""                 # Project directory name (e.g., "tickets")
  filename_pattern: "specification-{n}.md"  # Filename template ({n} = spec number)

review:
  save_in_project: false          # Save reviews to project directory
  filename_pattern: "review-{n}.txt"        # Filename template ({n} = review number)

cache:
  enabled: true  # Enable/disable caching globally

github:
  cache:
    disabled: false  # Provider-specific override
```

## Specification Storage

By default, specifications are stored only in the home directory (`~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/specifications/`).

To save specifications in your project directory (for version control):

```yaml
specification:
  save_in_project: true
  project_dir: "tickets"          # Creates tickets/<task-id>/
  filename_pattern: "SPEC-{n}.md" # Creates SPEC-1.md, SPEC-2.md, etc.
```

This creates a dual-storage system:
- **Internal storage** (home directory) - Always maintained as authoritative copy
- **Project storage** (e.g., `tickets/`) - Copy that can be committed to your repo

## Review Storage

Reviews follow the same pattern as specifications:

```yaml
review:
  save_in_project: true
  filename_pattern: "CODERABBIT-{n}.txt"  # Creates CODERABBIT-1.txt, etc.
```

Reviews are stored in the same project directory as specifications (uses `specification.project_dir`).

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
