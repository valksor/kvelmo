# Dashboard

The dashboard is the main interface for monitoring and managing your tasks.

## Starting the Dashboard

Start the Web UI server:

```bash
# Start server (random port)
mehr serve

# Start on specific port
mehr serve --port 3000

# Start and open browser automatically
mehr serve --open

# Global mode (all projects)
mehr serve --global
```

The dashboard opens at `http://localhost:XXXX`

## Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  Mehrhof                        [Dark Mode Toggle] [Settings]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Workflow State Diagram                                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Active Task Card                                         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────┐  ┌──────────────────────────────────┐    │
│  │  Quick Actions    │  │  Task History                    │    │
│  └───────────────────┘  │  (past tasks)                    │    │
│                         └──────────────────────────────────┘    │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Agent Output (Live)                                      │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────┐  ┌──────────────────────────────────┐    │
│  │  Costs            │  │  Specifications / File Changes   │    │
│  └───────────────────┘  └──────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Workflow State Diagram

At the top of the dashboard, an interactive diagram shows your current workflow state:

```
┌──────────────────────────────────────────────────────────────┐
│  ┌─────┐    plan    ┌───────┐    implement    ┌─────────┐    │
│  │ IDLE│ ──────────>│PLANING│ ──────────────> │IMPLEMENT│    │
│  └─────┘            └───────│                 └────┬────┘    │
│    ▲                        │                      │         │
│    │                        │ finish               │ review  │
│    └────────────────────────┴──────────────────────┴─────────┤
│                         ◀── DONE ◀──                         │
└──────────────────────────────────────────────────────────────┘
```

- **Current state** is highlighted in color
- **Valid transitions** are shown as arrows
- **Updates automatically** as the workflow progresses

This visualization helps you understand where you are and what actions are available.

## Active Task Card

Shows your current task with all key information:

```
┌──────────────────────────────────────────────────────────────┐
│  Active Task: Add User OAuth Authentication                  │
├──────────────────────────────────────────────────────────────┤
│  State: ● Implementing                                       │
│  Branch: feature/user-oauth                                  │
│  Worktree: ~/project-worktrees/a1b2c3d4                      │
│  Progress: ████████░░░░ 80%                                  │
│                                                              │
│  Last Action: "Implement" - 2 minutes ago                    │
│  Next Step: Review                                           │
│                                                              │
│  Actions:                                                    │
│    [Continue] [Plan] [Implement] [Review] [Finish]           │
│                                                              │
│  Cost: $0.45  |  Sessions: 3  |  Checkpoint: 5/7             │
└──────────────────────────────────────────────────────────────┘
```

### Information Shown

| Field           | Description                                       |
|-----------------|---------------------------------------------------|
| **Title**       | Task name                                         |
| **State**       | Current workflow state with color indicator       |
| **Branch**      | Git branch name                                   |
| **Worktree**    | Path if using worktrees                           |
| **Progress**    | Visual progress bar                               |
| **Last Action** | What happened last                                |
| **Next Step**   | Suggested next action                             |
| **Cost**        | Token usage cost                                  |
| **Sessions**    | Number of AI sessions                             |
| **Checkpoint**  | Current checkpoint number                         |
| **Labels**      | Colored badges for task categorization (optional) |

### Action Buttons

| Button        | Purpose                       |
|---------------|-------------------------------|
| **Continue**  | Auto-run next logical step    |
| **Plan**      | Generate specifications       |
| **Implement** | Execute specifications        |
| **Review**    | Run quality checks            |
| **Finish**    | Complete and merge task       |
| **Undo**      | Revert to previous checkpoint |
| **Redo**      | Reapply undone checkpoint     |
| **Abandon**   | Cancel and discard task       |

### Labels

Labels appear as colored badges on the task card for organization and filtering:

```
┌──────────────────────────────────────────────────────────────┐
│  🏷️ priority:high  ×  type:bug  ×  team:backend  [+ Add]     │
└──────────────────────────────────────────────────────────────┘
```

**Managing Labels:**

- **Add labels** - Click **+ Add** button to open label picker with autocomplete
- **Remove labels** - Click **×** on any label badge
- **View all labels** - API endpoint `/api/v1/labels` shows all labels with counts

