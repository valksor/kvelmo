This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# IT IS YEAR 2026 !!! Please use 2026 in web searches!!!
## No time estimates. Never say "this will take 1 day" or "a few weeks" - these are always wrong. If you must indicate complexity, use Fibonacci numbers (1, 2, 3, 5, 8, 13) for relative effort.
## PROJECT USES BUN NOT NODE OR NPM! PLEASE USE BUN OR BUNX WHEN CALLING SCRIPTS!

## Project Overview

kvelmo is a socket-first task lifecycle orchestration system for AI-assisted development. It manages the complete lifecycle of development tasks from loading requirements through implementation to PR submission, using Unix domain sockets for inter-process communication between CLI, web UI, and AI agents.

**Note:** The `prototype/` directory contains a fully working prototype implementation of kvelmo. This code is **read-only reference material** - not in active development. Use it to understand patterns and approaches when needed.

## Build & Development Commands

```bash
# Build (includes web frontend)
make build              # Full build: web + Go binary → ./build/kvelmo
make build-go           # Go-only build (faster, skip web)

# Run
make run                # Build and run (sockets + web UI)
make run-dev            # Run without rebuilding (uses existing binary)

# Tests
make test               # Run all tests
make test-v             # Verbose test output
make test-cover         # Coverage report → coverage.html
make test-race          # Tests with race detector
go test ./pkg/socket/...  # Run tests for specific package

# Quality
make lint               # golangci-lint with --fix
make quality            # fmt + vet + lint
make ci                 # quality + test + build

# Frontend
make web-dev            # Vite dev server with hot reload (port 5173)
make web-build          # Production build → web/dist/
```

## Frontend

**Use bun, not npm/node** for all frontend operations in `web/`.

## Architecture

### Socket Layer (`pkg/socket/`)
- **GlobalSocket** (`global.go`): Manages project registry and worker pool. Single instance at `~/.valksor/kvelmo/global.sock`
- **WorktreeSocket** (`worktree.go`): Per-project state machine and git operations. Created at `<project>/.kvelmo/worktree.sock`
- **Protocol**: JSON-RPC 2.0 over Unix domain sockets (`protocol.go`)

### State Machine (`pkg/conductor/`)
Task lifecycle with 11 states defined in `state.go`:
- Core flow: `none` → `loaded` → `planning` → `planned` → `implementing` → `implemented` → `reviewing` → `submitted`
- Auxiliary states: `optimizing`, `waiting`, `paused`, `failed`
- Transition guards ensure prerequisites (e.g., must have specifications before implementing)
- Undo/redo support via git checkpoints

### Agent System (`pkg/agent/`)
- **Interface**: `Agent` interface in `agent.go` with WebSocket (primary) and CLI (fallback) modes
- **Implementations**: `claude/`, `codex/`, `custom/` subdirectories
- **Events**: Streaming events (tokens, tool calls, permissions, completion)
- **Permissions**: `DefaultPermissionHandler` auto-approves read-only tools

### Worker Pool (`pkg/worker/`)
- Manages concurrent AI agent executions
- Job queue with worker assignment
- Real-time event streaming via WebSocket

### Providers (`pkg/provider/`)
Task sources: `file.go` (local markdown), `github.go`, `gitlab.go`, `wrike.go`
Pattern: `provider:reference` (e.g., `github:owner/repo#123`)

### Web Frontend (`web/`)
- React 19 + TypeScript + Vite 7
- State: Zustand stores in `src/stores/` (global, project, chat, screenshot, theme, layout)
- UI: Tailwind CSS 4 + DaisyUI 5
- Views: `GlobalView` (project picker) ↔ `ProjectView` (active project dashboard)
- WebSocket connection to backend for real-time updates

## Key Patterns

### Error Handling
Go: Return errors, wrap with context (`fmt.Errorf("action: %w", err)`)

### Configuration
- Global config: `~/.valksor/kvelmo/kvelmo.yaml` (managed by `pkg/settings/`)
- CLI: `kvelmo config show|init|set`
- Environment: `KVELMO_SOCKET_DIR`, `GITHUB_TOKEN`, etc.

### Testing
- Table-driven tests using `testing.T`
- Benchmark tests in `pkg/socket/bench_test.go`
- Frontend: Add `?demo` URL param for UI testing without backend
- **Never accept test failures.** If a test fails, fix it. No exceptions. Never rationalize failures as "pre-existing" or "not my problem."

## CLI Command Structure

Commands in `cmd/kvelmo/commands/`:
- `serve.go`: Main entry point, starts global socket + web server (default port 6337)
- Workflow: `start`, `plan`, `implement`, `optimize`, `review`, `submit`
- Navigation: `undo`, `redo`, `status`
- Management: `config`, `workers`, `projects`, `completion`

## Code Style

- **Imports**: stdlib → third-party → local (alphabetical within groups)
- **Naming**: PascalCase exported, camelCase unexported
- **Errors**: `fmt.Errorf("prefix: %w", err)`
- **Logging**: `log/slog`

### Import Discipline

Import packages directly. No type aliases, no wrapper functions, no re-exports.

```go
// ✅ GOOD - Direct import
import "github.com/gorilla/websocket"
conn, _ := websocket.Upgrade(...)

// ❌ BAD - Type alias or wrapper
type Conn = websocket.Conn  // Don't do this
func NewConn(...) *Conn { return websocket.Upgrade(...) }  // Don't do this
```

### Modern Go (1.23+)

- Use `slices.Contains()`, `maps.Clone()` instead of manual loops
- Always pass `context.Context` for cancelable operations

## Linting Guidelines

`//nolint` is a last resort. Always specify linter name and include justification.

**Acceptable**:
- `//nolint:unparam // Required by interface`
- `//nolint:errcheck // String builder WriteString won't fail`

**Never acceptable**:
- `//nolint:errcheck` without justification
- `//nolint:gosec` (fix the security issue)
- `//nolint:all`

## File Organization

Keep Go files under 500 lines. Split by feature or responsibility:

```go
// Split handlers.go (800 lines) into:
handlers_plan.go      // Planning handlers
handlers_implement.go // Implementation handlers
handlers_review.go    // Review handlers
```

### Commit Style

Plain imperative sentences. No conventional commits prefix (`feat:`, `fix:`, `chore:`, etc.).

```text
✅ Add project selector to GlobalView
✅ Fix socket reconnect race condition
✅ Improve TaskWidget keyboard navigation

❌ feat(web): add project selector
❌ fix(socket): reconnect race condition
```
