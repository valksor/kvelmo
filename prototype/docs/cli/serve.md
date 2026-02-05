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

| Flag            | Type   | Default   | Description                                      |
|-----------------|--------|-----------|--------------------------------------------------|
| `--port`, `-p`  | int    | 6337      | Server port (0 = random, auto-fallback if taken) |
| `--host`        | string | localhost | Host to bind to (use 0.0.0.0 for network access) |
| `--global`      | bool   | false     | Run in global mode showing all projects          |
| `--open`        | bool   | false     | Automatically open browser after server starts   |
| `--api`         | bool   | false     | API-only mode (no web UI, for IDE plugins)       |
| `--tunnel-info` | bool   | false     | Show SSH tunnel instructions for remote access   |

## Subcommands

### mehr serve auth

Manage authentication for network-accessible servers.

#### mehr serve auth add

Add a new user for authentication.

```bash
mehr serve auth add <username> <password>
```

##### --role flag

Specify the user role:

```bash
# Add full-access user (default)
mehr serve auth add admin mypassword

# Add read-only viewer
mehr serve auth add stakeholder viewpass123 --role viewer
```

**Available roles:**
- `user` - Full access (default)
- `viewer` - Read-only access (can view but not modify)

**Viewer permissions:**
- ✅ View dashboard, tasks, specifications, history, logs
- ✅ View settings and configuration values
- ❌ Cannot start, modify, or interact with workflows
- ❌ Cannot submit projects or quick tasks
- ❌ Cannot change settings
- ❌ Cannot run security/quality scans

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

#### mehr serve auth role

Change a user's role.

```bash
# Promote viewer to full user
mehr serve auth role stakeholder user

# Demote user to viewer
mehr serve auth role contractor viewer
```

**Valid roles:** `user`, `viewer`

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

# API-only with network binding (authentication required)
mehr serve --api --host 0.0.0.0 --port 8080
```

**What's available in API-only mode:**
- ✅ All `/api/v1/*` endpoints including `/api/v1/interactive/*`
- ✅ Health check at `/health`
- ✅ Server-Sent Events at `/api/v1/events`
- ✅ Agent log streaming at `/api/v1/agent/logs/*`

**What's disabled in API-only mode:**
- ❌ Web UI pages (`/`, `/project`, `/interactive`, etc.)
- ❌ Static assets (`/static/*`)
- ❌ UI partials (`/ui/partials/*`)

**Use case:** IDE plugins connect via API endpoints and don't require the web UI.

### Remote Access

#### Direct Network Binding

For machines exposed to the network or internet:

```bash
# First, set up authentication
mehr serve auth add admin mypassword

# Then start with network binding
mehr serve --host 0.0.0.0 --port 8080
```

**Important**: Authentication is **required** when using `--host 0.0.0.0` or any non-localhost address.

#### SSH Tunnel

For secure remote access without exposing ports:

```bash
# Show tunnel instructions (helper flag - exits without starting server)
mehr serve --tunnel-info

# Output includes SSH tunnel instructions:
# SSH Tunnel Instructions:
#   Access remote serve from your local machine (-L flag):
#     ssh -L 8080:localhost:6337 user@remote-server
#     Then open: http://localhost:8080 on YOUR local machine
#
#   Access local serve from remote server (-R flag):
#     ssh -R 8080:localhost:6337 user@remote-server
#     Then open: http://localhost:8080 on THE REMOTE server
```

#### Third-party Tunnels

Alternative tunneling solutions (no special flags needed):

**Cloudflare Tunnel** (free, no port forwarding):
```bash
mehr serve
cloudflared tunnel --url http://localhost:6337
# Gives you: https://random-name.trycloudflare.com
```

**Tailscale** (mesh VPN):
```bash
mehr serve --host 0.0.0.0
# Access via Tailscale IP: http://100.x.x.x:6337
```

**ngrok**:
```bash
mehr serve
ngrok http 6337
```

### Authentication Setup

```bash
# Add users before starting network-accessible server
mehr serve auth add admin secretpassword
mehr serve auth add developer devpass123
mehr serve auth add stakeholder viewpass123 --role viewer

# List configured users (shows roles)
mehr serve auth list

# Change password
mehr serve auth passwd admin newpassword

# Change user role
mehr serve auth role stakeholder user

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

## Security Middleware

When authentication is enabled (`--host 0.0.0.0`), the server activates additional security layers:

### CSRF Protection

All state-changing requests (POST, PUT, DELETE) require a valid `X-Csrf-Token` header. The token is:

- Returned in the login response JSON (`csrf_token` field)
- Available via `GET /api/v1/auth/csrf` for existing sessions
- Per-session and validated against the server-side session

**Exempt endpoints:** GET/HEAD/OPTIONS methods, `/api/v1/auth/login`, `/api/v1/webhooks/*`.

The Web UI handles CSRF automatically. IDE plugins include CSRF infrastructure for remote server connections.

### Rate Limiting

Per-IP rate limiting protects against brute-force and abuse:

| Endpoint Type  | Limit       |
|----------------|-------------|
| General API    | 120 req/min |
| Auth endpoints | 10 req/min  |

Exceeding the limit returns **HTTP 429 Too Many Requests**.

### Localhost Mode

Both CSRF and rate limiting are **automatically disabled** when the server runs on localhost (default). This keeps the development experience frictionless while protecting network-accessible deployments.

See [Web UI: Authentication](/web-ui/authentication.md) for full details.

## Notes

- The server binds to `localhost` by default for security
- Use `--host 0.0.0.0` only after configuring authentication
- In project mode, the conductor is initialized at startup
- In global mode, projects are discovered from `~/.valksor/mehrhof/workspaces/`
- Use `Ctrl+C` to stop the server gracefully

## See Also

- [status](status.md) - Check task status from CLI
- [workflow](workflow.md) - Understanding the workflow state machine
- [Web UI Documentation](/web-ui/) - Complete Web UI guide
