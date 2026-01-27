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
5. Click **"Create Breakdown"**

The AI will analyze your source and create a structured task list with dependencies.

### Source Type Differences

| Source | Description | Best For |
|--------|-------------|----------|
| **Directory** | Reads all file contents | Small codebases (<50 files) |
| **Research** | AI explores with tools | Large docs, existing tasks |
| **File** | Single file analysis | Individual requirements |
| **Provider** | Fetches from GitHub/Jira/etc. | External issue references |

### Viewing the Task Queue

After creating a breakdown, you'll see a table of all tasks:

| Column | Description |
|--------|-------------|
| ID | Task identifier |
| Title | Task name |
| Status | `ready`, `blocked`, `submitted` |
| Priority | 1 (high) to 5 (low) |
| Depends On | List of task IDs this task requires |

**Dependency Visualization:**
- Enable **"Show Dependencies"** to see the task relationship graph
- Blocked tasks are visually distinct
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
   - **Dependencies** - Comma-separated task IDs
   - **Labels** - Task tags
   - **Assignee** - Task owner (for providers)
4. Click **"Save"**

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
4. Click **"Submit"**

**Dependency Support by Provider:**
- **Wrike**: Native Finish-ToStart relationships
- **GitHub**: Task lists in epic body
- **GitLab**: Task lists in description
- **Jira**: Issue links (blocks/is-blocked-by)
- **Asana/ClickUp**: Native task dependencies

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
| `mehr project tasks` | View queue |
| `mehr project edit task-1` | Edit task |
| `mehr project reorder --auto` | AI reorder |
| `mehr project submit --provider wrike` | Submit to provider |
| `mehr project start` | Start task |

## Project Queue Storage

Project queues are stored in:
```
~/.valksor/mehrhof/workspaces/<project-id>/queues/<queue-id>.yaml
```

Each project plan generates a unique queue ID for tracking.
