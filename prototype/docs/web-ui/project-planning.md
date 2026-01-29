# Project Planning

The project planning interface provides a complete workflow for breaking down large requirements into manageable tasks with dependencies. Perfect for multi-task projects, epics, and coordinated feature development.

## Overview

Project planning lets you:
- **Create task breakdowns** from source files, directories, or external providers
- **View task queues** with status, priority, and dependencies
- **Edit task properties** inline
- **Reorder tasks** with AI assistance or manual control
- **Submit to providers** (GitHub, Jira, Wrike, etc.)
- **Start implementation** of individual tasks

## Accessing in the Web UI

Navigate to the project planning page at:

```
http://localhost:PORT/project
```

Or click **"Project Planning"** from the main dashboard.

## Using the Project Planning Interface

### Creating a Task Breakdown

1. Go to the project planning page (`/project`)
2. Click **"New Project Plan"**
3. Choose your source type:
   - **Directory** - Select a folder of specs/requirements
   - **File** - Choose a single requirements file
   - **Provider** - Enter a provider reference (e.g., `github:123`)
   - **URL** - Paste a URL to requirements
4. Configure options:
   - **Project Title** - Name for your epic/project
   - **Queue ID** - Specific queue (auto-generated if empty)
   - **Use schema-driven extraction** - Enables structured parsing of AI responses (enabled by default)
5. Click **"Create Breakdown"**

The AI will analyze your source and create a structured task list with dependencies.

### Source Type Differences

| Source | Description | Best For |
|--------|-------------|----------|
| **Directory** | Reads all file contents | Small codebases (<50 files) |
| **Research** | AI explores with tools | Large docs, existing tasks |
| **File** | Single file analysis | Individual requirements |
| **Provider** | Fetches from GitHub/Jira/etc. | External issue references |

### Schema-Driven Extraction

By default, the project planning interface uses **schema-driven extraction** to parse AI-generated task lists. This provides:

- **Flexible parsing**: Handles varied AI output formats without breaking
- **JSON validation**: Ensures extracted data matches expected structure
- **Automatic fallback**: Falls back to regex parsing if schema extraction fails

**When to disable:**
- You need faster parsing (schema extraction requires an additional LLM call)
- You're working with very standardized AI output formats
- You're debugging parsing issues

To disable schema-driven extraction, uncheck the **"Use schema-driven extraction"** checkbox before creating your breakdown.

**Note:** Schema extraction is enabled by default and recommended for most use cases. Only disable it if you encounter specific issues or need faster processing.

### Viewing the Task Queue

After creating a breakdown, you'll see a table of all tasks:

| Column | Description |
|--------|-------------|
| ID | Task identifier (indented for subtasks) |
| Title | Task name |
| Status | `ready`, `blocked`, `submitted` |
| Priority | 1 (high) to 5 (low) |
| Parent | Parent task ID (if this is a subtask) |
| Depends On | List of task IDs this task requires |

**Dependency Visualization:**
- Enable **"Show Dependencies"** to see the task relationship graph
- Blocked tasks are visually distinct
- Subtasks show with a tree indicator (└) next to their ID
- Click a task ID to jump to its dependencies

### Editing Tasks

Modify any task property:

1. Find the task in the queue
2. Click **"Edit"** or click directly on the field
3. Modify:
   - **Title** - Task name
   - **Description** - Full task details
   - **Priority** - 1-5 scale
   - **Status** - `ready`, `blocked`, `submitted`
   - **Parent** - Parent task ID (makes this a subtask; leave empty for top-level)
   - **Dependencies** - Comma-separated task IDs
   - **Labels** - Task tags
   - **Assignee** - Task owner (for providers)
4. Click **"Save"**

**Parent vs Dependencies:**

- **Parent** (hierarchical): Organizational grouping in providers (e.g., Wrike subtasks, Jira sub-issues)
- **Dependencies** (execution order): Task B cannot start until task A completes

A task can have BOTH a parent AND dependencies. For example, a subtask under "Authentication" might depend on "Database Schema" being completed first.

### Reordering Tasks

Use AI to optimize task order, or manually arrange tasks:

**AI Reordering:**
1. Click **"Auto Reorder"**
2. The AI analyzes:
   - Dependencies (blocking tasks go first)
   - Priorities (higher priority preferred)
   - Logical grouping (related tasks kept together)
3. Review the suggested order
4. Click **"Apply"** or **"Discard"**

**Manual Reordering:**
1. Click **"Reorder"** on a task
2. Choose:
   - **"Move before [task]"** - Reposition above another task
   - **"Move after [task]"** - Reposition below another task
3. Repeat until satisfied

### Submitting to Providers

Submit tasks to external project management systems:

1. Click **"Submit to Provider"**
2. Choose provider:
   - GitHub (creates issues/epics)
   - GitLab (creates issues/epics)
   - Jira (creates issues with links)
   - Wrike (creates tasks with dependencies)
   - Linear, Asana, ClickUp, and more
