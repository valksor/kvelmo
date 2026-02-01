# Task History

Browse, search, filter, and resume past tasks from the Task History section.

## Accessing Task History

The Task History section appears on the dashboard showing all tasks in your workspace:

```
┌──────────────────────────────────────────────────────────────┐
│  Task History                                                │
├──────────────────────────────────────────────────────────────┤
│  🔍 [Search tasks by title...]            Filter: [All ▼]    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 📋 Add User OAuth Authentication     │ [Done]       │      │
│  │ State: Done  Branch: main  Created: 2h ago          │      │
│  │                                    [View] [Load]    │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 📋 Health Check Endpoint            │ [Implementing]│      │
│  │ State: Implementing  Branch: feature/health         │      │
│  │                                    [View] [Load]    │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  Showing 2 of 12 tasks                                        │
└──────────────────────────────────────────────────────────────┘
```

## Task Card Information

Each task card displays:

| Field           | Description                                |
|-----------------|--------------------------------------------|
| **Icon/Emoji**  | Visual indicator for the task              |
| **Title**       | Task name                                  |
| **State Badge** | Current state with color                   |
| **Branch**      | Git branch name (or "deleted" if finished) |
| **Created**     | Time since creation                        |
| **Cost**        | Token cost for this task                   |
| **Sessions**    | Number of AI sessions                      |

```
┌─────────────────────────────────────────────┐
│ 📋 Add User OAuth Authentication          │
│                                             │
│ State: Done         Branch: main            │
│ Created: 2 hours ago                       │
│ $0.45 • 3 sessions                          │
│                                             │
│ [View Details]  [Load]  [Delete]           │
└─────────────────────────────────────────────┘
```

## Searching Tasks

Use the search box to find tasks by title:

```
┌──────────────────────────────────────────────────────────────┐
│  🔍 [Search tasks by title...]            Filter: [All ▼]    │
└──────────────────────────────────────────────────────────────┘
```

**Examples:**
- `oauth` - Finds all tasks with "oauth" in title
- `auth` - Finds all tasks with "auth" in title
- `bug` - Finds all bug fix tasks

Search is case-insensitive and matches partial titles.

## Filtering Tasks

Filter tasks by state:

| Filter        | Shows                                                       |
|---------------|-------------------------------------------------------------|
| **All**       | Every task in the workspace                                 |
| **Active**    | Currently running tasks (planning, implementing, reviewing) |
| **Completed** | Tasks that finished successfully                            |
| **Failed**    | Tasks that encountered errors                               |
| **Idle**      | Tasks ready for action                                      |

```
┌──────────────────────────────────────────────────────────────┐
│  Filter: [All ▼]                                              │
│          ├── All                                              │
│          ├── Active                                           │
│          ├── Completed                                        │
│          ├── Failed                                           │
│          └── Idle                                             │
└──────────────────────────────────────────────────────────────┘
```

## Sorting Tasks

Change the sort order:

| Sort         | Order                       |
|--------------|-----------------------------|
| **Date**     | Most recent first (default) |
| **Cost**     | Highest cost first          |
| **Duration** | Longest running first       |
| **Name**     | Alphabetical by title       |

## Viewing Task Details

Click **"View Details"** on any task card to see:

```
┌──────────────────────────────────────────────────────────────┐
│  Task Details: Add User OAuth Authentication                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Task ID: a1b2c3d4                                           │
│  State: Done                                                 │
│  Source: task.md                                             │
│  Branch: main (deleted after merge)                          │
│                                                              │
│  Timeline:                                                   │
│  • Created: 2 hours ago                                       │
│  • Planned: 2 hours ago (5 min)                              │
│  • Implemented: 2 hours ago (15 min)                         │
│  • Reviewed: 2 hours ago (2 min)                             │
│  • Finished: 2 hours ago                                      │
│                                                              │
│  Specifications (2):                                          │
│  • specification-1.md - OAuth Provider Setup                 │
│  • specification-2.md - Token Validation                     │
│                                                              │
│  Changes:                                                     │
│  • 3 files created                                           │
│  • 2 files modified                                           │
│  • 0 files deleted                                           │
│                                                              │
│  Cost: $0.45 (145,231 tokens)                                │
│  Sessions: 3                                                 │
│                                                              │
│  [Close]  [Load This Task]  [Delete]                         │
└──────────────────────────────────────────────────────────────┘
```

## Resuming Past Tasks

Click **"Load"** on any task to make it active again:

```
┌──────────────────────────────────────────────────────────────┐
│  Load Task                                                   │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Task: Add User OAuth Authentication                         │
│  State: Done                                                 │
│                                                              │
│  Loading this task will:                                     │
│  • Set it as the active task                                 │
│  • Switch to its branch (if exists)                          │
│  • Restore all specifications and notes                       │
│  • Allow you to review or continue work                      │
│                                                              │
│  Note: If the branch was deleted after finishing, you       │
│  cannot continue this task.                                  │
│                                                              │
│                                [Cancel]  [Load Task]         │
└──────────────────────────────────────────────────────────────┘
```

### When to Resume

Resume a task when you need to:
- Review what was done
- Make additional changes
- Reference the specifications
- Copy approach for a similar task

### Task States for Resuming

| Current State    | Can Resume? | What Happens                                   |
|------------------|-------------|------------------------------------------------|
| **Idle**         | ✅ Yes       | Task becomes active, ready for action          |
| **Done**         | ⚠️ Limited  | Can view, but cannot continue (branch deleted) |
| **Failed**       | ✅ Yes       | Can retry from last checkpoint                 |
| **Implementing** | ✅ Yes       | Can continue or cancel                         |

## Deleting Tasks

Click **"Delete"** to remove a task from history:

```
┌──────────────────────────────────────────────────────────────┐
│  Delete Task                                                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Are you sure you want to delete this task?                  │
│                                                              │
│  Task: Add User OAuth Authentication                         │
│                                                              │
│  This will:                                                  │
│  • Remove task from history                                  │
│  • Delete work directory                                    │
│  • Remove all checkpoints                                   │
│  • Cannot be undone                                         │
│                                                              │
│                                [Cancel]  [Delete]           │
└──────────────────────────────────────────────────────────────┘
```

⚠️ **Warning:** Deleting a task is permanent and cannot be undone.

## Task History Actions

| Action     | Description                              |
|------------|------------------------------------------|
| **View**   | See full task details                    |
| **Load**   | Make task active (for idle/failed tasks) |
| **Delete** | Remove task permanently                  |
| **Copy**   | Copy task title/description for new task |

## Working with Multiple Tasks

The task history shows all tasks in your workspace, making it easy to:

- **Switch between tasks** - Load different tasks as needed
- **Track progress** - See status of all your work
- **Find patterns** - Search for similar tasks
- **Review history** - Learn from past implementations

## Next Steps

- [**Dashboard**](dashboard.md) - Return to main view
- [**Creating Tasks**](creating-tasks.md) - Start a new task
- [**CLI: list**](../cli/list.md) - List tasks from command line

## CLI Equivalent

```bash
# List all tasks
mehr list

# List with details
mehr list --verbose

# List by state
mehr list --state done
mehr list --state active

# Search tasks
mehr list --search oauth
```

See [CLI: list](../cli/list.md) for all options.
