# Parallel Tasks

The Web UI provides a comprehensive interface for managing and monitoring tasks running in parallel.

## Overview

Parallel task execution allows you to run multiple tasks simultaneously, each in its own isolated goroutine with a dedicated conductor instance. The Web UI provides real-time monitoring, per-task communication, and control capabilities.

**Key features:**

- Real-time task status monitoring
- Per-task SSE (Server-Sent Events) streams
- Cancel individual running tasks
- Send notes to specific running tasks
- Track progress across all parallel tasks

## Starting Parallel Tasks

### Via API

Start multiple tasks in parallel using the `/api/v1/parallel` endpoint:

```bash
curl -X POST http://localhost:8080/api/v1/parallel \
  -H "Content-Type: application/json" \
  -d '{
    "references": ["file:a.md", "file:b.md", "file:c.md"],
    "max_workers": 3,
    "use_worktree": true
  }'
```

**Request Body:**

| Field          | Type     | Default | Description                                |
|----------------|----------|---------|--------------------------------------------|
| `references`   | string[] | -       | Array of task references to start          |
| `max_workers`  | int      | 2       | Maximum parallel workers (goroutines)      |
| `use_worktree` | bool     | false   | Create isolated git worktree for each task |

**Response:**

```json
{
  "success": true,
  "message": "parallel execution started",
  "task_count": 3,
  "max_workers": 3,
  "task_ids": ["abc123", "def456", "ghi789"]
}
```

**Important:** When `max_workers > 1`, you must set `use_worktree: true` to prevent file conflicts between concurrent tasks.

### Via CLI

From the CLI, use the `--parallel` flag:

```bash
mehr start file:a.md file:b.md file:c.md --parallel=3 --worktree
```

The Web UI will automatically display these tasks in the Running Tasks section.

## Monitoring Running Tasks

### Running Tasks List

The dashboard includes a Running Tasks section when parallel tasks are active:

```
┌──────────────────────────────────────────────────────────────┐
│  Running Tasks (3)                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 🟢 abc123  │  file:a.md  │  running  │  5m 30s   │  │      │
│  │  Worktree: ../worktrees/abc123                    │  │      │
│  │                            [View Stream] [Cancel] │  │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 🟢 def456  │  file:b.md  │  running  │  5m 28s   │  │      │
│  │  Worktree: ../worktrees/def456                    │  │      │
│  │                            [View Stream] [Cancel] │  │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────┐      │
│  │ ✅ ghi789  │  file:c.md  │  completed │  4m 15s  │  │      │
│  │  Worktree: ../worktrees/ghi789                    │  │      │
│  │                            [View Stream]          │  │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  2 running, 3 total                                          │
└──────────────────────────────────────────────────────────────┘
```

### API: List Running Tasks

```bash
curl http://localhost:8080/api/v1/running
```

**Response:**

```json
{
  "tasks": [
    {
      "id": "abc123",
      "reference": "file:a.md",
      "task_id": "task-001",
      "status": "running",
      "started_at": "2025-01-15T10:30:00Z",
      "duration": "5m30s",
      "worktree_path": "../worktrees/abc123"
    },
    {
      "id": "def456",
      "reference": "file:b.md",
      "task_id": "task-002",
      "status": "running",
      "started_at": "2025-01-15T10:30:02Z",
      "duration": "5m28s",
      "worktree_path": "../worktrees/def456"
    },
    {
      "id": "ghi789",
      "reference": "file:c.md",
      "task_id": "task-003",
      "status": "completed",
      "started_at": "2025-01-15T10:30:04Z",
      "finished_at": "2025-01-15T10:34:19Z",
      "duration": "4m15s",
      "worktree_path": "../worktrees/ghi789"
    }
  ],
  "count": 3,
  "running": 2
}
```

### Task Status Values

