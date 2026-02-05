# Storage Configuration

Controls where task data and caches are stored.

## Settings

```yaml
storage:
  save_in_project: false          # Store work in project (default: false = global)
  project_dir: ""                 # Project directory (default: ".mehrhof/work" when save_in_project=true)

specification:
  filename_pattern: "specification-{n}.md"  # Filename template ({n} = spec number)

review:
  filename_pattern: "review-{n}.txt"        # Filename template ({n} = review number)

cache:
  enabled: true  # Enable/disable caching globally

github:
  cache:
    disabled: false  # Provider-specific override
```

## Work Storage Location

By default, all work files (specs, reviews, sessions) are stored in the home directory (`~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/`).

To store work in your project directory (for version control):

```yaml
storage:
  save_in_project: true
  project_dir: "tickets"          # Creates tickets/<task-id>/
```

**Storage locations:**

| Config                                             | Work Location                                            |
|----------------------------------------------------|----------------------------------------------------------|
| `save_in_project: false`                           | `~/.valksor/mehrhof/workspaces/<name>/work/<taskid>/...` |
| `save_in_project: true`                            | `.mehrhof/work/<taskid>/...`                             |
| `save_in_project: true` + `project_dir: "tickets"` | `tickets/<taskid>/...`                                   |

## Filename Patterns

Customize filenames for specs and reviews (location is controlled by `storage.save_in_project`):

```yaml
specification:
  filename_pattern: "SPEC-{n}.md"         # Creates SPEC-1.md, SPEC-2.md, etc.

review:
  filename_pattern: "CODERABBIT-{n}.txt"  # Creates CODERABBIT-1.txt, etc.
```

Both specs and reviews are stored in the same task directory.

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

| Git Remote URL                              | Project ID                          |
|---------------------------------------------|-------------------------------------|
| `https://github.com/user/repo`              | `github.com-user-repo`              |
| `git@github.com:user/project.git`           | `github.com-user-project`           |
| `https://gitlab.com/group/subgroup/project` | `gitlab.com-group-subgroup-project` |
| No remote (local)                           | `local-<hash>`                      |

## See Also

- [Storage Structure Reference](/reference/storage.md) - Complete storage documentation
- [Configuration Overview](index.md) - All configuration options
