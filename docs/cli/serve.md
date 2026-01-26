# mehr serve

Start the web UI server for browser-based task management.

## Synopsis

```bash
mehr serve [flags]
mehr serve auth <subcommand>
mehr serve register [flags]
mehr serve unregister [project-id]
```

## Description

The `serve` command starts a local HTTP server providing a web-based UI for managing Mehrhof tasks. It supports two modes:

- **Project mode** (default): Shows the current workspace's tasks, workflow state, and allows triggering workflow actions
- **Global mode**: Shows all projects discovered across your system with task summaries

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port`, `-p` | int | 0 | Server port (0 = random available port) |
| `--host` | string | localhost | Host to bind to (use 0.0.0.0 for network access) |
| `--global` | bool | false | Run in global mode showing all projects |
| `--open` | bool | false | Automatically open browser after server starts |
| `--tunnel-info` | bool | false | Show SSH tunnel instructions for remote access |

## Subcommands

### mehr serve auth

Manage authentication for network-accessible servers.

#### mehr serve auth add

Add a new user for authentication.

```bash
mehr serve auth add <username> <password>
```

#### mehr serve auth list

List all configured users.

```bash
mehr serve auth list
```

#### mehr serve auth remove

Remove a user.

```bash
mehr serve auth remove <username>
```

#### mehr serve auth passwd

Change a user's password.

```bash
mehr serve auth passwd <username> <new-password>
```

### mehr serve register

Register the current project for remote access.

```bash
# Register current project
mehr serve register

# List all registered projects
mehr serve register --list
```

### mehr serve unregister

Unregister a project from remote access.

```bash
# Unregister current project
mehr serve unregister

# Unregister by project ID
mehr serve unregister <project-id>
```

## Examples

### Basic Usage

```bash
# Start server in project mode (random port)
mehr serve

# Start on a specific port
mehr serve --port 3000

# Start and open browser automatically
mehr serve --open

# Start in global mode to see all projects
mehr serve --global
```

### Remote Access

#### Option 1: Direct Network Binding

For machines exposed to the network or internet:

```bash
# First, set up authentication
mehr serve auth add admin mypassword

# Then start with network binding
mehr serve --host 0.0.0.0 --port 8080
```

**Important**: Authentication is **required** when using `--host 0.0.0.0` or any non-localhost address.

#### Option 2: SSH Tunnel

For secure remote access without exposing ports:

```bash
# Show tunnel instructions (helper flag - exits without starting server)
mehr serve --tunnel-info

# Output includes SSH tunnel instructions:
# SSH Tunnel Instructions:
#   Access remote serve from your local machine (-L flag):
#     ssh -L 8080:localhost:3000 user@remote-server
#     Then open: http://localhost:8080 on YOUR local machine
#
#   Access local serve from remote server (-R flag):
#     ssh -R 8080:localhost:3000 user@remote-server
#     Then open: http://localhost:8080 on THE REMOTE server
```

#### Option 3: Third-party Tunnels

Alternative tunneling solutions (no special flags needed):

**Cloudflare Tunnel** (free, no port forwarding):
```bash
mehr serve --port 3000
cloudflared tunnel --url http://localhost:3000
# Gives you: https://random-name.trycloudflare.com
```

**Tailscale** (mesh VPN):
```bash
mehr serve --host 0.0.0.0 --port 3000
# Access via Tailscale IP: http://100.x.x.x:3000
```

**ngrok**:
```bash
mehr serve --port 3000
ngrok http 3000
```

### Authentication Setup

```bash
# Add users before starting network-accessible server
mehr serve auth add admin secretpassword
mehr serve auth add developer devpass123

# List configured users
mehr serve auth list

# Change password
mehr serve auth passwd admin newpassword

# Remove user
mehr serve auth remove developer
```

### Project Registration

```bash
# Register current project for future remote features
cd ~/projects/my-app
mehr serve register

# List registered projects
mehr serve register --list

# Unregister
mehr serve unregister
```

### Integration with Workflows

The web UI provides real-time updates via Server-Sent Events (SSE) and exposes a REST API for workflow control:

```bash
# Start server in background
mehr serve &

