# Contributing

Thank you for your interest in contributing to Mehrhof! This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

- **Go 1.25+** - Required for building from source
- **Git** – Required for version control operations
- **make** – For build automation

### Getting Started

```bash
# Clone the repository
git clone https://github.com/valksor/go-mehrhof.git
cd go-mehrhof

# Download dependencies
make deps

# Build the binary
make build

# Install to $GOPATH/bin (optional)
make install
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage and race detection
make coverage

# Generate HTML coverage report
make coverage-html
```

### Code Quality

```bash
# Run linters and formatters (golangci-lint, gofmt, goimports, gofumpt, check-alias)
make quality

# Format code only
make fmt

# Tidy dependencies
make tidy

# Run quality checks for IDE plugins
make ide-quality        # Both VS Code and JetBrains
make vscode-quality     # VS Code extension only
make jetbrains-quality  # JetBrains plugin only
```

**Run checks for what you changed:**
- Go code (`cmd/`, `internal/`, `*.go`): `make quality`
- VS Code extension (`ide/vscode/`): `cd ide/vscode && make quality`
- JetBrains plugin (`ide/jetbrains/`): `cd ide/jetbrains && make quality`
- Docs only (`docs/`, `*.md`): No checks required

## Code Style

### Import Order

Imports should be grouped in the following order, with each group sorted alphabetically:

1. Standard library
2. Third-party packages
3. Local packages

```go
import (
    "fmt"           // standard library
    "os"            // standard library

    "github.com/spf13/cobra"  // third-party

    "github.com/valksor/go-mehrhof/internal/conductor"  // local
)
```

### Naming Conventions

- **Exported**: PascalCase (`MyFunction`, `MyType`)
- **Unexported**: camelCase (`myFunction`, `myType`)
- **Constants**: PascalCase (`MaxRetries`)
- **Interfaces**: Usually PascalCase, often ending with `-er` suffix (`Reader`, `Writer`)

### Error Handling

Always handle errors explicitly:

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

For multiple independent errors, use `errors.Join`:

```go
var errs []error
if err := op1(); err != nil {
    errs = append(errs, fmt.Errorf("op1: %w", err))
}
if err := op2(); err != nil {
    errs = append(errs, fmt.Errorf("op2: %w", err))
}
if len(errs) > 0 {
    return errors.Join(errs...)
}
```

### Modern Go Practices (Go 1.25+)

- Use `errors.Join()` for aggregating multiple errors
- Use `slices.Contains()`, `slices.Concat()`, `maps.Clone()` from stdlib
- Use `log/slog` for structured logging
- Always pass `context.Context` for cancelable operations
- Use `wg.Go(func() { ... })` instead of `wg.Add(1); go func() { defer wg.Done() }()`

## Critical Rules

These rules are enforced by CI. Violating them will cause your PR to fail.

### Multi-Interface Parity

**Every feature needs both CLI and Web UI implementations.** Shared logic goes in `internal/conductor/`. Both interfaces call the same conductor methods.

See [docs/reference/feature-parity.md](docs/reference/feature-parity.md) for the implementation checklist and status tables.

### go-toolkit Import Policy

Import `github.com/valksor/go-toolkit` packages directly. **No type aliases, no wrappers, no re-exports.**

```go
// ✅ GOOD - Direct import
import "github.com/valksor/go-toolkit/eventbus"
bus := eventbus.NewBus()

// ❌ BAD - Type alias or wrapper
type Bus = eventbus.Bus  // Don't do this
```

CI enforces this via `make check-alias`.

### No nolint Abuse

`//nolint` is a last resort. Never disable linters globally in `.golangci.yml`.

**Acceptable** (with justification):
```go
//nolint:unparam // Required by interface
//nolint:nilnil // No task found is not an error
//nolint:errcheck // String builder WriteString won't fail
```

**Never acceptable:**
- `//nolint:errcheck` without justification
- `//nolint:gosec` (fix the security issue)
- `//nolint:all` (never suppress all linters)

Always: specify linter name, include justification, place on specific line.

### File Size Limit

Keep all Go files under **500 lines**. Split by feature or responsibility:

```go
// Split handlers.go (1000 lines) into:
handlers_plan.go      // Planning handlers
handlers_implement.go // Implementation handlers
handlers_review.go    // Review handlers
```

## Testing

### Test-First Development

**Write tests FIRST (TDD).** This ensures your implementation meets requirements and catches regressions.

### Testing Strategy

**During development:** Run targeted tests for changed packages:
```bash
go test ./internal/storage/...           # Test a package
go test -run TestWorkspace ./internal/... # Test specific function
```

**Before committing:** Run the full test suite:
```bash
make test  # Only after implementation is complete
```

### Conventions

- Use the standard `testing` package
- Prefer table-driven tests for multiple cases
- Target 80%+ code coverage
- Place test files next to the code they test (`foo_test.go`)
- Shared test utilities are in `internal/helper_test/`

