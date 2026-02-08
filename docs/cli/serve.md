# mehr serve

Start the web UI server for browser-based task management.

## Synopsis

```bash
mehr serve [flags]
```

## Description

The `serve` command starts a local HTTP server providing a web-based UI for managing Mehrhof tasks. It supports two modes:

- **Project mode** (default): Shows the current workspace's tasks, workflow state, and allows triggering workflow actions
- **Global mode**: Shows all projects discovered across your system with task summaries

## Flags

| Flag            | Type   | Default   | Description                                      |
|-----------------|--------|-----------|--------------------------------------------------|
| `--port`, `-p`  | int    | 6337      | Server port (0 = random, auto-fallback if taken) |
| `--global`      | bool   | false     | Run in global mode showing all projects          |
| `--open`        | bool   | false     | Automatically open browser after server starts   |
| `--api`         | bool   | false     | API-only mode (no web UI, for IDE plugins)       |

## Subcommands

### mehr serve register

Register the current project in the global registry.

```bash
# Register current project
mehr serve register

# List all registered projects
mehr serve register --list
```

### mehr serve unregister

Remove a project from the global registry.

```bash
# Unregister current project
mehr serve unregister

# Unregister by project ID
mehr serve unregister <project-id>
```

## Examples

### Basic Usage

```bash
# Start server in project mode (port 6337, or random if taken)
mehr serve

# Start on a specific port
mehr serve --port 8080

# Start and open browser automatically
mehr serve --open

# Start in global mode to see all projects
mehr serve --global
```

### API-Only Mode

Start the server without the web UI for IDE plugin integration (JetBrains, VS Code):

```bash
# API-only mode - no web UI, all /api/v1/* endpoints available
mehr serve --api --port 8080

```

**What's available in API-only mode:**
- ✅ All `/api/v1/*` endpoints including `/api/v1/interactive/*`
- ✅ Health check at `/health`
- ✅ Server-Sent Events at `/api/v1/events`
- ✅ Agent log streaming at `/api/v1/agent/logs/*`

**What's disabled in API-only mode:**
- ❌ Web UI pages (`/`, `/project`, `/interactive`, etc.)
- ❌ Static assets (`/static/*`)

**Use case:** IDE plugins connect via API endpoints and don't require the web UI.

## Notes

- The server binds to `localhost` by default for security
- In project mode, the conductor is initialized at startup
- In global mode, projects are discovered from `~/.valksor/mehrhof/workspaces/`
- Use `Ctrl+C` to stop the server gracefully

## See Also

- [status](status.md) - Check task status from CLI
- [workflow](workflow.md) - Understanding the workflow state machine
- [Web UI Documentation](/web-ui/) - Complete Web UI guide