# The API can be used programmatically
curl http://localhost:PORT/api/v1/status
curl -X POST http://localhost:PORT/api/v1/workflow/plan
```

## REST API Endpoints

### Authentication Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/login` | GET | Login page (HTML) |
| `/api/v1/auth/login` | POST | Submit login credentials |
| `/logout` | GET | Logout and clear session |
| `/api/v1/auth/logout` | POST | Logout (API) |

### Read Operations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (public) |
| `/api/v1/status` | GET | Server and workflow status |
| `/api/v1/context` | GET | Server context (workspace root, current task) |
| `/api/v1/task` | GET | Active task details (includes pending question if any) |
| `/api/v1/tasks` | GET | List all tasks in workspace |
| `/api/v1/tasks/{id}` | GET | Get specific task details |
| `/api/v1/tasks/{id}/specs` | GET | Task specifications |
| `/api/v1/tasks/{id}/sessions` | GET | Task sessions |
| `/api/v1/tasks/{id}/notes` | GET | Task notes content |
| `/api/v1/tasks/{id}/costs` | GET | Task token usage and cost breakdown |
| `/api/v1/projects` | GET | List all projects (global mode only) |
| `/api/v1/events` | GET | SSE stream for real-time updates |
| `/api/v1/guide` | GET | Get state-specific guidance on next actions |
| `/api/v1/agents` | GET | List available AI agents with capabilities and models |
| `/api/v1/providers` | GET | List available task providers |
| `/api/v1/workflow/diagram` | GET | Get SVG workflow state diagram |
| `/api/v1/costs` | GET | Get costs for all tasks with totals |

### Workflow Actions (POST)

| Endpoint | Description |
|----------|-------------|
| `/api/v1/workflow/start` | Start a new task |
| `/api/v1/workflow/plan` | Run planning phase |
| `/api/v1/workflow/implement` | Run implementation phase |
| `/api/v1/workflow/review` | Run review phase |
| `/api/v1/workflow/finish` | Complete the task |
| `/api/v1/workflow/undo` | Undo to previous checkpoint |
| `/api/v1/workflow/redo` | Redo to next checkpoint |
| `/api/v1/workflow/answer` | Submit answer to pending question |
| `/api/v1/workflow/abandon` | Abandon current task |
| `/api/v1/workflow/continue` | Resume workflow with optional auto-execute |
| `/api/v1/workflow/auto` | Run full automation cycle |
| `/api/v1/workflow/sync` | Sync task from provider and detect changes |
| `/api/v1/workflow/simplify` | Auto-simplify content based on workflow state |
| `/api/v1/tasks/{id}/notes` | Add a note to a task |

### Template Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/templates` | GET | List available templates |
| `/api/v1/templates/{name}` | GET | Get template details |
| `/api/v1/templates/apply` | POST | Apply template to a file |

### Browser Automation Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/browser/status` | GET | Browser connection status and tabs |
| `/api/v1/browser/tabs` | GET | List open browser tabs |
| `/api/v1/browser/goto` | POST | Open URL in new tab |
| `/api/v1/browser/navigate` | POST | Navigate existing tab to URL |
| `/api/v1/browser/screenshot` | POST | Capture screenshot |
| `/api/v1/browser/click` | POST | Click element by selector |
| `/api/v1/browser/type` | POST | Type text into element |
| `/api/v1/browser/eval` | POST | Execute JavaScript |
| `/api/v1/browser/dom` | POST | Query DOM elements |
| `/api/v1/browser/reload` | POST | Reload page |
| `/api/v1/browser/close` | POST | Close tab |

### Security Scanning

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/scan` | POST | Run security scans (gosec, gitleaks, govulncheck) |

### Memory/Search Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/search` | GET | Search semantic memory (`?q=query&limit=5&types=code,spec`) |
| `/api/v1/memory/index` | POST | Index a task to memory |
| `/api/v1/memory/stats` | GET | Memory system statistics |

### Settings Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/settings` | GET | Settings page (HTML) |
| `/api/v1/settings` | GET | Get current configuration |
| `/api/v1/settings` | POST | Save configuration changes |

