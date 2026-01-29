# AGENTS.md / CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mehrhof is a **Go CLI tool + Web UI** for AI-powered task automation. It orchestrates AI agents (primarily Claude) to perform planning, implementation, and code review workflows with checkpointing, parallel task support, and multi-provider integrations.

**⚠️ ALL features must be implemented for BOTH CLI and Web UI interfaces. Interactive modes (CLI REPL + Web `/interactive`) support workflow commands based on usability considerations.** See the "Multi-Interface Implementation" section below.

---

## ⚠️ CRITICAL: go-toolkit Usage Guidelines

**DO NOT re-export or wrap go-toolkit functionality unnecessarily.**

go-mehrhof shares code with `github.com/valksor/go-toolkit` for reuse across Valksor projects. The purpose of go-toolkit is to **eliminate duplication**, not create additional abstraction layers.

### What NOT to Do:

```go
// ❌ BAD - Type alias re-export
type Bus = eventbus.Bus
type Result = validate.Result

// ❌ BAD - Wrapper function
func Slugify(title string, maxLen int) string {
    return slug.Slugify(title, maxLen)
}
```

### What to Do Instead:

```go
// ✅ GOOD - Import and use go-toolkit directly
import "github.com/valksor/go-toolkit/eventbus"

bus := eventbus.NewBus()

// ✅ GOOD - Domain-specific types that add value
type AgentConfig struct {
    Name string
    // ... mehrhof-specific agent configuration
}

// ✅ GOOD - Domain-specific functions with business logic
func ColorState(state, displayName string) string {
    // Maps mehrhof's workflow states to colors
}
```

### When to Add Code to go-toolkit vs. go-mehrhof:
- **go-toolkit**: Generic, reusable utilities with no mehrhof dependencies that could be used by other Valksor projects
- **go-mehrhof**: Domain-specific business logic with mehrhof dependencies

### Examples: Use `eventbus.Bus`, `validate.Result`, `jsonrpc.Request`, `slug.Slugify()` directly from go-toolkit. Keep domain-specific functions like `ColorState()` in mehrhof.

### Enforcement: CI runs `make check-alias` to detect unnecessary import aliases.

---

## ⚠️ CRITICAL: Multi-Interface Implementation

**ALL features must be implemented for BOTH CLI and Web UI unless explicitly CLI-only. Interactive modes (CLI REPL + Web `/interactive`) support workflow commands based on usability considerations.**

Mehrhof has four user interfaces:
1. **CLI** - Full command-line interface via `cmd/mehr/commands/`
2. **Interactive CLI** - REPL mode via `mehr interactive` (workflow command subset)
3. **Web UI** - Web interface pages via `internal/server/`
4. **Interactive Web** - Browser-based REPL at `/interactive` (workflow command subset)

### Implementation Checklist

When adding a new feature, complete ALL applicable items:

- [ ] **CLI Command**: Add command in `cmd/mehr/commands/*.go` using Cobra
- [ ] **Interactive CLI**: Add command to `interactive` allowed commands if workflow-relevant (see "When to Add to Interactive Modes" below)
- [ ] **Web UI Handler**: Add handler in `internal/server/handlers*.go` or `internal/server/api/`
- [ ] **Interactive Web**: Add to `/interactive` command parser if workflow-relevant (see "When to Add to Interactive Modes" below)
- [ ] **Router Registration**: Update `internal/server/router.go` to register new routes
- [ ] **Template/View**: Add template in `internal/server/templates/` or `internal/server/views/`
- [ ] **Navigation**: Update menus/navigation if feature is user-facing
- [ ] **SSE Streaming**: Add Server-Sent Events for long-running operations
- [ ] **Tests**: Write comprehensive tests for new functionality (see "Testing Requirements" section below)
- [ ] **Documentation**: Update docs/ (CLI and/or Web UI), README.md, and CLAUDE.md as needed (see "Documentation Requirements" section below)

### Implementation Patterns

Both interfaces should delegate to **shared core logic** in `internal/conductor/`:

```go
// CLI Pattern (cmd/mehr/commands/plan.go)
var planCmd = &cobra.Command{
    Use:   "plan [topic]",
    Short: "Enter planning phase",
    RunE: runPlan,
}

func runPlan(cmd *cobra.Command, args []string) error {
    cond, err := initializeConductor(ctx, opts...)
    if err != nil {
        return err
    }
    if err := cond.Plan(ctx); err != nil {
        return fmt.Errorf("plan: %w", err)
    }
    return nil
}
```

```go
// Web UI Pattern (internal/server/handlers.go)
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
    if s.config.Conductor == nil {
        s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")
        return
    }

    if err := s.config.Conductor.Plan(r.Context()); err != nil {
        s.writeError(w, http.StatusInternalServerError, "failed to enter planning: "+err.Error())
        return
    }

    s.writeJSON(w, http.StatusOK, map[string]any{
        "success": true,
        "message": "planning completed",
    })
}
```

