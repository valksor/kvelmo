# Reviewing

The review phase runs automated quality checks on your code.

## What Reviewing Does

When you click **"Review"**, Mehrhof:

1. **Runs quality checks** - Executes configured quality tools (linters, formatters)
2. **Analyzes code** - Checks for common issues and patterns
3. **Generates report** - Creates a review summary with findings
4. **Saves review** - Stores results in the task's review directory

## Starting Review

After implementation completes, click the **"Review"** button:

```
┌──────────────────────────────────────────────────────────────┐
│  Active Task: Add User OAuth Authentication                   │
├──────────────────────────────────────────────────────────────┤
│  State: ● Idle                                                │
│  Changes: 5 files modified                                    │
│                                                              │
│  Actions:                                                    │
│    [Plan] [Implement] [Review] [Finish] [Continue]           │
│                                                              │
│  [Review] ← Click this button                                │
└──────────────────────────────────────────────────────────────┘
```

## Review Phase Workflow

```mermaid
flowchart LR
    A[Idle + Code Ready] --> B[Click Review]
    B --> C[Reviewing State]
    C --> D[Quality Checks Run]
    D --> E[Report Generated]
    E --> F[Back to Idle - Review Complete]
```

## Real-Time Progress

Watch the review in the **Agent Output** section:

```
┌──────────────────────────────────────────────────────────────┐
│  Agent Output (Live)                                          │
├──────────────────────────────────────────────────────────────┤
│  $ Running quality checks...                                  │
│  ✓ go fmt - checked                                          │
│  ✓ go vet - passed                                           │
│  ✓ staticcheck - no issues found                             │
│  ✓ golangci-lint - passed                                    │
│  ✓ go test - all tests passing                               │
│                                                              │
│  Review complete: No issues found                            │
│  ▶ Streaming...                                               │
└──────────────────────────────────────────────────────────────┘
```

## The Reviewing State

During review, the task state changes to **"Reviewing"**:

| State | What's Happening | What You Can Do |
|-------|------------------|-----------------|
| **Reviewing** | Quality checks running | Watch progress, wait for completion |
| **Idle** | Review complete | Check findings, proceed to finish |

## Review Results

After review completes, you'll see a summary:

```
┌──────────────────────────────────────────────────────────────┐
│  Review Summary                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Status: ✅ Passed                                           │
│                                                              │
│  Checks Run: 5                                               │
│    ✓ gofmt - Code formatting                                 │
│    ✓ go vet - Static analysis                                │
│    ✓ staticcheck - Additional checks                         │
│    ✓ golangci-lint - Full lint                               │
│    ✓ go test - Unit tests                                    │
│                                                              │
│  Issues Found: 0                                             │
│                                                              │
│  Files Analyzed: 5                                           │
│  Test Coverage: 94.2%                                         │
│                                                              │
│  [View Full Report]                                          │
└──────────────────────────────────────────────────────────────┘
```

## When Issues Are Found

If the review finds issues:

```
┌──────────────────────────────────────────────────────────────┐
│  Review Summary                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Status: ⚠️ Issues Found                                     │
│                                                              │
│  Issues Found: 3                                             │
│                                                              │
│  🔴 Error: missing-return in func AuthMiddleware            │
│     File: internal/auth/middleware.go:45                     │
│                                                              │
│  🟡 Warning: exported function should have comment           │
│     File: internal/auth/oauth.go:23                          │
│                                                              │
│  🟡 Warning: unused variable 'sessionID'                     │
│     File: internal/auth/handlers.go:67                       │
│                                                              │
│  [View Full Report] [Fix Issues] [Ignore]                    │
└──────────────────────────────────────────────────────────────┘
```

### Options

- **Fix Issues** - Add a note with the issues and implement again
- **Ignore** - Proceed to finish (not recommended for errors)
- **Undo** - Go back and make corrections manually

## Configured Quality Tools

The review runs tools configured in your project. Common tools:

| Tool | What It Checks |
|------|----------------|
| **gofmt** | Code formatting |
| **go vet** | Static analysis |
| **staticcheck** | Additional static checks |
| **golangci-lint** | Comprehensive linting |
| **go test** | Unit tests |
| **gosec** | Security issues |
| **gitleaks** | Secret leaks |

Configure quality tools in your `Makefile` or project settings.

## Review Best Practices

1. **Always review** - Check findings before finishing
2. **Fix errors** - Don't finish with error-level issues
3. **Consider warnings** - Evaluate if warnings need fixing
4. **Check tests** - Ensure all tests pass
5. **Verify coverage** - Maintain good test coverage

## Next Steps

After review completes:

- [**Finishing**](finishing.md) - Complete and merge the task
- [**Implementing**](implementing.md) - Fix issues and implement again
- [**Undo & Redo**](undo-redo.md) - Navigate checkpoints if needed

## CLI Equivalent

```bash
# Run review
mehr review

# View review output
mehr review --verbose

# Continue after review
mehr continue
```

See [CLI: review](../cli/review.md) for all options.
