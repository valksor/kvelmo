# mehr list

List all tasks in the workspace.

## Synopsis

```bash
mehr list [flags]
```

## Description

The `list` command displays all tasks in the workspace with their worktree paths and states. This is particularly useful when running multiple parallel tasks across different terminals.

**Features:**

- Shows all tasks regardless of which directory you're in
- Displays worktree paths for parallel task management
- Indicates which task is active in the main repo (`*`)
- Indicates which worktree you're in (`→`)

## Flags

| Flag          | Short | Type   | Default | Description                                        |
|---------------|-------|--------|---------|----------------------------------------------------|
| `--search`    |       | string | ""      | Search tasks by title or description               |
| `--filter`    |       | string | ""      | Filter tasks (format: key:value, e.g., state:done) |
| `--sort`      |       | string | ""      | Sort tasks (date, cost, duration)                  |
| `--format`    |       | string | "table" | Output format (table, json, csv)                   |
| `--worktrees` | `-w`  | bool   | false   | Show only tasks with worktrees                     |
| `--running`   |       | bool   | false   | Show running parallel tasks (in-memory)            |
| `--json`      |       | bool   | false   | Output as JSON (deprecated, use --format json)     |

## Examples

### List All Tasks

```bash
mehr list
```

Output:

```
TASK ID     STATE           TITLE                    WORKTREE                         ACTIVE
a1b2c3d4    implementing    Add authentication       ../project-worktrees/a1b2c3d4    →
e5f6g7h8    planning        Fix database queries     ../project-worktrees/e5f6g7h8
c9d0e1f2    idle            Update config            -                                *
f3g4h5i6    done            Refactor logging         -

Legend: * = active task in main repo, → = current worktree
```

### Search Tasks

```bash
mehr list --search "auth"
```

Shows tasks matching "auth" in title or description.

### Filter Tasks

```bash
# Filter by state
mehr list --filter state:done

# Filter by cost (greater than)
mehr list --filter cost:>1.0

# Combine filters
mehr list --filter state:implementing --filter cost:>0.5
```

### Sort Tasks

```bash
# Sort by date (newest first)
mehr list --sort date

# Sort by cost (highest first)
mehr list --sort cost

# Sort by duration (longest first)
mehr list --sort duration
```

### List Only Worktree Tasks

```bash
mehr list --worktrees
```

Output:

```
TASK ID     STATE           TITLE                    WORKTREE                         ACTIVE
a1b2c3d4    implementing    Add authentication       ../project-worktrees/a1b2c3d4    →
e5f6g7h8    planning        Fix database queries     ../project-worktrees/e5f6g7h8

Legend: * = active task in main repo, → = current worktree
```

### List Running Parallel Tasks

When tasks are running in parallel goroutines (via `mehr start --parallel`), use `--running` to view them:

```bash
mehr list --running
```

Output:

```
ID       REFERENCE       STATUS     TASK ID     DURATION   WORKTREE
abc123   file:a.md       running    task-001    5m30s      ../worktrees/abc123
def456   file:b.md       running    task-002    5m28s      ../worktrees/def456
ghi789   file:c.md       completed  task-003    4m15s      ../worktrees/ghi789

2 running, 3 total
```

**Output Columns for `--running`:**

| Column    | Description                                          |
|-----------|------------------------------------------------------|
| ID        | Running task ID (for `--running` flag on other cmds) |
| REFERENCE | Task reference (e.g., file:a.md, github:123)         |
| STATUS    | pending, running, completed, failed, or cancelled    |
| TASK ID   | Internal task ID once started                        |
| DURATION  | Time elapsed since task started                      |
| WORKTREE  | Path to worktree (if using parallel isolation)       |

### CSV Output

```bash
mehr list --format csv
```

Output:

```csv
Task ID,State,Title,Worktree,Active,Cost
a1b2c3d4,implementing,Add authentication,../project-worktrees/a1b2c3d4,,1.235
e5f6g7h8,planning,Fix database queries,../project-worktrees/e5f6g7h8,*,0.450
```

### JSON Output

For programmatic access, use `--json`:

```bash
mehr list --json
```

Output:

```json
[
  {
    "task_id": "a1b2c3d4",
    "state": "implementing",
    "title": "Add authentication",
    "worktree_path": "../project-worktrees/a1b2c3d4",
    "is_active": false,
    "is_current_worktree": true
  },
  {
    "task_id": "c9d0e1f2",
    "state": "idle",
    "title": "Update config",
    "worktree_path": "",
    "is_active": true,
    "is_current_worktree": false
  }
]
```

## Output Columns

| Column   | Description                                                 |
|----------|-------------------------------------------------------------|
| TASK ID  | Unique 8-character task identifier                          |
| STATE    | Current workflow state (idle, planning, implementing, etc.) |
| TITLE    | Task title from source file                                 |
| WORKTREE | Path to worktree, or `-` if none                            |
| ACTIVE   | `*` for active task in main repo, `→` for current worktree  |

## Use Cases

### Managing Parallel Tasks

When working on multiple features simultaneously:

```bash
mehr list

cd ../project-worktrees/a1b2c3d4
mehr status
```

### Finding Your Worktrees

```bash
mehr list --worktrees
```

Shows only tasks with worktrees, making it easy to navigate between parallel tasks.

### From Any Location

The `list` command works from:

- Main repository
- Any worktree
- Any subdirectory within the project

It always shows all tasks in the workspace.

## Related Commands

- [status](cli/status.md) - Show detailed status of current task
- [start](cli/start.md) - Start a new task (with `--worktree` for parallel)
- [finish](cli/finish.md) - Complete a task and clean up worktree

## Web UI

Prefer a visual interface? See [Web UI: Task History](/web-ui/task-history.md).

## See Also

- [Parallel Tasks](/README.md#parallel-tasks) - Overview of parallel task workflow
