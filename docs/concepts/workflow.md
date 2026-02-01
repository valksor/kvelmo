# Workflow

Mehrhof uses a structured workflow to manage the task lifecycle. Understanding this process helps you use the tool effectively and stay in control at every step.

## Task Lifecycle

![Workflow State Diagram](../_media/img/workflow-state-diagram.png)

## States

### Primary States

| State            | Description                          | CLI Action  | Web UI Action         |
|------------------|--------------------------------------|-------------|-----------------------|
| **idle**         | Task registered, ready for action    | `mehr plan` | Click **Plan** button |
| **planning**     | Creating a structured plan           | Wait        | Watch output          |
| **implementing** | Executing the plan to create changes | Wait        | Watch output          |
| **reviewing**    | Quality checks in progress           | Wait        | Watch output          |
| **done**         | Task completed and merged            | None        | None                  |

### Auxiliary States

| State             | Description               |
|-------------------|---------------------------|
| **checkpointing** | Creating checkpoint       |
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
- Work directory is initialized
- Your task description is stored (read-only)

### 2. Planning Phase

Create a structured plan from your task description.

**CLI:**
```bash
mehr plan
```

**Web UI:** Click **"Plan"** button in Active Task card

What happens:
- Your task description and any notes are analyzed
- A structured plan is generated
- Plan files are saved for your review
- Git checkpoint is created for undo support

**Why this matters:** You review the plan before any changes are made. Nothing happens without your approval.

### 3. Creation Phase

Execute the plan to produce changes.

**CLI:**
```bash
mehr implement
```

**Web UI:** Click **"Implement"** or **"Create"** button in Active Task card

**Requirements:** At least one plan file must exist

What happens:
- The plan and any notes are used to guide creation
- Code, documentation, or configuration is generated/modified
- Changes are committed with checkpoint
- State returns to idle for review

### 4. Review Phase (Optional)

Run quality checks on the changes.

**CLI:**
```bash
mehr review
```

**Web UI:** Click **"Review"** button in Active Task card

What happens:
- Automated checks analyze the changes
- Review results are saved for your attention
- Issues are reported so you can address them

### 5. Finish Phase

Complete and merge the task.

**CLI:**
```bash
mehr finish
```

**Web UI:** Click **"Finish"** button in Active Task card

What happens:
- Quality checks run (if configured)
- Changes are squash-merged to target branch
- Task branch is deleted
- Work directory is cleaned up

## Guards

Guards are conditions that must be met for transitions:

| Guard     | Required For | Condition                    |
|-----------|--------------|------------------------------|
| HasSource | start        | Task has valid description   |
| HasSpecs  | implement    | Plan files exist             |
| CanUndo   | undo         | Checkpoint history available |
| CanRedo   | redo         | Redo stack not empty         |
| CanFinish | finish       | Task work exists             |

## Events

Events trigger state transitions:

| Event          | Description             |
|----------------|-------------------------|
| EventStart     | Begin task registration |
| EventPlan      | Enter planning phase    |
| EventImplement | Enter creation phase    |
| EventReview    | Enter review phase      |
| EventFinish    | Complete task           |
| EventUndo/Redo | Checkpoint operations   |
| EventError     | Handle errors           |
| EventAbort     | Abandon task            |

## Typical User Journey

**CLI:**
```
1. mehr start task.md     → idle (task registered)
2. mehr plan              → planning → idle (plan created)
3. [Optional: mehr simplify to clarify the plan]
4. [Review plan, add notes with mehr note]
5. mehr implement         → implementing → idle (changes created)
6. [Optional: mehr simplify to reduce complexity]
7. [Review changes, maybe undo/redo]
8. mehr review            → reviewing → idle (review done)
9. mehr finish            → done (merged)
```

**Web UI:**
```
1. Click "Create Task"    → idle (task registered)
2. Click "Plan"           → planning → idle (plan created)
3. [Review plan, add notes with "Add Note" button]
4. Click "Create"         → implementing → idle (changes created)
5. [Review changes, maybe use "Undo"]
6. Click "Review"         → reviewing → idle (review done)
7. Click "Finish"         → done (merged)
```

## Parallel Workflows

Each task runs in its own branch. You can:
- Switch between task branches
- Work on multiple tasks
- Use git worktrees for isolation

See [Tasks](tasks.md) for more on managing multiple tasks.

## See Also

- [Glossary](../glossary.md) - Plain-language definitions
- [CLI: workflow](../cli/workflow.md) - CLI workflow commands
- [Web UI: Overview](../web-ui/index.md) - Web UI guide
