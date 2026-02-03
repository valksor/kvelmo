# mehr find

AI-powered code search with focused results.

## Synopsis

```bash
mehr find <query> [flags]
```

## Description

The `find` command searches your codebase using AI with minimal fluff. Unlike asking Claude directly, this command uses a specialized prompt that instructs the agent to:

1. **Use Grep/Glob/Read tools efficiently** - Fast and precise code searches
2. **Focus on the task** - No tangential exploration or "helpful" context
3. **Concise output** - Results only, with multiple formatting options

The agent searches for code matching your query and returns results in a structured format with file paths, line numbers, code snippets, and context.

## When to Use

- **Find specific code** - "where is the archive_blade table used?"
- **Understand implementations** - "how does authentication work?"
- **Locate usages** - "where is the User struct defined?"
- **Trace patterns** - "all API endpoint handlers"
- **Quick exploration** - Without starting a full planning workflow

## Examples

### Basic Search

```bash
$ mehr find "archive_blade database table"

internal/models/archive.go:42: type ArchiveBlade struct
internal/db/query.go:128: db.Find(&ArchiveBlade{})
internal/handlers/archive.go:45: archiveBlade.Create()
```

### Structured Output

```bash
$ mehr find "archive_blade" --format structured

Found 3 match(es) for archive_blade

1. internal/models/archive.go:42
   type ArchiveBlade struct {
       ID      uint
       Name    string
   }
   → Model definition

2. internal/db/query.go:128
   db.Find(&ArchiveBlade{}).Where(...)
   → Database query usage

3. internal/handlers/archive.go:45
   archiveBlade.Create(&ArchiveBlade{...})
   → Create operation
```

### Restrict to Directory

```bash
$ mehr find "authentication" --path ./internal/auth/

internal/auth/middleware.go:15: func authenticate(w http.ResponseWriter)
internal/auth/handlers.go:42: return AuthResult{Success: true}
```

### With File Pattern

```bash
$ mehr find "User struct" --pattern "**/*.go

internal/models/user.go:25: type User struct {
internal/models/user.go:30:     Username string
internal/api/user.go:18: func GetUser(u *User) error {
```

### JSON for Scripting

```bash
$ mehr find "TODO comments" --format json | jq '.matches[] | .file'

"internal/auth/handlers.go"
"internal/db/query.go"
"internal/server/routes.go"
```

### Stream Results (Large Codebases)

```bash
$ mehr find "memory leak" --stream

internal/cache/cache.go:142: if cache.Contains(key) {
internal/mem/pool.go:89:  return p.Alloc()
internal/mem/pool.go:95:  return p.Alloc()
Found 3 matches
```

## Output Formats

| Format     | Description                                       |
|------------|---------------------------------------------------|
| concise    | `file.go:line: snippet` (default)                 |
| structured | Numbered list with file info, context, and reason |
| json       | Machine-readable for scripting and pipelines      |

## Flags

| Flag              | Default | Description                                              |
|-------------------|---------|----------------------------------------------------------|
| `--path`, `-p`    |         | Restrict search to directory (relative to project root)  |
| `--pattern`       |         | Glob pattern for files (e.g., `**/*.go`, `**/*test*.go`) |
| `--format`        | concise | Output format: `concise`, `structured`, or `json`        |
| `--stream`        | false   | Stream results as found (for large codebases)            |
| `--agent`         |         | Agent to use for search (e.g., `opus`, `claude-sonnet`)  |
| `--context`, `-C` | 3       | Lines of context to include in results                   |

## How It Works

The `find` command differs from using Claude directly in several key ways:

1. **Focused Prompt**: The agent receives instructions to use Grep/Glob/Read tools directly, avoiding exploratory behavior
2. **Structured Output**: Results are parsed and displayed in a consistent format
3. **No Task Required**: Works without an active task - searches the current project directory
4. **Efficient**: Agent is instructed NOT to explain its process or add "helpful" context

### Search Strategy

The agent follows this approach:
1. Extract key terms from your query (function names, variables, types)
2. Use Grep with regex patterns for code matches
3. Use Glob to narrow file scope if needed
4. Read confirmed matches to extract snippets with context

### Fallback Parsing

If the agent doesn't return structured results, the command falls back to extracting `file:line` patterns from the response.

## Web UI

Prefer a visual interface? See the search and filter features in [Task History](/web-ui/task-history.md).

## See Also

- [status](status.md) - Task status and workspace information
- [memory](memory.md) - Semantic search across past tasks
- [CLI Reference](/reference/cli.md) - All commands
