# mehr note

Add notes to the current task.

## Synopsis

```bash
mehr note [message]
mehr answer [message]
```

## Description

The `note` command saves notes to the task. Notes are appended to `notes.md` and included in subsequent planning and implementation prompts.

Unlike `plan` or `implement`, the `note` command does **not** run the AI agent. It simply saves your input as a note for future reference.

## Arguments

| Argument  | Description          |
|-----------|----------------------|
| `message` | Optional note to add |

## Flags

| Flag        | Short | Type   | Default | Description                             |
|-------------|-------|--------|---------|-----------------------------------------|
| `--running` |       | string | ""      | Send note to a running parallel task ID |

## Examples

### Interactive Mode

```bash
mehr note
```

Opens an interactive prompt for adding multiple notes:

```
Task: abc12345 (state: idle)

Entering interactive mode. Type 'exit' or 'quit' to leave.

> The API should use REST, not GraphQL
Note saved.

> Add error handling for edge cases
Note saved.

> exit
```

### Single Message

```bash
mehr note "Use PostgreSQL for the database, not SQLite"
```

Adds a single note without entering interactive mode.

### Answering Agent Questions

When the agent asks a question during planning, use the `answer` alias (which is `note` under a different name):

```bash
mehr plan
mehr answer "Use PostgreSQL - we already have it in production"
mehr plan
```

### Multiple Notes Before Planning

```bash
mehr start file:task.md
mehr note "Focus on security - this will handle payments"
mehr note "Use JWT for authentication"
mehr note "Add rate limiting to prevent abuse"
mehr plan
```

All notes are included when the agent runs during planning.

### Send Note to Running Parallel Task

When tasks are running in parallel (via `mehr start --parallel`), send notes to a specific running task:

```bash
# List running tasks to get IDs
mehr list --running

# Send note to a specific running task
mehr note --running=abc123 "Consider edge case X"
mehr note --running=def456 "Use the existing UserService"
```

This is useful for:
- Providing real-time guidance to parallel tasks
- Answering questions from a specific task
- Steering implementation direction mid-execution

## What Happens

1. **Note Saving**
   - Input appended to `notes.md`
   - Timestamp and current state added

2. **Answer Persistence** (when answering agent questions)
   - Q&A pair saved to `notes.md`
   - Answer recorded in latest session file (`sessions/`)
   - Full context archived to `transcripts/` for recovery

3. **No Agent Interaction**
   - The AI agent is NOT called
   - No tokens are consumed
   - Note is saved directly to disk

4. **Future Impact**
   - Notes included in `plan` prompts
   - Notes included in `implement` prompts
   - Notes included in `review` prompts
   - Q&A history from sessions available on resume

## Notes File

Notes are saved to `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/notes.md`:

```markdown
# Notes

## 2025-01-15 10:45:00 [idle]

Use PostgreSQL for the database, not SQLite.

## 2025-01-15 11:00:00 [idle]

Focus on security - this will handle payments.

## 2025-01-15 11:30:00 [waiting]

**Q:** Should we use PostgreSQL or MySQL?
**A:** Use PostgreSQL - we already have it in production.
```

## Interactive Commands

In interactive mode:

| Command | Action      |
|---------|-------------|
| `exit`  | End session |
| `quit`  | End session |
| Ctrl+D  | End session |

## Use Cases

### Add Requirements

```bash
mehr note "The search should support fuzzy matching with typo tolerance"
```

### Provide Context

```bash
mehr note "We're using the existing UserRepository for data access"
```

### Request Changes

```bash
mehr note "Remove the caching - we'll add it later"
```

### Answer Questions

```bash
mehr answer "Use dependency injection instead of global state"
```

## Workflow

Typical workflow for adding context before planning:

```bash
mehr start file:task.md

mehr note "Focus on performance"
mehr note "Add comprehensive error handling"
mehr note "Include unit tests for all new code"

mehr plan
```

## Web UI

Prefer a visual interface? See [Web UI: Notes](/web-ui/notes.md).

## See Also

- [plan](plan.md) - Create specifications (agent runs here)
- [implement](implement.md) - Generate code (agent runs here)
- [Workflow](/concepts/workflow.md) - Understanding states
