# Labels

Organize and filter tasks using custom labels for categorization, filtering, and team coordination.

## Overview

Labels are free-form tags that help you:

- **Organize** tasks by priority, type, or team
- **Filter** task lists to find relevant work
- **Group** related tasks together
- **Track** metadata for reporting

## Accessing Labels in the Web UI

Labels are available in multiple locations:

| Feature               | Location                     |
|-----------------------|------------------------------|
| **View Task Labels**  | Task Detail page (sidebar)   |
| **Edit Labels**       | Task Detail → Labels card    |
| **Filter by Labels**  | Task List → Filter dropdown  |

## Managing Labels

### Viewing Labels

Labels appear in the **Labels card** on any task detail page. Each label is displayed as a colored tag.

### Adding Labels

To add labels to a task:

1. Open the task detail page
2. Find the **Labels** card in the sidebar
3. Click the **Add** button or the **+** icon
4. Enter one or more labels (separated by commas or spaces)
5. Click **Save**

**Tip:** Use a consistent naming convention like `priority:high` or `type:bug` for easy filtering.

### Removing Labels

To remove a label:

1. Open the task detail page
2. Find the **Labels** card
3. Click the **X** on the label you want to remove

Or use the **Edit** mode to remove multiple labels at once.

### Replacing All Labels

To replace all labels with a new set:

1. Open the task detail page
2. Click **Edit** on the Labels card
3. Clear existing labels and enter new ones
4. Click **Save**

### Clearing All Labels

To remove all labels from a task:

1. Open the Labels card
2. Click **Clear All**
3. Confirm the action

## Filtering Tasks by Label

### In Task List

Use the filter dropdown to show only tasks with specific labels:

1. Go to **Tasks** from the navigation
2. Click the **Filter** dropdown
3. Select one or more labels to filter by
4. Tasks matching your selection appear below

### Filter Options

| Filter Type     | Behavior                              |
|-----------------|---------------------------------------|
| **Any label**   | Tasks matching any selected label     |
| **All labels**  | Tasks matching all selected labels    |
| **No labels**   | Tasks without any labels              |

## Label Conventions

While labels are free-form, common patterns include:

| Category      | Examples                                 | Purpose         |
|---------------|------------------------------------------|-----------------|
| **Priority**  | `priority:high`, `priority:low`          | Task urgency    |
| **Type**      | `type:bug`, `type:feature`, `type:docs`  | Work category   |
| **Team**      | `team:frontend`, `team:backend`          | Team ownership  |
| **Status**    | `status:blocked`, `status:in-review`     | Workflow status |
| **Component** | `component:auth`, `component:api`        | System affected |
| **Sprint**    | `sprint:2024-q1`, `sprint:backlog`       | Sprint planning |

## Best Practices

1. **Use consistent naming** - Agree on label format with your team (e.g., `category:value`)
2. **Keep labels meaningful** - Each label should serve a purpose
3. **Review periodically** - Clean up unused or outdated labels
4. **Combine with filters** - Use labels with the task list filters for powerful organization

---

## Also Available via CLI

For power users who prefer command-line access:

See [CLI: label](/cli/label.md) for all label commands and filtering options.
