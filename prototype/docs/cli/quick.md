# mehr quick

Create a simple task quickly without going through the full project planning workflow.

## Synopsis

```bash
mehr quick <description> [flags]
```

## Description

The `quick` command creates tasks in a lightweight queue for rapid task capture. Unlike the standard workflow which requires planning and specification files, quick tasks are designed for rapid entry and can be:

- **Iterated on** with notes
- **Optimized** by AI for better titles/descriptions
- **Submitted** to external providers (GitHub, Jira, Wrike, etc.)
- **Exported** to markdown files for the standard workflow
- **Started** directly with the standard workflow

Quick tasks are stored in a local queue (`quick-tasks` by default) and can be managed without leaving the terminal.

## Arguments

| Argument      | Description                 |
|---------------|-----------------------------|
| `description` | Task description (required) |

## Flags

| Flag         | Short | Description                                         |
|--------------|-------|-----------------------------------------------------|
| `--title`    |       | Custom task title (auto-extracted from description) |
| `--label`    |       | Task labels (can be specified multiple times)       |
| `--priority` |       | Task priority (1=high, 2=normal, 3=low, default: 2) |
| `--queue`    |       | Target queue ID (default: `quick-tasks`)            |
| `--agent`    |       | Agent to use for this task                          |

## Examples

### Basic Quick Task

```bash
mehr quick "fix typo in README.md line 42"
```

Creates a task with auto-extracted title from the description.

### With Priority

```bash
mehr quick --priority 1 "investigate crash in production"
```

Creates a high-priority task.

### With Labels

```bash
mehr quick --label bug --label urgent "users report login fails"
```

Creates a task with multiple labels.

### With Custom Title

```bash
mehr quick --title "Auth Fix" "users report login fails after password reset"
```

Uses a custom title instead of auto-extracting from the description.

### To Custom Queue

```bash
mehr quick --queue backlog "consider adding dark mode support"
```

Adds the task to a custom queue instead of the default `quick-tasks`.

## What Happens

1. **Task Creation**
   - Task ID auto-generated (e.g., `task-1`, `task-2`)
   - Title extracted from description (or use `--title`)
   - Task saved to queue file in workspace

2. **Queue Storage**
   - Tasks stored in `~/.valksor/mehrhof/workspaces/<project-id>/queues/<queue-id>.yaml`
   - Each queue maintains its own task list
   - Tasks preserve order of creation

3. **Next Steps Prompt**
   - After creation, you'll be prompted for next actions
   - Options: discuss, optimize, submit, start, exit

## Post-Creation Actions

After creating a quick task, you can:

```bash
# Add notes to provide more context
mehr note --task=quick-tasks/task-1 "This affects the authentication flow"

# AI optimizes the task based on accumulated notes
mehr optimize --task=quick-tasks/task-1

# Export to markdown for standard workflow
mehr export --task=quick-tasks/task-1 --output task.md

# Submit to external provider
mehr submit --task=quick-tasks/task-1 --provider github

# Start directly with standard workflow
mehr start queue:quick-tasks/task-1
```

## Workflow Examples

### Capture and Refine Workflow

```bash
# 1. Capture quickly
mehr quick "add search functionality to user list"

# 2. Add requirements
mehr note --task=quick-tasks/task-1 "Support fuzzy matching"
mehr note --task=quick-tasks/task-1 "Add debouncing to avoid excessive API calls"

# 3. AI optimizes the task
mehr optimize --task=quick-tasks/task-1

# 4. Export to proper spec
mehr export --task=quick-tasks/task-1 --output specs/user-search.md

# 5. Start standard workflow
mehr start file:specs/user-search.md
```

### Capture and Submit Workflow

```bash
# 1. Capture quickly
mehr quick --priority 1 --label bug "API returns 500 on empty user list"

# 2. Add details
mehr note --task=quick-tasks/task-1 "Happens when users table is empty"
mehr note --task=quick-tasks/task-1 "Stack trace points to nil pointer in User.FindAll"

# 3. Submit to GitHub
mehr submit --task=quick-tasks/task-1 --provider github --labels critical
```

### Batch Capture Workflow

```bash
# Capture multiple tasks quickly
mehr quick "add user profile page"
mehr quick "implement password reset"
mehr quick "add email notifications"

# Review all tasks
mehr list --queue=quick-tasks

# Work through them one by one
mehr start queue:quick-tasks/task-1
```

## Queue Management

### List Tasks in Queue

```bash
# List all quick tasks
mehr list --queue=quick-tasks

# Or use the project tasks command
mehr project tasks --queue-id=quick-tasks
```

### Delete Tasks

Tasks can be deleted via the Web UI or API. See [REST API Reference](/reference/rest-api.md#quick-tasks) for details.

## Task Priority Levels

| Level  | Flag Value | Description       |
|--------|------------|-------------------|
| High   | `1`        | Urgent, do soon   |
| Normal | `2`        | Standard priority |
| Low    | `3`        | Backlog item      |

## Integration with Standard Workflow

Quick tasks integrate seamlessly with the standard mehrhof workflow:

```bash
# Quick task queue
mehr quick "implement feature X"

# Export to markdown
mehr export --task=quick-tasks/task-1 --output feature-x.md

# Start standard workflow (creates worktree, session, etc.)
mehr start file:feature-x.md
mehr plan
mehr implement
mehr finish
```

## Web UI

Prefer a visual interface? See [Web UI: Quick Tasks](/web-ui/quick-tasks.md).

## See Also

- [optimize](optimize.md) - AI optimize a task
- [note](note.md) - Add notes to a task
- [export](export.md) - Export to markdown
- [submit](submit.md) - Submit to provider
- [start](start.md) - Start from queue reference
- [Queue Provider](/providers/queue.md) - Queue provider documentation