### Table-Driven Test Example

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "test", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Parse() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### E2E Tests

For changes affecting the core workflow, run fast E2E tests locally:

```bash
# Check prerequisites (ZAI_API_KEY, claude CLI)
make e2e-check

# Run fast E2E tests (~10 min)
make e2e
```

E2E tests validate the full workflow (start → plan → implement → review → finish) using your local configuration. See [E2E Testing](https://valksor.com/docs/mehrhof/nightly/#/advanced/e2e) for details.

## Pull Request Process

### Before Submitting

1. **Format your code**: `make fmt`
2. **Run linters**: `make quality` (fix any issues)
3. **Run tests**: `make test` (ensure all pass)
4. **Update documentation**: If adding features, update relevant docs

### Commit Messages

Follow clear, descriptive commit messages:

```
Add support for GitLab provider

- Implement GitLab API client
- Add issue parsing and MR creation
- Update documentation with examples
```

### Branch Naming

Use descriptive branch names:

- `feature/add-jira-provider`
- `fix/agent-session-restore`
- `docs/update-readme`

### Submitting a PR

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes and commit
4. Push to your fork
5. Open a pull request with a clear description

Your PR should:
- Pass all CI checks
- Include tests for new functionality
- Update documentation if needed
- Reference any related issues

### Review Process

Maintainers will review your PR and provide feedback. Please address review comments promptly. Once approved, your PR will be merged.

## Reporting Issues

### Bug Reports

Before submitting a bug report, please verify the issue against different versions:

1. **Check the latest release** - Confirm the bug exists in the latest stable release
2. **Check the nightly build** - Test against the latest nightly build to see if it's already fixed
3. **Decide whether to report**:
   - ✅ **Submit an issue** if: Bug exists in both release AND nightly
   - ✅ **Submit an issue** if: Bug exists ONLY in nightly (newly introduced)
   - ❌ **Do NOT submit** if: Bug exists only in release but is FIXED in nightly

### When Submitting Issues

1. **Search existing issues** first to avoid duplicates
2. **Provide clear details**:
   - Version tested (release version and/or nightly build date)
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version)
   - Relevant logs or error messages

### Feature Requests

For feature requests, please:
- Describe the use case clearly
- Explain why existing functionality doesn't meet your needs
- Consider if this would benefit most users or is specific to your workflow

## Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Documentation**: See the [docs/](docs/) directory for detailed guides

## Reference

For deeper context on project architecture and conventions:

- [CLAUDE.md](CLAUDE.md) — Complete project conventions and architecture overview
- [REFERENCE.md](REFERENCE.md) — Command, API, and package reference
- [docs/reference/feature-parity.md](docs/reference/feature-parity.md) — Interface parity checklist

## Documentation

### Documentation by Interface

Documentation is organized by interface with specific tone and content requirements:

| Directory | Audience | Tone |
|-----------|----------|------|
| `docs/web-ui/` | Non-technical users | Professional, accessible, corporate |
| `docs/cli/` | CLI-savvy developers | Technical, concise |
| `docs/ide/` | IDE users | Visual, task-oriented |
| `docs/concepts/` | All users | Accessible without being condescending |
| `docs/reference/` | Developers/integrators | Technical, comprehensive |

**Web UI docs (`docs/web-ui/`):**
- NO bash commands or CLI usage
- Use button names, screenshots, visual workflows

**CLI docs (`docs/cli/`):**
- Command syntax, flags, examples, exit codes
- Brief cross-reference to Web UI equivalent

**Link format:** Always use absolute paths (Docsify breaks relative links):
```markdown
# ✅ GOOD
See [CLI: note](/cli/note.md) for details.

# ❌ BAD - Breaks in Docsify
See [CLI: note](../cli/note.md) for details.
```

### Adding Diagrams

For diagrams in documentation, use **Mermaid** syntax but export to static images to avoid loading a 3MB JavaScript library.

**Generate a PNG from Mermaid:**

```bash
cat <<'EOF' | bunx @mermaid-js/mermaid-cli mmdc -i - -o docs/_media/img/diagram.png -b transparent
stateDiagram-v2
    [*] --> StateA
    StateA --> StateB
EOF
```

**Reference in markdown:**

```markdown
![Diagram Title](../_media/img/diagram.png)
```

**Why PNG instead of SVG?**
- Mermaid's SVG output uses `<foreignObject>` for text labels
- Browsers block `foreignObject` in SVGs loaded via `<img>` tags (security policy)
- PNG rasterization bypasses this limitation and renders consistently

**Alternative: Use Mermaid Live Editor**
- Visit [mermaid.live](https://mermaid.live)
- Write your diagram
- Export as PNG/SVG manually

## License

By contributing to Mehrhof, you agree that your contributions will be licensed under the [BSD 3-Clause License](https://github.com/valksor/go-mehrhof/blob/master/LICENSE).
