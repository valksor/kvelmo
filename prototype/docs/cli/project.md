# Project Planning Commands

The project commands provide a complete workflow for planning and executing multi-task projects with dependencies.

## Overview

```bash
mehr project plan <source>     # Create task breakdown from source
mehr project tasks             # List tasks in queue
mehr project edit <task-id>    # Edit task properties
mehr project reorder           # Reorder tasks
mehr project submit            # Submit tasks to provider
mehr project start             # Start implementing
```

## Commands

### plan {#plan}

Create a task breakdown from a source (directory, file, or provider reference).

```bash
mehr project plan <source> [flags]
```

**Arguments:**
- `source` - Source input:
  - `dir:/path` - Directory of files to scan (reads all content)
  - `file:spec.md` - Single file
  - `research:/path` - Directory for AI to research (agent explores selectively, best for large docs)
  - `github:123`, `jira:PROJ-123` - Provider reference (fetches task details from provider)
  - URL - Fetch requirements from a URL (via web API only)

**Flags:**
| Flag         | Short | Description                                    |
| ------------ | ----- | ---------------------------------------------- |
| `--title`     | `-t`  | Project/epic title                             |
| `--queue-id`   |       | Specific queue ID (default: auto)            |
| `--use-schema` |       | Use schema-driven extraction (default: true)    |
| `--instructions`|       | Custom instructions for AI                     |

**Examples:**

```bash
# Plan from a directory of specs (reads all files)
mehr project plan dir:/workspace/specs/

# Plan from large documentation base (agent explores selectively)
mehr project plan research:/workspace/.final/ --title "Reports System"

# Plan from a single file
mehr project plan file:requirements.md --title "Q1 Features"

# Plan from a provider reference
mehr project plan github:123

# Plan with custom instructions
mehr project plan dir:/workspace/specs/ --instructions "Focus on API design first, implementation second"

# Use regex-only parsing (faster, less flexible)
mehr project plan dir:/workspace/specs/ --use-schema=false
```

**Source Type Differences:**

| Source    | Description                              | Best For                    |
|-----------|------------------------------------------|-----------------------------|
| `dir:`     | Reads ALL file contents into prompt    | <50 files, small codebases |
| `research:`| Provides file manifest, AI uses tools  | Large docs, existing tasks |
| `file:`    | Single file analysis                   | Individual requirement files |
| `provider:`| Fetches from external task provider    | GitHub/Jira/Wrike issues      |

The AI will analyze the source and produce a structured task breakdown with dependencies.

**Schema-Driven Extraction:**

By default, `project plan` uses **schema-driven extraction** to parse AI responses. This provides:
- **Flexible parsing**: Handles varied AI output formats without breaking
- **JSON validation**: Ensures extracted data matches expected structure
- **Automatic fallback**: Falls back to regex parsing if schema extraction fails

To use regex-only parsing (faster but less flexible):

```bash
mehr project plan dir:/workspace/specs/ --use-schema=false
```

Use `--use-schema=false` if:
- You need faster parsing (schema extraction requires an additional LLM call)
- You're working with very standardized AI output formats
- You're debugging parsing issues

### tasks {#tasks}

List tasks in the current project queue.

```bash
mehr project tasks [flags]
```

**Flags:**
| Flag          | Short | Description                           |
| ------------- | ----- | ------------------------------------- |
| `--queue-id`  | `-q`  | Queue ID (default: most recent)       |
| `--status`    | `-s`  | Filter by status (ready, blocked, submitted) |
| `--show-deps` |       | Show dependency graph                 |

**Output:**

```
ID       Title                         Status   Priority  Parent   Depends On
task-1   Set up JWT authentication     ready    1         -        -
task-2   Create login endpoint         ready    2         task-1   -
task-3   Create user profile page      blocked  3         task-1   task-2
task-4   Add rate limiting             ready    4         -        -
task-5   Write integration tests       blocked  5         -        task-3,task-4
```

### edit {#edit}

Edit a task's properties.

```bash
mehr project edit <task-id> [flags]
```

