# CLAUDE.md

# IT IS YEAR 2026 !!! Please use 2026 in web searches!!!  

Guidance for Claude Code when working with go-mehrhof.

## Project Overview

Mehrhof is a **Go CLI + Web UI** for AI-powered task automation. It orchestrates AI agents to perform planning, implementation, and code review workflows with checkpointing, parallel tasks, and multi-provider integrations.

**Key constraint**: ALL features require BOTH CLI and Web UI implementations unless explicitly CLI-only.

**Note**: IDE plugins (JetBrains, VS Code) consume existing REST API + SSE endpoints. When adding workflow commands, ensure the `/interactive` API supports the new operation - plugins automatically inherit the functionality.

---

## Critical Rules

### 1. Multi-Interface Parity

Every feature needs CLI (`cmd/mehr/commands/`) + Web UI (`internal/server/`). Shared logic goes in `internal/conductor/`. Both interfaces call the same conductor methods.

**Interfaces**: CLI, Interactive CLI (`mehr interactive`), Web UI, Interactive Web (`/interactive`), JetBrains Plugin, VS Code Extension

See [docs/reference/feature-parity.md](docs/reference/feature-parity.md) for implementation checklist and status tables.

### 2. go-toolkit: Import Directly

Use `github.com/valksor/go-toolkit` packages directly. **NO type aliases, NO wrapper functions, NO re-exports.**

```go
// ✅ GOOD - Direct import
import "github.com/valksor/go-toolkit/eventbus"
bus := eventbus.NewBus()

// ❌ BAD - Type alias or wrapper
type Bus = eventbus.Bus  // Don't do this
```

**When to add to go-toolkit**: Generic utilities with no mehrhof dependencies.
**When to add to go-mehrhof**: Domain-specific business logic.

CI enforces via `make check-alias`.

### 3. Tests & Docs Required

Every feature MUST include:

| Requirement | Location | Target |
|-------------|----------|--------|
| Unit tests | `*_test.go` next to source | 80%+ coverage |
| Integration tests | `internal/helper_test/` | Critical paths |
| CLI docs | `docs/cli/feature.md` | Usage + examples |
| Web UI docs | `docs/web-ui/feature.md` | UI instructions |

Write tests FIRST (TDD). Use table-driven tests. Run `make test` before committing code.

### 4. Docs by Interface Type

Documentation is organized by interface:

| Directory | Content |
|-----------|---------|
| `docs/cli/` | CLI commands only |
| `docs/web-ui/` | Web UI features only |
| `docs/ide/` | IDE integrations (JetBrains, etc.) |
| `docs/concepts/` | Interface-agnostic architecture |
| `docs/reference/` | Technical reference, parity tables |

**Rule**: One interface per document. Cross-reference between CLI and Web UI docs.

### 5. Zero Broken Code

**ALL tests and quality checks MUST pass before committing code.**

```bash
# Before starting work - verify baseline
make quality && make test && make race

# Before committing code changes
make quality && make test && make race
```

If tests fail, fix them first. No exceptions for "not my code."

**Skip for docs-only changes** - no build/test needed for `.md` files.

### 6. Use Make Commands

Always use `make` commands, not direct `go` commands:

| Operation | Command              |
|-----------|----------------------|
| Build     | `make build`         |
| Test      | `make test`          |
| Race      | `make race`          |
| Quality   | `make quality`       |
| Format    | `make fmt`           |
| Coverage  | `make coverage-html` |
| Install   | `make install`       |

`make quality` runs: golangci-lint, gofmt, goimports, gofumpt, govulncheck, check-alias.

### 7. No nolint Abuse

**`//nolint` is a LAST RESORT.** Never disable linters in `.golangci.yml`.

**Acceptable**:
- `//nolint:unparam // Required by interface`
- `//nolint:nilnil // No task found is not an error`
- `//nolint:errcheck // String builder WriteString won't fail`

**Never acceptable**:
- `//nolint:errcheck` without justification
- `//nolint:gosec` (fix the security issue)
- `//nolint:all` (never suppress all linters)

Always: specify linter name, include justification, place on specific line.

### 8. File Size < 500 Lines

Keep all Go files under 500 lines. Split by feature or responsibility:

```go
// Split handlers.go (1000 lines) into:
handlers_plan.go      // Planning handlers
handlers_implement.go // Implementation handlers
handlers_review.go    // Review handlers
```

**Exceptions**: Generated code, single-responsibility modules, large templates.

---

## Commands

### Build & Development

```bash
make build | install | test | coverage | quality | fmt | tidy | hooks | race
```

All targets: `all`, `build`, `test`, `coverage`, `coverage-html`, `quality`, `fmt`, `install`, `clean`, `run`, `tidy`, `deps`, `version`, `hooks`, `lefthook`, `check-alias`, `help`, `race`.

### Workers Site

`workers-site/index.min.js` is auto-generated. Edit `index.js`, then run:
```bash
bun run workers:minify
```

### Workflow Commands

```bash
mehr start <ref> | plan | implement | review | finish | continue | abandon
mehr status | list | note <msg> | question <msg> | cost
mehr undo | redo | reset | browser | mcp | scan | serve | interactive
mehr project plan|submit|start|sync | stack | config validate
mehr agents | providers | templates | update | generate-secret
```

**Interactive mode** (`mehr interactive` or Web `/interactive`): workflow commands + chat.

