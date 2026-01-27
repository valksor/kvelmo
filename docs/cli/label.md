# mehr label

Manage task labels for organization, filtering, and grouping.

## Synopsis

```bash
mehr label <command> [flags]
```

## Description

The `label` command manages custom labels on tasks. Labels are free-form strings that can be used for:
- Filtering tasks in `mehr list`
- Grouping related work
- Adding metadata for reporting

## Commands

| Command          | Description                        |
| ---------------- | ---------------------------------- |
| `label add`      | Add labels to a task               |
| `label remove`   | Remove labels from a task          |
| `label set`      | Replace all labels on a task       |
| `label list`     | Show labels for a task             |

## Examples

### Add Labels

```bash
mehr label add a1b2c3d4 priority:high type:bug team:backend
```

Output:
```
Added 3 label(s) to task a1b2c3d4
```

### Remove Labels

```bash
mehr label remove a1b2c3d4 priority:high
```

Output:
```
Removed 1 label(s) from task a1b2c3d4
```

### Set Labels (Replace All)

```bash
# Replace all existing labels
mehr label set a1b2c3d4 priority:critical type:feature
```

Output:
```
Set 2 label(s) on task a1b2c3d4
```

```bash
# Clear all labels
mehr label set a1b2c3d4
```

Output:
```
Cleared all labels from task a1b2c3d4
```

### List Labels

```bash
mehr label list a1b2c3d4
```

Output:
```
Labels for Add user authentication (a1b2c3d4):
  - priority:high
  - type:bug
  - team:backend
```

If the task has no labels:
```
Labels for a1b2c3d4:
  (no labels)
```

### Filter by Label

```bash
# Exact match
mehr list --label priority:high

# Match any (OR logic)
mehr list --label-any priority:high type:bug

# Tasks without labels
mehr list --no-label
```

## Label Patterns

Labels are free-form, but common conventions include:

| Category    | Examples                                      | Purpose           |
| ----------- | --------------------------------------------- | ----------------- |
| **Priority** | `priority:critical`, `priority:high`, `priority:medium`, `priority:low` | Task urgency     |
| **Type**     | `type:bug`, `type:feature`, `type:refactor`, `type:docs`, `type:test` | Work category    |
| **Team**     | `team:frontend`, `team:backend`, `team:devops` | Team ownership    |
| **Status**   | `status:blocked`, `status:in-review`           | Workflow status   |
| **Component** | `component:auth`, `component:database`        | System affected   |
| **Sprint**   | `sprint:2024-q1`, `sprint:backlog`            | Sprint planning   |

## Web UI

Labels appear as colored badges on task cards. Use the **+ Add** button to:
- Add new labels (with autocomplete)
- Remove labels (click × on badge)

Colors are hash-based (consistent per label name).

## See Also

- [list](list.md) - List and filter tasks
- [status](status.md) - View task details
- [Web UI: Dashboard](../web-ui/dashboard.md#labels)