**Flags:**
| Flag            | Description                                    |
| --------------- | ---------------------------------------------- |
| `--title`       | Update task title                              |
| `--description` | Update task description                        |
| `--priority`    | Set priority (1-5, 1=highest)                  |
| `--status`      | Set status (pending, ready, blocked, submitted)|
| `--depends-on`  | Set dependencies (comma-separated task IDs)    |
| `--parent`      | Set parent task ID (makes this a subtask)      |
| `--clear-parent`| Remove parent relationship                     |
| `--labels`      | Set labels (comma-separated)                   |
| `--assignee`    | Set assignee                                   |

**Examples:**

```bash
# Update dependencies
mehr project edit task-3 --depends-on task-1,task-2

# Make task-2 a subtask of task-1
mehr project edit task-2 --parent task-1

# Remove subtask relationship
mehr project edit task-2 --clear-parent

# Change priority
mehr project edit task-2 --priority 1

# Mark as ready
mehr project edit task-3 --status ready
```

**Parent vs Dependencies:**

The `--parent` flag creates a **hierarchical relationship** (organizational grouping), while `--depends-on` creates an **execution ordering** (task B cannot start until task A completes). These are orthogonal concepts - a task can have both a parent AND dependencies:

```bash
# task-3 is a subtask of task-1, but depends on task-2 for execution order
mehr project edit task-3 --parent task-1 --depends-on task-2
```

### reorder {#reorder}

Reorder tasks in the queue manually or let AI suggest optimal ordering.

```bash
mehr project reorder [flags]
```

**Flags:**
| Flag       | Description                                   |
| ---------- | --------------------------------------------- |
| `--auto`   | AI suggests optimal order based on deps       |
| `--task`   | Task ID to move                               |
| `--before` | Move before this task ID                      |
| `--after`  | Move after this task ID                       |

**Examples:**

```bash
# Let AI suggest optimal order
mehr project reorder --auto

# Manual reorder
mehr project reorder --task task-3 --before task-1
```

**AI Reordering:**

The `--auto` flag uses AI to analyze your tasks and suggest an optimal execution order. The AI considers:

1. **Dependencies** - Tasks that are depended upon are scheduled earlier
2. **Priorities** - Higher priority tasks (lower numbers) are preferred
3. **Blocking relationships** - Tasks that block many others are scheduled early
4. **Logical grouping** - Related tasks are kept near each other

The AI returns both the new order and reasoning for the changes:

```
Tasks reordered by AI:

New order:
  1. task-3
  2. task-1
  3. task-2
  4. task-4

Reasoning:
task-3 has no dependencies and blocks task-1, so it should be done first.
Tasks were grouped by functional area for efficient context switching.
```

### submit {#submit}

Submit tasks to a provider (GitHub, Jira, Wrike, etc.).

```bash
mehr project submit [flags]
```

**Flags:**
| Flag            | Short | Description                                     |
| --------------- | ----- | ------------------------------------------------ |
| `--provider`    | `-p`  | Target provider (github, jira, etc.)            |
| `--create-epic` |       | Create parent epic/folder                        |
| `--labels`      |       | Additional labels for all tasks                  |
| `--dry-run`     |       | Preview without creating                          |
| `--task`        |       | Submit only specific tasks (comma-separated IDs)  |
| `--comment`     |       | Comment to add when tasks are already submitted |
| `--mention`     |       | Mention/notification to add to all submitted tasks |

**Examples:**

```bash
# Submit all tasks to Wrike
mehr project submit --provider wrike

# Submit only specific tasks
mehr project submit --provider github --task task-3,task-5

# Preview what would be created
mehr project submit --provider github --dry-run

# Submit with epic
mehr project submit --provider jira --create-epic

# Add comment when re-submitting already-submitted tasks
mehr project submit --provider wrike --task task-4 --comment "Updated per code review"

# Submit with manager notification
mehr project submit --provider jira --mention "@manager please review"
```

**Partial Task Submission:**

By default, `submit` submits all tasks in the queue. Use `--task` to submit specific tasks:

