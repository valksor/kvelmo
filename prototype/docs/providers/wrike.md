# Wrike Provider

> **⚠️ Third-Party Integration**: This integration depends on external APIs that may change. Not fully tested beyond unit tests. Behavior may vary depending on the third-party service. Manual validation recommended before production use.


**Schemes:** `wrike:`, `wk:`

**Capabilities:** `read`, `list`, `fetch_comments`, `comment`, `update_status`, `manage_labels`, `create_work_unit`, `download_attachment`, `snapshot`, `fetch_subtasks`

Integrates with Wrike for enterprise project management.

## Usage

```bash
mehr start wrike:IEAGI2D4I4AL7YNL
mehr plan wk:IEAGI2D4I4AL7YNL

mehr start wrike:https://app.wrike.com/open.htm?id=123456

mehr start wk:1234567890
```

## Configuration

```yaml
wrike:
  token: "${WRIKE_TOKEN}"
  host: "https://www.wrike.com/api/v4"  # Optional: override API base URL
  space: "824404493"                     # Optional: space ID for listing
  folder: "1635167041"                   # Optional: folder ID for task creation
  project: "4352950154"                  # Optional: project ID (takes priority over folder)
```

### Configuration Fields

| Field | Description |
|-------|-------------|
| `token` | Wrike API token (can use env var syntax) |
| `host` | API base URL override (default: `https://www.wrike.com/api/v4`) |
| `space` | Space ID for listing tasks across an entire space |
| `folder` | Folder ID for task lookup and creation (if no project) |
| `project` | Project ID for task creation (takes priority over folder) |

### Task Creation Priority

1. If `project` is configured → tasks are created in the project
2. Else if `folder` is configured → tasks are created in the folder
3. Else → error (no target configured)

### ID Formats

All ID fields (`space`, `folder`, `project`) accept:

- **Numeric ID** (from URL): e.g., `4352950154` — auto-resolved to API ID
- **API ID**: e.g., `IEAAJXXXX` — used directly

When you configure a numeric ID, it's automatically resolved on first use via the Wrike API and logged:

```
INFO Resolved Wrike project numeric_id=4352950154 api_id=IEAAJXXXX title="My Project" type=project
```

### Finding IDs from Wrike URLs

Extract numeric IDs from Wrike workspace URLs:

| URL Pattern | ID Location |
|-------------|-------------|
| `/folder/1635167041/` | Folder or project ID |
| `/task-view?id=123&pid=456` | Task ID (`id`), Parent folder (`pid`) |
| `?spaceId=824404493` | Space ID |

Example: From `https://www.wrike.com/workspace.htm#/folder/4352950154/tableV2?spaceId=824404493`
- Project ID: `4352950154`
- Space ID: `824404493`

## Token Resolution

1. `MEHR_WRIKE_TOKEN` environment variable
2. `WRIKE_TOKEN` environment variable
3. Token from `config.yaml`

## Features

- **Task Fetching**: Retrieves title, description, status, priority, permalink
- **Subtasks**: Recursively fetches nested subtasks (max depth: 5)
- **Comments**: Fetches all comments with automatic pagination support
- **Attachments**: Lists and downloads file attachments
- **Retry Logic**: Automatic exponential backoff for rate limit errors (429)
- **Multiple ID Formats**: Supports numeric IDs, API IDs (`IEAAJ...`), and permalink URLs
