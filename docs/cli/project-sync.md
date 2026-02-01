# `mehr project sync`

Sync entire project/epic structures from external providers into local implementation queues.

## Usage

```bash
mehr project sync <provider:reference> [flags]
```

## Description

The `project sync` command pulls all tasks and subtasks from a provider's project or epic and creates a local implementation queue. This is useful when you want to implement an entire feature or epic from tools like Wrike, Jira, or GitHub.

The synced queue includes:
- All tasks and subtasks from the project/epic
- Parent-child relationships preserved
- External IDs and URLs for traceability
- Smart status filtering (open + completed in last 30 days by default)

## Arguments

- `<provider:reference>` - Provider-specific project reference
  - Format: `provider:reference` (e.g., `wrike:123456`, `jira:PROJ-123`)
  - Or bare reference for default provider

## Flags

| Flag             | Description                        | Default         |
|------------------|------------------------------------|-----------------|
| `--status`       | Filter by status (comma-separated) | *smart default* |
| `--max-depth`    | Max depth for recursive fetch      | 0 (unlimited)   |
| `--preserve-ext` | Keep external IDs/URLs             | true            |

### Smart Status Filter

When `--status` is not specified, the smart default includes:
- All open tasks
- All in-progress tasks
- Tasks completed in the last 30 days

## Provider References

### Wrike

```
# By permalink URL
mehr project sync wrike:https://www.wrike.com/open.htm?id=123456

# By numeric ID
mehr project sync wrike:123456

# By folder ID (API ID)
mehr project sync wrike:IEAAJXXXXXXXX
```

### Jira

```
# Epic by key
mehr project sync jira:PROJ-123

# Epic with full URL
mehr project sync jira:https://domain.atlassian.net/browse/PROJ-123
```

### GitHub

```
# Issue with task list (fallback mode)
mehr project sync github:owner/repo#456
```

## Examples

```bash
# Sync an entire Wrike folder/project
mehr project sync wrike:https://www.wrike.com/open.htm?id=123456

# Sync a Jira epic with only open tasks
mehr project sync jira:PROJ-123 --status open,in_progress

# Sync with limited depth
mehr project sync wrike:123456 --max-depth 2
```

## Output

The command creates a new queue with a timestamp suffix:
- Queue name: `<project-name>-<timestamp>`
- Queue description includes source provider and URL
- Each task includes external reference metadata

Example output:
```
✓ Synced project: "Epic: User Authentication"
  Source: jira
  URL: https://example.atlassian.net/browse/AUTH-42
  Tasks: 12 total (8 new)

Queue created: auth-user-authentication-20250129-143052
```

## See Also

- [`mehr project plan`](./project-plan.md) - Create a project plan from scratch
- [`mehr project submit`](./project-submit.md) - Submit a project for implementation
- [`mehr queue list`](./queue-list.md) - List all queues
