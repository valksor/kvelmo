# mehr optimize

Use AI to improve a task description based on accumulated notes.

## Synopsis

```bash
mehr optimize --task <queue>/<task-id> [flags]
```

## Description

The `optimize` command uses AI to enhance a task based on its accumulated notes. The agent reviews the current task title, description, and any notes added via `mehr note`, then produces an improved version with:

- **Enhanced title** - More descriptive and clear
- **Expanded description** - Incorporates context from notes
- **Relevant labels** - Suggests appropriate labels
- **Improvement notes** - Explains what was changed and why

This is particularly useful for quick tasks that were captured rapidly and need refinement before being submitted to a provider or exported to a proper specification.

## Flags

| Flag               | Description                                               |
|--------------------|-----------------------------------------------------------|
| `--task`           | Queue task ID (format: `<queue-id>/<task-id>`) (required) |
| `--agent-optimize` | Agent to use for optimization (optional)                  |

## Examples

### Basic Optimization

```bash
mehr optimize --task=quick-tasks/task-1
```

Uses the default agent to optimize the task.

### With Specific Agent

```bash
mehr optimize --task=quick-tasks/task-1 --agent-optimize claude-opus
```

Uses a specific AI agent for optimization.

### After Adding Notes

```bash
# Add requirements as notes
mehr note --task=quick-tasks/task-1 "Need to handle edge cases"
mehr note --task=quick-tasks/task-1 "Must support pagination"

# Optimize to incorporate notes
mehr optimize --task=quick-tasks/task-1
```

The AI will incorporate the notes into the improved task description.

## What Happens

1. **Task Loading**
   - Current task title and description loaded
   - All accumulated notes loaded
   - Current labels preserved

2. **AI Analysis**
   - Agent reviews task and notes
   - Identifies areas for improvement
   - Suggests structural changes

3. **Task Update**
   - Title updated to be more descriptive
   - Description expanded with context from notes
   - Relevant labels suggested and added
   - Task saved with improvements

4. **Improvement Report**
   - Shows before/after for title
   - Lists added labels
   - Explains improvements made

## Output Example

```
Task: task-1
  Title: fix typo
  Description: typo in readme
  Notes: 2

✨ Optimizing task with AI...

✨ Task optimized:
  Title: fix typo → Fix typo: "Installaton" → "Installation" in README
  Description: enhanced
    Fix the typo in the README where "Installation" is misspelled as
    "Installaton" in the getting started section...

  Added labels: documentation, typo-fix

  Improvements:
    • Title expanded to include specific location and correction
    • Description includes file location and context
    • Added relevant labels for categorization

Next steps:
  mehr export --task=quick-tasks/task-1 --output task.md
  mehr submit --task=quick-tasks/task-1 --provider <provider>
  mehr start queue:quick-tasks/task-1
```

## Workflow Integration

### Before Export

```bash
# Capture quick task
mehr quick "add search"

# Add requirements as notes
mehr note --task=quick-tasks/task-1 "fuzzy matching"
mehr note --task=quick-tasks/task-1 "debouncing"

# Optimize before export
mehr optimize --task=quick-tasks/task-1

# Export improved version
mehr export --task=quick-tasks/task-1 --output specs/user-search.md
```

### Before Submission

```bash
# Capture task
mehr quick "api returns 500"

# Add debugging notes
mehr note --task=quick-tasks/task-1 "nil pointer in User.FindAll"
mehr note --task=quick-tasks/task-1 "only when users table is empty"

# Optimize for clearer issue description
mehr optimize --task=quick-tasks/task-1

# Submit with improved description
mehr submit --task=quick-tasks/task-1 --provider github
```

### Iterative Refinement

```bash
# Initial capture
mehr quick "user auth"

# First round of notes
mehr note --task=quick-tasks/task-1 "JWT tokens"
mehr note --task=quick-tasks/task-1 "refresh token rotation"
mehr optimize --task=quick-tasks/task-1

# More details
mehr note --task=quick-tasks/task-1 "store refresh tokens in database"
mehr note --task=quick-tasks/task-1 "revoke old tokens on issue"
mehr optimize --task=quick-tasks/task-1

# Export when satisfied
mehr export --task=quick-tasks/task-1 --output specs/auth-system.md
```

## Agent Selection

The optimize command respects the agent priority system:

1. `--agent` flag (highest priority)
2. Task frontmatter `agent:` field
3. Workspace config `agent.default`
4. Auto-detect first available agent

Different agents may produce different optimization styles. Experiment to find which works best for your workflow.

## Cost Considerations

Each optimization consumes tokens based on:

- Current task length
- Number of accumulated notes
- Agent model used

For cost-sensitive workflows, consider:
- Optimizing in batches rather than per-note
- Using a faster/cheaper model for optimization
- Manually editing exported markdown files

## See Also

- [quick](quick.md) - Create quick tasks
- [note](note.md) - Add notes to tasks
- [export](export.md) - Export to markdown
- [submit](submit.md) - Submit to provider
