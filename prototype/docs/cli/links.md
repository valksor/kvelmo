# mehr links

Manage bidirectional links between specs, notes, sessions, and decisions.

## Synopsis

```bash
mehr links <subcommand> [flags]
```

## Description

Mehrhof's links system provides Logseq-style bidirectional linking between:
- **Specifications** - Implementation plans
- **Notes** - Task notes and documentation
- **Sessions** - Agent conversation logs
- **Decisions** - Recorded decisions (marked with `decision:` prefix in notes)

This creates a knowledge graph where:
- **Outgoing links** show what an entity references
- **Backlinks** show what references an entity
- **Context** is preserved (surrounding text up to 200 chars)

## Configuration

Links are enabled by default. Configure in `.mehrhof/config.yaml`:

```yaml
links:
  enabled: true           # Enable link system (default: true)
  auto_index: true        # Auto-index on save (default: true)
  case_sensitive: false   # Name matching (default: false)
  max_context_length: 200 # Context chars for links (default: 200)
```

### When Links Are Disabled

Links are not parsed, indexed, or queried when disabled. The index file is preserved and can be rebuilt when re-enabled.

## Commands

### links list

Show outgoing links from an entity:

```bash
mehr links list [entity]
```

If no entity is specified, lists all entities with outgoing links.

**Entity IDs** use the format: `type:task-id:id`

| Entity Type   | Format                      | Example                               |
|---------------|-----------------------------|---------------------------------------|
| Specification | `spec:task-id:N`            | `spec:abc123:1`                       |
| Note          | `note:task-id:notes`        | `note:abc123:notes`                   |
| Session       | `session:task-id:timestamp` | `session:abc123:2024-01-29T10:00:00Z` |
| Decision      | `decision:task-id:id`       | `decision:abc123:cache-strategy`      |

**Examples:**
```bash
# List all entities with links
mehr links list

# Show outgoing links from spec 1
mehr links list spec:abc123:1

# Show outgoing links from a note
mehr links list note:abc123:notes
```

### links backlinks

Show incoming links (backlinks) to an entity:

```bash
mehr links backlinks <entity>
```

This shows what references this entity—useful for understanding dependencies and impact.

**Examples:**
```bash
# Show what references spec 1
mehr links backlinks spec:abc123:1

# Find all references to a decision
mehr links backlinks decision:abc123:cache-strategy
```

### links search

Find entities by human-readable name:

```bash
mehr links search <name>
```

Search is case-insensitive by default and supports partial matching.

**Examples:**
```bash
# Search for authentication-related entities
mehr links search "authentication"

# Find a specific spec
mehr links search "JWT"

# Search for decisions
mehr links search "cache"
```

### links stats

Show link graph statistics:

```bash
mehr links stats
```

**Example output:**
```
📊 Link Graph Statistics

Total links:     127
Total sources:   45
Total targets:   38
Orphan entities: 3

Most linked entities:
  1. spec:abc123:2 [42 total links]
  2. decision:abc123:cache-strategy [15 total links]
  3. note:abc123:notes [12 total links]
```

### links rebuild

Rebuild the link index from workspace content:

```bash
mehr links rebuild
```

This rescans:
- All specification files
- All task notes
- All session logs

Use after:
- Manual edits to files
- Migration from another system
- Index corruption

**Example output:**
```
🔄 Rebuilding link index from workspace content...

✓ Index rebuilt successfully!
  Total links: 127
  Total entities: 45
  Total targets: 38
```

## Reference Syntax

Links use the `[[...]]` syntax in markdown:

### Typed References

```markdown
# Fully qualified
[[spec:task-id:1]]
[[session:task-id:2024-01-29T10:30:00Z]]

# Task-scoped (uses active task)
[[spec:1]]
[[decision:cache-strategy]]
```

### Name-Based References

```markdown
# Human-readable names
[[Authentication Spec]]
[[JWT middleware decision]]

# With display alias
[[spec:1|see authentication flow]]
```

### In Notes

```markdown
# Implementation notes

See [[Authentication Spec]] for the login flow details.

For caching decisions, refer to [[decision:cache-strategy]].

Related to [[spec:2|API design]].
```

## How It Works

### Automatic Linking

When you save content that contains `[[references]]`:

1. Parser extracts all references with positions
2. Links are created between source and target entities
3. Context (surrounding text) is preserved
4. Index is updated automatically

### Name Resolution

Name-based references work via the **Name Registry**:

- Specifications: Extracted from YAML frontmatter `title` or first heading
- Sessions: Built from session type + timestamp
- Decisions: Extracted from `decision:` prefixed lines in notes

### Index Storage

Links are stored in:
```
~/.valksor/mehrhof/workspaces/<project>/links/
├── index.json   # Forward/backward link mappings
└── names.json   # Name registry for human-readable names
```

The index uses atomic writes (temp file + rename) to prevent corruption.

## Examples

### Creating Linked Specifications

```markdown
# specification-1.md
---
title: Authentication Flow
---

## Login Process

Users authenticate via JWT tokens. See [[spec:2|API Endpoints]] for token refresh.

# specification-2.md
---
title: API Endpoints
---

## Token Management

Tokens are issued during [[Authentication Flow]] and refreshed every hour.
```

Result: Both specs are cross-linked bidirectionally.

### Linking to Decisions

```markdown
# notes.md

## Decision: Cache Strategy

decision: Use Redis for session caching with 5-minute TTL.

## Implementation

Follow [[decision:cache-strategy]] for all session data.
```

Result: Decision is registered and linkable by name.

### Finding What Depends on a Spec

```bash
# Show backlinks to understand dependencies
mehr links backlinks spec:abc123:1

# Output shows all notes/specs referencing this spec
```

Useful before making breaking changes.

### Discovering Related Work

```bash
# Search for all authentication-related entities
mehr links search "authentication"

# See full link graph for an entity
mehr links list spec:abc123:1
```

## Web UI

Prefer a visual interface? See [Web UI: Links](/web-ui/links.md).

## See Also

- [Configuration Guide](/configuration/index.md) - Links settings in config.yaml
- [Linking Concepts](/concepts/linking.md) - Deep dive on linking architecture
- [CLI Reference](/cli/index.md) - All CLI commands
