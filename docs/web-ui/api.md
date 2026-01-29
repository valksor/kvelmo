# REST API

The Mehrhof Web UI exposes a REST API for programmatic access to all workflow features.

## Base URL

```
http://localhost:PORT/api/v1
```

Replace `PORT` with the actual port number shown when starting the server.

## Authentication

For network-accessible servers (`--host 0.0.0.0`), include session cookies:

```bash
# Login first to get session cookie
curl -c cookies.txt -X POST http://localhost:PORT/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Use session cookie for requests
curl -b cookies.txt http://localhost:PORT/api/v1/status
```

## Endpoints Overview

| Category | Endpoints |
|----------|-----------|
| **Authentication** | Login, logout |
| **Read** | Status, context, tasks, specs, sessions, notes, costs |
| **Workflow** | start, plan, implement, review, finish, undo, redo, continue, auto |
| **Project** | plan, tasks, submit, reorder, start |
| **Quick Tasks** | list, get, create, note, optimize, export, submit, start, delete, card |
| **Browser** | Status, tabs, goto, navigate, screenshot, click, type, eval, dom, close |
| **Settings** | Get, update settings |
| **Templates** | List, get, apply |
| **Utilities** | Guide, scan, memory search, providers, agents, diagram |

---

## Authentication Endpoints

### POST /api/v1/auth/login

Authenticate and create a session.

```bash
curl -X POST http://localhost:PORT/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

**Response:**
```json
{
  "success": true,
  "message": "login successful"
}
```

### POST /api/v1/auth/logout

End the current session.

```bash
curl -X POST http://localhost:PORT/api/v1/auth/logout
```

---

## Read Operations

### GET /health

Health check (public endpoint).

```bash
curl http://localhost:PORT/health
```

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### GET /api/v1/status

Server and workflow status.

```bash
curl http://localhost:PORT/api/v1/status
```

**Response:**
```json
{
  "server": {
    "port": 3000,
    "mode": "project",
    "workspace_root": "/path/to/project"
  },
  "task": {
    "id": "a1b2c3d4",
    "title": "Add user authentication",
    "state": "implementing",
    "branch": "feature/user-auth"
  }
}
```

### GET /api/v1/task

Active task details (includes pending question if any).

```bash
curl http://localhost:PORT/api/v1/task
```

### GET /api/v1/tasks

List all tasks in workspace.

```bash
curl http://localhost:PORT/api/v1/tasks
```

### GET /api/v1/tasks/{id}

Get specific task details.

```bash
curl http://localhost:PORT/api/v1/tasks/a1b2c3d4
```

### GET /api/v1/tasks/{id}/specs

Task specifications.

```bash
curl http://localhost:PORT/api/v1/tasks/a1b2c3d4/specs
```

### GET /api/v1/tasks/{id}/sessions

Task sessions.

```bash
curl http://localhost:PORT/api/v1/tasks/a1b2c3d4/sessions
```

### GET /api/v1/tasks/{id}/notes

Task notes content.

```bash
curl http://localhost:PORT/api/v1/tasks/a1b2c3d4/notes
```

### GET /api/v1/tasks/{id}/costs

Task token usage and cost breakdown.

```bash
curl http://localhost:PORT/api/v1/tasks/a1b2c3d4/costs
```

**Response (example):**
```json
{
  "task_id": "a1b2c3d4",
  "total_tokens": 12000,
  "input_tokens": 7000,
  "output_tokens": 5000,
  "cached_tokens": 1000,
  "total_cost_usd": 4.25,
  "budget": {
    "max_cost": 5.0,
    "currency": "USD",
    "on_limit": "pause",
    "warning_at": 0.8,
    "warned": true
  }
}
```

### GET /api/v1/task/labels

Get labels for the active task.

```bash
curl http://localhost:PORT/api/v1/task/labels
```

**Response:**
```json
{
  "task_id": "a1b2c3d4",
  "labels": ["priority:high", "type:bug", "team:backend"]
}
```

### POST /api/v1/task/labels

Manage labels for the active task.

```bash
curl -X POST http://localhost:PORT/api/v1/task/labels \
  -H "Content-Type: application/json" \
  -d '{"action": "add", "labels": ["priority:high", "type:bug"]}'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `action` | string | Operation: `add`, `remove`, or `set` |
| `labels` | array | Labels to operate on |

**Actions:**
- `add` - Add labels (ignores duplicates)
- `remove` - Remove labels (ignores non-existent)
- `set` - Replace all labels (empty array clears all)

