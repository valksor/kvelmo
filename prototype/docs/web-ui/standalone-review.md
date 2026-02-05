# Standalone Review

Review code changes without an active task using the Web UI API.

## Overview

The standalone review feature allows you to run AI-powered code reviews on:

- **Uncommitted changes** - Staged and unstaged changes in your working directory
- **Branch comparisons** - Current branch vs main/master/develop
- **Commit ranges** - Specific commit ranges (e.g., `HEAD~3..HEAD`)
- **Specific files** - Review only certain files

This is useful for quick code reviews, CI/CD pipelines, or reviewing changes before committing.

## API Endpoint

```
POST /api/v1/workflow/review/standalone
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
  "apply_fixes": false,
  "create_checkpoint": true
}
```

### Fields

| Field               | Type     | Required | Default       | Description                                                           |
|---------------------|----------|----------|---------------|-----------------------------------------------------------------------|
| `mode`              | string   | No       | `uncommitted` | Review mode: `uncommitted`, `branch`, `range`, `files`                |
| `base_branch`       | string   | No       | auto-detect   | Base branch for `branch` mode                                         |
| `range`             | string   | No       | -             | Commit range for `range` mode (e.g., `HEAD~3..HEAD`)                  |
| `files`             | string[] | No       | -             | Files to review for `files` mode                                      |
| `context`           | int      | No       | 3             | Lines of context in diff                                              |
| `agent`             | string   | No       | default       | Agent to use for review                                               |
| `apply_fixes`       | bool     | No       | false         | If true, apply suggested fixes to files                               |
| `create_checkpoint` | bool     | No       | true          | Create checkpoint before changes (only used if `apply_fixes` is true) |

## Response

### Synchronous Response

```json
{
  "success": true,
  "verdict": "NEEDS_CHANGES",
  "summary": "The code changes introduce a potential nil pointer dereference.",
  "issues": [
    {
      "severity": "high",
      "category": "correctness",
      "file": "handler.go",
      "line": 45,
      "description": "Missing nil check before accessing user.Name"
    }
  ],
  "changes": [
    {
      "path": "handler.go",
      "operation": "update"
    }
  ],
  "usage": {
    "input_tokens": 1234,
    "output_tokens": 567,
    "cached_tokens": 100,
    "cost_usd": 0.0042
  }
}
```

> **Note**: The `changes` array is only populated when `apply_fixes` is true and fixes were applied.
```

### SSE Streaming Response

When `Accept: text/event-stream` is set, the response streams events:

```
event: message
data: {"event":"progress","message":"Gathering diff..."}

event: message
data: {"event":"content","text":"Reviewing code changes..."}

event: message
data: {"event":"result","data":{"success":true,"verdict":"APPROVED",...}}

event: message
data: {"event":"done"}
```

## Review Modes

| Mode | Description |
|------|-------------|
| Uncommitted changes | Reviews changes not yet committed |
| Branch vs base | Compares current branch to a base branch (e.g., main) |
| Commit range | Reviews a specific range of commits |
| Specific files | Reviews only the specified files |

For API integration examples, see [REST API Reference](/reference/rest-api.md).

## Verdicts

| Verdict         | Description                                   |
|-----------------|-----------------------------------------------|
| `APPROVED`      | Code looks good, no significant issues found  |
| `NEEDS_CHANGES` | Issues found that should be addressed         |
| `COMMENT`       | General observations without a strong verdict |

## Issue Severities

| Severity   | Description                               |
|------------|-------------------------------------------|
| `critical` | Security vulnerabilities, data loss risks |
| `high`     | Bugs, correctness issues                  |
| `medium`   | Code quality, maintainability concerns    |
| `low`      | Style issues, minor improvements          |

## Error Response

```json
{
  "success": false,
  "error": "nothing to review: no changes found"
}
```

---

## Also Available via CLI

Run standalone code reviews from the command line for scripting or terminal workflows.

See [CLI: review](/cli/review.md) for all review modes, fix application, and output options.

## See Also

- [Reviewing](reviewing.md) - Task-based review in Web UI
- [CLI: review](/cli/review.md) - Command-line review
- [Standalone Simplify](standalone-simplify.md) - Standalone code simplification
