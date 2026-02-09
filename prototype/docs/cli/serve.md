# mehr serve

Start the web UI server for browser-based task management.

## Synopsis

```bash
mehr serve [flags]
```

## Description

The `serve` command starts a local HTTP server providing a web-based UI for managing Mehrhof tasks. It supports two modes:

- **Project mode** (default): Shows the current workspace's tasks, workflow state, and allows triggering workflow actions
- **Global mode**: Shows all recently accessed projects across your system with task summaries

Projects are automatically tracked when you run `mehr serve` in a directory. This means projects you work on will appear in global mode without any manual registration.

## Flags

| Flag            | Type   | Default   | Description                                      |
|-----------------|--------|-----------|--------------------------------------------------|
| `--port`, `-p`  | int    | 6337      | Server port (0 = random, auto-fallback if taken) |
| `--global`      | bool   | false     | Run in global mode showing all projects          |
| `--open`        | bool   | false     | Automatically open browser after server starts   |
| `--api`         | bool   | false     | API-only mode (no web UI, for IDE plugins)       |

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
- All `/api/v1/*` endpoints including `/api/v1/interactive/*`
- Health check at `/health`
- Server-Sent Events at `/api/v1/events`
- Agent log streaming at `/api/v1/agent/logs/*`

**What's disabled in API-only mode:**
- Web UI pages (`/`, `/project`, `/interactive`, etc.)
- Static assets (`/static/*`)

**Use case:** IDE plugins connect via API endpoints and don't require the web UI.

## Project Tracking

When you run `mehr serve` in a project directory, it is automatically added to your recent projects list. This enables:

- Quick access in global mode (`mehr serve --global`)
- Project selection in the Desktop App
- Synchronized project history across all interfaces

The tracking data is stored at `~/.valksor/mehrhof/projects.json`.

### Managing Projects

In global mode or the Desktop App:

- **Favorites**: Star frequently-used projects to keep them at the top
- **Remove**: Remove projects from the list that you no longer need

## Notes

- The server binds to `localhost` by default for security
- In project mode, the conductor is initialized at startup
- Use `Ctrl+C` to stop the server gracefully
- Projects are tracked automatically - no registration required

## See Also

- [status](status.md) - Check task status from CLI
- [workflow](workflow.md) - Understanding the workflow state machine
- [Web UI Documentation](/web-ui/) - Complete Web UI guide
- [Desktop App](/desktop/) - Native desktop application