**Key Point**: Both CLI and Web UI call `cond.Plan(ctx)` - the core logic is shared. The interfaces are just thin adapters.

### SSE Streaming for Long-Running Operations

For operations that take time (planning, implementing, reviewing), use SSE to stream progress:

```go
// Web UI SSE Pattern
func (s *Server) handleWorkflowPlan(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        s.writeError(w, http.StatusBadRequest, "streaming not supported")
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    fmt.Fprintf(w, "event: status\ndata: {\"message\": \"Starting planning...\"}\n\n")
    flusher.Flush()
    // ... execute operation ...
    fmt.Fprintf(w, "event: complete\ndata: {\"success\": true}\n\n")
    flusher.Flush()
}
```

### Feature Parity: CLI vs Web UI

| CLI Command | Web UI Status | Notes |
|-------------|---------------|-------|
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
| `cost` | ⚠️ | Basic only, detailed reporting missing |
| `list` | ✅ | Recent tasks sidebar |
| `undo/redo` | ✅ | Checkpoint navigation |
| `links` | ✅ | Full parity - `/links` page |
| `find` | ✅ | `/find` page |
| `browser` | ✅ | `/browser` page |
| `mcp` | ✅ | MCP server toggle |
| `scan` | ⚠️ | API endpoint exists, no UI |
| `memory` | ⚠️ | API endpoint exists, no UI |
| `project sync` | ✅ | API + SSE streaming |
| `stack` | ✅ | `/stack` page |
| `interactive` | ✅ | `/interactive` page |
| `serve` | N/A | Self-referential |
| `config validate` | ✅ | Settings page validation |
| `agents` | ✅ | Settings page |
| `providers` | ✅ | Settings page (login) |
| `templates` | ✅ | Settings page |
| `generate-secret` | ❌ | CLI-only utility |
| `update check/install` | ❌ | CLI-only utility |
| `hooks/lefthook` | ❌ | CLI-only development tool |
| `workflow` | ❌ | CLI-only diagnostic |
| `budget` | ❌ | Missing - only basic stats |
| `optimize` | ❌ | Missing |
| `export` | ❌ | Missing |

**Legend**: ✅ = Full support | ⚠️ = Partial support | ❌ = Not available | N/A = Not applicable

### Feature Parity: Interactive Modes (CLI REPL vs Web `/interactive`)

| Feature | Interactive CLI | Interactive Web | Notes |
|---------|-----------------|-----------------|-------|
| **Workflow Commands** |
| `start` | ✅ | ✅ | |
| `plan` | ✅ | ✅ | |
| `implement` | ✅ (alias: `impl`) | ✅ | |
| `review` | ✅ | ✅ | |
| `finish` | ✅ | ✅ | |
| `continue` | ✅ (alias: `cont`) | ✅ | |
| `abandon` | ✅ | ✅ | |
| **Session Commands** |
| `status` | ✅ (alias: `st`) | ✅ | |
| `note` | ✅ | ✅ | |
| `question` / `ask` | ✅ (via chat) | ✅ | |
| `answer` | ✅ (alias: `a`) | ✅ | |
| `specification` | ✅ (alias: `spec`) | ✅ | |
| `cost` | ✅ | ✅ | |
| `list` | ✅ | ✅ | |
| `quick` | ✅ | ✅ | |
| **Navigation** |
| `undo` | ✅ | ✅ | |
| `redo` | ✅ | ✅ | |
| `clear` | ✅ | N/A | Web has UI refresh instead |
| `help` / `?` | ✅ | ✅ | |
| `exit` / `quit` | ✅ | ✅ | Close tab/window |
| **Chat Interface** |
| `chat <msg>` | ✅ | ✅ (main input) | |
| **CLI-Only** (not suitable for REPL): |
| `project sync` | ❌ | ❌ | Flag-heavy operation |
| `browser` | ❌ | ❌ | Separate page for control |
| `mcp` | ❌ | ❌ | System operation |
| `config validate` | ❌ | ❌ | Setup task |
| `generate-secret` | ❌ | ❌ | Setup task |
| `update` | ❌ | ❌ | System operation |
| `hooks/lefthook` | ❌ | ❌ | Dev tool |
| `serve` | ❌ | ❌ | Server start |
| `workflow` | ❌ | ❌ | Diagnostic |

**Legend**: ✅ = Full support | ❌ = Not available | N/A = Not applicable

### When CLI-Only Is Appropriate

- **One-shot operations**: `generate-secret`, `update check/install`
- **Developer utilities**: `hooks`, `lefthook`, `config validate`
- **Debugging/diagnostic**: `status --diagram`, `cost --breakdown`

### When to Add Commands to Interactive Modes

**Interactive modes (CLI REPL + Web `/interactive`)** are designed for **conversational workflow sessions**. The key consideration is **usability in interactivity** - does this command make sense when the user is in a REPL/chat session?

