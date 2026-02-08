# Simplify

Automatically refine and clarify task content, specifications, or code using AI assistance.

## Overview

The Simplify feature helps you:

- **Clarify task descriptions** before planning
- **Refine specifications** for better implementation
- **Clean up code** while preserving functionality
- **Improve readability** across all content types

Simplification is context-aware and adapts based on your current workflow state.

## Accessing Simplify in the Web UI

Simplify is available through:

| Feature              | Location                          |
|----------------------|-----------------------------------|
| **Simplify Page**    | Tools → Simplify                  |
| **Task Simplify**    | Task Detail → Actions → Simplify  |
| **Standalone Mode**  | Tools → Simplify (no active task) |

## Using Simplify

### From a Task

When you have an active task:

1. Open the task detail page
2. Click **Actions** in the toolbar
3. Select **Simplify**
4. The system automatically determines what to simplify based on your workflow state

### What Gets Simplified

| Workflow State       | What's Simplified        | Result                        |
|----------------------|--------------------------|-------------------------------|
| **Before planning**  | Task description         | Clearer, more actionable text |
| **After planning**   | Specification files      | Refined implementation plans  |
| **After implementing** | Code changes           | Cleaner, more readable code   |

### Standalone Mode

Simplify code changes without an active task:

1. Go to **Tools → Simplify**
2. Select **Standalone Mode**
3. Choose what to simplify:
   - **Uncommitted changes** (default)
   - **Current branch vs main**
   - **Specific files**
4. Click **Simplify**

## Safety Features

### Automatic Checkpoints

Before any simplification, the system creates a checkpoint. If something goes wrong:

1. Go to the task detail page
2. Click **Undo** to revert to the previous state
3. Or click **Redo** to restore the simplified version

### Review Changes

After simplification completes:

1. Review the changes shown in the results panel
2. Check the diff view to see what changed
3. Accept or undo as needed

## Simplification Examples

### Task Description

**Before:**
> Add user auth with JWT tokens and OAuth providers also handle refresh tokens and session management

**After:**
> Implement JWT-based user authentication with OAuth integration. Support refresh token rotation and secure session management.

### Specifications

The AI refines your specifications to include:
- Clear step-by-step implementation plans
- Explicit file paths and function names
- Testable completion conditions

### Code

The AI improves code by:
- Adding meaningful comments
- Using descriptive variable names
- Improving error handling
- Following project conventions

## Configuration

Configure simplification behavior in **Settings → Workflow → Simplify**:

| Setting                   | Description                              |
|---------------------------|------------------------------------------|
| **Custom Instructions**   | Project-specific standards to follow     |
| **Skip Checkpoints**      | Skip safety checkpoints (not recommended)|
| **Agent**                 | Specific agent for simplification tasks  |

## Best Practices

1. **Simplify early** - Refine task descriptions before planning
2. **Review results** - Always check what the AI changed
3. **Keep checkpoints** - Don't disable safety features
4. **Iterate** - Run simplify multiple times for progressive refinement
5. **Add custom instructions** - Configure project-specific standards

---

## Also Available via CLI

For power users who prefer command-line access:

See [CLI: simplify](/cli/simplify.md) for all options including branch comparison, commit ranges, and verbose output.
