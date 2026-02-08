# Feature Parity Reference

This document tracks implementation status across Mehrhof's interfaces. Use this as a checklist when adding new features.

## Interface Overview

Mehrhof has seven user interfaces:

| Interface              | Entry Point              | Purpose                          |
|------------------------|--------------------------|----------------------------------|
| **CLI**                | `cmd/mehr/commands/`     | Full command-line interface      |
| **Interactive CLI**    | `mehr interactive`       | REPL mode for workflow sessions  |
| **Web UI**             | `internal/server/`       | Browser interface with dashboard |
| **Interactive Web**    | `/interactive`           | Browser REPL with SSE streaming  |
| **JetBrains Plugin**   | `ide/jetbrains/`         | IntelliJ/GoLand/WebStorm native  |
| **VS Code Extension**  | `ide/vscode/`            | VS Code sidebar integration      |
| **MCP Server**         | `internal/mcp/`          | AI agent tool interface          |

---

## Parity Rules

**CLI is the reference implementation** — all features start here.

**CLI-only commands** (intentionally NOT in other interfaces):
- `init` - Workspace initialization (one-time setup)
- `serve` - Starts web server (self-referential)
- `generate-secret` - One-time utility
- `update` - CLI self-update
- `hooks` / `lefthook` - Developer tooling

**CI/CD-only commands**:
- `review pr` - Only in automation context

**Everything else MUST have 1:1 parity across all interfaces.**

### Architecture: Unified Command Router

All interfaces route through `internal/conductor/commands/`:

```
CLI Interactive ─┐
Web Chat ────────┼──→ commands.Execute() ──→ Handler ──→ Conductor
IDE Plugins ─────┤     (unified router)
MCP Server ──────┘
```

This ensures consistent behavior and automatic parity for all commands.

---

## Current Parity Status

All interfaces now route through the unified command router (`internal/conductor/commands/`).

| Interface | Required     | Implemented | Parity      |
|-----------|--------------|-------------|-------------|
| CLI       | 100%         | 100%        | ✅ Reference |
| Web UI    | ~95 commands | ~95         | 100%        |
| Web Chat  | ~95 commands | ~95         | 100%        |
| VS Code   | ~95 commands | ~95         | 100%        |
| JetBrains | ~95 commands | ~95         | 100%        |
| MCP       | ~93 commands | ~95         | 100%+       |

---

## Implementation Checklist

When adding a new feature, complete ALL applicable items:

- [ ] **CLI Command**: Add in `cmd/mehr/commands/*.go` using Cobra
- [ ] **Interactive CLI**: Add to `interactive` allowed commands if workflow-relevant
- [ ] **Web UI Handler**: Add in `internal/server/handlers*.go` or `internal/server/api/`
- [ ] **Interactive Web**: Add to `handlers_interactive.go` command handler
- [ ] **JetBrains Plugin**: Add action in `ide/jetbrains/`
- [ ] **VS Code Extension**: Add command in `ide/vscode/`
- [ ] **MCP Server**: Verify tool is exposed (auto-mapped from CLI)
- [ ] **Router Registration**: Update `internal/server/router.go`
- [ ] **SSE Streaming**: Add for long-running operations
- [ ] **Tests**: Comprehensive tests (see Testing section in CLAUDE.md)
- [ ] **Documentation**: Update `docs/cli/` and/or `docs/web-ui/`

### Implementation Pattern

Both CLI and Web UI delegate to **shared core logic** in `internal/conductor/`:

```go
// CLI calls conductor
func runPlan(cmd *cobra.Command, args []string) error {
    return cond.Plan(ctx)
}

// Web UI calls the same conductor
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
    s.config.Conductor.Plan(r.Context())
}
```

### SSE Streaming Pattern

For long-running operations (planning, implementing, reviewing), use Server-Sent Events:

