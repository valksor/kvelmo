# Links

Explore the knowledge graph of linked content between specifications, notes, sessions, and decisions. Links provide Logseq-style bidirectional linking with automatic backlink tracking.

## Overview

Mehrhof's links system creates a knowledge graph where:

- **Outgoing links** show what an entity references
- **Backlinks** show what references an entity
- **Context** is preserved (surrounding text from the source)
- **Name-based references** enable human-readable links like `[[Authentication Spec]]`

## Accessing Links in the Web UI

Links features are available through:

| Feature | Location |
|---------|----------|
| **Search entities** | Navigate to **Links** in the sidebar |
| **View statistics** | Links → Statistics panel |
| **Rebuild index** | Links → Rebuild button |
| **Browse all links** | Links → Search results |

## Using Links in the Web UI

### Searching for Entities

Find entities by name:

1. Navigate to **Links** in the sidebar
2. Enter a search query in the search box
3. Click **Search** or press Enter

Search is case-insensitive and supports partial matching. Results show:
- Entity ID
- Type (spec, session, decision, note)
- Task ID (if applicable)
- Link counts (outgoing/incoming)

### Viewing Graph Statistics

See overall link graph statistics:

1. Navigate to **Links**
2. View the **Graph Statistics** panel on the right

Statistics include:
- **Total links** - Number of links in the graph
- **Total sources** - Entities with outgoing links
- **Total targets** - Entities with incoming links
- **Most connected** - Top 10 entities by total links

### Rebuilding the Link Index

Rebuild the index from all workspace content:

1. Navigate to **Links**
2. Click the **Rebuild** button in the sidebar
3. Wait for confirmation

Use rebuild after:
- Manual file edits
- Content migration
- Index corruption

## Link Syntax

Links use the `[[...]]` syntax in markdown content:

### Typed References

```markdown
# Fully qualified (specific task)
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

### Examples in Content

```markdown
# Implementation notes

See [[Authentication Spec]] for the login flow details.

For caching decisions, refer to [[decision:cache-strategy]].

Related to [[spec:2|API design]].
```

## How Links Work

### Automatic Link Creation

When content is saved:

1. Parser extracts all `[[references]]` with positions
2. Links are created between source and target entities
3. Context (surrounding text) is preserved (up to 200 chars)
4. Index is updated automatically

### Name Resolution

Name-based references work via the **Name Registry**:

| Entity Type | Name Source | Example |
|-------------|-------------|---------|
| **Specifications** | YAML frontmatter `title` or first heading | `Authentication Flow` |
| **Sessions** | Session type + timestamp | `planning session` |
| **Decisions** | `decision:` prefix in notes | `cache-strategy` |
| **Notes** | Entity ID (no human names) | `note:abc123:notes` |

### Link Context

Each link preserves context from the source content:

- **Up to 200 characters** of surrounding text
- Shows where the link was used
- Useful for understanding why entities are connected

## Common Workflows

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

**Result:** Both specs are cross-linked bidirectionally.

### Finding What Depends on Content

1. Navigate to **Links**
2. Search for the entity name
3. View the entity's **incoming links** count
4. Backlinks show what references this entity

Useful for:
- Understanding impact before changes
- Finding dependent code
- Auditing knowledge graph

### Discovering Related Work

1. Search for relevant terms (e.g., "authentication")
2. Browse results to find related entities
3. Check link counts to find highly-connected content

## CLI Equivalent

See [`mehr links`](../cli/links.md) for CLI usage.

| CLI Command | Web UI Action |
|-------------|---------------|
| `mehr links list [entity]` | Browse/search entities |
| `mehr links backlinks <entity>` | View entity details |
| `mehr links search <name>` | Search box |
| `mehr links stats` | Statistics panel |
| `mehr links rebuild` | Rebuild button |

## Link Storage

Link data is stored in:
```
~/.valksor/mehrhof/workspaces/<project>/links/
├── index.json   # Forward/backward link mappings
└── names.json   # Name registry for human-readable names
```

## Troubleshooting

### Links Not Being Created

Check that links are enabled in settings:
1. Navigate to **Settings**
2. Verify **Links → Enabled** is checked

### Search Not Finding Entities

- Try partial names instead of exact matches
- Check that the entity name is correctly formatted
- Use `mehr links rebuild` to refresh the index

### Backlinks Not Showing

- Ensure the linking content has been saved
- Check that the source content contains valid `[[reference]]` syntax
- Verify links are enabled in settings

### Index Corruption

If the link index appears corrupted:
1. Navigate to **Links**
2. Click **Rebuild** to recreate from source content
3. All links will be reparsed from markdown files
