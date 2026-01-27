# Auto Mode

Auto mode runs the entire Mehrhof workflow without user interaction—plan, implement, review, and finish in one click. Ideal for well-defined tasks where you trust the AI to handle the complete process autonomously.

## Overview

Auto mode orchestrates the complete task lifecycle:

1. **Start** - Register task and create git branch
2. **Plan** - Generate implementation specifications (agent questions skipped)
3. **Implement** - Execute the specifications
4. **Quality** - Run quality checks with automatic retry loop
5. **Review** - Code review with self-correction
6. **Finish** - Merge changes to target branch

## Accessing in the Web UI

Auto mode is available from the main dashboard:

```
http://localhost:PORT/
```

Look for the **"Auto"** button in the Active Task card or Quick Actions.

## Using Auto Mode

### Starting Auto Mode

1. Create or select a task (see [Creating Tasks](creating-tasks.md))
2. Click the **"Auto"** button on the dashboard
3. Confirm to start full automation

The workflow runs automatically without further interaction needed.

### What Happens During Auto

```
┌─────────────┐
│  Click Auto │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  1. Planning (questions skipped)    │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  2. Implementation                 │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────┐     ┌──────────┐
│ Quality     │────▶│ Failed?  │
└──────┬──────┘     └────┬─────┘
       │               │
    Passed          Yes │
       │               │ retry
       │               ▼
       │    ┌─────────────────────┐
       │    │ Add errors to notes │
       │    │ Re-implement        │
       │    └─────────┬───────────┘
       │              │
       │◄─────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  4. Review                         │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  5. Finish (merge to target)        │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│  Complete!                          │
└─────────────────────────────────────┘
```

### Progress Monitoring

While auto runs, the dashboard shows real-time progress:

- **Current Phase** - Which step is executing
- **Agent Output** - Live streaming from the AI
- **Quality Attempts** - Retry count for quality checks
- **File Changes** - Code being generated in real-time

### Quality Retry Loop

If quality checks fail, auto mode automatically:

1. Captures the quality check output (error messages)
2. Appends errors to task notes as feedback
3. Re-runs the implementation phase (agent sees what failed)
4. Re-runs quality checks
5. Repeats until quality passes or max retries exceeded

This creates a self-correcting feedback loop where the AI can fix its own mistakes.

**Default Max Retries:** 3 (configurable)

### Agent Question Handling

In auto mode, if the AI asks a clarifying question during planning:
- The question is **skipped**
- The agent proceeds with its best guess
- Skipped questions are logged for audit

This ensures fully non-interactive execution.

## Auto Mode Output

### Success

```
[AUTO] Task registered
[AUTO] Entering planning phase...
[AUTO] Planning complete
[AUTO] Entering implementation phase...
[AUTO] Implementation complete
[AUTO] Quality check attempt 1/3...
[AUTO] Quality checks passed
[AUTO] Entering review phase...
[AUTO] Review complete
[AUTO] Finishing task...
[AUTO] Task completed

AUTO complete!
  Quality attempts: 1
  Changes merged (not pushed)
```

### Failure

```
[AUTO] Task registered
[AUTO] Planning complete
[AUTO] Implementation complete
[AUTO] Quality check attempt 1/3...
[AUTO] Quality check attempt 2/3...
[AUTO] Quality check attempt 3/3...
[AUTO] Quality checks failed

AUTO failed at: quality
  Planning:       done
  Implementation: done
  Review:         pending
  Finish:         blocked
```

## When to Use Auto Mode

**Good for:**
- Well-defined, self-contained tasks
- Routine bug fixes with clear reproduction steps
- Tasks you've successfully run before
- Batch processing multiple similar tasks
- Tasks with clear acceptance criteria
- Refactoring with explicit goals

**Not recommended for:**
- Complex tasks requiring clarification
- Tasks with ambiguous requirements
- When you want to review specifications before implementation
- Tasks requiring domain-specific knowledge you need to provide
- First-time tasks in a new codebase

## Comparison with Manual Workflow

| Aspect | Auto | Manual |
|--------|------|--------|
| User interaction | None | At each step |
| Agent questions | Skipped | Answered |
| Quality failures | Auto-retry | Manual fix |
| Review specifications | No | Yes |
| Control | Less | Full |

## CLI Equivalent

See [`mehr auto`](../cli/auto.md) for CLI usage.

```bash
# CLI equivalent of clicking Auto
mehr auto task.md

# With quality retry limit
mehr auto --max-retries 5 task.md

# Skip quality checks entirely
mehr auto --no-quality task.md
```

## Common Workflows

### Well-Defined Bug Fix

```
1. Create task: "Fix null pointer in auth handler"
2. Click Auto
3. Mehrhof: plans → implements → checks → fixes → merges
4. Done!
```

### Batch Processing

```
1. Create multiple task files
2. Run auto on each sequentially
3. All tasks completed without intervention
```

### CI/CD Integration

```bash
# In your CI pipeline
mehr auto --max-retries 1 --quality-target lint task.md
```

## Notes

- Auto mode commits automatically (no checkpoint created)
- Changes are not pushed by default (configure with settings)
- Branch is deleted after merge (default, configurable)
- Use quality checks to catch issues before merge
- Review logs even on success to understand what changed