**Response (add action):**
```json
{
  "task_id": "a1b2c3d4",
  "action": "add",
  "added_labels": ["priority:high", "type:bug"],
  "labels": ["priority:high", "type:bug", "team:backend"]
}
```

**Response (remove action):**
```json
{
  "task_id": "a1b2c3d4",
  "action": "remove",
  "removed_labels": ["priority:high"],
  "labels": ["type:bug", "team:backend"]
}
```

**Response (set action):**
```json
{
  "task_id": "a1b2c3d4",
  "action": "set",
  "labels": ["priority:critical"]
}
```

### GET /api/v1/labels

List all unique labels across all tasks with counts.

```bash
curl http://localhost:PORT/api/v1/labels
```

**Response:**
```json
{
  "labels": {
    "priority:high": 5,
    "type:bug": 3,
    "team:backend": 7,
    "status:blocked": 1
  },
  "total_tasks": 12,
  "labeled_tasks": 10
}
```

### GET /api/v1/projects

List all projects (global mode only).

```bash
curl http://localhost:PORT/api/v1/projects
```

### GET /api/v1/events

SSE stream for real-time updates.

```bash
curl http://localhost:PORT/api/v1/events
```

### GET /api/v1/guide

Get state-specific guidance on next actions.

```bash
curl http://localhost:PORT/api/v1/guide
```

**Response:**
```json
{
  "has_task": true,
  "task_id": "abc123",
  "title": "Add health check",
  "state": "idle",
  "specifications": 2,
  "next_actions": [
    {
      "command": "mehr implement",
      "description": "Implement the specifications",
      "endpoint": "POST /api/v1/workflow/implement"
    }
  ]
}
```

### GET /api/v1/agents

List available AI agents with capabilities and models.

```bash
curl http://localhost:PORT/api/v1/agents
```

### GET /api/v1/providers

List available task providers.

```bash
curl http://localhost:PORT/api/v1/providers
```

### GET /api/v1/workflow/diagram

Get SVG workflow state diagram.

```bash
curl http://localhost:PORT/api/v1/workflow/diagram \
  --output workflow-diagram.svg
```

### GET /api/v1/costs

Get costs for all tasks with totals.

```bash
curl http://localhost:PORT/api/v1/costs
```

**Response (example):**
```json
{
  "tasks": [
    {
      "task_id": "a1b2c3d4",
      "total_cost_usd": 4.25
    }
  ],
  "grand_total": {
    "total_tokens": 12000,
    "cost_usd": 4.25
  },
  "monthly": {
    "month": "2026-01",
    "spent": 12.5,
    "max_cost": 100,
    "warning_at": 0.8
  }
}
```

---

## Workflow Actions (POST)

### POST /api/v1/workflow/start

Start a new task.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/start \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "file:task.md"
  }'
```

### POST /api/v1/workflow/plan

Run planning phase.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/plan
```

### POST /api/v1/workflow/implement

Run implementation phase.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/implement
```

### POST /api/v1/workflow/review

Run review phase.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/review
```

### POST /api/v1/workflow/finish

Complete the task.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/finish
```

### POST /api/v1/workflow/undo

Undo to previous checkpoint.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/undo
```

### POST /api/v1/workflow/redo

Redo to next checkpoint.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/redo
```

### POST /api/v1/workflow/answer

Submit answer to pending question.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/answer \
  -H "Content-Type: application/json" \
  -d '{"content": "Use PostgreSQL"}'
```

### POST /api/v1/workflow/resume

Resume a task paused due to budget limits.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/resume
```

### POST /api/v1/workflow/abandon

Abandon current task.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/abandon
```

### POST /api/v1/workflow/continue

Resume workflow with optional auto-execute.

```bash
# Get suggested next actions
curl -X POST http://localhost:PORT/api/v1/workflow/continue \
  -H "Content-Type: application/json" \
  -d '{"auto": false}'
```

**Response:**
```json
{
  "success": true,
  "state": "idle",
  "next_actions": ["POST /api/v1/workflow/plan"],
  "message": "task resumed"
}
```

### POST /api/v1/workflow/auto

Run full automation cycle (plan → implement → review → finish).

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/auto \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "file:task.md",
    "max_retries": 3,
    "quality_target": "quality",
    "no_push": false
  }'
```

**Response:**
```json
{
  "success": true,
  "planning_done": true,
  "implement_done": true,
  "quality_attempts": 1,
  "quality_passed": true,
  "finish_done": true
}
```

### POST /api/v1/workflow/sync

Sync task from provider and detect changes.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/sync \
  -H "Content-Type: application/json" \
  -d '{"task_id": "abc123"}'
```