3. Configure:
   - **Create Epic** - Group tasks under a parent
   - **Additional Labels** - Add tags to all tasks
   - **Dry Run** - Preview without creating
   - **Select Tasks** - Choose specific tasks for selective submit (optional)
4. Click **"Submit"**

**Selective Submit:**

You can submit specific tasks instead of the entire queue:
- Select individual tasks using checkboxes
- Or specify task IDs in the **"Task IDs"** field (comma-separated)
- Leave empty to submit all tasks

This is useful when:
- You want to review tasks before submitting in batches
- Some tasks need more work before being ready
- You're working with a provider that has rate limits

**Dependency and Subtask Support by Provider:**

| Provider | Dependencies | Subtasks |
|----------|--------------|----------|
| **Wrike** | Native FinishToStart | Native subtasks |
| **GitHub** | Task lists in epic body | Issues with milestone |
| **GitLab** | Task lists in description | Task notes |
| **Jira** | Issue links (blocks/is-blocked-by) | Native sub-issues |
| **Asana** | Native dependencies | Native subtasks |
| **ClickUp** | Native dependencies | Native subtasks |
| **Linear** | Description-based | Sub-issues |

When submitting tasks with parent relationships, parents are automatically created first and subtasks include the provider's parent ID.

### Starting Implementation

Begin working on a task from your queue:

1. Select a task with status `ready`
2. Click **"Start Task"**
3. The task loads into the main Mehrhof workflow
4. Proceed with planning, implementation, and completion

**After starting:**
- Task status changes to `in_progress`
- Other tasks dependent on it become `blocked`
- Navigate back to project planning to start the next task

### Task Status Flow

```
┌─────────┐     submit      ┌───────────┐
│  ready  │ ────────────────> │ submitted │
└────┬────┘                  └───────────┘
     │                            │
     │ start                     │
     ▼                            │
┌───────────┐                    │
│in_progress│ ────────────────────┘
└───────────┘
     │
     │ finish
     ▼
┌─────────┐
│  done   │
└─────────┘
```

## Common Workflows

### Multi-Feature Project

```
1. Gather requirements in ./specs/
2. Create breakdown from directory
3. Review AI-generated task list
4. Adjust dependencies as needed
5. Submit entire project to GitHub as epic
6. Work through tasks one by one
```

### Provider-Based Planning

```
1. Fetch from GitHub issue #123
2. AI breaks down into subtasks
3. Edit task details
4. Submit back to GitHub as checklist
5. Track progress in both systems
```

### Syncing from External Providers

Pull entire project/epic structures from external providers (Wrike, Jira, GitHub):

1. Go to the project planning page (`/project`)
2. Click **"Sync from Provider"**
3. Enter provider reference:
   - **Wrike**: `wrike:https://www.wrike.com/open.htm?id=123456` or `wrike:folder-789`
   - **Jira**: `jira:PROJ-123` (epic key)
   - **GitHub**: `github:owner/repo#456` (issue with task list)
4. Configure options:
   - **Status Filter** - Include tasks by status (empty = smart default)
   - **Max Depth** - Hierarchy depth limit (0 = unlimited)
   - **Preserve External IDs** - Keep provider IDs/URLs
5. Click **"Sync Project"**

**Progress Tracking:**

The sync process shows real-time progress via SSE streaming:
- `Connecting to provider...`
- `Fetching project structure...`
- `Found N tasks (including subtasks)`
- `Processing task hierarchy...`
- `Queue created: project-name-YYYYMMDD-HHMMSS`

**Smart Status Filter (Default):**

When no status filter is specified, includes:
- All open tasks
- All in-progress tasks
- Tasks completed in the last 30 days

Use `--status` to override:
- `open,in_progress` - Only active tasks
- `open,in_progress,done` - Include all completed tasks

### Iterative Refinement

```
1. Create initial breakdown
2. Manually reorder based on team input
3. Add detailed descriptions to each task
4. Set priorities for sprint planning
5. Submit highest priority tasks to Jira
```

## CLI Equivalent

See [`mehr project`](../cli/project.md) for CLI usage.

| CLI Command | Web UI Action |
|-------------|---------------|
| `mehr project plan dir:./specs` | Create breakdown |
| `mehr project plan dir:./specs --use-schema=false` | Create breakdown (regex-only) |
| `mehr project sync wrike:123456` | Sync from provider |
| `mehr project tasks` | View queue |
| `mehr project edit task-1` | Edit task |
| `mehr project reorder --auto` | AI reorder |
| `mehr project submit --provider wrike` | Submit all to provider |
| `mehr project submit --provider wrike --task task-1,task-2` | Selective submit |
| `mehr project start` | Start task |

## Project Queue Storage

Project queues are stored in:
```
~/.valksor/mehrhof/workspaces/<project-id>/queues/<queue-id>.yaml
```

Each project plan generates a unique queue ID for tracking.
