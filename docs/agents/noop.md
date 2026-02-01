# Noop Agent

> **Testing Only**
>
> The noop agent is for CI/testing environments only. It performs no actual AI operations and should never be used for production tasks.

The noop agent is a no-operation agent that's automatically registered when `MEHR_TEST_MODE=1` is set. It allows smoke tests and integration tests to run without requiring actual AI agents like Claude.

## When It's Available

The noop agent is **only registered** when the `MEHR_TEST_MODE` environment variable is set to `1`:

```bash
MEHR_TEST_MODE=1 mehr serve
```

This happens automatically in:
- Web UI smoke tests (Playwright)
- Integration tests that spawn the server

## Behavior

| Method | Behavior |
|--------|----------|
| `Available()` | Always returns `nil` (always available) |
| `Run()` | Returns empty response with "noop agent" message |
| `RunStream()` | Sends single `EventComplete` event |
| `RunWithCallback()` | Calls callback once with completion event |

## Why It Exists

In CI environments:
1. No AI CLI tools (claude, codex) are installed
2. Agent auto-detection fails with "no available agents found"
3. Server initialization fails, breaking smoke tests

The noop agent provides a fallback that passes availability checks, allowing the server to start for UI testing.

## Not For Production

The noop agent:
- Does not connect to any AI service
- Does not generate meaningful responses
- Does not execute any workflow logic

If you see "noop agent" in production logs, check that `MEHR_TEST_MODE` is not set in your environment.

## Implementation

The agent is implemented in `internal/agent/noop/noop.go` and registered in `internal/registration/standard.go` when test mode is detected.

## See Also

- [Claude Agent](claude.md) - Primary supported agent
- [Codex Agent](codex.md) - Alternative agent
- [Agent Configuration](index.md) - Full agent documentation