**Response:**
```json
{
  "success": true,
  "has_changes": true,
  "changes_summary": "Title changed, description updated",
  "spec_generated": ".mehrhof/workspaces/abc123/specs/delta-001.md",
  "message": "changes detected and delta specification generated"
}
```

### POST /api/v1/workflow/simplify

Auto-simplify content based on workflow state.

```bash
curl -X POST http://localhost:PORT/api/v1/workflow/simplify \
  -H "Content-Type: application/json" \
  -d '{}'
```

### POST /api/v1/tasks/{id}/notes

Add a note to a task.

```bash
curl -X POST http://localhost:PORT/api/v1/tasks/abc123/notes \
  -H "Content-Type: application/json" \
  -d '{"content": "Use the existing auth middleware"}'
```

---

## Template Endpoints

### GET /api/v1/templates

List available templates.

```bash
curl http://localhost:PORT/api/v1/templates
```

**Response:**
```json
{
  "templates": [
    {"name": "feature", "description": "Standard feature implementation"},
    {"name": "bug-fix", "description": "Bug fix with reproduction steps"}
  ],
  "count": 2
}
```

### GET /api/v1/templates/{name}

Get template details.

```bash
curl http://localhost:PORT/api/v1/templates/feature
```

### POST /api/v1/templates/apply

Apply template to a file.

```bash
curl -X POST http://localhost:PORT/api/v1/templates/apply \
  -H "Content-Type: application/json" \
  -d '{"template_name": "feature", "file_path": "task.md"}'
```

---

## Browser Automation Endpoints

### GET /api/v1/browser/status

Browser connection status and tabs.

```bash
curl http://localhost:PORT/api/v1/browser/status
```

### GET /api/v1/browser/tabs

List open browser tabs.

```bash
curl http://localhost:PORT/api/v1/browser/tabs
```

### POST /api/v1/browser/goto

Open URL in new tab.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/goto \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

### POST /api/v1/browser/navigate

Navigate existing tab to URL.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/navigate \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

### POST /api/v1/browser/screenshot

Capture screenshot.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/screenshot \
  -H "Content-Type: application/json" \
  -d '{"full_page": true}'
```

### POST /api/v1/browser/click

Click element by selector.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/click \
  -H "Content-Type: application/json" \
  -d '{"selector": "#submit-button"}'
```

### POST /api/v1/browser/type

Type text into element.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/type \
  -H "Content-Type: application/json" \
  -d '{"selector": "#email", "text": "user@example.com"}'
```

### POST /api/v1/browser/eval

Execute JavaScript.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/eval \
  -H "Content-Type: application/json" \
  -d '{"code": "document.title"}'
```

### POST /api/v1/browser/dom

Query DOM elements.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/dom \
  -H "Content-Type: application/json" \
  -d '{"selector": "a[href]"}'
```

### POST /api/v1/browser/reload

Reload page.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/reload
```

### POST /api/v1/browser/close

Close tab.

```bash
curl -X POST http://localhost:PORT/api/v1/browser/close \
  -H "Content-Type: application/json" \
  -d '{"tab_id": 1}'
```

---

## Security Scanning

### POST /api/v1/scan

Run security scans (gosec, gitleaks, govulncheck).

```bash
curl -X POST http://localhost:PORT/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanners": ["gosec", "gitleaks"],
    "fail_level": "high"
  }'
```

**Response:**
```json
{
  "findings": [...],
  "total_count": 5,
  "blocking_count": 2,
  "passed": false
}
```

---

## Memory/Search Endpoints

### GET /api/v1/memory/search

Search semantic memory.

```bash
curl "http://localhost:PORT/api/v1/memory/search?q=authentication&limit=5&types=code,spec"
```

**Response:**
```json
{
  "results": [
    {
      "task_id": "abc123",
      "type": "code_change",
      "score": 0.92,
      "content": "...",
      "metadata": {...}
    }
  ],
  "count": 1
}
```

### POST /api/v1/memory/index

Index a task to memory.

```bash
curl -X POST http://localhost:PORT/api/v1/memory/index
```

### GET /api/v1/memory/stats

Memory system statistics.

```bash
curl http://localhost:PORT/api/v1/memory/stats
```

---

## Settings Endpoints

### GET /settings

Settings page (HTML).

### GET /api/v1/settings

Get current configuration.

```bash
curl http://localhost:PORT/api/v1/settings
```

**Response:**
```json
{
  "git": {
    "auto_commit": true,
    "branch_pattern": "{type}/{key}--{slug}"
  },
  "agent": {
    "default": "claude",
    "timeout": 300
  }
}
```

### POST /api/v1/settings

Save configuration changes.

