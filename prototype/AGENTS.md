# AGENTS.md / CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mehrhof is a **Go CLI tool + Web UI** for AI-powered task automation. It orchestrates AI agents (primarily Claude) to perform planning, implementation, and code review workflows with checkpointing, parallel task support, and multi-provider integrations.

**⚠️ ALL features must be implemented for BOTH CLI and Web UI interfaces.** See the "Dual Interface Implementation" section below.

---

## ⚠️ CRITICAL: go-toolkit Usage Guidelines

**DO NOT re-export or wrap go-toolkit functionality unnecessarily.**

go-mehrhof shares code with `github.com/valksor/go-toolkit` for reuse across Valksor projects. The purpose of go-toolkit is to **eliminate duplication**, not create additional abstraction layers.

### What NOT to Do:

```go
// ❌ BAD - Type alias re-export
type Bus = eventbus.Bus
type Result = validate.Result
type Request = jsonrpc.Request

// ❌ BAD - Wrapper function
func Slugify(title string, maxLen int) string {
    return slug.Slugify(title, maxLen)
}

// ❌ BAD - Constructor wrapper
func NewBus() *Bus {
    return eventbus.NewBus()
}

// ❌ BAD - Variable re-export
var NewResult = validate.NewResult
```

### What to Do Instead:

```go
// ✅ GOOD - Import and use go-toolkit directly
import "github.com/valksor/go-toolkit/eventbus"

bus := eventbus.NewBus()

// ✅ GOOD - Domain-specific types that add value
type AgentConfig struct {
    Name        string
    Description string
    // ... mehrhof-specific agent configuration
}

// ✅ GOOD - Domain-specific functions with business logic
func ColorState(state, displayName string) string {
    // Maps mehrhof's workflow states to colors
    // This is domain-specific, not a simple wrapper
}
```

### When to Add Code to go-toolkit vs. go-mehrhof:

| Criteria | go-toolkit | go-mehrhof |
|----------|-----------|------------|
| Generic, reusable utilities? | ✅ Yes | ❌ No |
| Domain-specific business logic? | ❌ No | ✅ Yes |
| No dependencies on mehrhof internals? | ✅ Yes | ❌ No |
| Could be used by other Valksor projects? | ✅ Yes | ❌ No |

### Examples of Correct Usage:

- ✅ **eventbus**: Use `eventbus.Bus`, `eventbus.NewBus()` directly
- ✅ **validate**: Use `validate.Result`, `validate.NewResult()`, `validate.SeverityError` directly
- ✅ **jsonrpc**: Use `jsonrpc.Request`, `jsonrpc.Response`, `jsonrpc.NewRequest()` directly
- ✅ **slug**: Use `slug.Slugify()` directly
- ✅ **display colors**: Keep `ColorState()`, `ColorSpecStatus()` (domain-specific business logic)

### Enforcement:

- CI runs `make check-alias` to detect unnecessary import aliases
- Code review should flag any new type aliases or wrapper functions
- When in doubt, use go-toolkit directly

---

## ⚠️ CRITICAL: Dual Interface Implementation - CLI + Web UI

**ALL features must be implemented for BOTH CLI and Web UI unless explicitly CLI-only.**

Mehrhof has two user interfaces that must maintain feature parity:
1. **CLI** - Command-line interface via `cmd/mehr/commands/`
2. **Web UI** - Web interface via `internal/server/`

### Implementation Checklist

When adding a new feature, complete ALL applicable items:

