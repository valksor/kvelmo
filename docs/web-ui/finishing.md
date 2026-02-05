# Finishing

The finish phase completes your task and merges changes to the main branch.

## What Finishing Does

When you click **"Finish"**, Mehrhof:

1. **Runs quality checks** - Executes tests and quality tools (if configured)
2. **Squash-merges changes** - Merges your task branch to the target branch
3. **Deletes task branch** - Cleans up the feature branch
4. **Marks task complete** - Updates task state to "done"
5. **Cleans up work directory** - Optionally removes work files

## Starting Finish

When you're satisfied with the implementation, click **"Finish"** button:

When tests are passing and you're satisfied with the implementation, click **Finish** in the Active Task card to complete and merge your changes.

## Finish Phase Workflow

```text
┌──────────────────┐     ┌──────────────┐     ┌────────────────┐     ┌──────────────┐
│ Idle + Code Ready│ ──▶ │ Click Finish │ ──▶ │ Quality Checks │ ──▶ │ Checks Pass? │
└──────────────────┘     └──────────────┘     └────────────────┘     └──────┬───────┘
                                                                            │
                          ┌─────────────────────────────────────────────────┼──────────────────────┐
                          │ Yes                                             │                 No   │
                          ▼                                                 │                      ▼
                   ┌───────────────┐     ┌───────────────┐     ┌───────────┐│    ┌─────────────────────────────┐
                   │ Merge to Main │ ──▶ │ Delete Branch │ ──▶ │ Mark Done ││    │ Show Errors - Cancel Finish │
                   └───────────────┘     └───────────────┘     └───────────┘│    └─────────────────────────────┘
```

## Finish Confirmation Dialog

Clicking **"Finish"** shows a confirmation dialog:

The **Finish Task** confirmation dialog shows a summary of changes (files created, modified, deleted), what will happen (quality checks, merge, branch cleanup), and the target branch. Click **Confirm Finish** to proceed.

## Quality Checks

Before merging, finish runs quality checks (if configured):

The quality checks progress dialog shows each check as it runs (tests, formatting, linting, security), with status indicators for passed or failed checks.

### When Checks Fail

If quality checks fail, finish is cancelled:

If quality checks fail, the dialog lists each failure with details. You can click **Try Again** after fixing issues, **Force Finish** to proceed anyway (not recommended), or **Cancel** to return to the task.

## Merge Behavior

The finish step uses **squash merge** by default:

1. **All commits combined** - Your task branch commits are squashed into one
2. **Single commit on target** - Clean history with one merge commit
3. **Branch deleted** - Task branch is removed after successful merge

### Commit Message

The squash merge uses a generated commit message:

```
Add user OAuth authentication

- Implemented Google OAuth2 provider
- Added session management with PostgreSQL
- Created login/logout endpoints
- Added authentication middleware

Co-authored-by: Claude Opus 4.5 <noreply@anthropic.com>
```

## After Finishing

Once finish completes:

Once complete, the success dialog shows what happened: branch merged, feature branch deleted, and the new commit hash. Click **View in History** to see the completed task or **Create New Task** to start fresh.

## Finish Options

Configure finish behavior in [Settings](settings.md):

| Option            | Description                         | Default   |
|-------------------|-------------------------------------|-----------|
| **Target branch** | Branch to merge into                | `main`    |
| **Delete branch** | Delete task branch after merge      | `true`    |
| **Delete work**   | Clean up work directory             | `false`   |
| **Run quality**   | Execute quality checks before merge | `true`    |
| **Commit prefix** | Template for commit message         | `[{key}]` |

## Finish Best Practices

1. **Review changes first** - Use git diff or the File Changes section
2. **Run tests locally** - Ensure tests pass before finishing
3. **Check quality** - Run quality tools manually if needed
4. **Verify merge** - Check the target branch after finish
5. **Keep work directory** - Don't delete if you need to reference specs later

## What Happens to Work Directory

By default, the work directory is preserved for reference. Configure cleanup in settings:

```yaml
workflow:
  delete_work_on_finish: false  # Keep work directory
  delete_work_on_abandon: false  # Keep on abandon too
```

To manually clean up old work directories, see your workspace location:

```
~/.valksor/mehrhof/workspaces/<project-id>/work/
```

## Next Steps

After finishing:

- [**Creating Tasks**](creating-tasks.md) - Start a new task
- [**Task History**](task-history.md) - View completed tasks
- [**Dashboard**](dashboard.md) - Return to main dashboard

---

## Also Available via CLI

Complete tasks from the command line for terminal-based workflows or automation.

See [CLI: finish](/cli/finish.md) for all flags and options.
