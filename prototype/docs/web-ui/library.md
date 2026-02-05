# Library — Web UI

The library web interface provides documentation collection management with a visual interface for pulling, viewing, and organizing documentation.

## Access

From the navigation bar, open the **More** dropdown and click **Library**.

## Features

### Pull Documentation

The pull form allows adding documentation from three source types:

#### URL Source

Pull documentation from any public URL with automatic HTML-to-markdown conversion.

| Field | Description |
|-------|-------------|
| **Source** | URL to documentation (e.g., `https://docs.example.com/`) |
| **Name** | Collection name (optional, auto-detected from URL) |
| **Include Mode** | When to include: Auto, Explicit, or Always |
| **Path Patterns** | Comma-separated glob patterns for auto-include |
| **Tags** | Comma-separated tags for organization |
| **Max Crawl Depth** | Link depth for crawling (default: 3) |
| **Max Pages** | Maximum pages to fetch (default: 100) |
| **Shared** | Store globally (available to all projects) |

#### File Source

Pull from local directories or files:

```
Source: ./path/to/docs
Source: /absolute/path/to/file.md
```

#### Git Repository Source

Pull from git repositories:

```
Source: git@github.com:org/repo.git
Source: https://github.com/org/repo.git
```

Additional options for git:
- **Git Ref**: Branch or tag (e.g., `main`, `v1.0.0`)
- **Git Path**: Subdirectory to extract (e.g., `docs`)

### Collections List

View all documentation collections with:

- **Collection name** and source type icon
- **Include mode badge** (auto/explicit/always)
- **Location badge** (project/shared)
- **Page count**, size, and last update time
- **Tags** for organization
- **View** button - Opens detail modal
- **Remove** button - Delete collection with confirmation

### Collection Details Modal

Click **View** on any collection to see:

- **Metadata**: Source, type, mode, location
- **Statistics**: Page count, total size
- **Path patterns**: Glob patterns for auto-include
- **Pages list**: Searchable list of all pages

### Statistics Panel

View library statistics in the statistics panel:

- Total collections
- Total pages across all collections
- Total storage used
- Breakdown by location (project vs shared)

## REST API

All library operations are available via REST API.

### List Collections

```http
GET /api/v1/library
```

Query parameters:
- `shared=true` - Only shared collections
- `project=true` - Only project collections
- `tag=typescript` - Filter by tag

**Response (JSON):**
```json
{
  "collections": [
    {
      "id": "effective-go-abc123",
      "name": "Effective Go",
      "source": "https://go.dev/doc/effective_go",
      "source_type": "url",
      "include_mode": "auto",
      "page_count": 45,
      "total_size": 524288,
      "location": "project",
      "tags": ["golang", "best-practices"],
      "paths": ["**/*.go"],
      "pulled_at": "2026-01-15T10:30:00Z"
    }
  ],
  "count": 1
}
```

### Pull Documentation

```http
POST /api/v1/library/pull
Content-Type: application/x-www-form-urlencoded
```

Form fields:
- `source` (required) - URL, file path, or git repo
- `name` (optional) - Collection name
- `mode` (optional) - `auto`, `explicit`, `always`
- `paths` (optional) - Comma-separated glob patterns
- `tags` (optional) - Comma-separated tags
- `shared` (optional) - Store globally
- `max_depth` (optional) - Crawl depth
- `max_pages` (optional) - Max pages to crawl

**Response (JSON):**
```json
{
  "collection": { ... },
  "pages_written": 42,
  "pages_failed": 0,
  "pages_skipped": 5
}
```

### Show Collection

```http
GET /api/v1/library/{name}
```

**Response (JSON):**
```json
{
  "collection": { ... },
  "pages": ["index.md", "getting-started.md", ...]
}
```

### Remove Collection

```http
DELETE /api/v1/library/{name}
```

**Response (JSON):**
```json
{
  "success": true,
  "message": "collection removed successfully"
}
```

### Get Statistics

```http
GET /api/v1/library/stats
```

**Response (JSON):**
```json
{
  "total_collections": 5,
  "total_pages": 234,
  "total_size": 2097152,
  "project_count": 3,
  "shared_count": 2,
  "by_mode": {
    "auto": 3,
    "explicit": 1,
    "always": 1
  },
  "enabled": true
}
```

## Examples

### Pull Web Documentation

1. Navigate to **/library**
2. Fill in the pull form:
   - Source: `https://react.dev/learn`
   - Name: `React Learn`
   - Mode: `Auto`
   - Paths: `src/**/*.tsx, src/**/*.jsx`
   - Tags: `react, frontend`
3. Click **Preview** to see what will be pulled
4. Click **Pull** to create the collection

### View Collection Details

1. Click **View** on any collection
2. Modal opens with:
   - Collection metadata
   - Page list with search
3. Click **X** or outside modal to close

### Remove Collection

1. Click **Remove** on any collection
2. Confirm the removal
3. Collection is deleted from storage

## Integration with AI Workflows

Documentation collections are automatically used in AI workflows:

- **Planning phase**: Docs matching file paths are included in prompt
- **Implementation phase**: Context-aware documentation for code being written
- **Review phase**: API docs and style guides for review criteria

Enable auto-include via `--library` flag in CLI commands or configure collections with path patterns.

---

## Also Available via CLI

Manage documentation collections from the command line for scripting or terminal workflows.

| Command | What It Does |
|---------|--------------|
| `mehr library` | List all collections |
| `mehr library pull <source>` | Pull documentation from URL, file, or git |
| `mehr library show <name>` | View collection details and pages |
| `mehr library remove <name>` | Delete a collection |
| `mehr library stats` | View library statistics |

See [CLI: library](/cli/library.md) for all pull options, filtering, and batch operations.

## See Also

- [Configuration](/reference/storage.md#library-settings)