**Add to interactive modes when:**
- Workflow control: `start`, `plan`, `implement`, `review`, `finish`, `continue`, `abandon`
- Session context: `status`, `note`, `question`, `specification`, `cost`, `list`
- Session navigation: `undo`, `redo`, `clear`, `help`, `exit`
- Quick actions: `quick`, `answer` (respond to agent questions)

**Keep CLI-only when:**
- **Configuration workflows**: `config validate`, `generate-secret` - setup tasks, not session tasks
- **Developer utilities**: `hooks`, `lefthook` - development setup, not workflow-related
- **System operations**: `update check/install`, `serve`, `workflow` - system-level, not session-level
- **Flag-heavy operations**: Commands with many CLI flags don't translate well to REPL
- **One-shot diagnostics**: Operations you run once, not during a workflow session

**The key question**: "Would a user want to run this while in the middle of a conversational session with the agent?"

### Verification: Test both CLI and Web UI implementations, verify error handling, check CLI flags map to Web UI inputs, ensure SSE streaming works for long-running operations, update feature parity tables.

---

## ⚠️ CRITICAL: Feature Requirements - Tests, Docs, and CLAUDE.md

**ALL new features MUST include tests and documentation updates.**

### Testing Requirements

Every new feature MUST have corresponding tests:

| Test Type | When Required | Location | Target Coverage |
|-----------|---------------|----------|-----------------|
| Unit tests | Always | `*_test.go` next to source file | 80%+ for new code |
| Integration tests | For handlers, conductors, providers | `*_test.go` or `internal/helper_test/` | Critical paths |
| E2E tests | For user-facing workflows | `e2e/fast/` or `e2e/full/` | Main workflows |
| Table-driven tests | For functions with multiple input cases | `*_test.go` | All edge cases |

### Testing Guidelines

1. **Write tests FIRST** (TDD) when implementing features
2. **Use table-driven tests** for functions with multiple input scenarios (see pattern in `internal/helper_test/`)
3. **Test utilities** available in `internal/helper_test/` (mocks, fixtures, conductor helpers)
4. **Run `make test`** before committing code changes - all tests must pass (skip for docs-only changes)
5. **Run `make coverage-html`** to verify coverage - new code should be 80%+ covered

### Documentation Requirements

Every new feature MUST update relevant documentation:

| Documentation | When to Update | Location |
|---------------|----------------|----------|
| CLI docs | CLI commands added/changed | `docs/cli/*.md` |
| Web UI docs | Web UI features added/changed | `docs/web-ui/*.md` |
| README.md | User-facing features, installation changes | `README.md` |
| CLAUDE.md | New patterns, architecture changes, critical rules | `CLAUDE.md` |
| Configuration docs | New config options | `docs/configuration/*.md` |

### Documentation Checklist

Before considering a feature complete:

- [ ] **CLI doc**: Add/update `docs/cli/feature-name.md` with command usage, examples, flags
- [ ] **Web UI doc**: Add/update `docs/web-ui/feature-name.md` with UI usage, screenshots if applicable
- [ ] **README.md**: Update if feature is user-visible or affects installation/quickstart
- [ ] **CLAUDE.md**: Update if:
  - New architecture pattern is introduced
  - New critical rule or guideline is needed
  - Feature parity table changes
  - New package or major code organization change

### Documentation Content Standards

All documentation must include:

1. **Purpose**: What the feature does and why it exists
2. **Usage**: How to use it (commands, UI, config)
3. **Examples**: At least one working example
4. **Requirements**: Prerequisites or dependencies
5. **Troubleshooting**: Common issues and solutions (if applicable)

### When README.md Updates Are Required

Update `README.md` when:

- **New user-facing feature** is added
- **Installation process** changes
- **Quickstart** instructions need updating
- **Configuration** examples need new options
- **Breaking changes** are introduced

### When CLAUDE.md Updates Are Required

Update `CLAUDE.md` when:

- **New architecture pattern** is introduced (add to Architecture section)
- **New critical rule** is needed (add as "⚠️ CRITICAL" section)
- **Feature parity** changes (update the parity table)
- **New package** is added (update Core Packages table)
- **Code style** conventions change (update Code Style section)
- **Code organization guidelines** change (update relevant sections)
- **New workflow state** is added (update Workflow States table)

### Examples

#### ✅ GOOD: Complete Feature Delivery
```
Added budget feature:
✅ cmd/mehr/commands/budget.go + tests (85% coverage)
✅ internal/server/handlers_budget.go + tests
✅ internal/server/templates/budget.html
✅ docs/cli/budget.md, docs/web-ui/budget.md
✅ README.md updated
```

#### ❌ BAD: Incomplete Feature Delivery
```
Added budget feature:
✅ cmd/mehr/commands/budget.go
❌ No tests, no Web UI implementation, no documentation
```

### Verification

Before marking a feature as complete:

