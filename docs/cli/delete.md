# delete

Delete a task from a queue.

## Usage

```bash
mehr delete --task <queue>/<task-id>
```

## Description

Permanently removes a task and its associated notes from a queue. This action cannot be undone.

## Flags

| Flag     | Required | Description                                    |
| -------- | -------- | ---------------------------------------------- |
| `--task` | Yes      | Task reference in format `<queue-id>/<task-id>` |

## Examples

### Delete a Quick Task

```bash
mehr delete --task=quick-tasks/task-1
```

### Delete from a Project Queue

```bash
mehr delete --task=my-project/task-5
```

### View Tasks Before Deleting

```bash
# List all tasks to find the task ID
mehr list

# Delete the specific task
mehr delete --task=quick-tasks/task-3
```

## Output

On success:

```
✓ Deleted task task-1
  Queue: quick-tasks
  Title: Implement user authentication

Next steps:
  mehr list
  mehr quick <description>
```

## Errors

| Error                        | Cause                                      |
| ---------------------------- | ------------------------------------------ |
| `queue not found: <queue>`   | The specified queue does not exist         |
| `task not found: <queue>/<id>` | The task ID does not exist in the queue  |

## What Gets Deleted

- The task entry from the queue file
- The task's notes file (if any)

## See Also

- [quick](quick.md) - Create a quick task
- [list](list.md) - List all tasks in workspace
- [optimize](optimize.md) - AI optimize a task based on notes
- [export](export.md) - Export queue task to markdown file
