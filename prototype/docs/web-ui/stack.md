# Stacked Features

The stack page allows you to manage dependent features and their git branches through the Web UI.

## Accessing the Stack Page

From the navigation bar, click **Tools**, then select the **Stack** tab.

For a complete overview of the Tools page, see [Tools](/web-ui/tools.md).

## Stack Overview

The stack page shows all feature stacks with their tasks and status. Each stack card displays the stack name, task count, root task reference, and a list of tasks with their branch names and status indicators. A **Sync PR Status** button appears at the top to refresh status from providers. Stack cards show timestamps for creation and last update, and action buttons like **Needs Rebase** warning and **Rebase All** when applicable.

## Task States

Each task displays its current state with a visual indicator:

| Icon | State          | Color  | Description                    |
|------|----------------|--------|--------------------------------|
| ✓    | merged         | Green  | PR merged to target            |
| ⟳    | needs-rebase   | Yellow | Parent merged, needs rebasing  |
| ✗    | conflict       | Red    | Rebase failed due to conflicts |
| ◯    | pending-review | Blue   | PR open, awaiting review       |
| ◉    | approved       | Green  | PR approved, ready to merge    |
| ●    | active         | Gray   | Being worked on                |
| ○    | abandoned      | Gray   | PR closed without merge        |

## Syncing PR Status

Click **"Sync PR Status"** to fetch the latest PR status from your provider. A loading indicator shows during sync, and upon completion a success message shows how many tasks were updated.

The sync operation:
1. Fetches PR status for all stacked tasks
2. Updates states (pending-review → merged, etc.)
3. Marks children as needs-rebase when parents merge

## Rebasing Stacks

When a parent feature merges, its children need rebasing. Click **"Preview Rebase"** on a stack card to see what will happen before performing the rebase.

### Preview Modal

Clicking **"Preview Rebase"** opens a modal showing the conflict status. If no conflicts are detected, it shows how many tasks can be safely rebased and lists each branch with its target. Action buttons include **Cancel** and **Rebase Now**.

If conflicts are detected, the modal shows which tasks have conflicts and lists the conflicting files. The **Rebase Now** button is disabled until conflicts are resolved manually.

### Rebase Progress

During rebase, a progress panel shows real-time status updates for each task being rebased, with checkmarks for completed tasks and spinners for in-progress tasks.

### Rebase Success

When rebase completes successfully, a success message shows which tasks were rebased and to which branches. Next steps are suggested: push updated branches and update PRs if needed.

### Rebase Conflict

If a conflict occurs, the rebase aborts and shows conflict details including the task, target branch, conflict hint (which file), and step-by-step resolution instructions. A **Copy Resolution Steps** button helps you copy the git commands needed to resolve manually.

## Task Details

Click on a task to see more details including the branch name, state, PR link, dependencies (what it depends on), blocking relationships (what it blocks), and which stack it belongs to. Action buttons include **View PR** and **Rebase This Task**.

## Empty State

When no stacks exist, the page explains how to create dependent features using `mehr start <task> --depends-on <parent>` or by starting a task while on a feature branch to be prompted about creating a dependency.

## Status Badges

Stacks display status badges for quick visibility:

| Badge            | Meaning                          |
|------------------|----------------------------------|
| `⚠ Needs Rebase` | One or more tasks need rebasing  |
| `✗ Conflict`     | One or more tasks have conflicts |
| `✓ All Merged`   | All tasks in stack are merged    |

## Help Section

The page includes a help section explaining stacked features: how they allow you to work on Feature B while Feature A is waiting on code review, how to create dependent features, sync PR status, and rebase after parent merges. It also points to `mehr stack` in the CLI for more options including `--graph` and `--mermaid` visualization.

## API Endpoints

The stack page uses these API endpoints:

| Method | Endpoint                       | Description                      |
|--------|--------------------------------|----------------------------------|
| GET    | `/api/v1/stack`                | List all stacks                  |
| POST   | `/api/v1/stack/sync`           | Sync PR status                   |
| GET    | `/api/v1/stack/rebase-preview` | Preview rebase (check conflicts) |
| POST   | `/api/v1/stack/rebase`         | Rebase stacks/tasks              |

### Rebase Preview API

The preview endpoint accepts optional query parameters:

```
GET /api/v1/stack/rebase-preview?stack_id=<id>    # Preview specific stack
GET /api/v1/stack/rebase-preview?task_id=<id>     # Preview specific task
GET /api/v1/stack/rebase-preview                   # Preview all stacks
```

Returns JSON.

---

## Also Available via CLI

Manage stacked features from the command line for scripting or terminal workflows.

See [CLI: stack](/cli/stack.md) for visualization options and all flags.

## See Also

- [Stacked Features Concept](/concepts/stacked-features.md) - Architecture and design
- [CLI: stack](/cli/stack.md) - CLI equivalent
- [Settings](settings.md) - Configure stack behavior
