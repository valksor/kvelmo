# Docker Sandbox Integration

Run mehrhof inside Docker Sandbox for microVM-level isolation.

## Overview

Docker Sandbox is a microVM-based isolation solution from Docker Desktop that runs AI coding agents in isolated virtual machines. This is **different** from mehr's `--sandbox` flag which provides OS-level isolation (see [Agent Sandboxing](/advanced/sandbox.md)).

| Feature             | mehr `--sandbox`                   | Docker Sandbox       |
|---------------------|------------------------------------|----------------------|
| **Isolation level** | OS-level (namespaces/sandbox-exec) | MicroVM (hypervisor) |
| **Kernel**          | Shared with host                   | Separate kernel      |
| **Docker access**   | Host daemon                        | Private daemon       |
| **Network**         | Shared with host                   | Isolated with proxy  |
| **File access**     | Bind mounts                        | File sync            |
| **Platform**        | Linux, macOS                       | macOS, Windows       |

**Architecture:**
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

## Prerequisites

- **Docker Desktop 4.58+** with AI features enabled
  - Standard Docker CE/EE does NOT include `docker sandbox`
  - Enable: Settings → Features in development → Enable Docker AI features
- **macOS or Windows** (Linux uses legacy container-based sandboxes)
- **ANTHROPIC_API_KEY** environment variable set globally

## Quick Start

### Build the Template

```bash
make sandbox-build
```

This creates `mehr-sandbox:v1` with:
- mehr binary pre-installed
- Go 1.24+ toolchain
- golangci-lint, govulncheck, gofumpt, goimports
- Default mehrhof configuration
- Shell aliases (`m`, `ms`, `mp`, `mi`, `mr`, `mf`, `mu`, `mg`)

### Run with Docker Sandbox

```bash
# Run in current directory
make sandbox-run

# Or manually:
docker sandbox run --load-local-template -t mehr-sandbox:v1 ~/my-project
```

### Use mehr Inside the Sandbox

```bash
# Full workflow
mehr start https://github.com/org/repo/issues/123
mehr plan
mehr implement
mehr review
mehr finish

# Or use aliases
ms      # mehr status
mp      # mehr plan
mi      # mehr implement
mr      # mehr review
```

## Build Options

### From Release (Default)

Uses pre-built binary from GitHub releases:

```bash
make sandbox-build
# Equivalent to:
docker build -f sandbox/Dockerfile.mehr -t mehr-sandbox:v1 .
```

### From Local Source

Build from your local changes:

```bash
make sandbox-build-dev
# Equivalent to:
make build
cp build/mehr sandbox/mehr-local
docker build -f sandbox/Dockerfile.mehr --build-arg MEHR_VERSION=local -t mehr-sandbox:dev .
```

### Custom Version

Specify a release version:

```bash
docker build -f sandbox/Dockerfile.mehr \
  --build-arg MEHR_VERSION=v1.5.0 \
  -t mehr-sandbox:v1.5.0 .
```

## cagent Integration

Docker's [cagent](https://docs.docker.com/ai/cagent/) provides a conversational wrapper for mehr commands.

### Setup

```bash
# Start sandbox with cagent
docker sandbox run cagent ~/my-project

# Copy cagent config into sandbox
docker sandbox exec <sandbox-name> mkdir -p /workspace/.mehrhof
docker cp sandbox/mehrhof.cagent.yaml <sandbox-name>:/workspace/.mehrhof/

# Run mehrhof cagent inside sandbox
cagent run /workspace/.mehrhof/mehrhof.cagent.yaml
```

### How cagent Works

cagent is a conversation layer - it doesn't do AI work itself:

```
User: "Help me implement issue #45"

cagent: [calls mehr_start]     → mehr manages state
        [calls mehr_plan]      → mehr spawns Claude for planning
        [calls mehr_implement] → mehr spawns Claude for coding
        [calls mehr_review]    → mehr spawns Claude for review
        [calls mehr_finish]    → mehr completes workflow
```

The AI work happens when mehr spawns Claude/Codex internally.

### Available cagent Tools

The `mehrhof.cagent.yaml` exposes 35+ commands:

