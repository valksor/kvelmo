# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mehrhof is a Go CLI tool for AI-powered task automation. It orchestrates AI agents (primarily Claude) to perform planning, implementation, and code review workflows with checkpointing, parallel task support, and multi-provider integrations.

## Commands

### Build & Development

```bash
make build | install | test | coverage | quality | fmt | tidy | hooks | lefthook
```

Available make targets: `all`, `build`, `test`, `coverage`, `coverage-html`, `quality`, `fmt`, `install`, `clean`, `run`, `run-args`, `tidy`, `deps`, `version`, `hooks`, `lefthook`, `check-alias`, `help`

See [README.md](README.md) for full documentation.

### Workflow

```bash
mehr start <ref> | plan | implement | review | finish | continue | auto <ref>
```

Additional commands: `sync <task-id>`, `simplify`, `abandon`, `undo`, `redo`, `guide`, `status`, `list`, `note <msg>`, `browser`, `mcp`, `scan`, `serve`, `project plan|submit`, `config validate`, `agents`, `providers`, `templates`, `update check|install`, `generate-secret`, `cost`, `memory`

## Architecture

### Entry Point Flow

`cmd/mehr/main.go` → `commands.Execute()` → Cobra command handlers

### Core Packages

| Package | Responsibility |
|---------|----------------|
| `internal/conductor/` | Main orchestrator (Facade) - combines workflow, storage, VCS, agents, browser, MCP |
| `internal/workflow/` | State machine engine - states, events, guards, effects, transitions |
| `internal/agent/` | AI agent abstraction with streaming; claude implementation; orchestration modes (pipeline, consensus) |
| `internal/agent/claude/` | Claude CLI wrapper agent implementation |
| `internal/agent/browser/` | Browser automation tool adapter for agents (Chrome CDP integration) |
| `internal/coordination/` | Agent resolution - 7-level priority system for selecting agents per workflow step |
| `internal/provider/` | Task source abstraction; implementations: file, directory, github, gitlab, jira, linear, asana, notion, trello, wrike, youtrack, bitbucket, clickup, azuredevops, empty |
| `internal/storage/` | Split storage: `.mehrhof/` in project (config.yaml, .env); `~/.valksor/mehrhof/workspaces/<project-id>/` (work/, sessions/, .active_task) |
| `internal/vcs/` | Git operations: branches, worktrees, checkpoints for undo/redo |
| `internal/events/` | Pub/sub event bus for component decoupling |
| `internal/browser/` | Chrome automation controller (CDP) for testing, scraping, auth flows |
| `internal/mcp/` | Model Context Protocol server for AI agent integration |
| `internal/memory/` | Semantic memory with vector embeddings for past task context |
| `internal/ml/` | Machine learning predictions for task complexity and resources |
| `internal/server/` | Web UI server with REST API, SSE, authentication |
| `internal/security/` | Security scanning (SAST with gosec, secrets with gitleaks, vulns with govulncheck) |
| `internal/quality/` | Code quality tools (linters, formatters) |
| `internal/naming/` | Branch/commit name template parsing with slug generation |
| `internal/plugin/` | Plugin system for external agent and provider extensions |
| `internal/registration/` | Standard agent and provider registration functions |
| `internal/update/` | Self-update mechanism from GitHub releases |
| `internal/template/` | Template system for prompts and specifications |

### Key Patterns

**State Machine**: The workflow package implements an explicit FSM:
- States: `idle` → `planning` → `implementing` → `reviewing` → `done`/`failed`
- Additional states: `waiting`, `checkpointing`, `reverting`, `restoring`
- Guard conditions control valid transitions
- Effects execute side-effects (git commits, file changes)

**Registry Pattern**: Providers and agents register themselves and are looked up by name/scheme at runtime.

**Event-Driven**: Components communicate via `events.Bus`, enabling loose coupling.

### Web UI Architecture

The web UI uses Go's `html/template` package with:
- **HTMX** for real-time interactivity and SSE (Server-Sent Events)
- **Tailwind CSS** via CDN for styling with custom brand colors
- **Dark mode** via `class`-based toggle

