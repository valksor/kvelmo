# mehr stack

Manage stacked features and dependent branches.

## Synopsis

```bash
mehr stack [command]
```

## Description

Stacked features allow you to work on Feature B while Feature A is waiting on code review. When Feature A merges, Feature B automatically needs rebasing.

This command manages:

1. **Stack Listing** - View all stacks with their status
2. **PR Status Sync** - Fetch latest PR status from provider
3. **Stack Rebase** - Rebase dependent features when parents merge
4. **Graph Visualization** - ASCII and Mermaid diagrams

## Commands

### List Stacks

```bash
mehr stack
```

Shows all stacks with their tasks and states:

```
Stack: auth-system (3 tasks)
  ✓ issue-100  feature/auth-system       merged
  ⟳ issue-101  feature/auth-oauth        needs-rebase (parent merged)
  ● issue-102  feature/auth-oauth-google active (blocked by issue-101)
```

### Sync PR Status

```bash
mehr stack sync
```

Fetches latest PR status from the provider (GitHub, GitLab, etc.):

```
→ Syncing stack pr-status...
  ✓ issue-100: pending-review → merged
  ⚠ issue-101: marked needs-rebase (parent merged)
✓ Synced 2 tasks
```

### Rebase Stack

```bash
# Rebase all stacks that need it
mehr stack rebase

# Rebase specific stack
mehr stack rebase --stack stack-abc123

# Rebase specific task
mehr stack rebase --task issue-101
```

Rebases dependent features in topological order (parents before children):

```
→ Rebasing stack auth-system...
  ✓ issue-101: rebased onto main (was: feature/auth-system)
  ✓ issue-102: rebased onto feature/auth-oauth (was: feature/auth-oauth@old)
✓ Rebased 2 tasks

Next steps:
  1. Run tests on each branch
  2. Push updated branches to remote
  3. Update PRs if needed
```

### Graph Visualization

```bash
# ASCII graph
mehr stack --graph

# Mermaid diagram
mehr stack --mermaid
```

ASCII graph output:

```
Stack: auth-system

    main
      │
      ├── feature/auth-system [issue-100] ✓ merged
      │     │
      │     └── feature/auth-oauth [issue-101] ⟳ needs-rebase
      │           │
      │           └── feature/auth-oauth-google [issue-102] ● active
```

## Stack States

| State            | Icon | Description                    |
|------------------|------|--------------------------------|
| `active`         | ●    | Being worked on                |
| `pending-review` | ◯    | PR open, awaiting review       |
| `approved`       | ◉    | PR approved, ready to merge    |
| `merged`         | ✓    | PR merged to target            |
| `needs-rebase`   | ⟳    | Parent merged, needs rebasing  |
| `conflict`       | ✗    | Rebase failed due to conflicts |
| `abandoned`      | ○    | PR closed without merge        |

## Creating Stacked Features

Use `mehr start` with `--depends-on` to create dependent features:

```bash
# Start feature-b depending on feature-a
mehr start issue-102 --depends-on issue-101
```

Or, if you're on a feature branch:

```bash
# On feature/auth-oauth branch
mehr start issue-102
# Prompt: "You're on feature/auth-oauth (issue-101).
#          Does issue-102 depend on this? [Y/n]"
```

## Conflict Handling

When rebase fails due to conflicts:

```
→ Rebasing stack auth-system...
  ✓ issue-101: rebased onto main
  ✗ issue-102: CONFLICT rebasing onto feature/auth-oauth

⚠ Rebase aborted. Conflict in issue-102:

  Conflicting files:
    - internal/auth/oauth.go
    - internal/auth/handler.go

  To resolve manually:
    cd /path/to/repo
    git checkout feature/auth-oauth-google
    git rebase feature/auth-oauth
    # Resolve conflicts
    git rebase --continue
    mehr stack sync
```

The rebase is atomic: if any task fails, the entire operation aborts and no changes are made.

## Example Workflow

### 1. Start parent feature

```bash
mehr start issue-100
# Creates feature/auth-system from main
```

### 2. Start dependent feature while parent is in review

```bash
# On feature/auth-system branch
mehr start issue-101 --depends-on issue-100
# Creates feature/auth-oauth from feature/auth-system
```

### 3. Check stack status

```bash
mehr stack
# Shows both tasks in the auth-system stack
```

### 4. When parent PR merges

```bash
mehr stack sync
# Detects merge, marks issue-101 as needs-rebase
```

### 5. Rebase dependent features

```bash
mehr stack rebase
# Rebases issue-101 onto main (new target after merge)
```

## Flags

| Flag           | Description                    |
|----------------|--------------------------------|
| `--graph`      | Show ASCII graph visualization |
| `--mermaid`    | Show Mermaid diagram           |
| `--stack <id>` | Target specific stack          |
| `--task <id>`  | Target specific task           |
| `--json`       | Output as JSON                 |

## Configuration

Stack tracking is stored in workspace data:

```
~/.valksor/mehrhof/workspaces/<project>/stacks/index.yaml
```

## See Also

- [mehr start](start.md) - Create a new task (use `--depends-on` for stacked features)
- [mehr sync](sync.md) - Sync task data from provider
- [Stacked Features Concept](../concepts/stacked-features.md) - Architecture and design
- [Web UI: Stack](../web-ui/stack.md) - Web UI for stacks