```bash
# Submit only task-3 and task-5
mehr project submit --provider github --task task-3,task-5
```

This is useful when:
- You want to review tasks before submitting in batches
- Some tasks need more work before being ready
- You're working with a provider that has rate limits

**Re-submitting Tasks:**

If some tasks were already submitted, you can add a comment using `--comment`:

```bash
# Add comment to already-submitted tasks
mehr project submit --provider github --task task-3,task-5 --comment "Updated per feedback"
```

**Dependency and Subtask Support:**

| Provider   | Dependencies                        | Subtasks                    |
| ---------- | ----------------------------------- | --------------------------- |
| Wrike      | Native (FinishToStart relationships)| Native (ParentID on create) |
| GitHub     | Task lists in epic body             | Issues with milestone       |
| GitLab     | Task lists in epic description      | Task notes                  |
| Jira       | Issue links (blocks/is-blocked-by)  | Native subtasks             |
| Linear     | Description-based                   | Sub-issues                  |
| Asana      | Native task dependencies            | Native subtasks             |
| ClickUp    | Native task dependencies            | Native subtasks             |
| Azure DevOps | Work item links                   | Child work items            |
| Trello     | Description-based                   | Checklists                  |
| YouTrack   | Description-based                   | Sub-issues                  |
| Bitbucket  | Description-based                   | Task lists                  |

When submitting tasks with `--parent` relationships, the parent tasks are automatically created first, and the subtasks include the provider's parent ID reference.

### start {#start}

Start implementing tasks from the queue.

```bash
mehr project start [task-id] [flags]
```

**Arguments:**
- `task-id` - Optional specific task to start (default: next ready task)

**Flags:**
| Flag         | Short | Description                                      |
| ------------ | ----- | ------------------------------------------------ |
| `--auto`     |       | Auto-chain through all tasks                     |
| `--parallel` | `-p`  | Max parallel tasks (requires worktree isolation) |

**Examples:**

```bash
# Start next ready task
mehr project start

# Start specific task
mehr project start task-3

# Auto-implement all tasks
mehr project start --auto

# Execute tasks in parallel (up to 3 at a time)
mehr project start --parallel=3
```

**Parallel Execution:**

The `--parallel` flag enables executing multiple tasks simultaneously in separate goroutines. Each task runs in its own isolated git worktree to prevent file conflicts:

```bash
# Execute up to 3 tasks in parallel
mehr project start --parallel=3
```

This is ideal for:
- Independent tasks without code dependencies
- Maximizing throughput on large task queues
- CI/CD environments with multiple cores

**Monitoring parallel tasks:**

```bash
# List running parallel tasks
mehr list --running

# Send note to a specific running task
mehr note --running=abc123 "Consider edge case X"
```

## Workflow Example

```bash
# 1. Create project plan from requirements
$ mehr project plan dir:/workspace/.final/ --title "Auth System"
Created queue: auth-system-abc123
  5 tasks identified
  2 questions to resolve

# 2. View the tasks
$ mehr project tasks
ID      Title                        Status   Priority  Depends On
task-1  Set up JWT authentication    ready    1         -
task-2  Create login endpoint        ready    2         -
task-3  Create user profile page     blocked  3         task-1
task-4  Add rate limiting            ready    4         -
task-5  Write integration tests      blocked  5         task-1,task-3

# 3. Adjust dependencies
$ mehr project edit task-3 --depends-on task-1,task-2
$ mehr project edit task-5 --depends-on task-4

# 4. Submit to Wrike
$ mehr project submit --provider wrike
Creating tasks:
  EXT-101 Set up JWT authentication (ready)
  EXT-102 Create login endpoint (ready)
  EXT-103 Create user profile page (blocked by EXT-101, EXT-102)
  EXT-104 Add rate limiting (ready)
  EXT-105 Write integration tests (blocked by EXT-104)
5 tasks created with dependencies.

# 5. Start implementing
$ mehr project start
Starting task: EXT-101 "Set up JWT authentication"
```
