# CLAUDE.md

# IT IS YEAR 2026 !!! Please use 2026 in web searches!!!  

Guidance for Claude Code when working with go-mehrhof.

## Project Overview

Mehrhof is a **Go CLI + Web UI** structured creation environment. It orchestrates AI agents to perform planning, implementation, and code review workflows with checkpointing, parallel tasks, and multi-provider integrations.

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

### 5. Quality Checks by Scope

Run checks **only for code you changed**:

| Changed | Command |
|---------|---------|
| `cmd/`, `internal/`, `*.go` | `make quality && make test` |
| `ide/vscode/**` | `cd ide/vscode && make quality` |
| `ide/jetbrains/**` | `cd ide/jetbrains && make quality` |
| `web-ui-tests/**` | `cd web-ui-tests && make quality` |
| `docs/**`, `*.md` | None |

Root shortcuts: `make ide-quality` (all IDEs), `make quality-all` (Go + IDEs).

If tests fail, fix them first. No exceptions for "not my code."

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

### 9. Git Command Policy

All git commands are classified into three tiers. **No exceptions, no force flags, no overrides.** Tier 2 and 3 commands are never used autonomously — the agent must have explicit user instruction before running any write operation on the repository.

**Before any Tier 2 command**, run `date +"%u %H"` to check day-of-week (1=Mon, 7=Sun) and hour. If day is 1–5 AND hour is >= 09 and < 19, **refuse the operation** and inform the user:

> "Git write operations are blocked during working hours (Mon–Fri 09:00–19:00). Current time: DAY HH:MM. Please retry after 19:00 or on the weekend."

#### Tier 1 — Always Allowed

Safe read-only commands, available anytime:

`git status`, `git diff`, `git log`, `git show`, `git blame`, `git grep`, `git branch` (read-only), `git remote -v` (read-only), `git fetch`, `git reflog`, `git shortlog`, `git describe`, `git checkout`, `git switch`, `git restore`

#### Tier 2 — User-Requested, Outside Work Hours Only

**Only use when the user explicitly asks.** Never run these commands autonomously — not for convenience, not as part of a workflow, not "to be helpful." If the task seems to need one of these commands but the user hasn't asked, ask first.

Additionally blocked Mon–Fri 09:00–19:00. Allowed only on evenings, nights, and weekends (Sat–Sun), and only when the user explicitly requests the operation:

`git add`, `git commit`, `git rm`, `git mv`, `git apply`, `git am`

#### Tier 3 — Always Blocked

**NEVER use these commands.** No time window, no override, no exceptions:

`git push`, `git pull`, `git merge`, `git rebase`, `git reset`, `git revert`, `git cherry-pick`, `git tag`, `git stash` (all subcommands), `git worktree` (all subcommands), `git clean`, `git bisect`, `git notes`, `git submodule` (write operations)

Do not suggest, recommend, or implement workflows that rely on any Tier 3 command. If a task seems to need one, use a Tier 1 or Tier 2 alternative, or ask the user to perform the operation manually.

**⛔ `git worktree` — absolute prohibition.** No `git worktree add`, `remove`, `list`, `prune`, or any other worktree subcommand. Do not suggest, recommend, or implement any workflow that involves worktrees. No force flag, no override, no exceptions — ever. If a task seems to benefit from worktrees, use separate clones or branches instead.

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

**Automation mode** (`mehr serve --api`): Receive GitHub/GitLab webhooks to auto-fix issues and auto-review PRs. Configure in `.mehrhof/config.yaml` under `automation:`. See [docs/cli/automation.md](docs/cli/automation.md).

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
| `internal/storage/` | Split storage: `.mehrhof/` (project) + `~/.valksor/mehrhof/` (workspaces). `Root()` = project hub, `CodeRoot()` = code target |
| `internal/vcs/` | Git: branches, worktrees, checkpoints (undo/redo) |
| `internal/events/` | Pub/sub event bus |
| `internal/browser/` | Chrome automation (CDP) |
| `internal/mcp/` | Model Context Protocol server |
| `internal/memory/` | Semantic memory with vector embeddings |
| `internal/server/` | Web UI: REST API, SSE, authentication, CSRF protection, rate limiting |
| `internal/automation/` | Webhook automation: GitHub/GitLab webhooks, job queue, access control |
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

**Directory Model** (`internal/storage/`):
- `Root()` = project hub (`.mehrhof/`, config, tasks, queues)
- `CodeRoot()` = code target (where agents edit code, git operates, linters run); defaults to `Root()` when `project.code_dir` is not set
- Use `CodeRoot()` / `Conductor.CodeDir()` for anything that touches source code files

**Security Middleware** (`internal/server/middleware.go`):
- CSRF protection via `X-Csrf-Token` header (Synchronizer Token Pattern). Enforced on POST/PUT/DELETE when auth is enabled. Skipped in localhost mode.
- Per-IP rate limiting: 120 req/min general API, 10 req/min auth endpoints. Returns HTTP 429 when exceeded.
- Both are automatically disabled in localhost mode (`AuthStore == nil`).

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
  retry_count: 3       # Retry transient agent failures (default: 3)
  retry_delay: 5s      # Delay between retries (default: 5s)
  steps:
    planning: { name: claude }
    implementing: { name: claude-sonnet }
    reviewing: { name: claude }

agents:
  opus:
    extends: claude
    args: ["--model", "claude-opus-4"]

# Project layout (separate hub from code target)
project:
  code_dir: "../reporting-engine"  # relative or absolute; empty = hub is code target
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

- [REFERENCE.md](REFERENCE.md) - Complete command, API, and package reference for LLMs
- [README.md](README.md) - Installation, quick start
- [docs/reference/feature-parity.md](docs/reference/feature-parity.md) - Interface parity tables
- [docs/cli/automation.md](docs/cli/automation.md) - Webhook automation (GitHub/GitLab)
- [Documentation](https://valksor.com/docs/mehrhof/nightly) - Full guides