**Template Structure** (`internal/server/templates/`):
- `base.html` - Base layout with HTMX + Tailwind, dark mode support
- `login.html` - Authentication page
- `dashboard.html` - Main task dashboard with SSE streaming
- `project.html` - Project-specific task management view
- `history.html` - Session history and replay
- `browser.html` - Browser automation control panel
- `settings.html` - Workspace configuration management
- `partials/` - Reusable template components
  - `actions.html` - Action buttons and controls
  - `costs.html` - Token cost display
  - `question.html` - Agent question prompts
  - `specs.html` - Specification displays
  - `task_card.html` - Task summary cards

### Provider Capability System

Providers declare supported operations via capability interfaces. This enables runtime feature detection.

**Key capabilities**: `CapRead`, `CapList`, `CapFetchComments`, `CapComment`, `CapUpdateStatus`, `CapManageLabels`, `CapDownloadAttachment`, `CapSnapshot`, `CapCreatePR`, `CapLinkBranch`, `CapCreateWorkUnit`, `CapFetchSubtasks`, `CapFetchPR`, `CapPRComment`, `CapFetchPRComments`, `CapUpdatePRComment`, `CapCreateDependency`, `CapFetchDependencies`

Key components:
- `provider.Capability` - String type for capability constants (defined in `internal/provider/types.go`)
- `provider.CapabilitySet` - Map of capabilities to booleans
- `provider.InferCapabilities()` - Auto-detects capabilities via interface assertions

### Agent Configuration

**Priority resolution** (7 levels, highest to lowest):
1. CLI step-specific flag (`--agent-plan`, `--agent-implement`, `--agent-review`)
2. CLI global flag (`--agent`)
3. Task frontmatter step-specific (`agent_steps.planning.agent`)
4. Task frontmatter default (`agent:`)
5. Workspace config step-specific (`agent.steps.planning.name`)
6. Workspace config default (`agent.default`)
7. Auto-detect (first available agent)

Implemented in: `internal/coordination/agent.go`

**Per-step agents**: Different agents can be configured for planning vs implementing:

```yaml
# Workspace config (.mehrhof/config.yaml)
agent:
  default: claude
  steps:
    planning: { name: claude }
    implementing: { name: claude-sonnet }
    reviewing: { name: claude }
```

**Aliases**: Wrap agents with custom env/args in workspace config:

```yaml
agents:
  opus:
    extends: claude
    args: ["--model", "claude-opus-4"]
    env:
      ANTHROPIC_API_KEY: "${CUSTOM_KEY}"
```

**Step-specific args**: Agents implement `StepArgsProvider` to provide workflow-step-specific CLI args (e.g., Claude uses `--permission-mode plan` for planning, `--permission-mode acceptEdits` for implementing). See `internal/agent/claude/claude.go:348`.

### Workflow States

| State | Description |
|-------|-------------|
| `idle` | No active task |
| `planning` | AI generating specifications |
| `implementing` | AI executing specifications |
| `reviewing` | Code review in progress |
| `waiting` | Awaiting user response to agent question |
| `checkpointing` | Creating git checkpoint |
| `reverting` | Undo to previous checkpoint |
| `restoring` | Redo to checkpoint |
| `done` | Task completed successfully |
| `failed` | Task failed |

## Code Style

- **Imports**: standard library → third-party → local (each group sorted alphabetically)
- **Naming**: PascalCase for exported, camelCase for unexported
- **Errors**: `fmt.Errorf("prefix: %w", err)` for wrapping; `errors.Join(errs...)` for multiple
- **Logging**: Use `log/slog`
- **Formatting**: Run `make fmt` (uses gofmt, goimports, gofumpt)
- **Linting**: Configured in `.golangci.yml` - CI runs on every PR

### Modern Go Practices (Go 1.25+)

- Use `slices.Contains()`, `slices.Concat()`, `maps.Clone()` instead of manual loops
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done(); ... }()`
- Always pass `context.Context` for cancelable operations

## Testing

- Tests use the standard `testing` package
- Table-driven tests preferred: `tests := []struct{...}{...}`
- Test utilities in `internal/helper_test/` (mocks, fixtures, conductor helpers)
- Target 80%+ coverage

## See Also

- [README.md](README.md) - User-facing documentation, installation, quick start
- [Documentation](https://valksor.com/docs/mehrhof) - Full guides and API reference
