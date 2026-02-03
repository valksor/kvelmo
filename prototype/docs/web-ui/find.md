# Find

AI-powered code search with focused results. Unlike asking Claude directly, the Find page uses a specialized prompt that instructs the agent to search efficiently and return concise, actionable results.

## Overview

The Find feature searches your codebase using AI with minimal fluff. The AI agent:

- **Uses Grep/Glob/Read tools efficiently** - Fast and precise code searches
- **Focuses on the task** - No tangential exploration or "helpful" context
- **Returns concise output** - Results only, with file paths, line numbers, and snippets

When you search, the agent extracts key terms from your query, searches for code matches, and returns results in a structured format.

## Accessing in the Web UI

Navigate to the **Find** page from the sidebar or go to `/find` directly.

| Feature         | Location              |
|-----------------|-----------------------|
| **Code Search** | Sidebar → Find        |
| **Search Form** | Find page main area   |
| **Search Tips** | Find page sidebar     |

## Using Find in the Web UI

### Basic Search

1. Go to **Find** in the sidebar
2. Enter your search query in the **Search Query** field
   - Use natural language: "where is the User struct defined?"
   - Include specific terms: function names, variable names, type names
3. Click **Find**

Results appear below with file paths, line numbers, and code snippets.

### Narrowing Search Scope

Optionally refine your search:

| Field           | Description                                        | Example              |
|-----------------|----------------------------------------------------|----------------------|
| **Path**        | Restrict to a directory (relative to project root) | `./internal/auth`    |
| **File Pattern** | Glob pattern for files                            | `**/*.go`, `**/*_test.go` |
| **Context Lines** | Lines of surrounding context in results          | 1, 3 (default), 5, 10 |

### Streaming Results

For large codebases, check **Stream results** to see matches as they're found rather than waiting for the full search to complete.

### Understanding Results

Each search result shows:

| Field    | Description                              |
|----------|------------------------------------------|
| File     | Path to the matching file                |
| Line     | Line number of the match                 |
| Snippet  | The matching code snippet                |
| Context  | Surrounding lines (based on context setting) |
| Reason   | Why this result matches your query       |

Results are displayed in a structured format with clickable file paths for easy navigation.

### Search Examples

**Find specific code:**
```
archive_blade database table
```

**Understand implementations:**
```
how does authentication work
```

**Locate definitions:**
```
where is the User struct defined
```

**Trace patterns:**
```
all API endpoint handlers
```

**Find usages:**
```
where is GetWorkspace called
```

## Search Tips

### Be Specific

Include function names, variable names, or type names for best results:
- **Good:** "GetWorkspace function in storage package"
- **Vague:** "where data is stored"

### Use Context

Mention what the code does:
- "database connection handling"
- "API endpoint for user creation"
- "error handling in auth middleware"

### File Patterns

Use glob patterns to narrow scope:
- `**/*.go` - All Go files
- `**/*_test.go` - Test files only
- `internal/**/*.go` - Go files in internal/

### Path Restriction

Limit to specific directories:
- `./internal/auth` - Authentication code
- `./cmd/mehr/commands` - CLI commands
- `./internal/server/handlers` - Web handlers

## How Find Works

### Specialized Prompt

Unlike general AI chat, Find uses a specialized prompt that instructs the agent to:

1. Extract key terms from your query
2. Use Grep with regex patterns for code matches
3. Use Glob to narrow file scope if needed
4. Read confirmed matches to extract snippets with context
5. Return results immediately without explanation

### No Task Required

Find works **without an active task**. It searches the current project directory directly, making it ideal for:

- Quick code exploration
- Understanding unfamiliar codebases
- Finding implementations before starting a task

### Result Format

Results are returned in a consistent format:

```
internal/models/user.go:25
   type User struct {
       ID      uint
       Name    string
   }
   → Model definition for user data
```

---

## Also Available via CLI

Search code from the command line for scripting or terminal workflows.

| Command | What It Does |
|---------|--------------|
| `mehr find "query"` | Search with AI-powered code search |
| `mehr find "query" --path ./src` | Limit search to a directory |
| `mehr find "query" --pattern "*.go"` | Filter to specific file patterns |
| `mehr find "query" --context 5` | Include more context lines |
| `mehr find "query" --stream` | Stream results as they're found |

See [CLI: find](/cli/find.md) for all search options and output formats.

## API Endpoints

The Find page uses these API endpoints:

| Method | Endpoint        | Description              |
|--------|-----------------|--------------------------|
| GET    | `/api/v1/find`  | Search with query params |
| POST   | `/api/v1/find`  | Search with JSON body    |

Query parameters / JSON fields:
- `q` / `query` - Search query (required)
- `path` - Directory to search
- `pattern` - Glob pattern for files
- `context` - Lines of context (default: 3)
- `stream` - Enable SSE streaming (true/false)

## Troubleshooting

### No Results Found

If searches return no results:
- Try more specific terms (function names, variable names)
- Remove path/pattern restrictions
- Rephrase your query with different keywords

### Search Taking Too Long

For large codebases:
- Enable "Stream results" to see matches as they're found
- Restrict search with a path or file pattern
- Use more specific terms to reduce search space

### Results Not Relevant

If results don't match your intent:
- Be more specific about what you're looking for
- Include the context of how the code is used
- Mention the type of code (handler, model, test, etc.)

## See Also

- [CLI: mehr find](/cli/find.md) - Command-line code search
- [Memory](memory.md) - Semantic search across past tasks
- [Task History](task-history.md) - Search and filter past tasks
