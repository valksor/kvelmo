# Library

The library feature enables pulling and managing documentation collections that provide context for AI agents during planning, implementation, and review workflows.

## Overview

Documentation can be pulled from:
- **URLs**: Web documentation (with automatic HTML-to-markdown conversion and crawling)
- **Local files**: Directories or individual files on your filesystem
- **Git repositories**: Clone specific branches/directories from git repos

Collections can be:
- **Auto-included**: Automatically added to agent prompts when file paths match patterns
- **Explicit**: Only included when using the `--library` flag
- **Always**: Included in every prompt regardless of context

## Commands

### `mehr library pull <source>`

Pull documentation into a named collection.

```bash
# Pull from a URL
mehr library pull https://go.dev/doc/effective_go --name "Effective Go"

# Pull from local files
mehr library pull ./docs/api --name "API Docs" --mode explicit

# Pull from git repository
mehr library pull git@github.com:org/repo.git --name "Repo Docs" --git-path docs/

# Pull with crawling options
mehr library pull https://code.visualstudio.com/api \
  --name "VSCode API" \
  --max-depth 2 \
  --max-pages 50

# Pull to shared location (available to all projects)
mehr library pull https://docs.example.com/ --shared
```

**Options:**

| Flag          | Description                                                          |
|---------------|----------------------------------------------------------------------|
| `--name`      | Collection name (auto-generated if not provided)                     |
| `--mode`      | Include mode: `auto`, `explicit`, `always` (default: `auto`)         |
| `--paths`     | Comma-separated glob patterns for auto-include (e.g., `src/**/*.ts`) |
| `--tags`      | Comma-separated tags for organization                                |
| `--shared`    | Store in shared location (available to all projects)                 |
| `--max-depth` | Maximum crawl depth for URLs (default: 3)                            |
| `--max-pages` | Maximum pages to crawl for URLs (default: 100)                       |
| `--dry-run`   | Preview what would be pulled without saving                          |
| `--continue`  | Resume an interrupted crawl                                          |
| `--restart`   | Ignore existing state and start fresh                                |

### Resume Interrupted Crawls

If a crawl is interrupted (network error, timeout, Ctrl+C), the next `pull` command will detect the incomplete state:

```bash
$ mehr library pull https://docs.example.com
! Found incomplete crawl:
  Collection: docs-example-com-abc123
  Started:    2026-02-03 14:30:00
  Progress:   45/120 pages (3 failed, 72 pending)

Options:
  mehr library pull <source> --continue   # Resume from where it left off
  mehr library pull <source> --restart    # Start fresh, discard progress
```

Resume the crawl:
```bash
mehr library pull https://docs.example.com --continue
```

Or start fresh:
```bash
mehr library pull https://docs.example.com --restart
```

**How it works:**
- Crawl progress is saved to a state file (`.crawl-state.yaml`) during crawling
- Pages are written to disk as they're fetched, not batched at the end
- State is checkpointed every 10 pages or 30 seconds
- On successful completion, the state file is automatically deleted

### `mehr library list`

List all documentation collections.

```bash
mehr library list
mehr library list --verbose
mehr library list --tag typescript
mehr library list --shared
```

**Options:**

| Flag        | Description                          |
|-------------|--------------------------------------|
| `--verbose` | Show detailed collection information |
| `--tag`     | Filter by tag                        |
| `--shared`  | Only show shared collections         |
| `--project` | Only show project collections        |

### `mehr library show <name>`

View details of a specific collection.

```bash
mehr library show "Effective Go"
mehr library show "Effective Go" page/getting-started
```

### `mehr library remove <name>`

Remove a documentation collection.

```bash
mehr library remove "Effective Go"
mehr library remove "Effective Go" --force
```

### `mehr library update [name]`

Update one or all collections by re-pulling from their sources.

```bash
# Update all collections
mehr library update

# Update specific collection
mehr library update "Effective Go"

# Force full refresh (re-fetch all pages)
mehr library update "Effective Go" --full
```

By default, updates auto-continue any interrupted crawls. Use `--full` to re-fetch all pages regardless of existing state.

### `--library` Flag

Automatically include relevant library documentation in agent prompts.

