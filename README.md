# Valksor Mehrhof - AI-Powered Task Automation CLI

[![valksor](https://badgen.net/static/org/valksor/green)](https://github.com/valksor)
[![BSD-3-Clause](https://img.shields.io/badge/BSD--3--Clause-green?style=flat)](https://github.com/valksor/go-mehrhof/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/valksor/go-mehrhof.svg?style=flat)](https://github.com/valksor/go-mehrhof/releases/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/valksor/go-mehrhof.svg?style=flat)]() 
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/valksor/go-mehrhof)

[![Coverage Status](https://coveralls.io/repos/github/valksor/go-mehrhof/badge.svg?branch=master)](https://coveralls.io/github/valksor/go-mehrhof?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/valksor/go-mehrhof)](https://goreportcard.com/report/github.com/valksor/go-mehrhof)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/valksor/go-mehrhof)

---

**⚠️ EXPERIMENTAL INTEGRATIONS**

Mehrhof's core workflow engine is stable, but **provider and agent integrations with third-party services are experimental**.

Due to the large number of external APIs (GitHub, GitLab, Jira, Notion, Claude, Gemini, etc.), integrations may:
- Break without notice due to third-party API changes
- Have edge cases not covered by automated tests
- Require manual validation for production use

We are gradually testing and hardening integrations. Report issues at [github.com/valksor/go-mehrhof/issues](https://github.com/valksor/go-mehrhof/issues).

---

## Why Mehrhof?

Mehrhof is a command-line tool that orchestrates AI agents to perform **planning, implementation, and code review** workflows for software development tasks.

**Key benefits:**
- **Parallel tasks** - Run multiple AI tasks simultaneously using git worktrees for isolated development
- **Workflow engine** - Reliable plan → implement → review → finish cycle with checkpointing and undo/redo
- **Provider integrations** - Connect to 16+ task sources (Empty, Files, GitHub issues, Jira, Linear, Notion, etc.)
- **Browser automation** - Chrome automation for web testing, scraping, and authentication flows
- **MCP server** - Expose commands and workspace data to AI agents via Model Context Protocol
- **Semantic memory** - Store and search past tasks using vector embeddings for context-aware AI
- **Security scanning** - Integrated SAST (gosec), secret detection (gitleaks), and vulnerability scanning (govulncheck) with automatic tool downloading and caching
- **Multi-agent orchestration** - Run multiple agents in parallel, sequentially, or consensus modes
- **ML predictions** - Predict task complexity and resource requirements from historical data
- **State tracking** - Task state persists across sessions; resume anytime with `mehr continue`
- **Auto mode** - Fully automated workflow: `mehr auto file:task.md` handles everything
- **Prompt optimization** - Automatically refine prompts for clarity and effectiveness with `--optimize`
- **Self-updating** - Auto-update from GitHub releases, no manual reinstall

## Quick Start

### Prerequisites

- **Go 1.25+** (for building from source)
- **Git** (for version control operations)
- **Claude CLI** - Mehrhof wraps Claude CLI for AI operations (see [quickstart](https://mehrhof.valksor.com/docs/#/quickstart))
- **Chrome or Chromium** (optional, required for browser automation features)

> **Browser Features**: Browser automation (`mehr browser` commands) requires Chrome or Chromium to be installed. See [Browser Documentation](https://mehrhof.valksor.com/docs/#/cli/browser) for installation instructions.

### Installation

#### Option 1: Install Script (Recommended)

```bash
# Latest stable release
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash

# Nightly build (latest master)
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash -s -- --nightly

# Specific version
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash -s -- -v v1.2.3
```

The install script auto-detects your OS/architecture, verifies checksums (and Cosign signatures if available), and installs to `~/.local/bin` (or `/usr/local/bin` with sudo).

#### Option 2: Pre-built Binary

Download the latest release for your platform:

| Platform | Architecture | Binary Name |
|----------|--------------|-------------|
| Linux | AMD64 | `mehr-linux-amd64` |
| Linux | ARM64 | `mehr-linux-arm64` |
| macOS | AMD64 (Intel) | `mehr-darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `mehr-darwin-arm64` |

> **Windows Users:** Native Windows binaries are not provided. Use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/) (Windows Subsystem for Linux) and download the appropriate Linux binary (`mehr-linux-amd64` or `mehr-linux-arm64`).

```bash
# Download and install (example for macOS ARM64)
curl -L https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-arm64 -o mehr
chmod +x mehr
sudo mv mehr /usr/local/bin/

# Verify
mehr version
```

#### Option 3: Nightly Build (Pre-release)

Get the latest commit build (always available, use with caution):

```bash
# Download latest nightly (example for macOS ARM64)
curl -L https://github.com/valksor/go-mehrhof/releases/download/nightly/mehr-darwin-arm64 -o mehr
chmod +x mehr
sudo mv mehr /usr/local/bin/
```

**Note:** Nightly builds are pre-release and may contain untested changes. Prefer stable releases when available.

#### Option 4: Build from Source

```bash
git clone https://github.com/valksor/go-mehrhof.git
cd go-mehrhof
make install

# Verify
mehr version
```

### First Task

```bash
# 1. Initialize workspace (one-time)
mehr init

# 2. Create a task file
cat > task.md << 'EOF'
---
title: Add health check endpoint
---
Create a /health endpoint that returns HTTP 200 with JSON status.
EOF

# 3. Run workflow
mehr start task.md    # Creates git branch
mehr plan             # AI creates specifications (implementation plan)
mehr implement        # AI writes code following specifications
mehr finish           # Merge changes or create PR
```

**[See full guides](https://mehrhof.valksor.com/docs)** for:
- Parallel tasks with worktrees
- Provider integrations (GitHub, Jira, Notion, etc.)
- Configuration options
- Troubleshooting

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  mehr init  →  mehr start  →  mehr plan  →                  │
│       ↓                                                    │
│  mehr simplify  ←  (at any stage to refine content)         │
│       ↓                                                    │
│  →  mehr implement  →  mehr review  →  mehr finish          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

1. **Initialize** (`mehr init`) - Set up workspace (one-time)
2. **Start** (`mehr start`) - Begin a task; creates git branch automatically
3. **Plan** (`mehr plan`) - AI generates implementation specifications
4. **Simplify** (`mehr simplify`) - Refine content based on current state (optional)
5. **Implement** (`mehr implement`) - AI executes the specifications
6. **Review** (`mehr review`) - Run automated code review
7. **Finish** (`mehr finish`) - Merge changes and clean up

**Recovery commands**:
- `mehr continue` - Resume workflow, optionally auto-execute (`--auto`)
- `mehr undo` / `mehr redo` - Revert to previous checkpoint
- `mehr abandon` - Abandon task without merging
- `mehr simplify` - Auto-detects what to simplify (task input, specs, or code)

## Essential Commands

| Command | Description |
|---------|-------------|
| `mehr init` | Initialize workspace (creates `.mehrhof/config.yaml`; task data in `~/.valksor/mehrhof/`) |
| `mehr start <ref>` | Start task from file, directory, or provider |
| `mehr sync <task-id>` | Sync task from provider and generate delta specification if changed |
| `mehr auto <ref>` | Full automation: plan → implement → review → finish |
| `mehr plan` | Generate AI implementation specifications |
| `mehr implement` | Execute the specifications |
| `mehr simplify` | Refine content based on current workflow state (task input, specs, or code) |
| `mehr review` | Run automated code review |
| `mehr status` | Show full task details |
| `mehr guide` | What should I do next? (quick suggestion) |
| `mehr continue` | Resume work on task |
| `mehr finish` | Complete task and merge changes |
| `mehr list` | List all tasks in workspace |
| `mehr undo` / `mehr redo` | Navigate checkpoints |
| `mehr note <msg>` | Add notes for AI context |
| `mehr browser` | Browser automation commands (goto, screenshot, click, etc.) |
| `mehr mcp` | Start MCP server for AI agent integration |
| `mehr scan` | Run security scanners (SAST, secrets, dependencies) |
| `mehr serve` | Start web UI server (includes auth, register subcommands) |
| `mehr project plan` | Create task breakdown from source with dependencies |
| `mehr project submit` | Submit tasks to provider with dependencies |

**Tip:** Use command shortcuts for faster typing: `mehr gu` → `guide`, `mehr config:v` → `config validate`.

**See [CLI Reference](https://mehrhof.valksor.com/docs/#/cli/index) for all commands and flags.**

## Task Providers

Mehrhof supports 16+ task sources. Use provider schemes to load tasks:

> **Security**: Provider login commands (`mehr github login`, etc.) use secure password-style input. Tokens are masked with asterisks (`****`) when entered and never displayed in the terminal.

| Provider | Scheme | Example | Docs |
|----------|--------|---------|------|
| Empty | `empty:` | `empty:FEATURE-1` | [empty](https://mehrhof.valksor.com/docs/#/providers/empty) |
| File | `file:` | `file:task.md` | [file](https://mehrhof.valksor.com/docs/#/providers/file) |
| Directory | `dir:` | `dir:./tasks/` | [directory](https://mehrhof.valksor.com/docs/#/providers/directory) |
| GitHub | `github:` | `github:123` | [github](https://mehrhof.valksor.com/docs/#/providers/github) |
| GitLab | `gitlab:` | `gitlab:123` | [gitlab](https://mehrhof.valksor.com/docs/#/providers/gitlab) |
| Bitbucket | `bitbucket:` | `bitbucket:123` | [bitbucket](https://mehrhof.valksor.com/docs/#/providers/bitbucket) |
| Jira | `jira:` | `jira:PROJ-123` | [jira](https://mehrhof.valksor.com/docs/#/providers/jira) |
| Linear | `linear:` | `linear:ENG-123` | [linear](https://mehrhof.valksor.com/docs/#/providers/linear) |
| Asana | `asana:` | `asana:1234...` | [asana](https://mehrhof.valksor.com/docs/#/providers/asana) |
| ClickUp | `clickup:` | `clickup:abc123` | [clickup](https://mehrhof.valksor.com/docs/#/providers/clickup) |
| Azure DevOps | `azdo:` | `azdo:123` | [azure-devops](https://mehrhof.valksor.com/docs/#/providers/azure-devops) |
| Notion | `notion:` | `notion:<uuid>` | [notion](https://mehrhof.valksor.com/docs/#/providers/notion) |
| Trello | `trello:` | `trello:<id>` | [trello](https://mehrhof.valksor.com/docs/#/providers/trello) |
| Wrike | `wrike:` | `wrike:<id>` | [wrike](https://mehrhof.valksor.com/docs/#/providers/wrike) |
| YouTrack | `youtrack:` | `youtrack:ABC-123` | [youtrack](https://mehrhof.valksor.com/docs/#/providers/youtrack) |

**Default provider**: Configure in `.mehrhof/config.yaml` to use bare references:
```yaml
providers:
  default: file  # "mehr start task.md" works without "file:" prefix
```

## Project Planning with Dependencies

Plan multi-task projects with dependency tracking:

```bash
# Create task breakdown from specs
mehr project plan dir:/workspace/.final/ --title "Auth System"

# Plan from a provider task (fetches details from provider)
mehr project plan github:123 --title "From Issue"

# View tasks and dependencies
mehr project tasks --show-deps

# Let AI optimize task order based on dependencies
mehr project reorder --auto

# Submit to provider with dependencies
mehr project submit --provider wrike

# Auto-implement all tasks in order
mehr project start --auto
```

**Features:**
- **Multiple source types** - Plan from directories, files, provider references, or URLs (via web API)
- **AI task ordering** - Automatically optimize task order based on dependencies and priorities
- **Provider integration** - Fetch task details from any supported provider (GitHub, Jira, etc.)

**Supported dependency types by provider:**

| Provider | Dependency Support |
|----------|-------------------|
| Wrike | Native (FinishToStart) |
| GitHub | Task lists in epic |
| GitLab | Task lists in description |
| Jira | Issue links (blocks/is-blocked-by) |
| Asana | Native task dependencies |
| ClickUp | Native task dependencies |
| Azure DevOps | Work item links |
| Linear, Trello, YouTrack, Bitbucket | Description-based |

See [Project Planning documentation](https://mehrhof.valksor.com/docs/#/cli/project) for full workflow details.

## Parallel Tasks with Worktrees

A **git worktree** is a separate working directory linked to the same git repository, allowing you to work on multiple branches simultaneously without stashing.

Run multiple tasks simultaneously in isolated environments:

```bash
# Terminal 1
mehr start --worktree feature-a.md
cd ../project-worktrees/<task-id>
mehr plan && mehr implement

# Terminal 2 (from main repo)
mehr start --worktree feature-b.md
cd ../project-worktrees/<task-id>
mehr plan && mehr implement
```

Each worktree is an isolated git checkout. Mehrhof auto-detects which task you're working on based on your current directory.

## AI Agents

> **⚠️ Claude is the Primary Supported Agent**
>
> Mehrhof is designed and optimized for **Claude**. See AI Agents documentation for other options.

Mehrhof supports AI agent plugins for custom backends. The primary agent is **Claude**, which is fully integrated with Mehrhof's workflow engine.

| Agent | Description |
|-------|-------------|
| Claude | Primary agent via Claude CLI (recommended) |

**See [AI Agents documentation](https://mehrhof.valksor.com/docs/#/agents/index) for configuration and custom aliases.**

## Web UI

Mehrhof includes a web-based UI for browser-based task management with full CLI feature parity:

```bash
# Start web UI (project mode)
mehr serve

# Start on specific port and open browser
mehr serve --port 3000 --open

# Global mode - see all projects
mehr serve --global

# Remote access (requires authentication)
mehr serve auth add admin mypassword
mehr serve --host 0.0.0.0 --port 8080
```

The web UI provides:
- **Full workflow control** - Start, plan, implement, review, finish, undo/redo via REST API
- **Auto mode** - Run complete automation cycles (`POST /api/v1/workflow/auto`)
- **Continue/resume** - Resume work with optional auto-execute next step
- **Real-time updates** - Server-Sent Events for live state changes
- **Cost tracking** - View token usage and costs per task or across all tasks
- **Guidance** - State-specific suggestions on what to do next
- **Notes** - Add notes to tasks for AI context
- **Settings page** - Configure all workspace settings via web UI (Git, Agent, Workflow, Browser, Providers)
- **Global mode project picker** - In global mode, select and configure any registered project's settings
- **Browser automation** - Control Chrome for testing via API
- **Security scanning** - Run SAST, secrets, and vulnerability scans
- **Memory search** - Query semantic memory for past task context
- **Agent/provider info** - List available AI agents and task providers
- **Authentication** - Secure access for network-accessible servers
- **SSH tunnel support** - Secure remote access

**See [Web UI documentation](https://mehrhof.valksor.com/docs/#/cli/serve) for API details and examples.**

## Configuration

**Project-level** (`.mehrhof/config.yaml`):
```yaml
# Git integration
git:
  auto_commit: true
  commit_prefix: "[{key}]"
  branch_pattern: "{type}/{key}--{slug}"
  target_branch: "main"

# Agent configuration
agent:
  default: claude
  timeout: 300

# Default provider for bare references
providers:
  default: file

# Auto-update checks
update:
  enabled: true
  check_interval: 24  # hours
```

**Note**: Task data (specifications, sessions, notes) is stored in `~/.valksor/mehrhof/workspaces/<project-id>/` to keep project directories clean.

**See [Configuration Guide](https://mehrhof.valksor.com/docs/#/configuration/index) for all options including agent aliases, per-step agents, and provider settings.**

## Documentation

- 📖 [Full Documentation](https://mehrhof.valksor.com/docs)
- [Quickstart](https://mehrhof.valksor.com/docs/#/quickstart) - Install and first task in 5 minutes
- [Guides](https://mehrhof.valksor.com/docs/#/guides/first-task) - Step-by-step tutorials
- [Providers](https://mehrhof.valksor.com/docs/#/providers/index) - Task source integrations
- [AI Agents](https://mehrhof.valksor.com/docs/#/agents/index) - Agent configuration and aliases
- [CLI Reference](https://mehrhof.valksor.com/docs/#/cli/index) - All commands and flags
- [Configuration](https://mehrhof.valksor.com/docs/#/configuration/index) - Customize behavior
- [Concepts](https://mehrhof.valksor.com/docs/#/concepts/workflow) - Workflow, storage, architecture
- [Troubleshooting](https://mehrhof.valksor.com/docs/#/troubleshooting/index) - Common issues

## Development

```bash
make build        # Build binary to ./build/mehr
make install      # Install to $GOPATH/bin
make test         # Run tests with coverage
make coverage     # Generate coverage report
make quality      # Run golangci-lint + govulncheck
make fmt          # Format code (gofmt, goimports, gofumpt)
make tidy         # Tidy dependencies
make hooks        # Enable versioned git hooks
make lefthook     # Install pre-commit hooks (auto-format + lint)
```

**CI/CD**: PRs trigger lint/test/build via GitHub Actions. Releases use [GoReleaser](https://goreleaser.com/) with Cosign signing and SBOM generation.

### PR Review in CI/CD

Mehrhof can review pull/merge requests automatically in CI/CD pipelines:

```bash
# Set required environment variable for state verification
export MEHRHOF_STATE_SECRET="$(openssl rand -base64 32)"

# Run PR review
mehr review pr --pr-number 123 --token "$GITHUB_TOKEN"
```

**Important**: The `MEHRHOF_STATE_SECRET` environment variable is **REQUIRED** for PR review state verification. This secret:
- Enables secure HMAC signing of embedded review state
- Prevents tampering with review history
- Must be at least 32 characters long
- Should be generated with a cryptographically secure random generator

**CI/CD Example** (GitHub Actions):
```yaml
- name: Review PR
  env:
    MEHRHOF_STATE_SECRET: ${{ secrets.STATE_SECRET }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    mehr review pr --pr-number ${{ github.event.pull_request.number }}
```

See [https://mehrhof.valksor.com/docs/#/guides/pr-review-state](https://mehrhof.valksor.com/docs/#/guides/pr-review-state) for detailed documentation on PR review state management.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

By contributing to Mehrhof, you agree that your contributions will be licensed under the [BSD 3-Clause License](LICENSE).

