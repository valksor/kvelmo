# Mehrhof Docker Sandbox

Run mehrhof (`mehr`) inside Docker Sandbox for isolated, secure AI agent execution.

## Prerequisites

- **Docker Desktop 4.58+** with AI features enabled
  - Standard Docker CE/EE does NOT include the `docker sandbox` command
  - Enable in Docker Desktop: Settings → Features in development → Enable Docker AI features
- **macOS or Windows** (Linux uses legacy container-based sandboxes)
- **ANTHROPIC_API_KEY** set globally (see Troubleshooting)

> **Note**: This is different from mehr's `--sandbox` flag, which provides OS-level agent isolation (Linux namespaces, macOS sandbox-exec). Docker Sandbox provides microVM-level isolation for the entire mehr stack.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Docker Sandbox (microVM isolation)                      │
│                                                         │
│  cagent (optional) ─── CLI wrapper for conversation     │
│     │                                                   │
│     ▼ or user directly calls                            │
│  mehr ──────────────── workflow COORDINATOR             │
│     │                  (manages state machine)          │
│     ▼ spawns                                            │
│  Claude/Codex ──────── actual AI agents (do the work)   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Key insight**: `mehr` is NOT an AI agent - it's a coordinator that spawns Claude/Codex. The sandbox isolates the entire stack.

## Quick Start

### Option 1: Custom Template (Recommended)

```bash
# Build the template
make sandbox-build

# Run with Docker Sandbox
make sandbox-run

# Or manually:
docker sandbox run --load-local-template -t mehr-sandbox:v1 ~/my-project
```

### Option 2: cagent Wrapper

```bash
# Start sandbox with cagent
docker sandbox run cagent ~/my-project

# Copy the cagent config into the sandbox first
docker sandbox exec <sandbox-name> mkdir -p /workspace/.mehrhof
docker cp sandbox/mehrhof.cagent.yaml <sandbox-name>:/workspace/.mehrhof/

# Inside sandbox, run mehrhof cagent
cagent run /workspace/.mehrhof/mehrhof.cagent.yaml
```

> **Note**: The cagent YAML file is on your host machine. You need to copy it into the sandbox or use the custom template approach (Option 1) which pre-installs mehr.

## Files

| File | Purpose |
|------|---------|
| `Dockerfile.mehr` | Custom sandbox template with mehr pre-installed |
| `config.yaml.template` | Default mehrhof configuration for sandbox |
| `mehrhof.cagent.yaml` | cagent wrapper for conversational interaction |
| `README.md` | This file |

## Usage Patterns

### Direct CLI Usage

Inside the sandbox, use `mehr` commands directly:

```bash
mehr start https://github.com/org/repo/issues/123
mehr status
mehr plan
mehr implement
mehr review
mehr finish
```

### With cagent (Conversational)

cagent provides a conversational interface that calls mehr commands:

```
User: "Help me implement issue #45"

cagent: [calls mehr_start]
        [calls mehr_status]
        [calls mehr_plan]
        ... etc
```

### Shell Aliases

The template includes convenient aliases:

| Alias | Command |
|-------|---------|
| `m` | `mehr` |
| `ms` | `mehr status` |
| `mp` | `mehr plan` |
| `mi` | `mehr implement` |
| `mr` | `mehr review` |
| `mf` | `mehr finish` |
| `mu` | `mehr undo` |
| `mg` | `mehr guide` |

## Configuration

### Entrypoint Modes

The Dockerfile supports three entrypoint modes:

| Mode | Entrypoint | Use Case |
|------|------------|----------|
| **CLI (default)** | `bash` | Full shell access, cagent compatibility |
| **Interactive** | `mehr interactive` | REPL mode, mehr handles conversation |
| **Web UI** | `mehr serve` | Browser-based interface |

To change mode, edit `Dockerfile.mehr` and uncomment the desired entrypoint.

### Environment Variables

Set these before building or running:

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | API key for Claude |
| `MEHR_VERSION` | Version to install (default: latest) |
| `GO_VERSION` | Go version (default: 1.24.0) |

## Building from Source

To build with local source instead of release:

```bash
# Build with local source
docker build -f sandbox/Dockerfile.mehr \
  --build-arg MEHR_VERSION=local \
  -t mehr-sandbox:dev .
```

## Sharing

Push to Docker Hub for team sharing:

```bash
make sandbox-push

# Team members can then:
docker sandbox run --load-local-template -t valksor/mehr-sandbox:v1 ~/project
```

## Verification

After starting the sandbox:

```bash
# Verify mehr is installed
mehr version

# Check available commands
mehr --help

# Verify Go tools
golangci-lint --version
govulncheck -version
```

## Troubleshooting

### "mehr: command not found"

The PATH may not be set. Try:
```bash
export PATH=$PATH:/usr/local/bin
```

### "permission denied"

Ensure the workspace is owned by the agent user:
```bash
sudo chown -R agent:agent /workspace
```

### API key not working

Docker Sandbox daemon doesn't inherit shell environment. Set the key globally:
```bash
# Add to ~/.bashrc or ~/.zshrc on HOST
export ANTHROPIC_API_KEY=sk-ant-api03-xxxxx

# Then restart Docker Desktop
```

## Resources

- [Docker Sandboxes Docs](https://docs.docker.com/ai/sandboxes)
- [cagent Documentation](https://docs.docker.com/ai/cagent/)
- [Mehrhof Documentation](https://valksor.com/docs/mehrhof)
