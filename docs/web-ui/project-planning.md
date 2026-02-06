# Project Planning

The project planning interface provides a complete workflow for breaking down large requirements into manageable tasks with dependencies. Perfect for multitask projects, epics, and coordinated feature development.

## Overview

Project planning lets you:
- **Create task breakdowns** from source files, directories, or external providers
- **Manage queues** (list, select, delete)
- **Route queue task execution** through the dashboard

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

| Source        | Description                   | Best For                    |
|---------------|-------------------------------|-----------------------------|
| **Directory** | Reads all file contents       | Small codebases (<50 files) |
| **Research**  | AI explores with tools        | Large docs, existing tasks  |
| **File**      | Single file analysis          | Individual requirements     |
| **Provider**  | Fetches from GitHub/Jira/etc. | External issue references   |

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

### Queue Task Management Location

Queue task operations are intentionally centralized in one place:
- Open the **Dashboard**
- Go to **Tasks**
- Switch to the **Queue** view

From that single view, you can:
- Inspect queue tasks and dependency structure
- Edit task properties
- Reorder tasks with AI
- Submit tasks to providers
- Start implementation

The **Project Planning** page focuses on queue creation and queue-level management.

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

---

## Also Available via CLI

Manage project planning from the command line for scripting or terminal workflows.

See [CLI: project](/cli/project.md) for all options, selective submission, and provider-specific features.

## Project Queue Storage

Project queues are stored in:
```
~/.valksor/mehrhof/workspaces/<project-id>/queues/<queue-id>.yaml
```

Each project plan generates a unique queue ID for tracking.