1. **Code changes made?** → Run `make quality && make test` - must pass
2. **Tests**: `make coverage-html` shows 80%+ for new code
3. **CLI doc**: Documentation exists in `docs/cli/` with usage examples
4. **Web UI doc**: Documentation exists in `docs/web-ui/` with UI instructions
5. **README**: Updated if feature is user-facing
6. **CLAUDE.md**: Updated if new patterns/rules introduced
7. **Both interfaces tested**: CLI and Web UI both work

**Note**: For documentation-only updates (steps 3-6 without code changes), skip `make quality` and `make test`.

---

## ⚠️ CRITICAL: Documentation Organization - Separate by Interface Type

**Documentation MUST be organized by interface type.**

### Directory Structure

| Directory | Purpose |
|-----------|---------|
| `docs/cli/` | CLI-specific documentation only |
| `docs/web-ui/` | Web UI-specific documentation only |
| `docs/concepts/` | Interface-agnostic concepts (workflows, architecture) |
| `docs/guides/` | Procedural guides (split by interface if interface-specific) |
| `docs/configuration/` | Shared configuration (interface-specific config goes in `docs/cli/` or `docs/web-ui/`) |
| `docs/providers/` | Provider documentation (interface-agnostic) |
| `docs/agents/` | Agent documentation (interface-agnostic) |
| `docs/advanced/` | Advanced topics (interface-agnostic) |
| `docs/reference/` | Technical reference (interface-agnostic) |

### Rules

1. **Single Interface per Document**: A documentation file should cover EITHER CLI OR Web UI, never both.
2. **Use Interface-Specific Directories**: Place CLI docs in `docs/cli/`, Web UI docs in `docs/web-ui/`.
3. **Cross-Reference**: If a concept applies to both interfaces, document it in the interface-specific docs and cross-reference:
   - CLI doc: "See [Web UI equivalent](/docs/web-ui/feature.md) for the web interface."
   - Web UI doc: "See [CLI equivalent](/docs/cli/feature.md) for the command-line interface."

### Exceptions (When Both Interfaces in One Doc Is OK)

1. **Comparison Docs**: Files that explicitly compare the two interfaces (e.g., `docs/guides/web-ui-vs-cli.md`)
2. **Feature Parity Tables**: Documents that track implementation status across interfaces
3. **Architecture/Conceptual Docs**: Files that describe underlying architecture that both interfaces share (e.g., workflow state machine, storage model)

### Examples

#### ✅ GOOD: Interface-Specific Documentation
```
docs/cli/scan.md              # CLI scanning commands only
docs/web-ui/security-quality.md  # Web UI security/quality tools only
docs/cli/memory.md            # CLI memory commands
docs/web-ui/memory.md         # Web UI memory features
```

#### ❌ BAD: Mixed Interface Documentation
```
docs/configuration/index.md   # Covers both CLI flags AND Web UI settings
docs/guides/first-task.md     # If it covers both CLI and Web UI workflows
```

#### ✅ GOOD: Split Mixed Docs Into:
```
docs/configuration/cli-flags.md       # CLI-specific flags
docs/configuration/web-ui-settings.md # Web UI-specific settings
docs/configuration/shared.md          # Settings common to both
```

### When Creating New Documentation

1. **Identify the interface**: CLI-only → `docs/cli/`, Web UI-only → `docs/web-ui/`, Interface-agnostic → `docs/concepts/`
2. **Check for existing related docs** and add cross-references
3. **Avoid duplication**: If a feature works identically in both interfaces, document in `docs/concepts/` and reference from both interface-specific docs

### Verification: Confirm file is in correct directory, verify content covers only ONE interface (unless comparison/parity doc), check for cross-references, run `make docs` (if available).

---

## ⚠️ CRITICAL: Zero Tolerance for Broken Code

**ALL tests must pass and `make quality` must succeed before committing code changes.**

There is no such thing as "not my code" or "not my problem." If tests fail or quality checks fail, they MUST be fixed before proceeding—regardless of:
- Who wrote the broken code
- When the code was written (yesterday, last week, last year)
- Whether you touched the related files
- Whether you think it's "not your job"

### The Rule

**Broken code blocks everything.** You cannot:
- Commit code while tests are failing
- Submit a PR with quality check failures
- Say "I didn't break it" and move on
- Assume someone else will fix it

**Exception**: Documentation-only commits don't require running build/test/quality commands. See "When to Run Build, Test, and Quality Commands" section.

### What MUST Pass

| Check | Command | Exit Code |
|-------|---------|-----------|
| All tests | `make test` | MUST be 0 |
| Code quality | `make quality` | MUST be 0 |
| Linting | `golangci-lint run` | MUST be 0 |
| Formatting | `make fmt` | MUST produce no changes |
| Alias check | `make check-alias` | MUST be 0 |
| Vulnerabilities | `govulncheck ./...` | MUST find none |

### Workflow

1. **Before starting work**: Run `make quality && make test` once to verify baseline. If anything fails, FIX IT FIRST. Do not add new code on top of broken code.
2. **Before committing code changes**: Run `make quality && make test` - commit gatekeeper for code changes: nothing gets committed if these fail. For docs-only changes, skip this step.
3. **Before PR**: Run `make quality && make test` - PR gatekeeper: nothing gets submitted if these fail. Only needed if PR includes code changes.