- [ ] **CLI Command**: Add command in `cmd/mehr/commands/*.go` using Cobra
- [ ] **Web UI Handler**: Add handler in `internal/server/handlers*.go` or `internal/server/api/`
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

    // Stream events as operation progresses
    fmt.Fprintf(w, "event: status\ndata: {\"message\": \"Starting planning...\"}\n\n")
    flusher.Flush()

    // ... execute operation ...

    fmt.Fprintf(w, "event: complete\ndata: {\"success\": true}\n\n")
    flusher.Flush()
}
```

### Current Feature Parity Gaps

These CLI commands **lack Web UI equivalents** (candidates for future implementation):

| CLI Command | Web UI Status |
|-------------|---------------|
| `interactive` | ✅ Full REPL at `/interactive` |
| `budget status/set/task set/resume/reset` | ❌ Missing - only basic stats in dashboard |
| `memory search/index/stats` | ⚠️ Partial - API exists, no UI |
| `cost` (detailed reporting) | ⚠️ Partial - basic cost tracking only |
| `continue` | ✅ Available in interactive mode |
| `optimize` | ❌ Missing |
| `export` | ❌ Missing |
| `scan` | ⚠️ Partial - API endpoint exists, no UI |

### When CLI-Only Is Appropriate

Some commands are intentionally CLI-only:

- **One-shot operations**: `generate-secret`, `update check/install`
- **Developer utilities**: `hooks`, `lefthook`, `config validate`
- **Debugging/diagnostic**: `status --diagram`, `cost --breakdown`

If a feature is CLI-only, document the rationale in code comments.

### Verification

Before considering a feature "done":

1. Test both CLI and Web UI implementations
2. Verify error handling works for both interfaces
3. Check that CLI flags map to Web UI form inputs appropriately
4. Ensure SSE streaming works for long-running operations
5. Update feature parity table above if adding new dual-interface features

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
2. **Use table-driven tests** for functions with multiple input scenarios:
   ```go
   tests := []struct {
       name    string
       input   string
       want    string
       wantErr bool
   }{
       {"valid input", "foo", "bar", false},
       {"empty input", "", "", true},
   }
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           // test logic
       })
   }
   ```
3. **Test utilities** are available in `internal/helper_test/` (mocks, fixtures, conductor helpers)
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
- **New workflow state** is added (update Workflow States table)

### Examples

#### ✅ GOOD: Complete Feature Delivery

```
Added budget feature:
✅ cmd/mehr/commands/budget.go               # CLI command
✅ cmd/mehr/commands/budget_test.go           # Unit tests (85% coverage)
✅ internal/server/handlers_budget.go        # Web UI handler
✅ internal/server/handlers_budget_test.go   # Handler tests
✅ internal/server/templates/budget.html     # UI template
✅ docs/cli/budget.md                         # CLI documentation
✅ docs/web-ui/budget.md                      # Web UI documentation
✅ README.md                                  # Updated feature list
```

#### ❌ BAD: Incomplete Feature Delivery

```
Added budget feature:
✅ cmd/mehr/commands/budget.go               # CLI command
❌ No tests
❌ No Web UI implementation
❌ No documentation
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
| `docs/guides/` | Procedural guides (should be split by interface if interface-specific) |
| `docs/configuration/` | Shared configuration concepts (interface-specific config should be in `docs/cli/` or `docs/web-ui/`) |
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

The ONLY situations where a single document may cover both CLI and Web UI:

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

docs/cli/first-task.md        # CLI first task walkthrough
docs/web-ui/getting-started.md # Web UI getting started
```

### When Creating New Documentation

1. **Identify the interface**: Is this for CLI, Web UI, or both?
2. **Choose the right directory**:
   - CLI-only → `docs/cli/`
   - Web UI-only → `docs/web-ui/`
   - Interface-agnostic concept → `docs/concepts/`
3. **Check for existing related docs** and add cross-references
4. **Avoid duplication**: If a feature works identically in both interfaces, document in `docs/concepts/` and reference from both interface-specific docs

### Verification

Before submitting documentation changes:

1. Confirm the file is in the correct directory (`docs/cli/`, `docs/web-ui/`, or appropriate conceptual dir)
2. Verify content covers only ONE interface (unless it's a comparison/parity doc)
3. Check for cross-references to the equivalent interface's documentation
4. Run `make docs` (if available) to verify documentation builds correctly

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

1. **Before starting work**: Run `make quality && make test` once to verify baseline
   - If anything fails, FIX IT FIRST
   - Do not add new code on top of broken code

2. **Before committing code changes**: Run `make quality && make test`
   - Commit gatekeeper for code changes: nothing gets committed if these fail
   - For docs-only changes, skip this step

3. **Before PR**: Run `make quality && make test`
   - PR gatekeeper: nothing gets submitted if these fail
   - Only needed if PR includes code changes

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

#### ❌ BAD: Leave Someone Else to Fix It

```bash
$ make quality
internal/foo/bar.go:123:1: goimports: missing import
exit status 1

