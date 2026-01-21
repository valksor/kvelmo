# Git Configuration

Controls version control integration.

## Settings

```yaml
git:
  auto_commit: true
  commit_prefix: "[{key}]"
  branch_pattern: "{type}/{key}--{slug}"
  sign_commits: false
  stash_on_start: false
  auto_pop_stash: true
```

| Setting | Default | Description |
|---------|---------|-------------|
| `auto_commit` | `true` | Auto-commit after operations |
| `commit_prefix` | `[{key}]` | Commit message prefix template |
| `branch_pattern` | `{type}/{key}--{slug}` | Branch naming template |
| `sign_commits` | `false` | GPG-sign commits |
| `stash_on_start` | `false` | Auto-stash changes before creating task branch |
| `auto_pop_stash` | `true` | Auto-pop stash after branch creation |

## Stash Behavior

When `stash_on_start` is enabled, Mehrhof automatically stashes uncommitted changes (including untracked files) before creating a new task branch. The `auto_pop_stash` setting controls whether the stash is automatically restored:

- `auto_pop_stash: true` (default) - Stash is automatically restored after branch creation
- `auto_pop_stash: false` - Stash is preserved for manual restoration (use `git stash pop`)

This is useful when you have work-in-progress changes that aren't ready to commit.

```yaml
git:
  stash_on_start: true  # Auto-stash changes before creating branch
  auto_pop_stash: true  # Auto-pop stash after branch (default: true)
  # Set to false to preserve stash for manual restoration
```

See [`mehr start --stash`](../cli/start.md#start-with-stash-uncommitted-changes) for CLI usage.

## Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{key}` | External key (from filename/frontmatter) | `FEATURE-123` |
| `{task_id}` | Internal task ID | `a1b2c3d4` |
| `{type}` | Task type from filename prefix | `feature`, `fix` |
| `{slug}` | URL-safe slugified title | `add-user-auth` |

## See Also

- [Checkpoints](../concepts/checkpoints.md) - How commits are created
- [Configuration Overview](index.md) - All configuration options
