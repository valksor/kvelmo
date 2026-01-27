# mehr export

Export a queue task to a markdown file for use with the standard workflow.

## Synopsis

```bash
mehr export --task <queue>/<task-id> --output <file> [flags]
```

## Description

The `export` command converts a queue task into a markdown file that can be used with the standard mehrhof workflow. The exported file includes:

- **YAML frontmatter** with title, labels, priority
- **Task description** as the main content
- **All accumulated notes** appended as context

This bridges the gap between quick task capture and the full workflow, allowing you to start with a quick task and transition to a proper specification when ready.

## Flags

| Flag        | Description                              |
| ----------- | ---------------------------------------- |
| `--task`    | Queue task ID (format: `<queue-id>/<task-id>`) (required) |
| `--output`  | Output file path (required)               |

## Examples

### Basic Export

```bash
mehr export --task=quick-tasks/task-1 --output task.md
```

Exports the task to `task.md` in the current directory.

### To Specifications Directory

```bash
mehr export --task=quick-tasks/task-1 --output specs/user-search.md
```

Exports to a specifications directory.

### Use Exported File

```bash
# Export the task
mehr export --task=quick-tasks/task-1 --output feature.md

# Start standard workflow with exported file
mehr start file:feature.md
mehr plan
mehr implement
mehr finish
```

## What Happens

1. **Task Loading**
   - Task loaded from queue
   - All notes loaded
   - Current labels and priority preserved

2. **Markdown Generation**
   - YAML frontmatter with metadata
   - Task description as body
   - Notes section with all accumulated notes

3. **File Writing**
   - File created at specified path
   - Existing files are overwritten
   - Permissions set to `0644`

## Exported File Format

The exported file follows the standard task format:

```markdown
---
title: Fix typo in README
labels:
  - documentation
  - typo-fix
priority: 1
---

# Fix typo in README

## Description

The word "Installation" is misspelled as "Installaton" in the README getting started section on line 42.

## Context

This typo affects the first impression of the project and should be fixed to maintain professionalism.

---

## Notes

### 2025-01-15 10:30:00

Found this during the README review. The typo is in the first paragraph of the Getting Started section.

### 2025-01-15 11:00:00

Double-checked - it appears twice in the document.
```

## Workflow Examples

### Quick Task to Full Workflow

```bash
# 1. Capture quickly
mehr quick "implement user search with filters"

# 2. Add requirements via notes
mehr note --task=quick-tasks/task-1 "filter by name, email, status"
mehr note --task=quick-tasks/task-1 "support pagination"
mehr note --task=quick-tasks/task-1 "add fuzzy name matching"

# 3. Optimize with AI
mehr optimize --task=quick-tasks/task-1

# 4. Export to proper spec
mehr export --task=quick-tasks/task-1 --output specs/user-search.md

# 5. Start full workflow
mehr start file:specs/user-search.md
```

### Batch Export

```bash
# Export multiple tasks to individual files
mehr export --task=quick-tasks/task-1 --output specs/auth.md
mehr export --task=quick-tasks/task-2 --output specs/search.md
mehr export --task=quick-tasks/task-3 --output specs/api.md

# Work through them using standard workflow
mehr start file:specs/auth.md
```

### Quick Iteration Loop

```bash
# Capture and export in one session
mehr quick "add dark mode toggle"
mehr note --task=quick-tasks/task-1 "use CSS variables"
mehr export --task=quick-tasks/task-1 --output tasks/dark-mode.md

# Immediately start work
mehr start file:tasks/dark-mode.md
```

## Integration with Start Command

The exported file can be used directly with the `start` command:

```bash
# Export
mehr export --task=quick-tasks/task-1 --output feature.md

# Start using file: reference
mehr start file:feature.md

# Alternative: start using queue reference directly
# (exports internally and starts)
mehr start queue:quick-tasks/task-1
```

## File Organization

Common patterns for organizing exported files:

```
project/
├── specs/           # Full specifications
│   ├── auth.md
│   └── search.md
├── tasks/           # Smaller tasks
│   ├── typo-fix.md
│   └── config.md
└── backlog/         # Future work
    └── ideas.md
```

## See Also

- [quick](quick.md) - Create quick tasks
- [optimize](optimize.md) - AI optimize before exporting
- [start](start.md) - Start from exported file
- [Task Format](../reference/task-format.md) - Task file format reference