```go
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        s.writeError(w, http.StatusBadRequest, "streaming not supported")
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    fmt.Fprintf(w, "event: status\ndata: {\"message\": \"Starting...\"}\n\n")
    flusher.Flush()
    // ... execute operation ...
    fmt.Fprintf(w, "event: complete\ndata: {\"success\": true}\n\n")
    flusher.Flush()
}
```

---

## CLI vs. Web UI Parity

| CLI Command            | Web UI   | Notes                               |
|------------------------|----------|-------------------------------------|
| `start <ref>`          | ✅        | Dashboard + project pages           |
| `plan`                 | ✅        | SSE streaming                       |
| `implement`            | ✅        | SSE streaming                       |
| `implement review <n>` | ✅        | SSE streaming                       |
| `review`               | ✅        | SSE streaming                       |
| `review view <n>`      | ✅        | API endpoint                        |
| `review pr`            | CI/CD    | Automation context only             |
| `finish`               | ✅        | PR creation/merge                   |
| `continue`             | ✅        | Resume from waiting                 |
| `abandon`              | ✅        | Discard task                        |
| `auto`                 | ✅        | `/auto` page                        |
| `quick <desc>`         | ✅        | `/quick` page                       |
| `guide`                | ✅        | `/api/v1/guide` endpoint            |
| `status`               | ✅        | Dashboard display                   |
| `list`                 | ✅        | Recent tasks sidebar                |
| `note <msg>`           | ✅        | Quick note form                     |
| `note list/view`       | ✅        | API endpoints                       |
| `question <msg>`       | ✅        | Quick question + SSE                |
| `specification`        | ✅        | `/api/v1/tasks/{id}/specs`          |
| `label`                | ✅        | LabelsCard in TaskDetail            |
| `delete --task`        | ✅        | Quick task delete                   |
| `export --task`        | ✅        | Quick task export                   |
| `optimize --task`      | ✅        | Quick task optimization             |
| `submit`               | ✅        | Quick task submit                   |
| `sync`                 | ✅        | API + SSE streaming                 |
| `cost`                 | ✅        | Detailed breakdown by step          |
| `budget`               | ✅        | API + monthly status/reset          |
| `undo/redo`            | ✅        | Checkpoint navigation               |
| `reset`                | ✅        | `/api/v1/workflow/reset`            |
| `find`                 | ✅        | `/find` page                        |
| `memory`               | ✅        | `/memory` page                      |
| `links`                | ✅        | `/links` page                       |
| `library`              | ✅        | `/library` page with pull/list/show |
| `browser`              | ✅        | `/browser` page                     |
| `browser cookies`      | ✅        | Cookies tab in DevTools section     |
| `project`              | ✅        | Project planning pages              |
| `stack`                | ✅        | `/stack` page                       |
| `scan`                 | ✅        | `/scan` page with scanner selection |
| `simplify`             | ✅        | `/simplify` page                    |
| `commit`               | ✅        | `/commit` page with analyze/preview |
| `config validate`      | ✅        | Settings validation                 |
| `config explain`       | ✅        | Settings explanation                |
| `agents`               | ✅        | Settings page                       |
| `providers`            | ✅        | Settings (login)                    |
| `templates`            | ✅        | Settings page                       |
| `workflow`             | ✅        | `/api/v1/workflow/diagram`          |
| `license`              | ✅        | `/api/v1/license`                   |
| `mcp`                  | ✅        | MCP server toggle                   |
| `interactive`          | ✅        | `/interactive` page                 |
| `init`                 | CLI-only | Workspace setup                     |
| `serve`                | N/A      | Self-referential                    |
| `generate-secret`      | CLI-only | Utility                             |
| `update`               | CLI-only | CLI self-update                     |
| `hooks/lefthook`       | CLI-only | Dev tool                            |

**Legend**: ✅ Full | ⚠️ Partial | ❌ Missing | CLI-only | CI/CD | N/A

---

## Interactive Modes Parity

