# mehr memory

Semantic memory commands for storing and searching past task embeddings.

## Synopsis

```bash
mehr memory <subcommand> [flags]
```

## Description

Mehrhof's semantic memory system stores vector embeddings of:
- Code changes (git diffs)
- Specifications (implementation plans)
- Session logs (agent conversations)
- Solutions (fixes and corrections)

This enables finding similar past tasks and auto-suggesting solutions based on historical context using semantic search.

## Configuration

Memory is disabled by default. Enable it in `.mehrhof/config.yaml`:

```yaml
memory:
  enabled: true
  vector_db:
    backend: chromadb                    # Vector database backend
    connection_string: ./.mehrhof/vectors # Path or URL to vector DB
    collection: mehr_task_memory          # Collection name
    embedding_model: default
  retention:
    max_days: 90                          # Retention period
    max_tasks: 1000                       # Maximum tasks to store
  search:
    similarity_threshold: 0.8            # Minimum similarity for matches
    max_results: 5                       # Maximum results to return
  learning:
    auto_store: true                     # Automatically store task data
    learn_from_corrections: true         # Learn when user corrects agent
    suggest_similar: true                # Auto-suggest similar tasks
```

### Vector Database Backends

| Backend    | Description                              | Connection String                         |
|------------|------------------------------------------|-------------------------------------------|
| `chromadb` | Local in-memory vector storage (default) | File path (default: `./.mehrhof/vectors`) |

### Embedding Models

| Model     | Type       | Description                                           | Dimension |
|-----------|------------|-------------------------------------------------------|-----------|
| `default` | Hash-based | SHA256 deterministic embedding (no external APIs)     | 1536      |
| `onnx`    | Semantic   | Neural embedding using ONNX Runtime (download-on-use) | 384       |

**Hash-based embedding** (default) uses SHA256 for deterministic local embeddings. Fast and fully offline, but only matches identical or near-identical text.

**ONNX embedding** uses neural networks for true semantic similarity. "cat" will match "kitten" and "feline". Requires ONNX Runtime library (auto-installed on first use).

### Enabling Semantic Embeddings

To enable ONNX semantic embeddings, update your config:

```yaml
memory:
  enabled: true
  vector_db:
    embedding_model: onnx
    onnx:
      model: all-MiniLM-L6-v2    # Default model (22MB, good quality)
      # cache_path: ~/.valksor/mehrhof/models/  # Custom cache location
      # max_length: 256          # Max tokens per text (default: 256)
```

**Available ONNX models:**

| Model               | Size | Quality | Speed  |
|---------------------|------|---------|--------|
| `all-MiniLM-L6-v2`  | 22MB | Good    | Fast   |
| `all-MiniLM-L12-v2` | 33MB | Better  | Medium |

**First-run behavior**: On first use, Mehrhof downloads the model to `~/.valksor/mehrhof/models/`. Subsequent runs use the cached model.

**Important**: Switching between `default` (hash) and `onnx` embeddings invalidates existing vectors. Run `mehr memory clear` after changing embedding models.

For details on the ONNX sidecar architecture, platform support, and troubleshooting, see [Advanced: ONNX Embedder](/advanced/onnx-embedder.md).

## Commands

### memory search

Search for semantically similar past tasks:

```bash
mehr memory search <query> [--limit=N] [--type=TYPE] [--task=ID]
```

Flags:
- `--limit`, `-l` - Maximum results to return (default: 5)
- `--type`, `-t` - Filter by document type (can specify multiple)
- `--task` - Filter by task ID

Document types:
- `code_change` - Code diffs from git
- `specification` - Implementation plans
- `session` - Agent conversation logs
- `solution` - Stored fixes and corrections

Examples:
```bash
# Search for authentication-related tasks
mehr memory search "authentication"

# Search with higher result limit
mehr memory search "api endpoint" --limit 10

# Search only specifications
mehr memory search "database schema" --type specification

# Search within specific task
mehr memory search "error handling" --task abc123
```

### memory index

Manually index a task into memory:

```bash
mehr memory index --task <id>
```

This indexes:
- Task specifications
- Code changes (git diff from base branch)
- Session logs (agent conversations)

Example:
```bash
# Index a completed task
mehr memory index --task abc123
```

**Note**: Tasks are automatically indexed on completion if `auto_store: true`.

### memory stats

Show memory system statistics:

```bash
mehr memory stats
```

Example output:
```
=== Memory Statistics ===
Total Documents: 47

Documents by Type:
  code_change: 18
  specification: 12
  session: 14
  solution: 3

=== Configuration ===
Embedding Model: simple (hash-based)
Vector Store: ChromaDB (in-memory)
```

### memory clear

Clear all stored memory (requires confirmation):

```bash
mehr memory clear
```

Example:
```bash
$ mehr memory clear
Are you sure you want to clear all memory? This cannot be undone. [y/N]: y
Memory cleared successfully.
```

## How It Works

### Automatic Indexing

When a task completes, Mehrhof automatically indexes:

1. **Specifications** - All specs generated during planning
2. **Code Changes** - Git diff between base branch and task branch
3. **Sessions** - Full conversation logs from agent interactions

### Semantic Search

When you search, Mehrhof:
1. Generates an embedding vector for your query
2. Searches the vector database for similar embeddings
3. Returns the most similar documents above the threshold

### Agent Integration

The memory system can augment agent prompts with relevant context from similar past tasks:

```yaml
agent:
  instructions: |
    Context from similar past tasks is available to guide your approach.
    Use historical solutions to inform your implementation decisions.
```

## Examples

### Find Similar Bug Fixes

```bash
# Search for similar bug fixes
mehr memory search "null pointer exception" --type code_change

# Search within solution documents
mehr memory search "race condition fix" --type solution
```

### Review Past Approaches

```bash
# See how similar features were implemented
mehr memory search "user authentication flow" --type specification

# Review past testing approaches
mehr memory search "integration testing" --type session
```

### Memory-Driven Development

1. Start a new task
2. Search memory for similar tasks: `mehr memory search "your query"`
3. Review similar solutions before implementing
4. Let agent use memory context automatically

## Troubleshooting

### No Results Found

If searches return no results:
- Lower the `similarity_threshold` in config (try 0.65)
- Increase `max_results` in config
- Ensure tasks have been indexed (check with `memory stats`)

### Memory Not Auto-Indexing

Check that `learning.auto_store: true` in config. Manually index with:
```bash
mehr memory index --task <id>
```

## Web UI

Prefer a visual interface? See [Web UI: Memory](/web-ui/memory.md).

## See Also

- [Configuration Guide](/configuration/index.md) - Memory settings in config.yaml
- [Advanced: Semantic Memory](/advanced/semantic-memory.md) - Deep dive on memory architecture
- [storage Reference](/reference/storage.md) - Where memory data is stored