```bash
# Plan with library context
mehr plan --library

# Implement with library context
mehr implement --library

# Review with library context
mehr review --library
```

When enabled, documentation is automatically selected based on:
1. File paths being edited (for collections with `--paths` patterns)
2. Collection `include_mode` setting (auto/explicit/always)

## Collection Management

### Storage Locations

- **Project**: Stored in `.mehrhof/library/` within the project (default)
- **Shared**: Stored in `~/.valksor/mehrhof/library/` and available to all projects

### Downloaded Content Structure

When you pull documentation, Mehrhof downloads and converts content to markdown:

```
.mehrhof/library/                         # Project collections
  └── {collection-id}/
      ├── meta.json                        # Collection metadata
      └── pages/
          ├── index.md                     # Root page
          ├── getting-started.md           # Converted pages
          └── api/
              └── reference.md

~/.valksor/mehrhof/library/               # Shared collections (--shared)
  └── {collection-id}/
      └── ...
```

**Storage behavior:**
- **URL sources**: HTML converted to markdown, images stripped
- **Local files**: Copied as-is (markdown files) or converted
- **Git repos**: Cloned temporarily, specified path extracted

### Auto-Include Mechanism

When the `--library` flag is used, Mehrhof automatically selects relevant documentation:

1. **Collection filtering**: Scans collections matching path patterns against working files
2. **Page scoring**: Scores each page by relevance (keyword matching or semantic similarity if ONNX enabled)
3. **Token budgeting**: Includes top-scoring pages within the configured token budget
4. **Truncation**: High-scoring pages may be truncated if they exceed remaining budget

Configure scoring behavior via the [memory embedding model](/cli/memory.md#embedding-models). When ONNX embeddings are enabled, library uses semantic similarity for more accurate relevance scoring.

### Path Patterns

Glob patterns control auto-include behavior:

```bash
# Single directory pattern
mehr library pull ./vscode-extension --paths "ide/vscode/**"

# Multiple patterns
mehr library pull ./docs --paths "api/**,client/**/*.ts"

# File extension pattern
mehr library pull ./golang-docs --paths "**/*.go"
```

### Include Modes

- **auto** (default): Include when files match path patterns
- **explicit**: Only include with `--library` flag
- **always**: Include in every agent prompt

## Examples

### Web Documentation with Crawling

```bash
# Pull VSCode API documentation with crawling
mehr library pull https://code.visualstudio.com/api \
  --name "VSCode API" \
  --paths "ide/vscode/**" \
  --max-depth 2 \
  --max-pages 50 \
  --tags vscode,api
```

### Local Documentation

```bash
# Pull local API documentation
mehr library pull ./docs/api \
  --name "Internal API" \
  --mode explicit \
  --paths "api/**,client/**"
```

### Git Repository Documentation

```bash
# Pull docs from a specific branch/path
mehr library pull git@github.com:golang/go.git \
  --name "Go Docs" \
  --git-ref go1.22 \
  --git-path doc \
  --paths "**/*.go"
```

### Using with AI Workflows

```bash
# Auto-include relevant docs during planning
mehr plan --library

# Auto-include during implementation
mehr implement --library

# Review with documentation context
mehr review --library
```

## Configuration

Library settings can be configured in `.mehrhof/config.yaml`:

```yaml
library:
  auto_include_max: 3          # Max collections to auto-include
  max_pages_per_prompt: 20     # Max pages from a single collection
  max_crawl_pages: 100         # Default max pages per crawl
  max_crawl_depth: 3           # Default max crawl depth
  max_page_size_bytes: 1048576 # Max size per page (1MB)
  lock_timeout: "10s"          # File lock timeout
  max_token_budget: 8000       # Token budget for library context
```

## MCP Tools

When using `mehr serve --api`, the following MCP tools are available:

- **library_list**: List all documentation collections
- **library_show**: Show details for a specific collection
- **library_get_docs**: Get relevant documentation for file paths

## Known Limitations

- Large PDFs are skipped (binary detection)
- JavaScript-heavy sites may have incomplete content
- Rate limiting is applied during crawling (500ms delay between requests)
- robots.txt is respected by default

## See Also

- [Web UI documentation](web-ui/library.md)
- [Configuration reference](reference/storage.md#library-settings)