## API Examples

### Continue Workflow

Resume work on the active task with optional auto-execution:

```bash
# Resume and get suggested next actions
curl -X POST http://localhost:8080/api/v1/workflow/continue \
  -H "Content-Type: application/json" \
  -d '{"auto": false}'

# Response:
{
  "success": true,
  "state": "idle",
  "next_actions": ["POST /api/v1/workflow/plan", "POST /api/v1/tasks/{id}/notes"],
  "message": "task resumed"
}

# Auto-execute next logical step
curl -X POST http://localhost:8080/api/v1/workflow/continue \
  -H "Content-Type: application/json" \
  -d '{"auto": true}'
```

### Full Automation

Run complete plan → implement → review → finish cycle:

```bash
curl -X POST http://localhost:8080/api/v1/workflow/auto \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "file:task.md",
    "max_retries": 3,
    "quality_target": "quality",
    "no_push": false
  }'

# Response:
{
  "success": true,
  "planning_done": true,
  "implement_done": true,
  "quality_attempts": 1,
  "quality_passed": true,
  "finish_done": true
}
```

### Get Guidance

Get state-specific suggestions for next actions:

```bash
curl http://localhost:8080/api/v1/guide

# Response:
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

### Add Notes

Add notes for AI context:

```bash
curl -X POST http://localhost:8080/api/v1/tasks/abc123/notes \
  -H "Content-Type: application/json" \
  -d '{"content": "Use the existing auth middleware"}'

# Response:
{
  "success": true,
  "was_answer": false,
  "message": "note added"
}
```

### Get Costs

View token usage and costs:

```bash
# Single task
curl http://localhost:8080/api/v1/tasks/abc123/costs

# All tasks with totals
curl http://localhost:8080/api/v1/costs
```

### Security Scan

Run security scans on the codebase:

```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanners": ["gosec", "gitleaks"],
    "fail_level": "high"
  }'

# Response:
{
  "findings": [...],
  "total_count": 5,
  "blocking_count": 2,
  "passed": false
}
```

### Memory Search

Search semantic memory:

```bash
curl "http://localhost:8080/api/v1/memory/search?q=authentication&limit=5&types=code,spec"

# Response:
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

### Sync Task

Sync a task from its provider to detect changes:

```bash
curl -X POST http://localhost:8080/api/v1/workflow/sync \
  -H "Content-Type: application/json" \
  -d '{"task_id": "abc123"}'

# Response:
{
  "success": true,
  "has_changes": true,
  "changes_summary": "Title changed, description updated",
  "spec_generated": ".mehrhof/workspaces/abc123/specs/delta-001.md",
  "message": "changes detected and delta specification generated"
}
```

### Simplify

Auto-simplify content based on workflow state:

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify \
  -H "Content-Type: application/json" \
  -d '{}'

# Response:
{
  "success": true,
  "simplified": "specifications",
  "message": "simplification complete"
}
```

### Templates

List, view, and apply task templates:

```bash
# List all templates
curl http://localhost:8080/api/v1/templates

# Response:
{
  "templates": [
    {"name": "feature", "description": "Standard feature implementation"},
    {"name": "bug-fix", "description": "Bug fix with reproduction steps"}
  ],
  "count": 2
}

# Get template details
curl http://localhost:8080/api/v1/templates/feature

# Apply template to file
curl -X POST http://localhost:8080/api/v1/templates/apply \
  -H "Content-Type: application/json" \
  -d '{"template_name": "feature", "file_path": "task.md"}'
```

### List Agents

View available AI agents with their capabilities and models:

```bash
curl http://localhost:8080/api/v1/agents

