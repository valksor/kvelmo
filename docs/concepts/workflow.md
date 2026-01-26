# Workflow

Mehrhof uses a state machine to manage the task lifecycle. Understanding the workflow helps you use the tool effectively.

## Task Lifecycle

![Workflow State Diagram](../_media/img/workflow-state-diagram.png)

## States

### Primary States

| State            | Description                       | CLI Action | Web UI Action |
| ---------------- | --------------------------------- | ----------- | ------------ |
| **idle**         | Task registered, ready for action | `mehr plan` | Click **Plan** button |
| **planning**     | AI creating specifications        | Wait | Watch output |
| **implementing** | AI generating code                | Wait | Watch output |
| **reviewing**    | Code review in progress           | Wait | Watch output |
| **done**         | Task completed and merged         | None | None |

### Auxiliary States

| State             | Description               |
| ----------------- | ------------------------- |
| **checkpointing** | Creating git checkpoint   |
| **reverting**     | Undo in progress          |
| **restoring**     | Redo in progress          |
| **failed**        | Error occurred (terminal) |

## Workflow Phases

### 1. Start Phase

Register a task and prepare the workspace.

**CLI:**
```bash
mehr start task.md
```

**Web UI:** Click **"Create Task"** button on dashboard

What happens:
- Task ID is generated
- Git branch `task/<id>` is created
- Work directory `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/` is initialized
- Source content is stored (read-only)

### 2. Planning Phase

AI analyzes requirements and creates specifications.

**CLI:**
```bash
mehr plan
```

**Web UI:** Click **"Plan"** button in Active Task card

What happens:
- AI reads the source content and any notes
- Specifications (specification files) are generated
- Files are saved to `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/specifications/`
- Git checkpoint is created for undo support

### 3. Implementation Phase

AI implements the specifications.

**CLI:**
```bash
mehr implement
```

**Web UI:** Click **"Implement"** button in Active Task card

**Requirements:** At least one specification file must exist

What happens:
- AI reads all specification files and notes
- Code is generated or modified
- Changes are committed with checkpoint
- State returns to idle for review

### 4. Review Phase (Optional)

Automated code review.

**CLI:**
```bash
mehr review
```

**Web UI:** Click **"Review"** button in Active Task card

What happens:
- CodeRabbit analyzes the changes
- Review saved to `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/reviews/`
- Issues are reported for your attention

### 5. Finish Phase

Complete and merge the task.

**CLI:**
```bash
mehr finish
```

**Web UI:** Click **"Finish"** button in Active Task card

What happens:
- Quality checks run (if `make quality` exists)
- Changes squash-merged to target branch
- Task branch deleted
- Work directory cleaned up

## Guards

Guards are conditions that must be met for transitions:

| Guard     | Required For | Condition                       |
| --------- | ------------ | ------------------------------- |
| HasSource | start        | Task has valid source reference |
| HasSpecs  | implement    | specification files exist       |
| CanUndo   | undo         | Checkpoint history available    |
| CanRedo   | redo         | Redo stack not empty            |
| CanFinish | finish       | Task work exists                |

## Events

Events trigger state transitions:

| Event          | Description             |
| -------------- | ----------------------- |
| EventStart     | Begin task registration |
| EventPlan      | Enter planning phase    |
| EventImplement | Enter implementation    |
| EventReview    | Enter code review       |
| EventFinish    | Complete task           |
| EventUndo/Redo | Checkpoint operations   |
| EventError     | Handle errors           |
| EventAbort     | Abandon task            |

## Typical User Journey

**CLI:**
```
1. mehr start task.md     → idle (task registered)
2. mehr plan              → planning → idle (specifications created)
3. [Optional: mehr simplify to clarify specifications]
4. [Review specifications, add notes with mehr note]
5. mehr implement         → implementing → idle (code generated)
6. [Optional: mehr simplify to reduce code complexity]
7. [Review changes, maybe undo/redo]
8. mehr review            → reviewing → idle (review done)
9. mehr finish            → done (merged)
```

**Web UI:**
```
1. Click "Create Task"    → idle (task registered)
2. Click "Plan"           → planning → idle (specifications created)
3. [Review specifications, add notes with "Add Note" button]
4. Click "Implement"     → implementing → idle (code generated)
5. [Review changes, maybe use "Undo"]
6. Click "Review"        → reviewing → idle (review done)
7. Click "Finish"        → done (merged)
```

## Parallel Workflows

Each task runs in its own branch. You can:
- Switch between task branches
- Work on multiple tasks
- Use git worktrees for isolation

See [Tasks](tasks.md) for more on managing multiple tasks.

## See Also

- [CLI: workflow](../cli/workflow.md) - CLI workflow commands
- [Web UI: Overview](../web-ui/index.md) - Web UI guide
