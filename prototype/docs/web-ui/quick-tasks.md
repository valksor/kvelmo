# Quick Tasks

Quick tasks provide a lightweight way to capture and manage tasks without going through the full planning workflow. Perfect for rapid task entry, quick fixes, and backlog items.

## Overview

Quick tasks are stored in a local queue and can be:
- **Iterated on** with notes
- **Optimized** by AI for better titles/descriptions
- **Submitted** to external providers (GitHub, Jira, Wrike, etc.)
- **Exported** to markdown files for the standard workflow
- **Started** directly with the standard workflow
- **Deleted** when no longer needed

## Accessing in the Web UI

Navigate to the quick tasks page at:

```
http://localhost:PORT/quick
```

Or click **"Quick Tasks"** from the main dashboard.

## Using the Quick Tasks Interface

### Creating a Quick Task

1. Go to the quick tasks page (`/quick`)
2. Enter your task description in the input field
3. Optionally add:
   - **Title** - Custom task title (auto-extracted from description if empty)
   - **Labels** - Task labels for categorization
   - **Priority** - 1 (high), 2 (normal), 3 (low)
4. Click **"Create Task"**

The task will be added to the queue with an auto-generated ID (e.g., `task-1`, `task-2`).

### Submitting From a Source

You can submit a task directly from a file or directory without pre-creating it:

1. Open the **Submit From Source** card
2. Enter a file path, directory path, or provider reference
3. Choose a provider (GitHub, Jira, Wrike, etc.)
4. Optionally add:
   - **Notes** to guide the task drafting
   - **Instructions** for how to interpret the source
   - **Labels** to apply
   - **Optimize** to refine the task before submission
5. Click **Submit**

The task is created in your quick task queue and submitted to the provider.

### Viewing Tasks

The quick tasks page displays all tasks in your queue:

| Field | Description |
|-------|-------------|
| ID | Auto-generated task identifier |
| Title | Task title (extracted or custom) |
| Description | Full task description |
| Priority | 1 (high), 2 (normal), 3 (low) |
| Labels | Categorization tags |
| Status | Current task state |

### Adding Notes

Add context or requirements to a task:

1. Find the task in the queue
2. Click **"Add Note"**
3. Enter your additional context
4. Click **"Save"**

Notes help the AI optimize the task and provide better context when you start working on it.

### Optimizing Tasks

Let AI refine the task title and description:

1. Find the task in the queue
2. Click **"Optimize"**
3. The AI will analyze accumulated notes and improve the task definition

**What optimization does:**
- Improves title clarity
- Enhances description specificity
- Organizes requirements
- Suggests implementation considerations

**Agent Configuration:**

The optimization uses the agent configured for the `optimizing` step. Configure in `.mehrhof/config.yaml`:

```yaml
agent:
  steps:
    optimizing:
      name: claude-opus  # Use a specific agent for task optimization
```

For CLI usage with explicit agent override, see [CLI optimize documentation](../cli/optimize.md).

### Exporting to Markdown

Convert a quick task to a markdown file for the standard workflow:

1. Find the task in the queue
2. Click **"Export"**
3. Choose an output filename (or use the default)
4. The task is saved as a `.md` file

You can then start the standard workflow with:
```bash
mehr start file:your-task.md
```

### Submitting to Providers

Submit quick tasks to external task providers (GitHub, Jira, Wrike, etc.):

1. Find the task in the queue
2. Click **"Submit"**
3. Choose the target provider
4. Configure submission options:
   - Labels
   - Priority
   - Assignee (if supported)
5. Click **"Submit"**

The task is created in the external system and linked back to your queue.

### Starting a Task

Start the standard Mehrhof workflow directly from a quick task:

1. Find the task in the queue
2. Click **"Start"**
3. The task is loaded into the active workflow
4. Proceed with planning, implementation, and completion

### Deleting Tasks

Remove tasks from the queue:

1. Find the task in the queue
2. Click **"Delete"**
3. Confirm the deletion

**Note:** Deleted tasks cannot be recovered. Export important tasks before deleting.

## Task Queue Storage

Quick tasks are stored in:
```
~/.valksor/mehrhof/workspaces/<project-id>/queues/<queue-id>.yaml
```

The default queue ID is `quick-tasks`. You can create multiple queues for different purposes.

## Common Workflows

### Capture and Refine

```
1. Capture quickly: "fix login bug"
2. Add context: "happens after password reset"
3. Optimize: AI refines the task
4. Export: Save as markdown
5. Start: Begin standard workflow
```

### Rapid Backlog Entry

```
1. Enter multiple tasks quickly
2. Set priorities as you go
3. Add labels for organization
4. Export highest priority tasks
5. Work through them systematically
```

### Provider Submission

```
1. Capture tasks from meeting
2. Add notes for requirements
3. Optimize for clarity
4. Submit to GitHub/Jira
5. Track in external system
```

## CLI Equivalent

See [`mehr quick`](../cli/quick.md) for CLI usage.

| CLI Command | Web UI Action |
|-------------|---------------|
| `mehr quick "description"` | Create task |
| `mehr note --task=... "text"` | Add note |
| `mehr optimize --task=...` | Optimize task |
| `mehr export --task=...` | Export to markdown |
| `mehr submit --task=...` | Submit to provider |
| `mehr start queue:...` | Start task |

## Managing Multiple Queues

You can work with multiple task queues:

```bash
# CLI: Create task in specific queue
mehr quick --queue backlog "future feature"

# Web UI: Select queue from dropdown
```

Switch between queues using the queue selector at the top of the quick tasks page.
