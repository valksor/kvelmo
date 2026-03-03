# Getting Started with Web UI

The kvelmo Web UI provides a comfortable browser-based interface for managing development tasks. It's ideal for non-technical contributors and anyone who prefers visual workflows.

## Starting the Web UI

Start the kvelmo server:

```bash
kvelmo serve
```

Then open http://localhost:6337 in your browser.

**Tip:** Use `kvelmo serve --open` to automatically open the browser.

## Dashboard Overview

When you open the Web UI, you'll see the dashboard with:

- **Project selector** — Choose which project to work on
- **Task status** — Current task state and progress
- **Actions panel** — Workflow buttons (Plan, Implement, Review, Submit)
- **Output panel** — Real-time agent output
- **Sidebar** — Navigation to different views

## Creating a Task

1. Click **New Task** in the actions panel
2. Enter a **Title** — A short description of what you want to build
3. Enter a **Description** — Detailed requirements
4. Click **Start**

The task will be loaded and a branch created automatically.

## Running the Workflow

The workflow has five phases. Each phase requires your approval before proceeding:

### 1. Plan

Click **Plan** to generate a specification. The agent will analyze your task and create a structured implementation plan.

Review the specification in the **Specifications** panel.

### 2. Implement

Once you're happy with the plan, click **Implement**. The agent will execute the specification and make changes to your code.

Watch the progress in the **Output** panel.

### 3. Review

Click **Review** to start the review phase. This includes:
- Viewing the changes in the **File Changes** panel
- Running security scans
- Approving or rejecting the implementation

### 4. Submit

When satisfied, click **Submit** to create a PR.

## Sidebar Panels

The sidebar provides access to additional features:

| Panel           | Description                 |
|-----------------|-----------------------------|
| **Files**       | Browse project files        |
| **Changes**     | View file diffs             |
| **Checkpoints** | Navigate undo/redo history  |
| **Workers**     | Monitor worker pool         |
| **Memory**      | Semantic memory management  |
| **Screenshots** | Screenshot gallery          |
| **Browser**     | Browser automation controls |
| **Settings**    | Configuration               |

## Undo and Redo

Every workflow step creates a git checkpoint. Use the **Checkpoints** panel to:

- **Undo** — Revert to a previous checkpoint
- **Redo** — Restore a checkpoint you undid

## Also Available via CLI

Prefer the command line? All Web UI functionality is available via CLI commands.

See [CLI Reference](/cli/index.md) for the full command list.
