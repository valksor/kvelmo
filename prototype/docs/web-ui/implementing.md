# Implementing

The implementation phase is where the AI executes your specifications and writes code.

## What Implementing Does

When you click **"Implement"**, the AI:

1. **Reads all specifications** - Every `specification-*.md` file is analyzed
2. **Analyzes existing code** - Understands your codebase structure and patterns
3. **Creates or modifies files** - Writes new code and updates existing files
4. **Writes tests** - Adds test coverage for new functionality
5. **Creates checkpoints** - Saves progress for undo/redo support

**Requirement:** At least one specification file must exist before implementing.

## Starting Implementation

After planning completes, and you've reviewed the specifications, click the **"Implement"** button:

The Active Task card shows the number of specifications ready. Click **Implement** to start the implementation phase.

## Implementation Phase Workflow

```text
┌───────────────────┐     ┌──────────────────┐     ┌────────────────────┐     ┌────────────────┐
│ Idle + Specs Ready│ ──▶ │ Click Implement  │ ──▶ │ Implementing State │ ──▶ │ AI Reads Specs │
└───────────────────┘     └──────────────────┘     └────────────────────┘     └───────┬────────┘
                                                                                      │
                                                                                      ▼
┌────────────────────────────┐     ┌────────────────────┐     ┌────────────────┐
│ Back to Idle - Code Ready  │ ◀── │ Checkpoint Created │ ◀── │ AI Writes Code │
└────────────────────────────┘     └────────────────────┘     └────────────────┘
```

## Real-Time Progress

Watch the AI work in the **Agent Output** section:

The **Agent Output** section shows real-time progress as the AI reads specifications, creates new files, and modifies existing ones. Each file change shows what was added or modified.

## The Implementing State

During implementation, the task state changes to **"Implementing"**:

| State            | What's Happening        | What You Can Do                     |
|------------------|-------------------------|-------------------------------------|
| **Implementing** | AI is writing code      | Watch progress, wait for completion |
| **Waiting**      | AI has a question       | Answer in the Questions section     |
| **Idle**         | Implementation complete | Review changes, run tests           |

## Reviewing File Changes

After implementation completes, review what changed:

The **File Changes** section lists all files that were created or modified, with line counts and expandable diffs. Click any file to see the full diff.

Click any file to see the full diff.

## Viewing Diffs From Specifications

In the **Task Detail** page, expand a specification and go to **Implemented Files**.

Each file includes a **View Diff** action that opens a read-only modal with:
- A **Visual** split view (Before vs After) with line numbers and color highlighting
- A **Raw** tab with the original unified patch text

This is useful when you want to inspect exactly what changed for one implemented file without starting a review workflow.

## Adding Notes Before Implementation

You can add notes right before implementing to provide additional guidance:

1. Click **"Add Note"** button
2. Enter instructions for the AI
3. Click **"Implement"**

```
Use the existing session store from internal/session/
Don't add Redis - we're using PostgreSQL only
```

The notes will be included in the implementation prompt.

## What If Something Goes Wrong?

### Use Undo

If you're not happy with the implementation:

1. Click **"Undo"** button
2. Add a note explaining what went wrong
3. Click **"Implement"** again

```text
                                           Yes ──▶ ┌────────────────────┐
                                      ┌────────────│ Continue to Review │
┌────────────────────────┐     ┌──────┴──┐         └────────────────────┘
│ Implementation Complete│ ──▶ │ Happy?  │
└────────────────────────┘     └──────┬──┘         ┌────────────┐     ┌────────────────┐     ┌─────────────────┐
                                      └────────────│ Click Undo │ ──▶ │ Add Note: Fix X│ ──▶ │ Implement Again │
                                           No ──▶  └────────────┘     └────────────────┘     └─────────────────┘
```

### Add Notes and Implement Again

Instead of reverting everything, you can:
1. Add a note with corrections
2. Click **"Implement"** again

The AI will build on the existing implementation and apply your corrections.

## Implementation Best Practices

1. **Review specs first** - Make sure specifications are complete
2. **Add context** - Use notes to guide the AI
3. **Check the output** - Review file changes after completion
4. **Run tests** - Verify everything works
5. **Use checkpoints** - Undo/redo if needed

## What Gets Implemented

The AI will create:
- **New files** - Based on specification requirements
- **Modifications** - Updates to existing files
- **Tests** - Unit tests for new functionality
- **Documentation** - Comments and docstrings

The AI follows your project's existing patterns and conventions.

## Next Steps

After implementation completes:

- [**Reviewing**](reviewing.md) - Run automated code review
- [**Finishing**](finishing.md) - Complete and merge the task
- [**Undo & Redo**](undo-redo.md) - Navigate checkpoints if needed
- [**Notes**](notes.md) - Add feedback for next iteration

---

## Also Available via CLI

Run implementation from the command line for terminal-based workflows or automation.

See [CLI: implement](/cli/implement.md) for all flags and options.
