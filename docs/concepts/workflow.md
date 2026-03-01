# Workflow

kvelmo manages tasks through a structured workflow that keeps you in control while your AI agent handles the details.

## Philosophy

**You decide what ships. The agent handles the mechanics.**

The workflow is designed around human oversight:
- Every phase requires your approval before proceeding
- Nothing ships without explicit confirmation
- You can undo at any point to try a different approach

Whether you use the CLI, Web UI, or Desktop App — the workflow is identical. You can switch between interfaces at any time without losing your place.

## The Five Phases

```
START → PLAN → IMPLEMENT → REVIEW → SUBMIT
```

### 1. Start

Load a task from a provider and prepare the workspace.

**What happens:**
- Task is fetched from the source (file, GitHub, GitLab, Linear, Wrike)
- A new git branch is created
- Workspace state transitions to `loaded`

**Why this matters:**
Starting from an issue tracker or file means everyone works from the same requirements. The automatic branch keeps your work isolated until you're ready to share.

**Commands:**
- CLI: `kvelmo start --from <provider>:<reference>`
- Web UI: Click **New Task** and enter details

### 2. Plan

Generate a structured specification that describes how to implement the task.

**What happens:**
- Agent analyzes the task requirements
- Agent explores the codebase for context
- A specification file is generated in `.kvelmo/specifications/`
- A git checkpoint is created
- Workspace state transitions to `planned`

**Why this matters:**
You review the plan before any code changes. This catches misunderstandings early.

**Commands:**
- CLI: `kvelmo plan`
- Web UI: Click **Plan**

### 3. Implement

Build your changes based on the approved plan.

**What happens:**
- Agent follows the specification you approved
- Code is written, modified, or deleted
- A git checkpoint is created after completion
- Workspace state transitions to `implemented`

**Why this matters:**
Because you approved the plan first, there are no surprises. If the implementation goes wrong, you can undo and try again without losing the original specification.

**Optional refinements:**
- `kvelmo simplify` — Simplify code for clarity
- `kvelmo optimize` — Optimize code quality

**Commands:**
- CLI: `kvelmo implement`
- Web UI: Click **Implement**

### 4. Review

Review the changes before they ship.

**What happens:**
- Changes are displayed for human review
- Security scanning runs (if configured)
- You approve or reject the implementation
- Workspace state transitions to `reviewing`

**Why this matters:**
Nothing leaves your machine without your explicit approval. Review catches issues before they become problems for your team.

**If you reject:**
Use `kvelmo undo` to revert, then try a different approach.

**Commands:**
- CLI: `kvelmo review`
- Web UI: Click **Review**

### 5. Submit

Create a PR and submit to the provider.

**What happens:**
- PR is created with the changes
- Task is marked as submitted in your issue tracker
- Workspace state transitions to `submitted`

**Why this matters:**
Your work is now visible to your team. The PR includes everything needed for code review — your original requirements and the changes that implement them.

**Commands:**
- CLI: `kvelmo submit`
- Web UI: Click **Submit**

---

## After Submit

Once your PR is merged:

- `kvelmo refresh` — Check PR status and update task state
- `kvelmo finish` — Clean up the branch and return to main

## Undo and Redo

Every phase creates a checkpoint. If something goes wrong, you can step back:

```
kvelmo undo    # Revert to previous checkpoint
kvelmo redo    # Restore a checkpoint you undid
```

**Why this matters:**
You can experiment freely. Try an approach, undo if it doesn't work, try something different. Your history is preserved — nothing is lost until you explicitly clean up.

## Recovery

If something goes wrong, you have options:

| Situation | Solution |
|-----------|----------|
| Bad implementation | `kvelmo undo` to revert to the previous checkpoint |
| Stuck state | `kvelmo reset` to recover without losing work |
| Want to start over | `kvelmo abandon` for full cleanup |

**The key insight:** Every step is reversible. You're never stuck with a bad outcome.

---

## Technical Details

For developers interested in the underlying mechanics, see [State Machine](/concepts/state-machine.md) for the full state transition diagram.