**Label Colors:**

Colors are hash-based (consistent per label name):
- Same label always has same color
- Visual distinction for easy scanning

**Common Label Patterns:**

| Category | Examples                                                                |
|----------|-------------------------------------------------------------------------|
| Priority | `priority:critical`, `priority:high`, `priority:medium`, `priority:low` |
| Type     | `type:bug`, `type:feature`, `type:refactor`, `type:docs`, `type:test`   |
| Team     | `team:frontend`, `team:backend`, `team:devops`                          |
| Status   | `status:blocked`, `status:in-review`                                    |

### Hierarchical Context

When working on a subtask (e.g., a GitHub issue that's a child of another issue), the active task card shows hierarchical context:

```
┌──────────────────────────────────────────────────────────────┐
│  👤 Parent Task                                              │
├──────────────────────────────────────────────────────────────┤
│  Implement User Authentication System                        │
│                                                              │
│  Add OAuth2 authentication with support for Google and...    │
│                                                              │
│  [View in provider →]                                        │
└──────────────────────────────────────────────────────────────┘
```

And sibling subtasks:

```
┌──────────────────────────────────────────────────────────────┐
│  🔗 Related Subtasks                                         │
├──────────────────────────────────────────────────────────────┤
│  ● Implement OAuth2 Provider Interface                       │
│  ○ Add Token Refresh Logic                                   │
│  ○ Create Login/Logout Endpoints                             │
└──────────────────────────────────────────────────────────────┘
```

**Legend:**
- `●` (green) - Completed
- `◐` (blue) - In progress
- `○` (gray) - Todo
- `✗` (red) - Failed
- `⏸` (yellow) - Paused

This context helps you understand:
- The broader goal (parent task)
- What related work is being done by others (siblings)
- How your subtask fits into the overall plan

See [Context Configuration](../configuration/context.md) to customize hierarchical context inclusion.

## Quick Actions

Context-aware buttons for common tasks:

```
┌──────────────────────────────────────────────────────────────┐
│  Quick Actions                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Task: Add User OAuth Authentication                         │
│  State: Idle                                                 │
│                                                              │
│  Ready for: Implement (2 specifications ready)               │
│                                                              │
│  [Implement] [Undo] [Add Note]                               │
└──────────────────────────────────────────────────────────────┘
```

## Task History

Browse and resume past tasks:

```
┌──────────────────────────────────────────────────────────────┐
│  Task History                                                │
├──────────────────────────────────────────────────────────────┤
│  🔍 [Search tasks...]       Filter: [All ▼]    Sort: [Date ▼]│
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ 📋 Add User OAuth Authentication     │ [Done]       │     │
│  │ State: Done  Branch: main  Created: 2h ago          │     │
│  │ $0.45 • 3 sessions                                  │     │
│  │                                    [View] [Load]    │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ 📋 Health Check Endpoint            │ [Implementing]│     │
│  │ State: Implementing  Branch: feature/health         │     │
│  │ $0.12 • 1 session                                   │     │
│  │                                    [View] [Load]    │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  Showing 2 of 12 tasks                                       │
└──────────────────────────────────────────────────────────────┘
```

See [Task History](task-history.md) for details on filtering and search.

## Agent Output

Real-time streaming of AI responses:

```
┌──────────────────────────────────────────────────────────────┐
│  Agent Output (Live)                                         │
├──────────────────────────────────────────────────────────────┤
│  $ Analyzing codebase structure...                           │
│  $ Reading specification files...                            │
│  ✓ Found 2 specification files to process                    │
│                                                              │
│  → Creating internal/auth/oauth.go                           │
│    • Defined OAuthConfig struct                              │
│    • Added GoogleProvider implementation                     │
│  ✓ Created successfully                                      │
│                                                              │
│  → Modifying internal/auth/middleware.go                     │
│    • Added AuthMiddleware function                           │
│  ✓ Modified successfully                                     │
│                                                              │
│  ▶ Streaming... (scrolls automatically)                      │
└──────────────────────────────────────────────────────────────┘
```

Features:
- **Auto-scroll** - Follows output as it streams
- **Color-coded** - Different colors for different message types
- **Timestamps** - Shows when each action occurred
- **Copy button** - Copy output to clipboard

## Costs Section

Track token usage and costs:

```
┌──────────────────────────────────────────────────────────────┐
│  Costs & Usage                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  This Session: $0.45  |  Tasks: 3  |  Tokens: 145,231        │
│                                                              │
│  Cost by Task:                                               │
│  Add Health Check              $0.12  ███████░░░░            │
│  User OAuth Auth               $0.28  ████████████████       │
│  Fix Login Bug                  $0.05  ██░░░░░░░░░░░░        │
│                                                              │
│  Token Usage (Last 7 Days):                                  │
│  Mon   ████████░░  92K tokens                                │
│  Tue   ██████████  115K tokens                               │
│  Wed   ██████░░░░  78K tokens                                │
│  Thu   ████████░░  95K tokens                                │
│  Fri   ███████░░░  88K tokens                                │
│                                                              │
│  Estimated: $0.68/day  |  Budget: $5.00/day  (13.6% used)    │
└──────────────────────────────────────────────────────────────┘
```

Budget limits show a progress bar and warning state when thresholds are reached.

### Cost Tracking Details

The Costs section provides real-time tracking of token usage and associated costs:

| Metric            | Description                                |
|-------------------|--------------------------------------------|
| **Input Tokens**  | Tokens sent to the AI (prompt + context)   |
| **Output Tokens** | Tokens generated by the AI (response)      |
| **Cached Tokens** | Previously processed tokens (90% discount) |
| **Total Cost**    | USD cost based on agent model pricing      |

**Cost Data Location:** Stored in `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/work.yaml`

**API Access:**
```bash
# Get cost data via API
curl http://localhost:8080/api/v1/costs

# Get task-specific costs
curl http://localhost:8080/api/v1/tasks/{id}/costs
```

For more detailed cost analysis, see [`mehr cost`](../cli/cost.md) in the CLI reference.

## Specifications Section

When specifications exist, shows them with status:

```
┌──────────────────────────────────────────────────────────────┐
│  Specifications (2 files)                                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  📄 specification-1.md                                       │
│     ✓ OAuth Provider Setup                                   │
│     ✓ Database Schema for Sessions                           │
│     ✓ Login/Logout Endpoints                                 │
│     [View Full Content]                                      │
│                                                              │
│  📄 specification-2.md                                       │
│     ✓ Token Validation Middleware                            │
│     ✓ Session Management                                     │
│     ✓ Security Considerations                                │
│     [View Full Content]                                      │
│                                                              │
│  [+ Add another specification]                               │
└──────────────────────────────────────────────────────────────┘
```

## File Changes Section

After implementation, shows what changed:

```
┌───────────────────────────────────────────────────────────────────┐
│  File Changes (5 files)                                           │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ▼ internal/auth/oauth.go                 [+ Created, 87 lines]   │
│  ▼ internal/auth/middleware.go            [+ Modified, 23 lines]  │
│  ▼ internal/auth/oauth_test.go            [+ Created, 54 lines]   │
│  ▼ cmd/server/main.go                     [+ Modified, 8 lines]   │
│  ▼ go.lock                                [+ Modified, 2 lines]   │
│                                                                   │
│  [View All Diffs]                                                 │
└───────────────────────────────────────────────────────────────────┘
```

Click any file to see the diff view.

## Dashboard Features

### Dark Mode

Toggle between light and dark themes using the button in the top-right.

### Mobile Responsive

Full feature support on mobile devices with:
- Collapsible navigation (hamburger menu)
- Touch-friendly controls
- Stacked layouts on small screens

### Real-Time Updates

All dashboard sections update automatically via Server-Sent Events (SSE):
- Task state changes
- Agent output streaming
- Cost updates
- New checkpoints

### Notifications

Stay informed with:
- Toast notifications for operation updates
- Browser notifications for task completion (requires permission)
- Notification center showing recent alerts

## Next Steps

- [**Getting Started**](getting-started.md) - First time walkthrough
- [**Creating Tasks**](creating-tasks.md) - Create your first task
- [**Settings**](settings.md) - Configure workspace
- [**Context Configuration**](../configuration/context.md) - Hierarchical task context settings
- [**CLI: serve**](../cli/serve.md) - Server command options