### Examples

#### ✅ GOOD: Fix Broken Tests First
```bash
$ make test
--- FAIL: TestFoo (0.00s)
    foo_test.go:42: expected "bar", got "baz"
FAIL
# Don't add new code. Fix this first.
$ vim foo_test.go  # or foo.go
$ make test
PASS
# NOW you can proceed with your work
```

#### ❌ BAD: Ignore Broken Tests
```bash
$ make test
--- FAIL: TestFoo (0.00s)
FAIL
# WRONG: "I didn't write this, not my problem"
$ git commit -m "add new feature"
```

### Escalation: If you encounter a test failure or quality issue you truly cannot fix: 1) Document it with reproduction steps, 2) Block the codebase - do not merge anything until resolved, 3) Alert the team immediately, 4) DO NOT work around it.

### Verification

Before committing code changes or creating a PR with code:

```bash
make quality && make test
```

For documentation-only changes, no build/test/quality commands are needed.

If the command does not exit with code 0 for code changes, you are NOT ready to commit.

---

## ⚠️ CRITICAL: Use Make Commands for Build Operations

**ALWAYS use `make` commands instead of direct `go` commands.**

The Makefile provides standardized, consistent commands that may run multiple operations (e.g., `make quality` runs linting, formatting, AND vuln checking).

### Common Commands

| Operation | Make Command | What It Does |
|-----------|--------------|--------------|
| Build | `make build` | Compiles binary with embedded licenses |
| Install | `make install` | Builds and installs to GOPATH/bin |
| Format | `make fmt` | Runs gofmt, goimports, gofumpt |
| Quality | `make quality` | Runs linter, formatter, vuln check, alias check |
| Test | `make test` | Runs all tests with coverage |
| Coverage | `make coverage` | Generates coverage profile |
| Coverage HTML | `make coverage-html` | Generates HTML coverage report |
| Clean | `make clean` | Removes build artifacts |
| Tidy | `make tidy` | Cleans and tidies dependencies |

### What NOT to Do / What to Do Instead

```bash
# ❌ BAD - Direct go commands
go build ./cmd/mehr
go fmt ./...
go test ./...

# ✅ GOOD - Use make commands
make build
make fmt
make test
make quality
```

### When to Run Build, Test, and Quality Commands

**ONLY run build, test, and quality commands when CODE changes are involved.**

| Command | Run When... | Do NOT Run When... |
|---------|-------------|-------------------|
| `make build` | You've modified `.go` files | Documentation-only changes |
| `make test` | You've modified `.go` or `*_test.go` files | Documentation-only changes |
| `make quality` | You've modified `.go` files | Documentation-only changes |
| `make fmt` | You've modified `.go` files | Documentation-only changes |
| `make coverage-html` | You've added/modified tests | Documentation-only changes |

### Code vs. Non-Code Changes

**Code changes** (run build/test/quality): Modified `.go` source files, added/removed imports, changed package structure, modified test files.

**Non-code changes** (DO NOT run build/test/quality): Documentation updates (`*.md`, `docs/`), git operations (commits, merges, rebases), configuration changes (`.yaml`, `.toml`, `.json`), asset changes (images, templates, scripts), comments-only changes to `.go` files.

### Workflow Decision Tree

```
Did you change any .go files?
├── YES → Run `make quality && make test`
│       └── Did tests pass? → Proceed with commit/PR
│       └── Did tests fail? → Fix before committing
└── NO → Skip build/test/quality (docs, config, git ops only)
```

### Exception: Verify Health Before Starting Work

**Always run `make quality && make test` once before starting new work**, regardless of what you're doing. This ensures you're not building on top of already-broken code.

### Verification

Before committing or creating a PR:

1. **Did you change .go files?** → Run `make quality && make test`
2. **Only documentation?** → Commit directly, no build/test needed
3. **Only git operations?** → No build/test needed
4. **Starting a new session?** → Run `make quality && make test` once to verify baseline

---

## ⚠️ CRITICAL: golangci-lint Configuration and nolint Usage

**NEVER disable golangci-lint linters in configuration. Use `//nolint` as a LAST RESORT, not a first resort.**

The `.golangci.yml` configuration enables 39 linters that catch bugs, security issues, and code quality problems. Disabling linters or overusing `//nolint` erodes code quality and allows issues to accumulate.

### Never Disable Linters in Configuration

**DO NOT modify `.golangci.yml` to disable linters.**

If a linter is flagging legitimate issues, the solution is to:
1. Fix the code (preferred)
2. Use targeted `//nolint` comments for specific, justified exceptions (as last resort)

```yaml
# ❌ BAD - Disabling linters to avoid fixing issues
linters:
    disable:
        - errcheck     # "too many false positives"
        - gosec        # "security rules too strict"
        - nilerr       # "don't want to fix these"
```

