# Memory

The semantic memory system stores vector embeddings of your past tasks—code changes, specifications, session logs, and solutions. This enables finding similar past work and auto-suggesting solutions based on historical context.

## Overview

Mehrhof's semantic memory stores embeddings of:

- **Code changes** (git diffs) - What code was written
- **Specifications** (implementation plans) - What was planned
- **Session logs** (agent conversations) - How problems were solved
- **Solutions** (fixes and corrections) - What went wrong and how it was fixed

When you search, Mehrhof finds semantically similar past tasks using vector similarity—not just keyword matching.

## Accessing in the Web UI

Memory features are available through:

| Feature | Location |
|---------|----------|
| **Memory Search** | Settings → Memory tab |
| **Memory Statistics** | Settings → Memory → Stats |
| **Memory Configuration** | Settings → Memory → Config |

Memory search results appear automatically in the dashboard when starting similar tasks.

## Using Memory in the Web UI

### Searching Memory

Find similar past tasks:

1. Go to **Settings → Memory**
2. Enter your search query in the search box
3. Optionally filter by:
   - **Document type** - `code_change`, `specification`, `session`, `solution`
   - **Task ID** - Search within a specific task
   - **Result limit** - Number of results (default: 5)
4. Click **"Search"**

Results show similar documents with similarity scores.

### Understanding Results

Each search result shows:

| Field | Description |
|-------|-------------|
| Similarity | Score from 0-1 (higher = more similar) |
| Document Type | What kind of content |
| Task ID | Source task |
| Excerpt | Relevant content snippet |
| Date | When the document was created |

**Similarity Threshold:** Only results above 0.8 (default) are shown. Adjust this in settings.

### Viewing Memory Statistics

See what's stored in memory:

1. Go to **Settings → Memory → Stats**
2. View:
   - Total documents stored
   - Documents by type
   - Embedding model in use
   - Vector store configuration

Example:
```
Total Documents: 47

Documents by Type:
  code_change: 18
  specification: 12
  session: 14
  solution: 3

Embedding Model: simple (hash-based)
Vector Store: ChromaDB (in-memory)
```

### Configuring Memory

Memory is configured in **Settings → Memory → Config**:

```yaml
memory:
  enabled: true
  vector_db:
    backend: chromadb
    connection_string: ./.mehrhof/vectors
    collection: mehr_task_memory
  retention:
    max_days: 90
    max_tasks: 1000
  search:
    similarity_threshold: 0.8
    max_results: 5
  learning:
    auto_store: true
    learn_from_corrections: true
    suggest_similar: true
```

**Key Settings:**
- **similarity_threshold** - Lower = more results (try 0.65 if none found)
- **max_results** - How many matches to return
- **auto_store** - Automatically index completed tasks
- **suggest_similar** - Show similar tasks when starting new work

### Clearing Memory

Remove all stored memory:

1. Go to **Settings → Memory**
2. Scroll to **Danger Zone**
3. Click **"Clear All Memory"**
4. Confirm the action

**Warning:** This cannot be undone.

## How Memory Works

### Automatic Indexing

When a task completes, Mehrhof automatically indexes:

1. **Specifications** - All specs from planning phase
2. **Code Changes** - Git diff between base and task branch
3. **Sessions** - Full agent conversation logs

This happens if `auto_store: true` in settings (default).

### Semantic Search

When you search, Mehrhof:

1. Generates an embedding vector for your query
2. Searches the vector database for similar embeddings
3. Returns documents above the similarity threshold

Vector similarity means:
- "authentication bug" finds "login fix"
- "database schema" finds "model changes"
- "race condition" finds "concurrency fix"

### Agent Integration

When `suggest_similar: true`, the AI agent receives context from similar past tasks automatically:

```
Agent context includes:
  - Similar task titles
  - Relevant code snippets
  - Past solutions that worked
  - Common pitfalls to avoid
```

This helps the AI apply lessons learned from previous work.

## Common Workflows

### Finding Similar Bug Fixes

```
1. Search: "null pointer exception"
2. Filter by: code_change
3. Review similar fixes
4. Apply the same approach
```

### Reviewing Past Approaches

```
1. Search: "user authentication flow"
2. Filter by: specification
3. See how similar features were designed
4. Inform your current implementation
```

### Memory-Driven Development

```
1. Start new task
2. Dashboard shows: "3 similar tasks found"
3. Review suggestions automatically
4. Agent uses memory context in planning
```

## CLI Equivalent

See [`mehr memory`](../cli/memory.md) for CLI usage.

| CLI Command | Web UI Action |
|-------------|---------------|
| `mehr memory search "query"` | Search memory |
| `mehr memory stats` | View statistics |
| `mehr memory index --task abc123` | Manually index task |
| `mehr memory clear` | Clear all memory |

## Memory Storage

Memory data is stored in:
```
./.mehrhof/vectors/              # Vector database
~/.valksor/mehrhof/memory/        # Memory index
```

## Troubleshooting

### No Results Found

If searches return no results:
- Lower the `similarity_threshold` in settings (try 0.65)
- Increase `max_results` in settings
- Ensure tasks have been indexed (check stats)

### Memory Not Auto-Indexing

Check that `auto_store: true` in settings. Manually index from CLI:
```bash
mehr memory index --task <id>
```

### Similar Suggestions Not Appearing

Check that `suggest_similar: true` in settings. This enables automatic suggestions on the dashboard.
