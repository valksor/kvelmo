# Linking

Bidirectional linking system for creating a knowledge graph across specifications, notes, sessions, and decisions.

## Overview

The linking system enables Logseq-style `[[reference]]` syntax with automatic backlink tracking. This creates an emergent knowledge graph where:

- **Forward links** show what content references
- **Backlinks** show what references content
- **Context** is preserved for understanding connections
- **Name-based references** provide human-readable links

## Key Concepts

### References

References are the fundamental unit of linking. They use the `[[...]]` syntax in markdown:

```markdown
# Typed reference
[[spec:1]]

# Name-based reference
[[Authentication Spec]]

# With alias
[[spec:1|see authentication]]
```

### Entity Types

| Type | Description | Entity ID Format |
|------|-------------|------------------|
| `spec` | Specification | `spec:task-id:N` |
| `note` | Task notes | `note:task-id:notes` |
| `session` | Session log | `session:task-id:timestamp` |
| `decision` | Decision record | `decision:task-id:id` |
| `solution` | Solution record | `solution:task-id:id` |
| `error` | Error record | `error:task-id:id` |

### Links

A **link** represents a directional connection between two entities:

```go
type Link struct {
    Source    string    // Source entity ID
    Target    string    // Target entity ID
    Context   string    // Surrounding text (up to 200 chars)
    CreatedAt time.Time // When the link was created
}
```

Links are created when:
- Specifications are saved
- Notes are appended
- Sessions are saved

### Bidirectional Index

The link index maintains both directions:

| Index Type | Purpose | Example |
|------------|---------|---------|
| **Forward** | What entity X references | `spec:task-123:1 → spec:task-123:2` |
| **Backward** | What references entity X | `note:task-123:notes → spec:task-123:1` |

This enables efficient queries in both directions:
- **Outgoing links**: Show related content
- **Backlinks**: Show dependencies and usage

## Architecture

### Storage Model

Links use a hybrid storage approach:

| Storage | Location | Purpose |
|---------|----------|---------|
| **JSON Index** | `~/.valksor/mehrhof/workspaces/<project>/links/index.json` | O(1) link lookups |
| **Name Registry** | `~/.valksor/mehrhof/workspaces/<project>/links/names.json` | Human name → ID mapping |

The index uses atomic writes (temp file + rename) to prevent corruption.

### Parsing Pipeline

```
Markdown Content
    ↓
Parse with regex [[.*?]]
    ↓
Extract: type, task-id, id, name, alias, position
    ↓
Create: Link(source, target, context, created_at)
    ↓
Index: Add to forward/backward indices
    ↓
Save: Atomic write to index.json
```

### Name Resolution

Name-based references are resolved via the **Name Registry**:

1. **Specifications**: Extracted from YAML frontmatter or first heading
2. **Sessions**: Built from session type + timestamp
3. **Decisions**: Parsed from `decision:` prefix in notes
4. **Notes**: No human names (use entity ID)

Resolution order:
1. Check name registry
2. If not found, scan workspace (specs → sessions → decisions)
3. Register found names for future lookups

### Query System

Links support structured queries beyond semantic search:

| Query Type | Description | Example |
|------------|-------------|---------|
| **FindLinks** | Search with filters | `FindLinks(From("spec:123:1"), OfType("decision"))` |
| **FindBacklinks** | Reverse direction lookup | `FindBacklinks("spec:123:1")` |
| **FindOrphans** | Entities with no outgoing links | `FindOrphans()` |
| **FindPath** | Shortest path between entities | `FindPath("spec:123:1", "decision:456:abc")` |
| **FindConnectedEntities** | Reachability analysis | `FindConnectedEntities("spec:123:1", 3)` |

## Advanced Features

### Context Extraction

Each link preserves surrounding context:

```markdown
We need to implement [[spec:2|authentication]] before
proceeding to the API endpoints. The auth flow should
handle JWT tokens with proper validation.
```

Link context: `"We need to implement [[spec:2|authentication]] before proceeding to..."`

### Task-Scoped References

Within a task, you can use shorter references:

```markdown
# Fully qualified
[[spec:task-abc123:1]]

# Task-scoped (uses active task)
[[spec:1]]  # Resolves to spec:task-abc123:1
```

Task-scoped references:
- Reduce verbosity
- Automatically resolve to active task
- Useful for task-local linking

### Mutual Links

Entities that link to each other create mutual connections:

```markdown
# spec-1.md
See [[spec:2]] for API design.

# spec-2.md
Extends [[spec:1]] authentication flow.
```

Detected via: `FindMutualLinks(entityID)`

### Circular Paths

Circular references are detected and can be analyzed:

```
spec:1 → spec:2 → spec:3 → spec:1 (cycle)
```

Detected via: `FindCircularPaths(entityID, maxDepth)`

## Integration Points

### With Specifications

Specifications automatically create links when:

1. Content is saved to `work/<task>/specifications/specification-N.md`
2. Title extracted from YAML frontmatter or first heading
3. `[[references]]` parsed from content
4. Links created and indexed

### With Notes

Notes are indexed when appended:

1. Content added to `work/<task>/notes.md`
2. Entity ID: `note:<task-id>:notes`
3. `[[references]]` parsed and indexed
4. Decisions extracted from `decision:` prefix

### With Sessions

Sessions are indexed when saved:

1. Session YAML saved to `work/<task>/sessions/<timestamp>.yaml`
2. Entity ID: `session:<task-id>:<timestamp>`
3. `[[references]]` parsed from session content

## Performance Considerations

### Index Size

For large workspaces:
- Index scales with O(N) where N = total links
- Forward/backward lookups are O(1)
- Memory usage depends on link count

### Rebuild Performance

Rebuilding the index:
- Scans all specifications, notes, sessions
- O(N) where N = total content size
- Atomic write prevents corruption
- Can be run on-demand or after migrations

### Query Performance

| Operation | Complexity | Notes |
|------------|------------|-------|
| FindLinks | O(L) | L = links from source |
| FindBacklinks | O(L) | L = links to target |
| FindPath | O(V+E) | V = entities, E = edges (BFS) |
| FindOrphans | O(V) | V = entities |

## Best Practices

### Naming Conventions

Use clear, consistent names for specifications:

```yaml
# specification-1.md
---
title: Authentication Flow
---
```

This enables: `[[Authentication Flow]]` references

### Decision Recording

Mark decisions clearly in notes:

```markdown
## Design Decisions

decision: Use Redis for session caching with 5-minute TTL.
Rationale: Fast reads, simple invalidation.

decision: Store JWT tokens in HTTP-only cookies.
Rationale: Prevents XSS attacks.
```

### Link Granularity

Link at appropriate granularity:

- ✅ **Good**: `[[Authentication Flow]]` - Concept-level link
- ✅ **Good**: `[[spec:1|login section]]` - Specific section
- ❌ **Avoid**: Over-linking to generic terms

### Circular References

Avoid unintended circular dependencies:

```
spec:1 → spec:2 → spec:3 → spec:1 (cycle)
```

Detected via: `FindCircularPaths()`

## See Also

- [CLI: links command](../cli/links.md) - CLI usage
- [Web UI: Links](../web-ui/links.md) - Web UI usage
- [Configuration](../configuration/index.md) - Links settings
