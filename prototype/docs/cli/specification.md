# mehr specification

View and manage task specifications.

## Synopsis

```bash
mehr specification <subcommand> [flags]
```

## Description

Specifications are detailed implementation plans created by the AI during the planning phase. Each specification contains what needs to be built and how to implement it.

The `specification` command lets you view specification content with metadata like status, component, and timestamps.

## Commands

### specification view

Display the full content of a specification with metadata.

```bash
mehr specification view <number> [flags]
```

**Flags:**

| Flag       | Short | Type   | Default | Description                      |
|------------|-------|--------|---------|----------------------------------|
| `--number` | `-n`  | int    | 0       | Specification number             |
| `--all`    | `-a`  | bool   | false   | View all specifications          |
| `--output` | `-o`  | string | ""      | Save to file instead of printing |

The specification number can be provided as a positional argument or via the `--number` flag.

### specification diff

Display a read-only unified diff for a file listed in a specification's `implemented_files`.

```bash
mehr specification diff <number> --file <path> [flags]
```

**Flags:**

| Flag        | Short | Type   | Default | Description                                  |
|-------------|-------|--------|---------|----------------------------------------------|
| `--number`  | `-n`  | int    | 0       | Specification number                          |
| `--file`    | `-f`  | string | ""      | Implemented file path (required)              |
| `--context` | `-c`  | int    | 3       | Number of context lines in the unified diff   |
| `--output`  | `-o`  | string | ""      | Save diff to file instead of printing         |

## Examples

### View a specific specification

```bash
mehr specification view 1
```

Output:

```
─ Specification 1: Authentication Flow

Status:     ✅ completed
Component:  internal/auth
Created:    2026-01-30 10:15
Completed:  2026-01-30 10:42

────────────────────────────────────────────────────────────────────────────────

## Plan
1. Add `internal/auth/` package for authentication
2. Implement JWT token generation and validation
...
```

### View all specifications

```bash
mehr specification view --all
```

Displays all specifications separated by dividers.

### Save to file

```bash
# Save a single specification
mehr specification view 1 -o spec.md

# Save all specifications (creates spec-1.md, spec-2.md, etc.)
mehr specification view --all -o spec.md
```

### View implemented file diff

```bash
mehr specification diff 1 --file web/src/components/task/SpecificationsList.tsx
```

### Save diff to file

```bash
mehr specification diff 1 --file internal/server/handlers.go -o spec.diff
```

### No specifications yet

```bash
mehr specification view 1
```

If no specifications exist, the command suggests running `mehr plan`:

```
No specifications yet. Run 'mehr plan' to create them.
```

### Specification not found

If the requested number does not exist, available specifications are listed:

```
Specification 5 not found. Available specifications:
  ✅ specification-1: Authentication Flow [completed]
  🔄 specification-2: API Endpoints [in-progress]
```

## Web UI

Prefer a visual interface? See [Web UI: Implementing](/web-ui/implementing.md) for viewing implemented file diffs from the Specifications section.

## See Also

- [plan](plan.md) - Create specifications
- [implement](implement.md) - Implement specifications
- [status](status.md) - View task state including specification count