# Response:
{
  "agents": [
    {
      "name": "claude",
      "type": "builtin",
      "description": "Claude AI via Claude CLI",
      "version": "1.0.0",
      "available": true,
      "capabilities": {
        "streaming": true,
        "tool_use": true,
        "file_operations": true,
        "code_execution": false,
        "multi_turn": true
      },
      "models": [
        {"id": "claude-sonnet-4-20250514", "name": "Claude 3.5 Sonnet", "max_tokens": 200000, "default": true},
        {"id": "claude-opus-4-20250514", "name": "Claude 3 Opus", "max_tokens": 200000}
      ]
    },
    {
      "name": "work-fast",
      "type": "alias",
      "description": "Fast implementation with Sonnet",
      "extends": "claude",
      "available": true,
      "capabilities": {
        "streaming": true,
        "tool_use": true,
        "file_operations": true,
        "multi_turn": true
      },
      "models": [
        {"id": "claude-sonnet-4-20250514", "name": "Claude 3.5 Sonnet", "max_tokens": 200000}
      ]
    }
  ],
  "count": 2
}
```

### List Providers

View available task providers:

```bash
curl http://localhost:8080/api/v1/providers

# Response:
{
  "providers": [
    {
      "scheme": "file",
      "shorthand": "f",
      "name": "File",
      "description": "Load tasks from markdown files",
      "example": "file:task.md or f:task.md"
    },
    {
      "scheme": "github",
      "name": "GitHub",
      "description": "Load tasks from GitHub issues",
      "example": "github:123"
    },
    {
      "scheme": "jira",
      "name": "Jira",
      "description": "Load tasks from Jira tickets",
      "example": "jira:PROJ-123"
    }
  ],
  "count": 16
}
```

### Workflow Diagram

Get an SVG visualization of the workflow state machine:

```bash
# Get the diagram (returns SVG image)
curl http://localhost:8080/api/v1/workflow/diagram \
  --output workflow-diagram.svg

# Or embed directly in HTML
<img src="/api/v1/workflow/diagram" alt="Workflow State Diagram" />

# The SVG highlights the current state
# and shows all valid state transitions
```

The workflow diagram automatically highlights:
- **Current state** - Filled with accent color
- **Valid transitions** - Shown as arrows from current state
- **All states** - idle, planning, implementing, reviewing, done, failed, waiting, checkpointing, reverting, restoring

### Settings

View and modify workspace configuration:

```bash
# Get current settings
curl http://localhost:8080/api/v1/settings

# Response:
{
  "git": {
    "auto_commit": true,
    "branch_pattern": "{type}/{key}--{slug}"
  },
  "agent": {
    "default": "claude",
    "timeout": 300
  },
  ...
}

# Update settings (JSON)
curl -X POST http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/json" \
  -d '{
    "git": {"auto_commit": false},
    "agent": {"timeout": 600}
  }'

# Update settings (form)
curl -X POST http://localhost:8080/api/v1/settings \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "git.auto_commit=false&agent.timeout=600"
```

### Browser Automation

Control browser for testing:

```bash
# Check status
curl http://localhost:8080/api/v1/browser/status

# Navigate
curl -X POST http://localhost:8080/api/v1/browser/goto \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Take screenshot
curl -X POST http://localhost:8080/api/v1/browser/screenshot \
  -H "Content-Type: application/json" \
  -d '{"full_page": true}'

# Click element
curl -X POST http://localhost:8080/api/v1/browser/click \
  -H "Content-Type: application/json" \
  -d '{"selector": "#submit-button"}'
```

## Authentication Behavior

| Host Setting | Auth Required | Behavior |
|--------------|---------------|----------|
| `localhost` (default) | No | Direct access, no login |
| `127.0.0.1` | No | Direct access, no login |
| `0.0.0.0` | **Yes** | Must configure users first |
| Specific IP | **Yes** | Must configure users first |

When auth is enabled:
- All non-public endpoints require a valid session
- Sessions last 7 days by default
- Session cookies are HTTP-only and secure (when using HTTPS)

## Credential Storage

User credentials are stored at `~/.valksor/mehrhof/auth.yaml`:

```yaml
version: "1"
users:
  admin:
    username: admin
    password_hash: "$2a$10$..."  # bcrypt hash
    created_at: 2024-01-15T10:30:00Z