```yaml
# ✅ GOOD - Keep all linters enabled
linters:
    enable:
        - errcheck
        - gosec
        - nilerr
        # ... all other linters remain enabled
```

### nolint Comment Guidelines

**`//nolint` is a LAST RESORT, not a first resort.**

Before using `//nolint`, you MUST:
1. **Understand the warning**: Research why the linter is flagging this code
2. **Fix the issue**: Refactor code to address the linter's concern
3. **Document justification**: Only use `//nolint` if the warning is truly a false positive

#### Acceptable nolint Use Cases

| Scenario | Example | Justification |
|----------|---------|---------------|
| **API compliance** | Unused parameter required by interface | `//nolint:unparam // Required by interface SomeInterface` |
| **Intentional nil-nil returns** | Function returns nil,nil for "not found" | `//nolint:nilnil // No task found is not an error` |
| **Benchmark code** | No context needed in benchmarks | `//nolint:noctx // Benchmark: no cancellation needed` |
| **String builders** | Error check won't fail | `//nolint:errcheck // String builder WriteString won't fail` |

#### Unacceptable nolint Use Cases

| Scenario | ❌ Wrong Approach | ✅ Correct Approach |
|----------|------------------|---------------------|
| **Unchecked error** | `//nolint:errcheck` | Handle the error properly |
| **Security issue** | `//nolint:gosec` | Fix the security vulnerability |
| **Complex function** | `//nolint:gocyclo` | Refactor into smaller functions |
| **Unused code** | `//nolint:deadcode` | Remove the dead code |
| **Lazy fix** | `//nolint:all` | Fix each issue individually |

### nolint Best Practices

1. **Always specify the linter name**: Never use `//nolint:all`
   ```go
   // ❌ BAD - Suppresses all linters
   //nolint:all

   // ✅ GOOD - Specific linter
   //nolint:unparam // Error return reserved for future validation needs
   ```

2. **Always include a justification**: Explain WHY the suppression is necessary
   ```go
   // ❌ BAD - No explanation
   //nolint:errcheck

   // ✅ GOOD - Explains the reasoning
   //nolint:errcheck // Writing to string builder cannot fail
   ```

3. **Keep scope minimal**: Place `//nolint` on the specific line, not the entire function
   ```go
   // ❌ BAD - Suppresses for entire function
   //nolint:errcheck
   func foo() {
       bar()
       baz()
   }

   // ✅ GOOD - Suppresses only the specific line
   func foo() {
       bar()
       _ = strings.Builder.WriteString(buf, "text") //nolint:errcheck // Safe
       baz()
   }
   ```

4. **Prefer code fixes over nolint**: Refactor to avoid the warning (e.g., use sentinel values instead of nil,nil returns)

### Enforcement & Verification

The `nolintlint` linter is enabled to catch `//nolint` without specifying which linter, without an explanation, or machine-generated comments. CI will fail if new `//nolint` comments don't specify the linter, lack justification, or if linters are disabled in `.golangci.yml`.

Before committing code with `//nolint`: 1) Verify the warning is a false positive, 2) Prefer refactoring over suppression, 3) Specify the exact linter name, 4) Document WHY the suppression exists. If you cannot justify the `//nolint`, **fix the code instead.**

---

## ⚠️ CRITICAL: File Size Limits - Keep Files Under 500 Lines

**ALL Go files (source and test) should be kept under 500 lines.**

Large files degrade AI agent performance, increase error rates, and reduce code maintainability. When a file grows beyond 500 lines, split it into smaller, more focused files.

### The Rule

**Keep files under 500 lines.** If a file exceeds this limit, split it into logical categories.

### File Size Guidelines

| File Size | Action |
|-----------|--------|
| **< 300 lines** | Ideal - fast, accurate |
| **300-500 lines** | Acceptable - works well |
| **500-700 lines** | Warning - consider splitting |
| **700+ lines** | **MUST split** - too large |

### How to Split Source Files

Organize by **responsibility** or **feature**:

```go
// ❌ BAD - One 1000-line file
internal/conductor/handlers.go (1000+ lines)

// ✅ GOOD - Split by feature
internal/conductor/handlers_plan.go      // Planning handlers
internal/conductor/handlers_implement.go // Implementation handlers
internal/conductor/handlers_review.go    // Review handlers
```

### How to Split Test Files

Organize by **feature** or **test category**:

```go
// ❌ BAD - One 3000-line test file
internal/storage/workspace_test.go (3000+ lines)

// ✅ GOOD - Split by feature
internal/storage/workspace_save_test.go      // Save operations
internal/storage/workspace_load_test.go      // Load operations
internal/storage/workspace_config_test.go    // Config operations
internal/storage/workspace_integration_test.go // Integration tests
```

All test files in the same package share a namespace - tests will run normally when split.

### When Larger Files Are Acceptable

Some files may legitimately exceed 500 lines:

