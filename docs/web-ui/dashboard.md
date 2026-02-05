# Dashboard

The dashboard is the main landing page for the Web UI, providing quick access to task creation, active task status, and recent work.

## Accessing the Dashboard

Once the Web UI server is running, open your browser to the displayed URL (typically `http://localhost:XXXX`). The dashboard is the default landing page.

## Dashboard Modes

The dashboard operates in two modes depending on how the server was started.

### Project Mode (Default)

When the server is started from a project directory, the dashboard shows:
- **Active Task Summary** — Current task status with a link to full details
- **Task Creation** — Tabs for starting tasks (Quick Start, Quick Tasks, Planning)
- **Budget Overview** — Cost tracking (if budget is enabled)
- **Recent Tasks** — Your recent task history with quick access

### Global Mode

When the server is started in global mode, the dashboard shows a project selector:
- List of all registered projects
- Project paths and statistics
- Click any project to view its dashboard

## Dashboard Layout

### Active Task Summary

When you have an active task, a summary card appears at the top showing:

| Field | Description |
|-------|-------------|
| **Title** | Task name (click to view full details) |
| **State** | Current workflow state (planning, implementing, etc.) |
| **Branch** | Git branch for this task |

Click **View Task** or the task title to open the Task Detail page with full workflow controls.

### Task Creation Tabs

Three tabs for starting new work:

| Tab | Purpose |
|-----|---------|
| **Start** | Begin a new task from a reference (file path, GitHub issue, Jira ticket) |
| **Quick** | Manage quick tasks queue — lightweight tasks for rapid execution |
| **Plan** | Create project plans from requirements or specifications |

### Budget Overview

If monthly budget tracking is enabled in [Settings](/web-ui/settings.md), this section shows:
- Current month spending
- Budget limit and remaining
- Warning indicators when approaching limits

### Recent Tasks

A list of your recent tasks with:
- Task title and state
- Creation date
- Quick links to view or resume

See [Task History](/web-ui/task-history.md) for full filtering and search capabilities.

## Task Detail Page

**For active task management, click a task to access the Task Detail page.** This page provides:

- **Workflow Actions** — Buttons for Plan, Implement, Review, Finish, Undo, Redo, Abandon
- **Workflow Diagram** — Visual state machine showing current progress
- **Specifications** — View generated specs with expandable content
- **Reviews** — Code review results and issues
- **Notes** — Add context notes for the AI agent
- **Agent Terminal** — Real-time streaming of AI responses
- **Costs** — Token usage and cost tracking
- **Quick Questions** — Ask the agent questions without changing workflow state

This separation keeps the dashboard lightweight while providing full control on the task detail page.

## Connection Status

The dashboard header shows your real-time connection status:
- **Connected** (green) — SSE connection active, receiving live updates
- **Reconnecting** (yellow) — Connection lost, attempting to reconnect

Real-time updates include:
- Task state changes
- Agent activity indicators
- Cost updates
- New checkpoint notifications

## Dashboard Features

### Dark Mode

Toggle between light and dark themes using the button in the top-right corner.

### Mobile Responsive

Full feature support on mobile devices with:
- Collapsible navigation (hamburger menu)
- Touch-friendly controls
- Stacked layouts on small screens

### Notifications

Stay informed with:
- Toast notifications for operation updates
- Browser notifications for task completion (requires permission)
- Notification center showing recent alerts

---

## Also Available via CLI

Prefer working from the terminal? See [CLI: serve](/cli/serve.md) for server options and flags.

## Next Steps

- [**Getting Started**](/web-ui/getting-started.md) — First time walkthrough
- [**Creating Tasks**](/web-ui/creating-tasks.md) — Create your first task
- [**Settings**](/web-ui/settings.md) — Configure workspace
- [**Task History**](/web-ui/task-history.md) — Browse past tasks