```bash
curl -X POST http://localhost:PORT/api/v1/settings \
  -H "Content-Type: application/json" \
  -d '{
    "git": {"auto_commit": false},
    "agent": {"timeout": 600}
  }'
```

---

## Project Planning

The project planning API provides a complete workflow for planning and executing multi-task projects with dependencies.

### POST /api/v1/project/plan

Create a project plan from a source (directory, file, or provider reference).

```bash
curl -X POST http://localhost:PORT/api/v1/project/plan \
  -H "Content-Type: application/json" \
  -d '{
    "source": "file:requirements.md",
    "title": "Q1 Features"
  }'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `source` | string | Source: `dir:/path`, `file:/path`, `github:123`, `jira:PROJ-123`, URL |
| `title` | string | Optional project title |
| `instructions` | string | Optional custom instructions for AI |

**Response:**
```json
{
  "queue_id": "auth-system-abc123",
  "title": "Q1 Features",
  "tasks": [...],
  "questions": [],
  "blockers": []
}
```

### GET /api/v1/project/tasks

List tasks in a project queue.

```bash
curl "http://localhost:PORT/api/v1/project/tasks?queue_id=auth-system-abc123"
```

**Query Parameters:**
| Parameter | Description |
|-----------|-------------|
| `queue_id` | Queue ID (default: most recent) |
| `status` | Filter by status (ready, blocked, submitted) |
| `show_deps` | Show dependency graph |

### POST /api/v1/project/submit

Submit tasks to an external provider (GitHub, Jira, Wrike, etc.).

```bash
# Submit all tasks
curl -X POST http://localhost:PORT/api/v1/project/submit \
  -H "Content-Type: application/json" \
  -d '{
    "queue_id": "auth-system-abc123",
    "provider": "wrike",
    "create_epic": true,
    "labels": ["q1", "feature"],
    "dry_run": false,
    "mention": "@manager please review"
  }'

# Submit specific tasks only (selective submit)
curl -X POST http://localhost:PORT/api/v1/project/submit \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "wrike",
    "task_ids": ["task-1", "task-3", "task-5"],
    "labels": ["sprint-1"]
  }'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `queue_id` | string | Queue ID (default: most recent) |
| `provider` | string | Provider name (wrike, github, jira, etc.) |
| `create_epic` | boolean | Create parent epic/project |
| `labels` | array | Labels to apply to all tasks |
| `dry_run` | boolean | Preview without creating |
| `task_ids` | array | Submit only specific task IDs (e.g., `["task-1", "task-3"]`). Omit to submit all. |
| `mention` | string | Mention/notification to add to all submitted tasks |

**Response:**
```json
{
  "dry_run": false,
  "epic": {
    "external_id": "EXT-100",
    "external_url": "https://wrike.com/...",
    "title": "Auth System"
  },
  "tasks": [
    {
      "local_id": "task-1",
      "external_id": "EXT-101",
      "external_url": "https://wrike.com/...",
      "title": "Set up JWT authentication"
    }
  ]
}
```

### POST /api/v1/project/reorder

Reorder tasks in the queue.

**AI-based reordering:**
```bash
curl -X POST http://localhost:PORT/api/v1/project/reorder \
  -H "Content-Type: application/json" \
  -d '{"auto": true}'
```

**Manual reordering:**
```bash
curl -X POST http://localhost:PORT/api/v1/project/reorder \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-3",
    "position": "before",
    "reference_id": "task-1"
  }'
```

### POST /api/v1/project/start

Start implementing tasks from the queue.

```bash
curl -X POST http://localhost:PORT/api/v1/project/start \
  -H "Content-Type: application/json" \
  -d '{
    "queue_id": "auth-system-abc123",
    "auto": false
  }'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `queue_id` | string | Queue ID (default: most recent) |
| `task_id` | string | Optional specific task to start |
| `auto` | boolean | Auto-chain through all tasks |

---

## Quick Tasks

The quick tasks API provides rapid task capture and management without the full project planning workflow.

### GET /api/v1/quick

List all quick tasks.

```bash
curl http://localhost:PORT/api/v1/quick
```

**Response:**
```json
{
  "tasks": [
    {
      "id": "task-1",
      "title": "Fix typo in README",
      "priority": 1,
      "labels": ["documentation", "typo-fix"],
      "status": "pending",
      "note_count": 2
    }
  ],
  "count": 1
}
```

### GET /api/v1/quick/{taskId}

Get a specific quick task.

```bash
curl http://localhost:PORT/api/v1/quick/task-1
```

**Response:**
```json
{
  "id": "task-1",
  "title": "Fix typo in README",
  "description": "The word Installation is misspelled...",
  "priority": 1,
  "labels": ["documentation"],
  "status": "pending",
  "created_at": "2025-01-15T10:30:00Z",
  "notes": [
    {"timestamp": "2025-01-15T10:35:00Z", "content": "Found in getting started section"}
  ]
}
```

### POST /api/v1/quick

Create a new quick task.

```bash
curl -X POST http://localhost:PORT/api/v1/quick \
  -H "Content-Type: application/json" \
  -d '{
    "description": "fix typo in README line 42",
    "title": "Fix README typo",
    "priority": 1,
    "labels": ["documentation", "typo-fix"]
  }'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Task description (required) |