```

Passwords are hashed using bcrypt and never stored in plain text.

## Web UI Features

The web UI uses:
- **HTMX** for dynamic updates without full page reloads
- **Tailwind CSS** for styling with dark mode support
- **Server-Sent Events** for real-time state updates and agent output streaming

### Dashboard

The main dashboard shows:
- **Current server mode** (project/global)
- **Workflow state diagram** - Interactive visualization of current state and available transitions
- **Active task information** - Title, state, specifications progress
- **Real-time agent output** - Terminal-like view with color-coded logs and auto-scroll
- **Quick actions** - Start, plan, implement, review, finish, undo/redo buttons
- **Cost tracking** - Token usage with interactive charts
- **Guidance** - Context-aware suggestions on what to do next

### Browser Control Panel

The browser control panel (`/browser`) provides a web interface for browser automation:

- **Screenshot preview** - Live screenshot with auto-refresh option
- **Tab management** - List, switch between, and close browser tabs
- **DOM inspector** - Query DOM elements with CSS selectors
- **JavaScript console** - View console output (log levels: info, warn, error)
- **Navigation controls** - Back, forward, reload page
- **URL navigation** - Navigate to any URL
- **Element interaction** - Click and type into elements

### Task History

The task history page (`/history`) allows browsing and managing past tasks:

- **Search** - Find tasks by title or description
- **Filter** - Filter by state (done, failed, idle, etc.), date range, agent
- **Sort** - Sort by date, cost, duration
- **Timeline view** - Visual representation of task completion over time
- **Cost trends** - Charts showing cost patterns across tasks

### Settings Page

The settings page (`/settings`) allows you to configure all workspace settings through the web UI:

- **Git Settings**: auto_commit, sign_commits, commit_prefix, branch_pattern, stash_on_start, auto_pop_stash
- **Agent Settings**: default agent, timeout, max_retries, global instructions, step-specific agents
- **Agent Resolution** - Debug which agent is selected for each workflow step
- **Workflow Settings**: auto_init, session_retention_days, delete_work_on_finish, delete_work_on_abandon
- **Browser Settings**: enabled, headless, port, timeout, screenshot_dir, cookie settings
- **Provider Settings**: GitHub, GitLab, Jira, Linear, Notion, Bitbucket tokens and configuration
- **Provider Health** - Monitor connection status and rate limits for all configured providers

Changes are saved to `.mehrhof/config.yaml`.

### User Interface Features

- **Dark mode** - Toggle theme with automatic system preference detection
- **Mobile responsive** - Full feature support on mobile and tablet devices
  - Collapsible sidebar navigation (hamburger menu)
  - Touch-friendly controls (min-height 44px)
  - Stacked layouts on small screens
- **Notifications** - Stay informed of task events
  - Toast notifications for operation updates
  - Browser notifications for task completion (requires permission)
  - Notification center showing recent alerts

#### Global Mode Project Picker

In global mode (`mehr serve --global`), the settings page includes a project picker dropdown at the top of the page. This allows you to:

1. **Select a project** - Choose from all registered projects discovered in `~/.valksor/mehrhof/workspaces/`
2. **View project settings** - See the configuration for the selected project
3. **Edit non-sensitive settings** - Modify settings like timeouts, branch patterns, etc.
4. **Save changes** - Save settings to the selected project's config file

The project picker shows each project's ID and task count. When you select a project, the URL updates to include `?project=<id>` and the form displays that project's settings.

```bash
# Example: Start global server and access settings
mehr serve --global --port 3000

# Open in browser: http://localhost:3000/settings
# Select a project from the dropdown to view/edit its settings
```

**Security Note**: In Project mode, all settings including API tokens are editable. In Global mode, sensitive fields (API tokens, passwords) are hidden for security, even when a project is selected.

## Notes

- The server binds to `localhost` by default for security
- Use `--host 0.0.0.0` only after configuring authentication
- In project mode, the conductor is initialized at startup
- In global mode, projects are discovered from `~/.valksor/mehrhof/workspaces/`
- Use `Ctrl+C` to stop the server gracefully

## See Also

- [status](status.md) - Check task status from CLI
- [workflow](workflow.md) - Understanding the workflow state machine
