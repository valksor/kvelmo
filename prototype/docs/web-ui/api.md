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