- **Generated code** - Not meant for human editing
- **Single-responsibility modules** - If truly cohesive (e.g., state machine with many states)
- **Prompt/template files** - Large templates are acceptable

These are exceptions, not the norm. Justify with a comment if exceeding 500 lines.

### Splitting Guidelines

When splitting files:

1. **Identify boundaries** - Group related functions/types together
2. **Use descriptive names** - `handlers_plan.go`, not `handlers_2.go`
3. **Keep package namespace** - All files in same package share types
4. **Move imports** - Each split file needs its own imports
5. **Update tests** - Split test files alongside source

### Verification

Before considering work complete:
1. **Check file size**: `wc -l path/to/file.go`
2. **Verify imports**: Each file has necessary imports
3. **Run tests**: `make test` - all tests must pass
4. **Run quality**: `make quality` - must pass

---

## Commands

### Build & Development

```bash
make build | install | test | coverage | quality | fmt | tidy | hooks | lefthook
```

Available make targets: `all`, `build`, `test`, `coverage`, `coverage-html`, `quality`, `fmt`, `install`, `clean`, `run`, `run-args`, `tidy`, `deps`, `version`, `hooks`, `lefthook`, `check-alias`, `help`

See [README.md](README.md) for full documentation.

### Workers Site JavaScript

The `workers-site/index.min.js` file is **auto-generated** from `workers-site/index.js`.

**⚠️ DO NOT edit `index.min.js` directly.** Any changes will be overwritten.

To rebuild after modifying `index.js`:

```bash
bun run workers:minify
```

This runs esbuild with minification to produce the production-ready bundle.

### Workflow

```bash
mehr start <ref> | plan | implement | review | finish | continue | auto <ref> | interactive
```

Additional commands: `sync <task-id>`, `simplify`, `abandon`, `undo`, `redo`, `guide`, `status`, `list`, `note <msg>`, `question <msg>`, `browser`, `mcp`, `scan`, `serve`, `project plan|submit|start|sync`, `stack`, `config validate`, `agents`, `providers`, `templates`, `update check|install`, `generate-secret`, `cost`, `memory`, `links`, `find`, `review_pr`, `migrate_tokens`

**Question Command**: `mehr question <query>` (aliases: `ask`, `q`)
- Ask the agent a question during planning, implementing, or reviewing
- Does NOT change the workflow state - agent responds and work continues
- Useful for: understanding decisions, discussing alternatives, getting clarification
- Web UI: Quick Question input form + SSE streaming response

**Web UI Access**: Run `mehr serve` or navigate to the web interface at the configured port. Most workflow commands have Web UI equivalents. See "Multi-Interface Implementation" section above for parity status.

**Interactive Mode**: Use `mehr interactive` for CLI REPL mode or navigate to `/interactive` in the Web UI for real-time agent chat with workflow control.

**Interactive Mode Commands** (subset):
- Workflow: `start`, `plan`, `implement` (alias: `impl`), `review`, `finish`, `continue` (alias: `cont`), `abandon`
- Info: `status` (alias: `st`), `note`, `cost`, `list`, `specification` (alias: `spec`), `answer` (alias: `a`)
- Nav: `undo`, `redo`, `clear`, `help` (alias: `?`), `exit` (alias: `quit`, `q`)
- Chat: `chat <msg>`, `ask <msg>`
- Quick: `quick <desc>`

## Architecture

### Entry Point Flow

**CLI Path**: `cmd/mehr/main.go` → `commands.Execute()` → Cobra command handlers
**Interactive CLI Path**: `cmd/mehr/main.go` → `interactive` command → `REPL` → command dispatcher
**Web UI Path**: `cmd/mehr/main.go` → `serve` command → `internal/server/server.go` → HTTP handlers → templates
**Interactive Web Path**: `cmd/mehr/main.go` → `serve` → `/interactive` handler → SSE + HTMX

### Core Packages

| Package | Responsibility |
|---------|----------------|
| `internal/conductor/` | Main orchestrator (Facade) - combines workflow, storage, VCS, agents, browser, MCP |
| `internal/links/` | Bidirectional linking between entities using `[[reference]]` syntax with Logseq-style backlink tracking and graph queries |
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
| `internal/export/` | AI task plan output parsing into structured format |
| `internal/cost/` | ASCII chart generation for cost visualization |
| `internal/validation/` | Workspace configuration validation with error codes |
| `internal/project/` | Dependency graph generation for task visualization |
| `internal/display/` | Display formatting utilities (wraps go-toolkit display) |

### Key Patterns

**go-toolkit Integration**: Shared utilities live in `github.com/valksor/go-toolkit` for reuse across Valksor projects. **Always use go-toolkit packages directly** - do NOT create type aliases, wrapper functions, or re-exports. See the warning section above for detailed guidelines.

**State Machine**: The workflow package implements an explicit FSM:
- States: `idle` → `planning` → `implementing` → `reviewing` → `done`/`failed`
- Additional states: `waiting`, `checkpointing`, `reverting`, `restoring`
- Guard conditions control valid transitions
- Effects execute side-effects (git commits, file changes)

