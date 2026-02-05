# Undo & Redo

Navigate checkpoint history to revert or reapply changes.

## What Undo Does

When you click **"Undo"**, Mehrhof:

1. **Finds previous checkpoint** - Locates the most recent checkpoint
2. **Reverts changes** - Restores files to that checkpoint state
3. **Updates task state** - Returns to the state from that checkpoint
4. **Preserves future** - Saves current state for potential redo

## What Redo Does

When you click **"Redo"**, Mehrhof:

1. **Finds next checkpoint** - Locates the checkpoint after current state
2. **Reapplies changes** - Restores files to that checkpoint state
3. **Updates task state** - Returns to the future state
4. **Updates history** - Maintains checkpoint stack

## Checkpoints

Checkpoints are automatic snapshots created at key workflow moments:

| Event                    | Checkpoint Created        |
|--------------------------|---------------------------|
| After **Planning**       | Saves specification files |
| After **Implementation** | Saves code changes        |
| After **Review**         | Saves review results      |
| Before **Finish**        | Final state before merge  |

## Using Undo

Click **"Undo"** to go back one step:

The Active Task card shows your current checkpoint position. Click **Undo** to revert to the previous checkpoint.

### Undo After Implementation

The **Undo Checkpoint** dialog shows which checkpoint you're at and which you'll revert to, lists files that will be reverted, and confirms the current state will be saved for redo. Click **Confirm Undo** to proceed.

## Using Redo

After undoing, click **"Redo"** to go forward:

The **Redo Checkpoint** dialog shows which state you'll restore and lists the files that will be reapplied. Click **Confirm Redo** to proceed.

## Undo/Redo Workflow

```text
                                        Yes ──▶ ┌──────────┐
                                   ┌────────────│ Continue │◀────────────────────────────────────┐
┌────────────────┐     ┌───────────┴┐           └──────────┘                                     │
│ Implementation │ ──▶ │   Happy?   │                                                       Yes │
└────────────────┘     └───────────┬┘                                                           │
                                   │                                                     ┌──────┴─────┐
                              No   │                                                     │ Happy now? │
                                   ▼                                                     └──────┬─────┘
                            ┌────────────┐     ┌──────────────────┐     ┌──────────┐     ┌──────┴──────────┐
                            │ Click Undo │ ──▶ │ Back to Planning │ ──▶ │ Add Note │ ──▶ │ Implement Again │
                            └──────┬─────┘     └──────────────────┘     └──────────┘     └─────────────────┘
                                   │                                                            ▲
                                   │  ┌───────────────┐                                    No   │
                                   └──│ Redo Available│◀── (if you change your mind) ──────────┘
                                      └───────────────┘
```

## Checkpoint History

View all checkpoints in the Active Task card:

The **Checkpoints** section lists all checkpoints with descriptions and file counts. The current checkpoint is highlighted. Click **View Details** to see what changed, or **Restore** to jump directly to any checkpoint.

Click **"Restore"** on any checkpoint to jump directly to that state.

## When to Use Undo

### Fix Implementation Issues

1. Implementation completes with bugs
2. Click **"Undo"** to revert
3. Add a note: "Fix the error handling"
4. Click **"Implement"** again

### Try Different Approach

1. Review specifications and want a different direction
2. Click **"Undo"** to go back to planning
3. Add a note with new requirements
4. Click **"Plan"** again

### Recover from Mistakes

1. Accidentally made wrong changes
2. Click **"Undo"** to revert
3. Make corrections
4. Continue

## Undo Best Practices

1. **Review before undo** - Check what will be reverted
2. **Add notes** - Explain what to fix before re-implementing
3. **Use checkpoints** - Each major step creates a checkpoint
4. **Don't undo too far** - You might lose progress
5. **Consider redo** - After undo, you can always redo

## Limitations

- **Undo before finish** - Cannot undo after task is finished
- **No partial undo** - Undo reverts entire checkpoint
- **Linear history** - Cannot branch checkpoint history

## Next Steps

After using Undo/Redo:

- [**Planning**](planning.md) - Create new specifications
- [**Implementing**](implementing.md) - Try implementation again
- [**Notes**](notes.md) - Add context for next attempt

---

## Also Available via CLI

Navigate checkpoints from the command line.

See [CLI: undo](/cli/undo.md) and [CLI: redo](/cli/redo.md) for all options.
