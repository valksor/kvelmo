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

| Header | Value | Description |
|--------|-------|-------------|
| `Content-Type` | `application/json` | Required |
| `Accept` | `text/event-stream` | Optional - enables SSE streaming |

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

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `mode` | string | No | `uncommitted` | Simplify mode: `uncommitted`, `branch`, `range`, `files` |
| `base_branch` | string | No | auto-detect | Base branch for `branch` mode |
| `range` | string | No | - | Commit range for `range` mode (e.g., `HEAD~3..HEAD`) |
| `files` | string[] | No | - | Files to simplify for `files` mode |
| `context` | int | No | 3 | Lines of context in diff |
| `agent` | string | No | default | Agent to use for simplification |
| `create_checkpoint` | bool | No | true | Create a git checkpoint before changes |

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

## Examples

### Simplify Uncommitted Changes

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "uncommitted"
  }'
```

### Simplify Branch vs Main

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "branch",
    "base_branch": "main"
  }'
```

### Simplify Commit Range

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "range",
    "range": "HEAD~5..HEAD"
  }'
```

### Simplify Specific Files

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "files",
    "files": ["src/handler.go", "src/validation.go"]
  }'
```

### Skip Checkpoint (Not Recommended)

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "uncommitted",
    "create_checkpoint": false
  }'
```

### Stream Simplify Results

```bash
curl -X POST http://localhost:8080/api/v1/workflow/simplify/standalone \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "mode": "uncommitted"
  }'
```

## File Operations

| Operation | Description |
|-----------|-------------|
| `modify` | File content was modified |
| `create` | New file was created |
| `delete` | File was removed |
| `rename` | File was renamed |

## Safety

### Automatic Checkpoints

By default, simplification creates a git checkpoint before modifying files. This allows you to undo changes using `git checkout` or the `mehr undo` command.

To skip checkpoint creation (not recommended):

```json
{
  "create_checkpoint": false
}
```

### Recovery

If simplification produces unwanted results:

```bash
# Using git
git checkout .

# Using mehr CLI
mehr undo
```

## Error Response

```json
{
  "success": false,
  "error": "nothing to simplify: no changes found"
}
```

## CLI Equivalent

```bash
mehr simplify --standalone
mehr simplify --standalone --branch main
mehr simplify --standalone --range HEAD~5..HEAD
mehr simplify --standalone src/handler.go src/validation.go
mehr simplify --standalone --no-checkpoint
```

See [CLI: simplify](../cli/simplify.md) for the command-line interface.

## See Also

- [CLI: simplify](../cli/simplify.md) - Command-line simplification
- [Standalone Review](standalone-review.md) - Standalone code review
- [Undo & Redo](undo-redo.md) - Recovering from changes