| Category       | Commands                                                                                                   |
|----------------|------------------------------------------------------------------------------------------------------------|
| **Workflow**   | `mehr_start`, `mehr_plan`, `mehr_implement`, `mehr_review`, `mehr_finish`, `mehr_abandon`, `mehr_continue` |
| **Navigation** | `mehr_status`, `mehr_guide`, `mehr_undo`, `mehr_redo`, `mehr_reset`, `mehr_cost`                           |
| **Context**    | `mehr_note`, `mehr_question`, `mehr_memory`, `mehr_links`                                                  |
| **Project**    | `mehr_list`, `mehr_project`, `mehr_stack`, `mehr_sync`, `mehr_submit`, `mehr_export`, `mehr_delete`        |
| **Quality**    | `mehr_scan`, `mehr_find`, `mehr_optimize`, `mehr_simplify`, `mehr_commit`                                  |
| **Config**     | `mehr_agents`, `mehr_providers`, `mehr_config`, `mehr_templates`, `mehr_plugins`, `mehr_init`              |
| **Services**   | `mehr_browser`, `mehr_workflow`                                                                            |

## Entrypoint Modes

The Dockerfile supports three entrypoint modes:

| Mode              | Entrypoint         | Use Case                                |
|-------------------|--------------------|-----------------------------------------|
| **CLI (default)** | `bash`             | Full shell access, cagent compatibility |
| **Interactive**   | `mehr interactive` | REPL mode, mehr handles conversation    |
| **Web UI**        | `mehr serve`       | Browser-based interface                 |

To change mode, edit `sandbox/Dockerfile.mehr` and uncomment the desired entrypoint.

### Interactive Mode

```bash
make sandbox-interactive
# Or:
docker sandbox run --load-local-template -t mehr-sandbox:v1 ~/my-project -- mehr interactive
```

## Configuration

### Default Config

The sandbox uses `sandbox/config.yaml.template`:

```yaml
agent:
  default: claude
  retry_count: 3
  retry_delay: 5s
  steps:
    planning: { name: claude }
    implementing: { name: claude-sonnet }
    reviewing: { name: claude }

security:
  allow_shell: true
  allow_file_write: true
```

### Custom Config

Mount your own config:

```bash
docker sandbox run --load-local-template -t mehr-sandbox:v1 ~/my-project
# Then inside sandbox:
cp /workspace/.mehrhof/config.yaml /home/agent/.mehrhof/config.yaml
```

## Sharing

### Push to Docker Hub

```bash
make sandbox-push
# Pushes to: valksor/mehr-sandbox:v1
```

### Team Usage

```bash
# Team members can run directly:
docker sandbox run --load-local-template -t valksor/mehr-sandbox:v1 ~/project
```

## CLI Reference

| Command                    | Description                         |
|----------------------------|-------------------------------------|
| `make sandbox-build`       | Build sandbox template from release |
| `make sandbox-build-dev`   | Build from local source             |
| `make sandbox-run`         | Run sandbox in current directory    |
| `make sandbox-interactive` | Run in interactive mode             |
| `make sandbox-push`        | Push to Docker Hub                  |
| `make sandbox-ls`          | List running sandboxes              |
| `make sandbox-clean`       | Remove sandbox VM                   |

## Troubleshooting

### "mehr: command not found"

PATH may not be set. Try:
```bash
export PATH=$PATH:/usr/local/bin
```

### "permission denied"

Ensure workspace is owned by agent user:
```bash
sudo chown -R agent:agent /workspace
```

### API key not working

Docker Sandbox daemon doesn't inherit shell environment. Set globally:
```bash
# Add to ~/.bashrc or ~/.zshrc on HOST
export ANTHROPIC_API_KEY=sk-ant-api03-xxxxx

# Then restart Docker Desktop
```

### "docker sandbox: command not found"

Docker Sandbox requires Docker Desktop 4.58+ with AI features enabled:
- Settings → Features in development → Enable Docker AI features
- Standard Docker CE/EE doesn't include this feature

## Security Model

Docker Sandbox provides stronger isolation than mehr's `--sandbox` flag:

| Feature              | Protection                       |
|----------------------|----------------------------------|
| **MicroVM boundary** | Agents can't escape to host      |
| **Separate kernel**  | Kernel exploits contained        |
| **Private Docker**   | Can build/run containers safely  |
| **Network proxy**    | HTTP/HTTPS filtering available   |
| **File sync**        | Only synced workspace accessible |

### When to Use Which

| Scenario                 | Recommended              |
|--------------------------|--------------------------|
| Quick local development  | mehr `--sandbox`         |
| Untrusted code execution | Docker Sandbox           |
| CI/CD pipelines          | Docker Sandbox           |
| Team sharing             | Docker Sandbox (via Hub) |
| Offline work             | mehr `--sandbox`         |

## See Also

- [Agent Sandboxing](/advanced/sandbox.md) - OS-level `--sandbox` flag
- [Security Scanning](/advanced/security.md) - SAST, secrets, vulnerabilities
- [Docker Sandboxes Docs](https://docs.docker.com/ai/sandboxes)
- [cagent Documentation](https://docs.docker.com/ai/cagent/)