**Registry Pattern**: Providers and agents register themselves and are looked up by name/scheme at runtime.

**Event-Driven**: Components communicate via `events.Bus`, enabling loose coupling.

**Links System**: The `internal/links/` package provides Logseq-style bidirectional linking:
- **Reference syntax**: `[[spec:1]]`, `[[decision:cache-strategy]]`, `[[Authentication Spec]]`
- **Entity types**: `spec` (specifications), `note` (task notes), `session` (conversation logs), `decision` (recorded decisions)
- **Query operations**: `FindLinks()`, `FindBacklinks()`, `FindPath()`, `FindOrphans()`, `FindConnectedEntities()`
- **Storage**: JSON index at `~/.valksor/mehrhof/workspaces/<project>/links/` with atomic writes
- **Integration**: Automatically indexed when specs, notes, or sessions are saved
- **CLI/Web UI**: `mehr links` commands and `/links` page for browsing the knowledge graph

**Plugin System**: External agents and providers can be added via plugins. Plugins use JSON-RPC over stdio and are configured via `plugin.yaml` manifests.

**Plugin manifest structure:**
```yaml
name: my-provider
version: 1.0.0
type: provider
entry: ./bin/my-provider
```

See `internal/plugin/` for protocol details and registration.

### Web UI Architecture

The web UI uses Go's `html/template` package with:
- **HTMX** for real-time interactivity and SSE (Server-Sent Events)
- **Tailwind CSS** via CDN for styling with custom brand colors
- **Dark mode** via `class`-based toggle
- **SVG Workflow Diagram** at `/api/v1/workflow/diagram` - generates visual state diagram with current state highlighted

**Template Structure** (`internal/server/templates/`):
- `base.html` - Base layout with HTMX + Tailwind, dark mode support
- `login.html` - Authentication page
- `dashboard.html` - Main task dashboard with SSE streaming
- `project.html` - Project-specific task management view
- `history.html` - Session history and replay
- `browser.html` - Browser automation control panel
- `settings.html` - Workspace configuration management
- `quick.html` - Quick tasks page
- `license.html` - License information page
- `partials/` - Reusable template components (loaded via HTMX)
  - `actions.html` - Workflow action buttons
  - `active_work.html` - Current task/quick/project display
  - `costs.html` - Token usage and cost display
  - `question.html` - Agent question prompts
  - `specifications.html` - Specification list with progress
  - `stats.html` - Workspace statistics
  - `recent_tasks.html` - Recent tasks list
  - `labels.html` - Task labels
  - `task_card.html` - Task summary cards
- `partials/empty_states/` - Empty state displays
  - `no_task.html`, `no_stats.html`, `no_project.html`, `no_recent_tasks.html`

**Views Package** (`internal/server/views/`):
- `data.go` - View data structures for all pages
- `render.go` - Template rendering with type-safe methods
- `compute.go` - Data computation from conductor/storage
- `constants.go` - State displays, colors, SSE event names
- `format.go` - Formatting utilities (time, numbers, etc.)

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

**Agent Metadata**: Agents implement `MetadataProvider` to expose capabilities and models to the Web UI:
- Capabilities: `Streaming`, `ToolUse`, `FileOperations`, `CodeExecution`, `MultiTurn`, `SystemPrompt`, `AllowedTools`
- Models: `ID`, `Name`, `Default`, `MaxTokens`, `InputCost`, `OutputCost`

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

- **Multi-Interface**: ALL features must have both CLI and Web UI implementations (see "Multi-Interface Implementation" section)
- **Imports**: standard library → third-party → local (each group sorted alphabetically)
- **Naming**: PascalCase for exported, camelCase for unexported
- **Errors**: `fmt.Errorf("prefix: %w", err)` for wrapping; `errors.Join(errs...)` for multiple
- **Logging**: Use `log/slog`
- **Formatting**: Run `make fmt` (uses gofmt, goimports, gofumpt)
- **Linting/Quality**: Run `make quality` (runs golangci-lint, gofmt, goimports, gofumpt, govulncheck, check-alias)

### Modern Go Practices (Go 1.25+)

- Use `slices.Contains()`, `slices.Concat()`, `maps.Clone()` instead of manual loops
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done(); ... }()`
- Always pass `context.Context` for cancelable operations

## Testing

- **Run tests**: Use `make test` (runs all tests with coverage)
- **Coverage report**: Use `make coverage-html` (generates HTML report at `.coverage/coverage.html`)
- Tests use the standard `testing` package
- Table-driven tests preferred: `tests := []struct{...}{...}`
- Test utilities in `internal/helper_test/` (mocks, fixtures, conductor helpers)
- Target 80%+ coverage

## See Also

- [README.md](README.md) - User-facing documentation, installation, quick start
- [Documentation](https://valksor.com/docs/mehrhof/nightly) - Full guides and API reference
