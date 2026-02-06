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

| Feature                  | Location                             |
|--------------------------|--------------------------------------|
| **Memory Search**        | Tools → Memory                       |
| **Memory Configuration** | Settings → Advanced → Memory System  |

Memory search results appear automatically in the dashboard when starting similar tasks.

## Using Memory in the Web UI

### Searching Memory

Find similar past tasks:

1. Go to **Tools → Memory** from the navigation
2. Enter your search query in the search box
3. Optionally filter by document type:
   - **Code** — Code changes from past tasks
   - **Specifications** — Implementation plans
   - **Sessions** — Agent conversation logs
   - **Solutions** — Fixes and corrections
   - **Decisions** — Architectural decisions
   - **Errors** — Past errors and resolutions
4. Set a results limit (default: 5)
5. Click **"Search"**

Results show similar documents with similarity scores (percentage match).

### Understanding Results

Each search result shows:

| Field         | Description                            |
|---------------|----------------------------------------|
| Similarity    | Score from 0-1 (higher = more similar) |
| Document Type | What kind of content                   |
| Task ID       | Source task                            |
| Excerpt       | Relevant content snippet               |
| Date          | When the document was created          |

**Similarity Threshold:** Only results above 0.8 (default) are shown. Adjust this in settings.

### Configuring Memory

Memory is configured in **Settings → Advanced → Memory System**:

```yaml
memory:
  enabled: true
  vector_db:
    backend: chromadb
    connection_string: ./.mehrhof/vectors
    collection: mehr_task_memory
    embedding_model: default    # or "onnx" for semantic embeddings
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
- **embedding_model** - `default` (hash-based) or `onnx` (semantic neural embeddings)
- **similarity_threshold** - Lower = more results (try 0.65 if none found)
- **max_results** - How many matches to return
- **auto_store** - Automatically index completed tasks
- **suggest_similar** - Show similar tasks when starting new work

### Embedding Models

Mehrhof supports two embedding approaches:

| Model       | How It Works          | Best For                                          |
|-------------|-----------------------|---------------------------------------------------|
| **default** | Hash-based (SHA256)   | Fast, offline, exact/near-exact matches           |
| **onnx**    | Neural network (ONNX) | True semantic similarity ("cat" matches "kitten") |

#### Enabling Semantic Embeddings

For better semantic search, enable ONNX embeddings:

```yaml
memory:
  vector_db:
    embedding_model: onnx
    onnx:
      model: all-MiniLM-L6-v2    # 22MB, good quality (default)
```

**Available models:**

| Model             | Size | Quality            |
|-------------------|------|--------------------|
| all-MiniLM-L6-v2  | 22MB | Good (recommended) |
| all-MiniLM-L12-v2 | 33MB | Better             |

**First-run download**: ONNX models download automatically on first use to `~/.valksor/mehrhof/models/`. No manual setup required.

**Switching models**: Changing from `default` to `onnx` (or vice versa) invalidates existing vectors. You must clear memory after switching models.

### Clearing Memory

To remove all stored memory, use the CLI. See [CLI: memory](/cli/memory.md) for the clear command.

**Warning:** Clearing memory cannot be undone.

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

**With ONNX embeddings** (true semantic similarity):
- "authentication bug" finds "login fix"
- "database schema" finds "model changes"
- "race condition" finds "concurrency fix"
- "cat" matches "kitten" and "feline"

**With default embeddings** (hash-based):
- Works best for exact or near-exact text matches
- Good for finding specific error messages or function names

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

---

## Also Available via CLI

Prefer working from the terminal? See [CLI: memory](/cli/memory.md) for search, indexing, and management options.

## Troubleshooting

### No Results Found

If searches return no results:
- Lower the `similarity_threshold` in settings (try 0.65)
- Increase `max_results` in settings
- Ensure tasks have been indexed (check stats)

### Memory Not Auto-Indexing

Check that `auto_store: true` in settings. You can manually index tasks from the CLI if needed.

### Similar Suggestions Not Appearing

Check that `suggest_similar: true` in settings. This enables automatic suggestions on the dashboard.