# WRONG: "Someone else's mess, let them fix it"
$ git commit -m "fix typo"
```

### Philosophy

**You are responsible for the health of the codebase.**

- Professionalism means fixing problems, not ignoring them
- A 5-minute fix today saves hours of debugging tomorrow
- CI will catch it anyway—fix it locally first
- "Not mine" is not an acceptable answer

### Escalation

If you encounter a test failure or quality issue you truly cannot fix:

1. **Document it**: Open an issue with reproduction steps
2. **Block the codebase**: Do not merge anything until it's resolved
3. **Communicate**: Alert the team immediately
4. **DO NOT work around it**: Broken code blocks everything

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

### What NOT to Do:

```bash
# ❌ BAD - Direct go commands
go build ./cmd/mehr
go fmt ./...
go test ./...
golangci-lint run
```

### What to Do Instead:

```bash
# ✅ GOOD - Use make commands
make build
make fmt
make test
make quality
```

### Why Use Make?

1. **Consistency**: All developers use the same commands
2. **Multi-step operations**: `make quality` runs linter, formatter, vuln check, and alias check in one command
3. **Embedded assets**: `make build` includes license generation and asset bundling
4. **Version injection**: Build metadata (version, commit, build time) is automatically injected

### When to Use These Commands

See the "When to Run Build, Test, and Quality Commands" section below for guidance on when to run these commands. In short:

- **Code changes**: Run `make build`, `make test`, `make quality`
- **Documentation-only changes**: Skip these commands
- **Baseline verification**: Run `make quality && make test` once per session before starting work

---

## ⚠️ CRITICAL: When to Run Build, Test, and Quality Commands

**ONLY run build, test, and quality commands when CODE changes are involved.**

Build, test, and quality commands verify the CODEBASE. Running them after documentation-only changes, git operations, or other non-code changes is wasteful and illogical.

### When to Run These Commands

| Command | Run When... | Do NOT Run When... |
|---------|-------------|-------------------|
| `make build` | You've modified `.go` files | Documentation-only changes |
| `make test` | You've modified `.go` or `*_test.go` files | Documentation-only changes |
| `make quality` | You've modified `.go` files | Documentation-only changes |
| `make fmt` | You've modified `.go` files | Documentation-only changes |
| `make coverage-html` | You've added/modified tests | Documentation-only changes |

### Code vs. Non-Code Changes

**Code changes** (run build/test/quality):
- Modified `.go` source files
- Added/removed imports
- Changed package structure
- Modified test files

**Non-code changes** (DO NOT run build/test/quality):
- Documentation updates (`*.md`, `docs/`)
- Git operations (commits, merges, rebases)
- Configuration changes (`.yaml`, `.toml`, `.json`)
- Asset changes (images, templates, scripts)
- Comments-only changes to `.go` files

### Workflow Decision Tree

```
Did you change any .go files?
├── YES → Run `make quality && make test`
│       └── Did tests pass? → Proceed with commit/PR
│       └── Did tests fail? → Fix before committing
└── NO → Skip build/test/quality (docs, config, git ops only)
```

### Examples

#### ✅ GOOD: Code Changes

```bash
# Modified internal/conductor/conductor.go
$ make quality && make test
# All pass
$ git commit -m "fix: handle nil conductor"
```

#### ✅ GOOD: Documentation-Only Changes

```bash
# Updated docs/cli/scan.md
# NO need to run make quality or make test
$ git commit -m "docs: improve scan command documentation"
```

#### ❌ BAD: Unnecessary Commands

```bash
# Only changed CLAUDE.md
$ make quality && make test  # WASTEFUL - no code changed
$ git commit -m "docs: clarify nolint usage"
```

### Exception: Verify Health Before Starting Work

**Always run `make quality && make test` once before starting new work**, regardless of what you're doing. This ensures you're not building on top of already-broken code.

```bash
# Before starting work (run once per session)
$ make quality && make test

# Now do your work...
# If only docs changed: commit without running again
# If code changed: run again before committing
```

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

4. **Prefer code fixes over nolint**: Refactor to avoid the warning
   ```go
   // ❌ BAD - Using nolint for nil check pattern
   //nolint:nilnil
   func getTask(id string) (*Task, error) {
       return nil, nil // "not found" is not an error
   }

   // ✅ GOOD - Use a sentinel value or custom error type
   var ErrNotFound = errors.New("task not found")

   func getTask(id string) (*Task, error) {
       return nil, ErrNotFound
   }
   ```

### Enforcement

The `nolintlint` linter is enabled to catch:
- `//nolint` without specifying which linter
- `//nolint` without an explanation
- Machine-generated `//nolint` comments

CI will fail if:
- New `//nolint` comments don't specify the linter
- New `//nolint` comments lack justification
- Linters are disabled in `.golangci.yml`

### Verification

Before committing code with `//nolint`:

1. **Is the linter correct?** Verify the warning is a false positive
2. **Can you fix the code?** Prefer refactoring over suppression
3. **Is the nolint specific?** Specify the exact linter name
4. **Is there a justification?** Document WHY the suppression exists

If you cannot justify the `//nolint`, **fix the code instead.**

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

Additional commands: `sync <task-id>`, `simplify`, `abandon`, `undo`, `redo`, `guide`, `status`, `list`, `note <msg>`, `question <msg>`, `browser`, `mcp`, `scan`, `serve`, `project plan|submit`, `config validate`, `agents`, `providers`, `templates`, `update check|install`, `generate-secret`, `cost`, `memory`, `review_pr`, `migrate_tokens`

**Question Command**: `mehr question <query>` (aliases: `ask`, `q`)
- Ask the agent a question during planning, implementing, or reviewing
- Does NOT change the workflow state - agent responds and work continues
- Useful for: understanding decisions, discussing alternatives, getting clarification
- Web UI: Quick Question input form + SSE streaming response

**Web UI Access**: Run `mehr serve` or navigate to the web interface at the configured port. Most workflow commands have Web UI equivalents. See "Dual Interface Implementation" section above for parity status.

**Interactive Mode**: Use `mehr interactive` for CLI REPL mode or navigate to `/interactive` in the Web UI for real-time agent chat with workflow control.

## Architecture

### Entry Point Flow

**CLI Path**: `cmd/mehr/main.go` → `commands.Execute()` → Cobra command handlers
**Web UI Path**: `cmd/mehr/main.go` → `serve` command → `internal/server/server.go` → HTTP handlers

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

- **Dual Interface**: ALL features must have both CLI and Web UI implementations (see "Dual Interface Implementation" section)
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
