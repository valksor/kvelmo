# Task History

Browse, search, filter, and resume past tasks from the Task History section.

## Accessing Task History

Open **Task History** from:
- The dashboard **Tasks** block using **Recent → View All**
- Direct navigation to `/history`

Task History provides full list management with search, state filtering, sorting, and direct links to task details.

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

Each task card displays the title, state with colored badge, branch name, creation time, cost, and session count. Action buttons include **View Details**, **Load**, and **Delete**.

## Searching Tasks

Use the search box at the top of the Task History section to find tasks by title.

**Examples:**
- `oauth` - Finds all tasks with "oauth" in title
- `auth` - Finds all tasks with "auth" in title
- `bug` - Finds all bug fix tasks

Search is case-insensitive and matches partial titles.

## Filtering Tasks

Filter tasks by state using the dropdown next to the search box:

| Filter        | Shows                                                       |
|---------------|-------------------------------------------------------------|
| **All**       | Every task in the workspace                                 |
| **Active**    | Currently running tasks (planning, implementing, reviewing) |
| **Completed** | Tasks that finished successfully                            |
| **Failed**    | Tasks that encountered errors                               |
| **Idle**      | Tasks ready for action                                      |

## Sorting Tasks

Change the sort order:

| Sort         | Order                       |
|--------------|-----------------------------|
| **Date**     | Most recent first (default) |
| **Cost**     | Highest cost first          |
| **Duration** | Longest running first       |
| **Name**     | Alphabetical by title       |

## Viewing Task Details

Click **"View Details"** on any task card to open a modal with complete task information:

- **Task ID** - Unique identifier
- **State** - Current workflow state
- **Source** - Original task file
- **Branch** - Git branch (or "deleted after merge")
- **Timeline** - When each phase occurred (Created, Planned, Implemented, Reviewed, Finished)
- **Specifications** - List of spec files created during planning
- **Changes** - Summary of files created, modified, and deleted
- **Cost** - Total cost and token count
- **Sessions** - Number of AI sessions

The modal has **Close**, **Load This Task**, and **Delete** buttons.

## Resuming Past Tasks

Click **"Load"** on any task to make it active again. A confirmation dialog opens explaining what will happen:

- Set it as the active task
- Switch to its branch (if exists)
- Restore all specifications and notes
- Allow you to review or continue work

The dialog notes that if the branch was deleted after finishing, you cannot continue the task. Click **Load Task** to proceed or **Cancel** to close.

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

Click **"Delete"** to remove a task from history. A confirmation dialog warns you that this action:

- Removes task from history
- Deletes work directory
- Removes all checkpoints
- Cannot be undone

Click **Delete** to confirm or **Cancel** to keep the task.

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
- [**CLI: list**](/cli/list.md) - List tasks from command line

---

## Also Available via CLI

Browse and search tasks from the command line.

See [CLI: list](/cli/list.md) for all filters and options.
