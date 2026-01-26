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
#     ssh -L 8080:localhost:3000 user@remote-server
#     Then open: http://localhost:8080 on YOUR local machine
#
#   Access local serve from remote server (-R flag):
#     ssh -R 8080:localhost:3000 user@remote-server
#     Then open: http://localhost:8080 on THE REMOTE server
```

#### Third-party Tunnels

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

## Web UI Documentation

The Web UI provides comprehensive features including:

- **Dashboard** - Task management, real-time updates, workflow visualization
- **Settings** - Configure agents, providers, git, and workflow options
- **Browser Control** - Automate Chrome for testing and authentication
- **REST API** - Programmatic access to all features
- **Remote Access** - SSH tunnels, Cloudflare, Tailscale integration

For complete Web UI documentation, see:
- [Web UI Overview](../web-ui/index.md)
- [Getting Started](../web-ui/getting-started.md)
- [REST API Reference](../web-ui/api.md)
- [Remote Access](../web-ui/remote-access.md)
- [Authentication](../web-ui/authentication.md)

## Notes

- The server binds to `localhost` by default for security
- Use `--host 0.0.0.0` only after configuring authentication
- In project mode, the conductor is initialized at startup
- In global mode, projects are discovered from `~/.valksor/mehrhof/workspaces/`
- Use `Ctrl+C` to stop the server gracefully

## See Also

- [status](status.md) - Check task status from CLI
- [workflow](workflow.md) - Understanding the workflow state machine
- [Web UI Documentation](../web-ui/) - Complete Web UI guide
