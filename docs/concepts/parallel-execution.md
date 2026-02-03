# Parallel Task Execution

This document describes the internal architecture of Mehrhof's parallel task execution system.

## Overview

Parallel task execution allows running multiple tasks simultaneously, each in its own isolated goroutine with a dedicated conductor instance. This architecture enables efficient batch processing while maintaining isolation between tasks.

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
│  │              Goroutine Pool                      │        │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │        │
│  │  │ Worker 1 │  │ Worker 2 │  │ Worker 3 │       │        │
│  │  │ (Task A) │  │ (Task B) │  │ (Task C) │       │        │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘       │        │
│  │       │             │             │             │        │
│  │       ▼             ▼             ▼             │        │
│  │  ┌─────────┐   ┌─────────┐   ┌─────────┐       │        │
│  │  │Conductor│   │Conductor│   │Conductor│       │        │
│  │  └─────────┘   └─────────┘   └─────────┘       │        │
│  └─────────────────────────────────────────────────┘        │
│                                                             │
│  ┌─────────────────────────────────────────────────┐        │
│  │           Git Worktrees (Isolation)              │        │
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

## Thread Safety

The registry uses `sync.RWMutex` to ensure thread-safe access:

- Multiple readers can access task state simultaneously
- Writers get exclusive access for state updates
- Event bus notifies subscribers of state changes

## Isolation Model

Each parallel task receives:

1. **Dedicated Conductor** - Independent orchestrator with its own state
2. **Separate Storage** - Task-specific work directory
3. **Git Worktree** - Isolated working copy (when `use_worktree: true`)
4. **Context** - Individual cancellation context per task

## Worker Pool

The task runner uses a semaphore-based worker pool:

- `max_workers` controls maximum concurrent tasks
- Tasks queue when all workers are busy
- Context cancellation propagates to running tasks
- Failed tasks don't affect other workers

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
