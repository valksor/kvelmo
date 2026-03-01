# Recovery Guide

How to recover from common problems in kvelmo.

## Quick Recovery Commands

| Problem | Solution |
|---------|----------|
| Bad implementation | `kvelmo undo` |
| Stuck state | `kvelmo reset` |
| Want to start over | `kvelmo abandon` |
| Server not responding | Restart `kvelmo serve` |

## Undo a Bad Implementation

If the implementation doesn't look right:

```bash
# Revert to previous checkpoint
kvelmo undo

# Now you're back to planned state
kvelmo status
# State: planned

# Try again with more context
kvelmo implement
```

You can undo multiple times:
```bash
kvelmo undo  # Back to planned
kvelmo undo  # Back to loaded
```

## Redo After Undo

If you undid too far:

```bash
kvelmo redo
```

This restores the next checkpoint in the redo stack.

## Reset Stuck State

If kvelmo is stuck (e.g., after a crash):

```bash
kvelmo reset
```

This:
- Resets state to `loaded`
- Preserves your task data
- Keeps checkpoints intact

## Abandon a Task

To completely abandon a task:

```bash
kvelmo abandon
```

This:
- Deletes the task branch
- Cleans up `.kvelmo/` files
- Resets to no active task

**Caution:** This is destructive. Your work on this task will be lost.

## Restart the Server

If the server is unresponsive:

```bash
# Stop any running server (Ctrl+C in the terminal)

# Remove stale socket file
rm ~/.valksor/kvelmo/global.sock

# Start fresh
kvelmo serve
```

## Fix Corrupted State

If `.kvelmo/` is corrupted:

```bash
# Backup first
cp -r .kvelmo .kvelmo.bak

# Reset state
kvelmo reset

# If still broken, remove and re-create
rm -rf .kvelmo
kvelmo start --from file:task.md
```

## Recover Lost Work

kvelmo creates git checkpoints. Even after `abandon`, your work may be in git:

```bash
# List recent commits
git log --all --oneline -20

# Find the checkpoint commit
# It will have a message like "[kvelmo] Checkpoint: implement"

# Create a branch from it
git checkout -b recovered <commit-sha>
```

## Debug Mode

For more information on what's happening:

```bash
# Run with verbose output
kvelmo status --json

# Check server logs
kvelmo serve  # Watch the terminal output
```

## Common Error Messages

### "no active task"

You need to start a task first:
```bash
kvelmo start --from file:task.md
```

### "guards failed for transition"

You're trying to do something out of order. Check the state:
```bash
kvelmo status
```

Then follow the [workflow](/concepts/workflow.md).

### "socket not found"

The server isn't running:
```bash
kvelmo serve
```

### "permission denied"

Socket file permissions issue:
```bash
chmod 600 ~/.valksor/kvelmo/global.sock
```

## Getting Help

- Check [FAQ](/faq.md)
- Review [State Machine](/concepts/state-machine.md)
- Open an issue on [GitHub](https://github.com/valksor/kvelmo/issues)
