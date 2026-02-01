# Queue Provider

**Schemes:** `queue:`

**Capabilities:** `read`, `snapshot`

Manages tasks from local task queues stored in the workspace. Queues allow you to organize and prioritize work items within a project.

## Usage

```bash
# Start a task from a queue
mehr start queue:backlog/fix-auth
mehr start queue:sprint-1/task-42

# List queues (via project command)
mehr project queues
```

## Queue Format

Queue tasks use the format `queue:queue-id/task-id`:

- **queue-id**: Name/identifier of the queue (e.g., `backlog`, `sprint-1`, `bugs`)
- **task-id**: Unique identifier within the queue (e.g., `fix-auth`, `task-42`)

## Creating Queues

Queues are stored in the workspace and can be managed via the project command:

```bash
# Create a new queue
mehr project queue create backlog

# Add a task to a queue
mehr project queue add backlog "Fix authentication bug"

# List queues
mehr project queues
```

## Queue Priority

Tasks in queues have priority levels that map to Mehrhof's priority system:

| Queue Priority   | Mehrhof Priority |
|------------------|------------------|
| 0-1 (High)       | High             |
| 2 (Normal)       | Normal           |
| 3+ (Low)         | Low              |
| Invalid/Negative | Normal (default) |

## Workflow Integration

Queues integrate with Mehrhof's workflow:

1. **Plan**: `mehr plan` - Creates specifications from the queued task
2. **Implement**: `mehr implement` - Executes the specifications
3. **Review**: `mehr review` - Reviews the implementation
4. **Finish**: `mehr finish` - Marks the task as complete

## Configuration

Queue storage is managed through the workspace. Queues are persisted in `.mehrhof/work/queues/`:

```
.mehrhof/
├── work/
│   └── queues/
│       ├── backlog.json
│       ├── sprint-1.json
│       └── bugs.json
```

## See Also

- [Project Command](../cli/project.md) - Queue management commands
- [Providers Overview](index.md) - All available providers