| `title` | string | Optional custom title (auto-extracted from description if not provided) |
| `priority` | number | Priority level (1=high, 2=normal, 3=low, default: 2) |
| `labels` | array | Labels to apply |
| `queue` | string | Target queue ID (default: "quick-tasks") |

**Response:**
```json
{
  "id": "task-1",
  "title": "Fix README typo",
  "queue_id": "quick-tasks",
  "message": "Task created"
}
```

### POST /api/v1/quick/{taskId}/note

Add a note to a quick task.

```bash
curl -X POST http://localhost:PORT/api/v1/quick/task-1/note \
  -H "Content-Type: application/json" \
  -d '{"note": "The typo is on line 42"}'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `note` | string | Note content (preferred field name) |
| `content` | string | Legacy field name (still supported for backward compatibility) |

### POST /api/v1/quick/{taskId}/optimize

AI optimize a task based on its notes.

```bash
curl -X POST http://localhost:PORT/api/v1/quick/task-1/optimize \
  -H "Content-Type: application/json" \
  -d '{"agent": "claude-opus"}'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `agent` | string | Optional agent override for optimization |

**Response:**
```json
{
  "original_title": "fix typo",
  "optimized_title": "Fix typo: Installaton → Installation in README",
  "original_description": "...",
  "optimized_description": "...",
  "added_labels": ["documentation", "typo-fix"],
  "improvement_notes": [
    "Title expanded to include specific location and correction",
    "Description includes file location and context"
  ]
}
```

### POST /api/v1/quick/{taskId}/export

Export a quick task to a markdown file.

```bash
curl -X POST http://localhost:PORT/api/v1/quick/task-1/export \
  -H "Content-Type: application/json" \
  -d '{"output": "specs/readme-typo.md"}'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `output` | string | Output file path (required) |

**Response:**
```json
{
  "file": "specs/readme-typo.md",
  "content": "---\ntitle: Fix typo...\n---\n\n# Fix typo...",
  "message": "Task exported"
}
```

### POST /api/v1/quick/{taskId}/submit

Submit a quick task to an external provider.

```bash
curl -X POST http://localhost:PORT/api/v1/quick/task-1/submit \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "github",
    "labels": ["bug", "urgent"],
    "dry_run": false
  }'
```

**Request Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Provider name (required) - github, jira, wrike, etc. |
| `labels` | array | Additional labels to apply |
| `dry_run` | boolean | Preview without submitting (default: false) |

**Response:**
```json
{
  "dry_run": false,
  "tasks": [{
    "local_id": "task-1",
    "external_id": "valksor/go-mehrhof#123",
    "external_url": "https://github.com/valksor/go-mehrhof/issues/123",
    "title": "Fix typo in README"
  }],
  "message": "Task submitted successfully"
}
```

### POST /api/v1/quick/{taskId}/start

Start standard workflow from a quick task.

```bash
curl -X POST http://localhost:PORT/api/v1/quick/task-1/start \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:**
```json
{
  "task_id": "abc12345",
  "message": "Workflow started",
  "state": "idle"
}
```

### DELETE /api/v1/quick/{taskId}

Delete a quick task.

```bash
curl -X DELETE http://localhost:PORT/api/v1/quick/task-1
```

**Response:**
```json
{
  "message": "Task deleted"
}
```

### GET /api/v1/quick/{taskId}/card

Get task card HTML for UI rendering.

```bash
curl http://localhost:PORT/api/v1/quick/task-1/card
```

**Response:** HTML fragment for embedding in UI.

---

## Common Response Format

Most endpoints return JSON:

```json
{
  "success": true,
  "data": {...},
  "message": "operation completed"
}
```

Error responses:

```json
{
  "success": false,
  "error": "error message",
  "code": "ERROR_CODE"
}
```

---

## Next Steps

- [**CLI: serve**](../cli/serve.md) - Server command options
- [**Authentication**](authentication.md) - Setup for remote access
- [**Dashboard**](dashboard.md) - Web UI interface
