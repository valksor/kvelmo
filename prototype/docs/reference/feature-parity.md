# Feature Parity Reference

This document tracks implementation status across Mehrhof's interfaces. Use this as a checklist when adding new features.

## Interface Overview

Mehrhof has four user interfaces:

| Interface | Entry Point | Purpose |
|-----------|-------------|---------|
| **CLI** | `cmd/mehr/commands/` | Full command-line interface |
| **Interactive CLI** | `mehr interactive` | REPL mode for workflow sessions |
| **Web UI** | `internal/server/` | Browser interface with dashboard |
| **Interactive Web** | `/interactive` | Browser REPL with SSE streaming |

---

## Implementation Checklist

When adding a new feature, complete ALL applicable items:

- [ ] **CLI Command**: Add in `cmd/mehr/commands/*.go` using Cobra
- [ ] **Interactive CLI**: Add to `interactive` allowed commands if workflow-relevant
- [ ] **Web UI Handler**: Add in `internal/server/handlers*.go` or `internal/server/api/`
- [ ] **Interactive Web**: Add to `/interactive` command parser if workflow-relevant
- [ ] **Router Registration**: Update `internal/server/router.go`
- [ ] **Template/View**: Add in `internal/server/templates/` or `internal/server/views/`
- [ ] **Navigation**: Update menus if user-facing
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

## CLI vs Web UI Parity

| CLI Command | Web UI | Notes |
|-------------|--------|-------|
| `start <ref>` | ✅ | Dashboard + project pages |
| `plan` | ✅ | SSE streaming |
| `implement` | ✅ | SSE streaming |
| `review` | ✅ | SSE streaming |
| `finish` | ✅ | PR creation/merge |
| `continue` | ✅ | Resume from waiting |
| `abandon` | ✅ | Discard task |
| `status` | ✅ | Dashboard display |
| `note <msg>` | ✅ | Quick note form |
| `question <msg>` | ✅ | Quick question + SSE |
| `cost` | ✅ | Detailed breakdown by step |
| `list` | ✅ | Recent tasks sidebar |
| `undo/redo` | ✅ | Checkpoint navigation |
| `links` | ✅ | `/links` page |
| `find` | ✅ | `/find` page |
| `browser` | ✅ | `/browser` page |
| `mcp` | ✅ | MCP server toggle |
| `scan` | ✅ | `/scan` page with scanner selection |
| `memory` | ✅ | `/memory` page |
| `commit` | ✅ | `/commit` page with analyze/preview |
| `project sync` | ✅ | API + SSE streaming |
| `stack` | ✅ | `/stack` page |
| `interactive` | ✅ | `/interactive` page |
| `budget` | ✅ | API + monthly status/reset |
| `optimize` | ✅ | Quick task optimization |
| `export` | ✅ | Quick task export |
| `serve` | N/A | Self-referential |
| `config validate` | ✅ | Settings validation |
| `agents` | ✅ | Settings page |
| `providers` | ✅ | Settings (login) |
| `templates` | ✅ | Settings page |
| `generate-secret` | ❌ | CLI-only utility |
| `update` | ❌ | CLI-only utility |
| `hooks/lefthook` | ❌ | CLI-only dev tool |
| `workflow` | ❌ | CLI-only diagnostic |

**Legend**: ✅ Full | ⚠️ Partial | ❌ Missing | N/A Not applicable

---

## Interactive Modes Parity

| Feature | CLI REPL | Web `/interactive` | Notes |
|---------|----------|-------------------|-------|
| **Workflow** |
| `start` | ✅ | ✅ | |
| `plan` | ✅ | ✅ | |
| `implement` | ✅ (`impl`) | ✅ | |
| `review` | ✅ | ✅ | |
| `finish` | ✅ | ✅ | |
| `continue` | ✅ (`cont`) | ✅ | |
| `abandon` | ✅ | ✅ | |
| **Session** |
| `status` | ✅ (`st`) | ✅ | |
| `note` | ✅ | ✅ | |
| `question`/`ask` | ✅ | ✅ | |
| `answer` | ✅ (`a`) | ✅ | |
| `specification` | ✅ (`spec`) | ✅ | |
| `cost` | ✅ | ✅ | |
| `list` | ✅ | ✅ | |
| `quick` | ✅ | ✅ | |
| **Navigation** |
| `undo` | ✅ | ✅ | |
| `redo` | ✅ | ✅ | |
| `clear` | ✅ | N/A | Web uses UI refresh |
| `help`/`?` | ✅ | ✅ | |
| `exit`/`quit` | ✅ | ✅ | Close tab |
| **Chat** |
| `chat <msg>` | ✅ | ✅ (main input) | |

---

## When CLI-Only Is Appropriate

Commands that should NOT be added to interactive modes:

- **One-shot utilities**: `generate-secret`, `update check/install`
- **Developer tools**: `hooks`, `lefthook`, `config validate`
- **System operations**: `serve`, `workflow`
- **Flag-heavy operations**: Commands with many CLI flags
- **Setup tasks**: Operations run once, not during workflow sessions

**The key question**: "Would a user run this during a conversational workflow session?"

---

## When to Add to Interactive Modes

Add commands to both CLI REPL and Web `/interactive` when:

- **Workflow control**: `start`, `plan`, `implement`, `review`, `finish`, `continue`, `abandon`
- **Session context**: `status`, `note`, `question`, `specification`, `cost`, `list`
- **Session navigation**: `undo`, `redo`, `clear`, `help`, `exit`
- **Quick actions**: `quick`, `answer`
