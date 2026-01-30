# Commit Page

The commit page provides a web interface for creating logically grouped commits from uncommitted changes using AI.

## Access

Navigate to `/commit` in the Web UI, or click "Commit" in the navigation menu.

## Features

- **Analyze Changes**: Click to analyze uncommitted changes and group them into logical commits
- **Include Unstaged**: Toggle to include unstaged changes in the analysis
- **Preview Groups**: See proposed commits with their messages and file lists
- **Create Commits**: Execute the commits after reviewing
- **Auto Push**: Option to push commits to remote after creating

## Workflow

1. Navigate to `/commit`
2. Optionally check "Include unstaged" to include unstaged changes
3. Click "Analyze Changes" to group your changes
4. Review the proposed commit groups
5. Optionally check "Push after creating"
6. Click "Create Commits" to execute

## Agent Configuration

The commit grouping and message generation uses the agent configured for the `checkpointing` step. You can configure this in your workspace config:

```yaml
# .mehrhof/config.yaml
agent:
  steps:
    checkpointing:
      name: claude-opus  # Use a specific agent for commit operations
```

For CLI usage with explicit agent override, see [CLI commit documentation](../cli/commit.md).

## API Endpoints

### `GET /commit`

Renders the commit page UI.

### `GET /api/v1/commit/plan?all={true|false}`

Analyzes changes and returns commit groups for preview.

**Response**:
```json
{
  "groups": [
    {
      "files": ["path1", "path2"],
      "message": "Add authentication feature",
      "reason": "All related to auth"
    }
  ],
  "total": 1
}
```

### `POST /api/v1/commit/execute`

Creates commits for the provided groups.

**Request**:
```json
{
  "groups": [
    {
      "message": "Add authentication feature",
      "files": ["path1", "path2"]
    }
  ],
  "push": false
}
```

**Response**:
```json
{
  "results": [
    {
      "hash": "abc123",
      "message": "Add authentication feature"
    }
  ]
}
```