| Status      | Description                      |
|-------------|----------------------------------|
| `pending`   | Task registered, not yet started |
| `running`   | Task actively executing          |
| `completed` | Task finished successfully       |
| `failed`    | Task encountered an error        |
| `cancelled` | Task was manually cancelled      |

## Per-Task Streaming

### SSE Stream Endpoint

Get real-time updates for a specific task:

```bash
curl -N http://localhost:8080/api/v1/running/abc123/stream
```

The stream sends Server-Sent Events with task-specific updates:

```
event: connected
data: {"status":"connected","task_id":"abc123","reference":"file:a.md","state":"running"}

event: task_progress
data: {"id":"abc123","message":"Analyzing codebase..."}

event: task_output
data: {"id":"abc123","content":"Creating internal/auth/handler.go"}

event: task_complete
data: {"id":"abc123","status":"completed","duration":"5m30s"}
```

### Dashboard Stream View

Click **View Stream** on any running task to open a dedicated output panel showing:

- Real-time agent output
- Progress indicators
- File changes as they happen
- Completion status

## Cancelling Tasks

### Via Dashboard

Click the **Cancel** button on any running task. The task will:

1. Receive a context cancellation signal
2. Clean up resources
3. Mark status as `cancelled`

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/running/abc123/cancel
```

**Response:**

```json
{
  "success": true,
  "message": "task cancellation requested",
  "task_id": "abc123",
  "reference": "file:a.md"
}
```

**Note:** Cancellation is asynchronous. The task will stop at the next safe checkpoint.

## Sending Notes to Running Tasks

### Via Dashboard

1. Click on a running task to select it
2. Type your note in the input field
3. Click **Send Note**

The note is delivered to the task's conductor and included in subsequent agent prompts.

### Via API

Use the standard notes endpoint with the running task ID:

```bash
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{
    "running_task_id": "abc123",
    "message": "Consider edge case X"
  }'
```

### Via CLI

```bash
mehr note --running=abc123 "Consider edge case X"
```

## Best Practices

### When to Use Parallel Execution

**Good use cases:**
- Independent tasks with no code dependencies
- Batch processing multiple features
- CI/CD pipelines with ample resources

**Avoid when:**
- Tasks modify the same files
- Tasks have sequential dependencies
- Limited system resources

### Recommended Settings

| Scenario             | `max_workers` | Notes                         |
|----------------------|---------------|-------------------------------|
| Development machine  | 2-3           | Leave headroom for other work |
| CI server            | 4-8           | Based on available cores      |
| Sequential execution | 1             | Default, no worktree needed   |

### Monitoring Tips

1. **Watch the Running Tasks section** for stuck tasks (excessive duration)
2. **Check failed tasks** for error messages
3. **Use per-task streams** to debug issues
4. **Cancel unresponsive tasks** rather than waiting indefinitely

---

## Also Available via CLI

Manage parallel task execution from the command line for scripting or terminal workflows.

| Command | What It Does |
|---------|--------------|
| `mehr start <refs> --parallel=N` | Start multiple tasks in parallel with N workers |
| `mehr start <refs> --parallel=N --worktree` | Use isolated git worktrees per task |
| `mehr list --running` | List currently running parallel tasks |
| `mehr note --running=<id> "message"` | Send a note to a specific running task |
| `mehr project start --parallel=N` | Start project queue tasks in parallel |

See [CLI: start](/cli/start.md) for parallel execution options, and [CLI: list](/cli/list.md) for monitoring running tasks.

## Related Documentation

- [Parallel Execution Architecture](/concepts/parallel-execution.md) - Technical architecture and internals
- [CLI: start --parallel](/cli/start.md#start-multiple-tasks-in-parallel)
- [CLI: list --running](/cli/list.md#list-running-parallel-tasks)
- [CLI: note --running](/cli/note.md#send-note-to-running-parallel-task)
- [CLI: project start --parallel](/cli/project.md#start)
- [Dashboard](dashboard.md) - Main Web UI interface
