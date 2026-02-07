# Standalone Simplify

Simplify code changes without an active task using the Web UI API.

## Overview

The standalone simplify feature allows you to run AI-powered code simplification on:

- **Uncommitted changes** - Staged and unstaged changes in your working directory
- **Branch comparisons** - Current branch vs main/master/develop
- **Commit ranges** - Specific commit ranges (e.g., `HEAD~3..HEAD`)
- **Specific files** - Simplify only certain files

This is useful for quick refactoring, cleaning up code before commits, or improving code quality in feature branches.

## API Endpoint

```
POST /api/v1/workflow/simplify/standalone
```

## Request

### Headers

| Header         | Value               | Description                      |
|----------------|---------------------|----------------------------------|
| `Content-Type` | `application/json`  | Required                         |
| `Accept`       | `text/event-stream` | Optional - enables SSE streaming |

### Body

```json
{
  "mode": "uncommitted",
  "base_branch": "",
  "range": "",
  "files": [],
  "context": 3,
  "agent": "",
  "create_checkpoint": true
}
```

### Fields

| Field               | Type     | Required | Default       | Description                                                                       |
|---------------------|----------|----------|---------------|-----------------------------------------------------------------------------------|
| `mode`              | string   | No       | `uncommitted` | Simplify mode: `uncommitted`, `branch`, `range`, `files`                          |
| `base_branch`       | string   | No       | auto-detect   | Base branch for `branch` mode                                                     |
| `range`             | string   | No       | -             | Commit range for `range` mode (e.g., `HEAD~3..HEAD`)                              |
| `files`             | string[] | No       | -             | Files to simplify for `files` mode                                                |
| `context`           | int      | No       | 3             | Lines of context in diff                                                          |
| `agent`             | string   | No       | default       | Agent to use for simplification (uses `simplifying` step config if not specified) |
| `create_checkpoint` | bool     | No       | true          | Create a git checkpoint before changes                                            |

## Response

### Synchronous Response

```json
{
  "success": true,
  "summary": "Refactored handler functions to reduce complexity and improve readability.",
  "changes": [
    {
      "path": "internal/handler.go",
      "operation": "modify"
    },
    {
      "path": "internal/validation.go",
      "operation": "modify"
    }
  ],
  "usage": {
    "input_tokens": 2345,
    "output_tokens": 890,
    "cached_tokens": 200,
    "cost_usd": 0.0067
  }
}
```

### SSE Streaming Response

When `Accept: text/event-stream` is set, the response streams events:

```
event: message
data: {"event":"progress","message":"Gathering diff..."}

event: message
data: {"event":"content","text":"Simplifying code..."}

event: message
data: {"event":"result","data":{"success":true,"summary":"...",...}}

event: message
data: {"event":"done"}
```

## Simplification Modes

| Mode                | Description                                               |
|---------------------|-----------------------------------------------------------|
| Uncommitted changes | Simplifies changes not yet committed                      |
| Branch vs base      | Simplifies the difference between current branch and base |
| Commit range        | Simplifies code within a specific commit range            |
| Specific files      | Simplifies only the specified files                       |

For API integration examples, see [REST API Reference](/reference/rest-api.md).

## File Operations

| Operation | Description               |
|-----------|---------------------------|
| `modify`  | File content was modified |
| `create`  | New file was created      |
| `delete`  | File was removed          |
| `rename`  | File was renamed          |

## Safety

### Automatic Checkpoints

By default, simplification creates a git checkpoint before modifying files. This allows you to undo changes using the **Undo** button or git commands.

### Recovery

If simplification produces unwanted results, you can recover using the **Undo** button to restore from a checkpoint.

## Error Response

```json
{
  "success": false,
  "error": "nothing to simplify: no changes found"
}
```

---

## Also Available via CLI

Run standalone code simplification from the command line for scripting or terminal workflows.

See [CLI: simplify](/cli/simplify.md) for all modes, agent selection, and output options.

## See Also

- [CLI: simplify](/cli/simplify.md) - Command-line simplification
- [Standalone Review](standalone-review.md) - Standalone code review
- [Undo & Redo](undo-redo.md) - Recovering from changes
