# Semantic Memory System

Store and search vector embeddings of code changes, decisions, and solutions for semantic similarity matching.

## Architecture

The memory system consists of four main components:

1. **Embedding Model** - Converts text to vector embeddings
2. **Vector Store** - Stores and searches embeddings
3. **Indexer** - Automatically indexes completed tasks
4. **Memory Tool** - Provides search and context to agents

```
┌─────────────┐
│ Completed   │
│   Task      │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│   Indexer   │────▶│ Vector Store │
└─────────────┘     └──────┬───────┘
                          │
                ┌─────────┴─────────┐
                ▼                   ▼
         ┌──────────┐         ┌──────────┐
         │  Search  │         │  Store   │
         └──────────┘         └──────────┘
```

## Vector Embeddings

Embeddings are numerical representations of text that capture semantic meaning:

- **Dimension**: 1536 (default)
- **Similarity**: Measured using cosine similarity (0-1 range)
- **Models**: SHA256 hash-based embedding

### Embedding Models

**Hash-based embedding** uses SHA256 for deterministic local embeddings without requiring external APIs.

## Document Types

The memory system stores different document types:

| Type            | Description             | Content                               |
|-----------------|-------------------------|---------------------------------------|
| `code_change`   | Git diffs               | Unified diff of code changes          |
| `specification` | Implementation plans    | Specification file content            |
| `session`       | Agent conversations     | Full transcript of agent interactions |
| `solution`      | Fixes and corrections   | Problem → solution pairs              |
| `decision`      | Architectural decisions | Decision records and rationale        |

### Document Structure

```go
type Document struct {
    ID        string                 // Unique document ID
    TaskID    string                 // Source task ID
    Type      DocumentType           // Document type
    Content   string                 // Text content
    Metadata  map[string]interface{} // Additional metadata
    Embedding []float32              // Vector embedding
    CreatedAt time.Time              // Creation timestamp
    Tags      []string               // Searchable tags
}
```

## Vector Storage

### Type-Safe Filtering

The memory system supports type-safe filtering by document type:

```go
// Filter by single type
filter := map[string]interface{}{"type": "code_change"}

// Filter by multiple types
filter := map[string]interface{}{"type": []string{"code_change", "solution"}}
```

The filter matching function:
- Handles both string and `[]string` type filters
- Performs safe type assertions to prevent panics
- Falls back to string conversion for unknown types

### ChromaDB Backend

The default backend is ChromaDB (in-memory mode):

```yaml
memory:
  vector_db:
    backend: chromadb
    connection_string: ./.mehrhof/vectors  # Local storage
    collection: mehr_task_memory
```

**Features**:
- Persistent local storage
- Cosine similarity search
- Metadata filtering
- No external dependencies

### Storage Location

```
./.mehrhof/
  └── vectors/
      └── chroma/
          └── mehr_task_memory/
              ├── chroma.sqlite3     # Metadata index
              └── *.bin              # Vector data
```

## Similarity Search

### Cosine Similarity

Cosine similarity measures the angle between two vectors:

```
similarity = cos(θ) = (A · B) / (|A| × |B|)

Range: 0 (no similarity) to 1 (identical)
```

### Search Process

1. **Query Embedding** - Convert search query to vector
2. **Vector Search** - Find nearest vectors in database
3. **Score Filtering** - Filter by similarity threshold
4. **Type Filtering** - Filter by document type
5. **Ranking** - Sort by similarity score

### Search Options

```yaml
memory:
  search:
    similarity_threshold: 0.8  # Minimum similarity (0-1)
    max_results: 5             # Maximum results to return
    include_code: true         # Include code changes
    include_specs: true        # Include specifications
    include_sessions: false    # Exclude sessions
```

## Automatic Indexing

The indexer automatically processes completed tasks:

### Indexing Process

```
Task Completion
       │
       ▼
┌──────────────────┐
│  Specifications  │ → Generate embeddings → Store
└──────────────────┘
       │
       ▼
┌──────────────────┐
│  Code Changes    │ → Generate diff → Embed → Store
└──────────────────┘
       │
       ▼
┌──────────────────┐
│  Session Logs    │ → Concatenate → Embed → Store
└──────────────────┘
```

### Indexer Configuration

```yaml
memory:
  learning:
    auto_store: true               # Automatically index tasks
    learn_from_corrections: true    # Store user corrections
    suggest_similar: true           # Auto-suggest to agents
```

## Agent Integration

### Memory Tool

The memory tool provides agents with context from similar past tasks:

```yaml
agent:
  instructions: |
    Context from similar past tasks is available.
    Use historical solutions to inform your approach.
```

### Prompt Augmentation

When a task starts, Mehrhof automatically:

1. Searches for similar past tasks
2. Formats the results as context
3. Augments the agent prompt with relevant context

**Example Augmented Prompt**:

```
[Original Task Description]

## Relevant Context from Similar Tasks

### Task abc123 (Similarity: 85%)
**Type**: specification
**Content**:
The authentication system should use JWT tokens stored in
httpOnly cookies to prevent XSS attacks...

### Task def456 (Similarity: 78%)
**Type**: solution
**Content**:
Problem: Session fixation attacks
Solution: Implement session regeneration after login...

Use this context to inform your approach. These are past
solutions that worked for similar problems.
```

## Learning from Corrections

When you correct an agent's output, it can be stored as a solution:

```yaml
memory:
  learning:
    learn_from_corrections: true
```

**Stored as**:
- Document type: `solution`
- Content: Problem → Solution pair
- Tags: `["solution", "fix", "learned"]`

## Performance

### Indexing Performance

| Operation          | Time  | Notes                |
|--------------------|-------|----------------------|
| Store document     | <10ms | Hash-based embedding |
| Generate embedding | <5ms  | SHA256-based         |
| Search (1000 docs) | <50ms | In-memory ChromaDB   |

### Storage Requirements

| Documents | Storage | Notes                   |
|-----------|---------|-------------------------|
| 100       | ~5MB    | Depends on content size |
| 1000      | ~50MB   | Typical project         |
| 10000     | ~500MB  | Large project           |

### Optimization Tips

1. **Set retention limits** to prune old documents
2. **Use type filtering** to reduce search space
3. **Tune similarity threshold** to balance precision/recall
4. **Consider batch indexing** for large imports

## Troubleshooting

### No Search Results

**Problem**: Search returns no results

**Solutions**:
- Lower `similarity_threshold` (try 0.65)
- Increase `max_results`
- Verify documents are indexed: `mehr memory stats`
- Check document types match filter

### Poor Quality Matches

**Problem**: Search results aren't relevant

**Solutions**:
- Improve document content quality
- Add more descriptive tags
- Tune similarity threshold upward

### Memory Not Auto-Indexing

**Problem**: Tasks aren't being indexed

**Solutions**:
- Check `auto_store: true` in config
- Verify memory is enabled
- Check for indexing errors in logs
- Manually index: `mehr memory index --task <id>`

## See Also

- [CLI: memory](../cli/memory.md) - Memory commands reference
- [Configuration Guide](../configuration/index.md) - Memory settings
- [Vector Databases](https://en.wikipedia.org/wiki/Vector_database) - General concepts
