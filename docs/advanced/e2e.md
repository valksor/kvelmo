# E2E Testing

Mehrhof includes fast end-to-end tests that validate the core AI workflow using your local configuration.

## Overview

The fast E2E suite tests the complete workflow:
- **Start** a task from a file
- **Plan** - generate specifications
- **Implement** - apply changes
- **Review** - review implemented code
- **Finish** - complete the task

Tests run in temporary directories using your local `.mehrhof/config.yaml` and `.mehrhof/.env`.

## Prerequisites

```bash
# Required
export ZAI_API_KEY="your-zai-key"  # For glm agent
which claude                        # Claude CLI must be in PATH

# Optional - uses your local config
ls .mehrhof/config.yaml             # Your workspace config
ls .mehrhof/.env                    # Contains ZAI_API_KEY
```

## Running Tests

```bash
# Check prerequisites
make e2e-check

# Run fast E2E tests (~10 minutes)
make e2e
# or
make e2e-fast
```

## Test Coverage

| Test | Description | Duration |
|------|-------------|----------|
| `TestHappyPath` | Full workflow: start → plan → implement → review → finish | ~10 min |
| `TestBasicCommands` | Commands that don't need agent (version, help, init, etc.) | ~1 min |
| `TestStartAndPlan` | Start task and run plan | ~5 min |
| `TestImplementDryRun` | Implement with --dry-run flag | ~8 min |

## What's Validated

- ✅ CLI argument parsing
- ✅ Workspace initialization
- ✅ Local config loading
- ✅ Agent invocation (glm via ZAI)
- ✅ Planning phase output
- ✅ Implementation phase (file modifications)
- ✅ Review phase
- ✅ Finish phase
- ✅ Basic commands

## What's Not Tested

The fast E2E suite intentionally skips:
- Git operations (branches, commits, worktrees)
- GitHub provider integration
- Pull request creation
- Browser automation
- Project workflows
- Checkpoint/undo/redo

These are planned for the comprehensive `e2e-full` suite.

## Isolation

Each test:
1. Creates a temporary directory via `t.TempDir()`
2. Copies your local `.mehrhof/config.yaml` and `.env`
3. Runs commands in isolation
4. Auto-cleans on completion

No git repository is created/modified. No GitHub API calls are made.

## Adding Tests

Tests are simple Go functions in `e2e/fast/`:

```go
//go:build e2e_fast
// +build e2e_fast

package fast

func TestMyFeature(t *testing.T) {
    dir := t.TempDir()
    h := NewHelper(t, dir)

    h.InitWithLocalConfig()
    h.WriteTask("task.md", "---\ntitle: Test\n---\nDo something")

    h.Run("start", "file:task.md", "--no-branch")
    h.AssertSuccess()
}
```

See `e2e/fast/e2e_test.go` for examples.