**Recovery tip:** If an agent hangs and you kill it, use `mehr reset` to reset state to idle without losing work. Or use `--force` on step commands (e.g., `mehr plan --force`).

---

## Architecture

### Entry Points

| Path | Description |
|------|-------------|
| CLI | `cmd/mehr/main.go` → `commands.Execute()` → Cobra handlers |
| Interactive CLI | → `interactive` → REPL → command dispatcher |
| Web UI | → `serve` → `internal/server/server.go` → handlers → templates |
| Interactive Web | → `/interactive` handler → SSE + HTMX |
| JetBrains Plugin | → `ide/jetbrains/` → Kotlin plugin → REST API + SSE |
| VS Code Extension | → `ide/vscode/` → TypeScript extension → REST API + SSE |

### Core Packages

| Package | Responsibility |
|---------|----------------|
| `internal/conductor/` | Main orchestrator (Facade) - workflow, storage, VCS, agents, browser, MCP |
| `internal/workflow/` | State machine - states, events, guards, effects, transitions |
| `internal/agent/` | AI agent abstraction with streaming; Claude implementation |
| `internal/agent/claude/` | Claude CLI wrapper agent |
| `internal/coordination/` | Agent resolution - 7-level priority system |
| `internal/provider/` | Task sources: file, github, gitlab, jira, linear, notion, etc. |
| `internal/storage/` | Split storage: `.mehrhof/` (project) + `~/.valksor/mehrhof/` (workspaces) |
| `internal/vcs/` | Git: branches, worktrees, checkpoints (undo/redo) |
| `internal/events/` | Pub/sub event bus |
| `internal/browser/` | Chrome automation (CDP) |
| `internal/mcp/` | Model Context Protocol server |
| `internal/memory/` | Semantic memory with vector embeddings |
| `internal/server/` | Web UI: REST API, SSE, authentication |
| `ide/jetbrains/` | JetBrains IDE plugin - Kotlin, native integration via REST API + SSE |
| `ide/vscode/` | VS Code extension - TypeScript, webview-based UI via REST API + SSE |
| `internal/links/` | Bidirectional linking (`[[reference]]` syntax) |
| `internal/plugin/` | External agent/provider extensions (JSON-RPC) |
| `internal/security/` | SAST (gosec), secrets (gitleaks), vulns (govulncheck) |
| `internal/quality/` | Linters, formatters |

### Key Patterns

**State Machine** (`internal/workflow/`):
- States: `idle` → `planning` → `implementing` → `reviewing` → `done`/`failed`
- Additional: `waiting`, `checkpointing`, `reverting`, `restoring`
- Guards control transitions; effects execute side-effects

**Registry Pattern**: Providers and agents register themselves, looked up by name at runtime.

**Event-Driven**: Components communicate via `events.Bus`.

**Links System**: Logseq-style `[[spec:1]]`, `[[decision:cache-strategy]]` linking. Query with `FindLinks()`, `FindBacklinks()`, `FindPath()`.

**Plugin System**: JSON-RPC over stdio, configured via `plugin.yaml`.

### Agent Configuration

Priority resolution (highest to lowest):
1. CLI step flag: `--agent-plan`, `--agent-implement`, `--agent-review`
2. CLI global: `--agent`
3. Task frontmatter step: `agent_steps.planning.agent`
4. Task frontmatter default: `agent:`
5. Workspace config step: `agent.steps.planning.name`
6. Workspace config default: `agent.default`
7. Auto-detect

```yaml
# .mehrhof/config.yaml
agent:
  default: claude
  steps:
    planning: { name: claude }
    implementing: { name: claude-sonnet }
    reviewing: { name: claude }

agents:
  opus:
    extends: claude
    args: ["--model", "claude-opus-4"]
```

### Workflow States

| State | Description |
|-------|-------------|
| `idle` | No active task |
| `planning` | AI generating specifications |
| `implementing` | AI executing specifications |
| `reviewing` | Code review in progress |
| `waiting` | Awaiting user response |
| `checkpointing` | Creating git checkpoint |
| `reverting` | Undo to checkpoint |
| `restoring` | Redo to checkpoint |
| `done` | Completed |
| `failed` | Failed |

---

## Code Style

- **Imports**: stdlib → third-party → local (alphabetical within groups)
- **Naming**: PascalCase exported, camelCase unexported
- **Errors**: `fmt.Errorf("prefix: %w", err)`; `errors.Join(errs...)`
- **Logging**: `log/slog`
- **Formatting**: `make fmt` (gofmt, goimports, gofumpt)
- **Quality**: `make quality`

### Modern Go (1.25+)

- Use `slices.Contains()`, `maps.Clone()` instead of manual loops
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done() }()`
- Always pass `context.Context` for cancelable operations

---

## Testing

- Run: `make test`
- Coverage: `make coverage-html` (output: `.coverage/coverage.html`)
- Style: Table-driven with `tests := []struct{...}{...}`
- Utilities: `internal/helper_test/` (mocks, fixtures)
- Target: 80%+ coverage
- Race detector: `make race`

---

## See Also

- [README.md](README.md) - Installation, quick start
- [docs/reference/feature-parity.md](docs/reference/feature-parity.md) - Interface parity tables
- [Documentation](https://valksor.com/docs/mehrhof/nightly) - Full guides
