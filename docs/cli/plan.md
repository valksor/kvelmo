# mehr plan

Create implementation specifications for the active task.

## Synopsis

```bash
mehr plan [flags]
```

**Aliases:** `p`

## Description

The `plan` command runs the planning phase where the AI agent:

1. Reads the task source content
2. Reviews any existing notes
3. Analyzes requirements
4. Generates specification files with implementation details

specification files are saved to `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/specifications/`.

## Flags

| Flag             | Short | Type   | Default | Description                                |
|------------------|-------|--------|---------|--------------------------------------------|
| `--standalone`   |       | bool   | false   | Start standalone planning (no task)        |
| `--seed`         | `-s`  | string |         | Initial topic for standalone planning      |
| `--verbose`      | `-v`  | bool   | false   | Show agent output in real-time             |
| `--agent-plan`   |       | string |         | Override agent for planning step           |
| `--full-context` |       | bool   | false   | Load full transcripts instead of summaries |
| `--optimize`     |       | bool   | false   | Optimize prompt before sending to agent    |
| `--library`      |       | bool   | false   | Include relevant library docs in prompts   |

**Note:** For standalone mode, you can also provide the seed topic as a positional argument:
```bash
mehr plan --standalone "build a CLI"           # positional argument
mehr plan --standalone --seed "build a CLI"    # flag argument (equivalent)
```

## Examples

### Plan Active Task

```bash
mehr plan
```

Output:

```
Planning task a1b2c3d4...
Created: specification-1.md
Created: specification-2.md
Planning complete. 2 specifications created.
```

### Verbose Output

```bash
mehr plan --verbose
```

Shows the agent's reasoning and progress in real-time.

### Resume with Full Context

```bash
mehr plan --full-context
```

When resuming after answering an agent question, use `--full-context` to load the complete exploration transcript instead of summaries. This provides more context to the agent but uses more tokens.

### Standalone Planning

```bash
mehr plan --standalone
```

Start an interactive planning session without a formal task:

```
Enter topic: How should I implement user authentication?
...
Plan saved to: .mehrhof/planned/xyz789/
```

### Standalone with Seed

```bash
mehr plan --standalone "Design a REST API for products"
```

Skip the topic prompt by providing it directly.

### Override Planning Agent

```bash
mehr plan --agent-plan claude-opus
```

Use a specific agent for this planning session. See [AI Agents](/agents/index.md#per-step-agent-configuration).

### Optimize Prompt

```bash
mehr plan --optimize
```

Optimize the planning prompt using an optimizer agent before sending to the working agent. This can improve clarity and effectiveness of the prompt.

## What Happens

### For Active Tasks

1. **Context Gathering**
   - Reads source from `.mehrhof/work/<id>/`
   - Includes notes from `notes.md`
   - Reviews any existing specifications
   - Loads Q&A history from previous sessions (if any)

2. **Agent Execution**
   - Prompts agent with context
   - Agent analyzes requirements
   - Agent generates structured specifications

3. **Output**
   - specification files written to `specifications/` directory
   - Session logged to `sessions/` (with Q&A exchanges)
   - Full transcript archived to `transcripts/`
   - Checkpoint created for undo support

### For Standalone Planning

1. **Input**
   - Interactive or seed-provided topic
   - No task context required

2. **Output**
   - Plan saved to `.mehrhof/planned/<id>/`
   - Includes `PLAN_HISTORY.md` with conversation

## Specification File Format

Generated specifications follow this format:

```markdown
---
title: Implementation Title
status: draft
created_at: 2025-01-15T10:30:00Z
---

# Component Name

## Overview

Brief description of what this spec covers.

## Implementation Details

1. Step one
2. Step two

## Files to Modify

- `path/to/file.go` - Description
- `path/to/other.go` - Description
```

## Multiple Specifications

The agent may create multiple specification files for complex tasks:

```
specifications/
├── specification-1.md    # Core functionality
├── specification-2.md    # API endpoints
└── specification-3.md    # Tests
```

## Iterating on Plans

You can run `mehr plan` multiple times:

- Additional specifications are appended
- Use `mehr note` to add requirements first
- Delete unwanted specifications manually

## After Planning

Review the specifications:

```bash
cat .mehrhof/work/*/specifications/specification-1.md
```

If satisfied, implement:

```bash
mehr implement
```

If changes needed, add notes:

```bash
mehr note "Simplify the authentication flow"
```

## Web UI

Prefer a visual interface? See [Web UI: Planning](/web-ui/planning.md).

## See Also

- [implement](cli/implement.md) - Implement specifications
- [note](cli/note.md) - Add notes or clarifications
- [Specification Format](/reference/specification-format.md) - Specification file details
