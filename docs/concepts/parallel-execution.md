# Parallel Task Execution

Run multiple tasks at the same time, each completely isolated from the others.

## Overview

Parallel task execution lets you work on several tasks simultaneously. Each task runs independently with its own workspace, so changes in one task never interfere with another. This is useful for:

- **Batch processing** - Complete a backlog of independent tasks quickly
- **Team workflows** - Different team members can work on separate features
- **CI/CD pipelines** - Process multiple issues or PRs in parallel

Under the hood, Mehrhof creates a separate environment for each task with its own orchestrator and isolated file system.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Web Server                           │
│                                                             │
│  ┌─────────────────┐                                        │
│  │ Task Registry   │  In-memory tracking of running tasks   │
│  └────────┬────────┘                                        │
│           │                                                 │
│  ┌────────▼────────┐                                        │
│  │  Task Runner    │  Worker pool with semaphore            │
│  └────────┬────────┘                                        │
│           │                                                 │
│  ┌────────▼────────────────────────────────────────┐        │
│  │              Goroutine Pool                     │        │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │        │
│  │  │ Worker 1 │  │ Worker 2 │  │ Worker 3 │       │        │
│  │  │ (Task A) │  │ (Task B) │  │ (Task C) │       │        │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘       │        │
│  │       │             │             │             │        │
│  │       ▼             ▼             ▼             │        │
│  │  ┌─────────┐   ┌─────────┐   ┌─────────┐        │        │
│  │  │Conductor│   │Conductor│   │Conductor│        │        │
│  │  └─────────┘   └─────────┘   └─────────┘        │        │
│  └─────────────────────────────────────────────────┘        │
│                                                             │
│  ┌─────────────────────────────────────────────────┐        │
│  │           Git Worktrees (Isolation)             │        │
│  │  ../worktrees/abc123/  ../worktrees/def456/  ...│        │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Components

| Component     | Description                                    |
|---------------|------------------------------------------------|
| Task Registry | Thread-safe map tracking all running tasks     |
| Task Runner   | Worker pool with configurable parallelism      |
| Conductor     | Per-task orchestrator (AI agent, storage, VCS) |
| Git Worktrees | Isolated working directories per task          |

## Safe Concurrent Access

The system safely handles multiple tasks running at the same time:

- Multiple clients can view task status without conflicts
- Updates to task state are handled one at a time to prevent corruption
- Status changes are broadcast to all connected clients in real-time

## Isolation Model

Each parallel task receives:

1. **Dedicated Conductor** - Independent orchestrator with its own state
2. **Separate Storage** - Task-specific work directory
3. **Git Worktree** - Isolated working copy (when `use_worktree: true`)
4. **Context** - Individual cancellation context per task

## Worker Pool

Tasks are processed by a pool of workers:

- **max_workers** setting controls how many tasks run at once
- Additional tasks wait in a queue until a worker becomes available
- Cancelling a task stops it without affecting other running tasks
- If one task fails, other tasks continue running normally

## Event Flow

```
Task Start Request
       │
       ▼
Task Registry.Register()
       │
       ▼
Worker Pool.Acquire()
       │
       ▼
Goroutine Spawned
       │
       ├──▶ Conductor.Initialize()
       │
       ├──▶ Conductor.Execute()
       │         │
       │         ├──▶ Events broadcast via Event Bus
       │         │
       │         └──▶ SSE streams to connected clients
       │
       └──▶ Task Registry.Complete()
                  │
                  ▼
           Worker Pool.Release()
```

## Worktree Management

When `use_worktree: true`:

1. **Creation**: New worktree created at `../worktrees/<task-id>/`
2. **Isolation**: Each task operates on independent file system
3. **Cleanup**: Worktrees removed after task completion
4. **Conflict Prevention**: Required when `max_workers > 1` to prevent file conflicts

## See Also

- [Web UI: Parallel Tasks](/web-ui/parallel-tasks.md) - User interface documentation
- [CLI: start --parallel](/cli/start.md#start-multiple-tasks-in-parallel) - CLI usage
- [Workflow Concept](workflow.md) - State machine documentation