| Feature           | CLI REPL   | Web Chat | JetBrains | VS Code | Notes                                                                       |
|-------------------|------------|----------|-----------|---------|-----------------------------------------------------------------------------|
| **Workflow**      |
| `start`           | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `plan`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `implement`       | ✅ (`impl`) | ✅        | ✅         | ✅       |                                                                             |
| `review`          | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `finish`          | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `continue`        | ✅ (`cont`) | ✅        | ✅         | ✅       |                                                                             |
| `abandon`         | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `auto`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| **Session**       |
| `status`          | ✅ (`st`)   | ✅        | ✅         | ✅       |                                                                             |
| `note`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `question`        | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `answer`          | ✅ (`a`)    | ✅        | ✅         | ✅       |                                                                             |
| `specification`   | ✅ (`spec`) | ✅        | ✅         | ✅       |                                                                             |
| `cost`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `list`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `quick`           | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `budget`          | ✅          | ✅        | ✅         | ✅       | status/reset                                                                |
| `reset`           | ✅          | ✅        | ✅         | ✅       |                                                                             |
| **Search**        |
| `find`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `memory`          | ✅          | ✅        | ✅         | ✅       | search/index/stats                                                          |
| `library`         | ✅          | ✅        | ✅         | ✅       | list/show/pull/remove/stats                                                 |
| `links`           | ✅          | ✅        | ✅         | ✅       | list/search/stats/rebuild                                                   |
| **Queue Tasks**   |
| `delete --task`   | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `export --task`   | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `optimize --task` | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `submit`          | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `sync`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| **Tools**         |
| `simplify`        | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `label`           | ✅          | ✅        | ✅         | ✅       | list/add/remove                                                             |
| `scan`            | ❌          | ✅        | ✅         | ✅       | CLI redirect                                                                |
| `commit`          | ❌          | ✅        | ✅         | ✅       | CLI redirect                                                                |
| **Browser**       |
| `browser *`       | ❌          | ✅        | ✅         | ✅       | status/tabs/goto/navigate/reload/screenshot/click/type/eval/console/network |
| **Project**       |
| `project *`       | ❌          | ✅        | ✅         | ✅       | plan/tasks/edit/submit/start/sync                                           |
| **Stack**         |
| `stack *`         | ❌          | ✅        | ✅         | ✅       | list/rebase/sync                                                            |
| **Config**        |
| `config *`        | ❌          | ✅        | ✅         | ✅       | validate only; explain redirects to CLI                                     |
| `agents`          | ❌          | ✅        | ✅         | ✅       | list/explain                                                                |
| `providers`       | ❌          | ✅        | ✅         | ✅       | list/info; status redirects to CLI                                          |
| `templates`       | ❌          | ✅        | ✅         | ✅       | list/show; apply redirects to CLI                                           |
| **Navigation**    |
| `undo`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `redo`            | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `clear`           | ✅          | N/A      | N/A       | N/A     | Web uses UI refresh                                                         |
| `help`/`?`        | ✅          | ✅        | ✅         | ✅       |                                                                             |
| `exit`/`quit`     | ✅          | ✅        | N/A       | N/A     | Close tab/panel                                                             |
| **Chat**          |
| `chat <msg>`      | ✅          | ✅ (main) | ✅         | ✅       |                                                                             |

---

## MCP Server Parity

| CLI Command   | MCP Tool            | Status | Notes                           |
|---------------|---------------------|--------|---------------------------------|
| All commands  | Via CLI passthrough | ✅      | Full parity via Cobra mapping   |
| `init`        | N/A                 | Remove | CLI-only, should not be exposed |
| `serve`       | N/A                 | Remove | CLI-only, should not be exposed |
| `workspace_*` | Dedicated tools     | ✅      | AI-friendly data access         |
| `library_*`   | Dedicated tools     | ✅      | Documentation access            |
| `agents_*`    | Dedicated tools     | ✅      | Registry access                 |
| `providers_*` | Dedicated tools     | ✅      | Registry access                 |

