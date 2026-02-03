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
# Preview what would happen (check for conflicts)
mehr stack rebase --preview

# Dry run (alias for --preview)
mehr stack rebase --dry-run

# Rebase all stacks that need it (prompts for confirmation)
mehr stack rebase

# Skip confirmation (for scripts/automation)
mehr stack rebase --yes

# Rebase specific stack
mehr stack rebase --stack stack-abc123

# Rebase specific task
mehr stack rebase --task issue-101
```

The rebase command always checks for conflicts before executing. If conflicts are detected, the rebase is blocked with clear instructions for manual resolution.

**Preview output:**

```
Stack: auth-system
┌──────────────────────────────────────────────────────┐
│ feature/auth-oauth    → main           ✓ Safe
│ feature/auth-google   → auth-oauth     ✗ CONFLICT
│   └─ internal/auth/oauth.go
│   └─ internal/auth/handler.go
└──────────────────────────────────────────────────────┘

⚠ 1 task(s) have conflicts. Resolve manually before rebasing.
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

### List Flags

| Flag           | Description                    |
|----------------|--------------------------------|
| `--graph`      | Show ASCII graph visualization |
| `--mermaid`    | Show Mermaid diagram           |
| `--json`       | Output as JSON                 |

### Rebase Flags

| Flag           | Description                                      |
|----------------|--------------------------------------------------|
| `--preview`    | Preview what would happen (check for conflicts)  |
| `--dry-run`    | Alias for `--preview`                            |
| `--yes`, `-y`  | Skip confirmation prompt (for scripts)           |

### Rebase Arguments

| Argument       | Description                                      |
|----------------|--------------------------------------------------|
| `[task-id]`    | Optional: rebase a specific task only            |

## Configuration

Stack tracking is stored in workspace data:

```
~/.valksor/mehrhof/workspaces/<project>/stacks/index.yaml
```

### Auto-Rebase on Finish

Configure automatic rebase when finishing a parent task:

```yaml
# .mehrhof/config.yaml
stack:
  auto_rebase: on_finish    # "disabled" (default) | "on_finish"
  block_on_conflicts: true  # Always block if conflicts detected (default)
```

When enabled, `mehr finish` will:
1. Create the PR for the finished task
2. Preview rebase for any dependent tasks
3. If no conflicts: prompt for confirmation
4. If conflicts: warn and skip (manual resolution required)

**Requirements:**
- Git 2.38+ for conflict detection (`git merge-tree --write-tree`)
- Interactive mode (auto-rebase requires user confirmation per Tier 3 policy)

If Git is older than 2.38, conflict detection will be unavailable but manual rebase still works.

## Web UI

Prefer a visual interface? See [Web UI: Stacked Features](/web-ui/stack.md).

## See Also

- [mehr start](start.md) - Create a new task (use `--depends-on` for stacked features)
- [mehr sync](sync.md) - Sync task data from provider
- [Stacked Features Concept](/concepts/stacked-features.md) - Architecture and design
