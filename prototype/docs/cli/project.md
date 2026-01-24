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
  - `dir:/path` - Directory of files to scan
  - `file:spec.md` - Single file
  - `github:123`, `jira:PROJ-123` - Provider reference (fetches task details from provider)
  - URL - Fetch requirements from a URL (via web API only)

**Flags:**
| Flag        | Short | Description                      |
| ----------- | ----- | -------------------------------- |
| `--title`   | `-t`  | Project/epic title               |
| `--queue-id`|       | Specific queue ID (default: auto)|

**Examples:**

```bash
# Plan from a directory of specs
mehr project plan dir:/workspace/.final/

# Plan from a single file
mehr project plan file:requirements.md --title "Q1 Features"

# Plan from a provider reference
mehr project plan github:123
```

The AI will analyze the source and produce a structured task breakdown with dependencies.

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
ID       Title                         Status   Priority  Depends On
task-1   Set up JWT authentication     ready    1         -
task-2   Create login endpoint         ready    2         -
task-3   Create user profile page      blocked  3         task-1
task-4   Add rate limiting             ready    4         -
task-5   Write integration tests       blocked  5         task-1,task-3
```

### edit {#edit}

Edit a task's properties.

```bash
mehr project edit <task-id> [flags]
```

**Flags:**
| Flag           | Description                                    |
| -------------- | ---------------------------------------------- |
| `--title`      | Update task title                              |
| `--description`| Update task description                        |
| `--priority`   | Set priority (1-5, 1=highest)                  |
| `--status`     | Set status (pending, ready, blocked, submitted)|
| `--depends-on` | Set dependencies (comma-separated task IDs)    |
| `--labels`     | Set labels (comma-separated)                   |
| `--assignee`   | Set assignee                                   |

**Examples:**

```bash
# Update dependencies
mehr project edit task-3 --depends-on task-1,task-2

# Change priority
mehr project edit task-2 --priority 1

# Mark as ready
mehr project edit task-3 --status ready
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
| Flag            | Short | Description                          |
| --------------- | ----- | ------------------------------------ |
| `--provider`    | `-p`  | Target provider (github, jira, etc.) |
| `--create-epic` |       | Create parent epic/folder            |
| `--labels`      |       | Additional labels for all tasks      |
| `--dry-run`     |       | Preview without creating             |

**Examples:**

```bash
# Submit to Wrike with dependencies
mehr project submit --provider wrike

# Preview what would be created
mehr project submit --provider github --dry-run

# Submit with epic
mehr project submit --provider jira --create-epic
```

**Dependency Support:**

| Provider   | Dependencies                        |
| ---------- | ----------------------------------- |
| Wrike      | Native (FinishToStart relationships)|
| GitHub     | Task lists in epic body             |
| GitLab     | Task lists in epic description      |
| Jira       | Issue links (blocks/is-blocked-by)  |
| Linear     | Description-based                   |
| Asana      | Native task dependencies            |
| ClickUp    | Native task dependencies            |
| Azure DevOps | Work item links                   |
| Trello     | Description-based                   |
| YouTrack   | Description-based                   |
| Bitbucket  | Description-based                   |

### start {#start}

Start implementing tasks from the queue.

```bash
mehr project start [task-id] [flags]
```

**Arguments:**
- `task-id` - Optional specific task to start (default: next ready task)

**Flags:**
| Flag     | Description                           |
| -------- | ------------------------------------- |
| `--auto` | Auto-chain through all tasks          |

**Examples:**

```bash
# Start next ready task
mehr project start

# Start specific task
mehr project start task-3

# Auto-implement all tasks
mehr project start --auto
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

## Web API

All project commands are also available via REST API when running `mehr serve`:

| CLI Command            | API Endpoint                     |
| ---------------------- | -------------------------------- |
| `project plan`         | `POST /api/v1/project/plan`      |
| `project tasks`        | `GET /api/v1/project/tasks`      |
| `project edit`         | `PUT /api/v1/project/tasks/{id}` |
| `project reorder`      | `POST /api/v1/project/reorder`   |
| `project submit`       | `POST /api/v1/project/submit`    |
| `project start`        | `POST /api/v1/project/start`     |
| (source handling)      | `POST /api/v1/project/source`    |
| (file upload)          | `POST /api/v1/project/upload`    |

### Source API

The `/api/v1/project/source` endpoint handles different source types:

```bash
# Text content
curl -X POST localhost:8080/api/v1/project/source \
  -H "Content-Type: application/json" \
  -d '{"type":"text","value":"# Requirements\n\n- Feature 1"}'

# Provider reference
curl -X POST localhost:8080/api/v1/project/source \
  -H "Content-Type: application/json" \
  -d '{"type":"reference","value":"github:123"}'

# URL fetch
curl -X POST localhost:8080/api/v1/project/source \
  -H "Content-Type: application/json" \
  -d '{"type":"url","value":"https://raw.githubusercontent.com/user/repo/main/requirements.md"}'
```

### Reorder API

```bash
# AI-based reordering
curl -X POST localhost:8080/api/v1/project/reorder \
  -H "Content-Type: application/json" \
  -d '{"auto":true}'

# Manual reordering
curl -X POST localhost:8080/api/v1/project/reorder \
  -H "Content-Type: application/json" \
  -d '{"task_id":"task-3","position":"before","reference_id":"task-1"}'
```

See [Web UI documentation](../web/api.md) for full API details.
