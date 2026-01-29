# Interactive Mode

Interactive mode provides a REPL (Read-Eval-Print Loop) interface for continuous interaction with the AI agent.

## Usage

```bash
mehr interactive
```

## Features

- **Real-time chat**: Type messages directly to the agent without entering commands
- **Workflow commands**: Execute all workflow commands within the session
- **Agent streaming**: See agent responses in real-time as they're generated
- **Command history**: Use arrow keys to navigate previous commands (tab completion for commands)
- **State-aware prompts**: The prompt shows the current workflow state

## Commands

### Chat

| Command | Aliases | Description |
|---------|----------|-------------|
| `chat <msg>` | `ask`, `c` | Chat with the agent |
| `answer <resp>` | `a` | Answer agent's question |
| `note <msg>` | - | Add a note to the current task |

### Workflow

| Command | Aliases | Description |
|---------|----------|-------------|
| `start <ref>` | - | Start a new task from reference |
| `plan [prompt]` | - | Enter planning phase |
| `implement` | `impl` | Execute specifications |
| `review` | - | Review code |
| `continue` | `cont` | Resume from waiting/paused |
| `finish` | - | Complete the task |
| `abandon` | - | Discard the task |

### Info

| Command | Aliases | Description |
|---------|----------|-------------|
| `status` | `st` | Show task status |
| `cost` | - | Show token usage and costs |
| `budget` | - | Show token budget status |
| `list` | - | List all tasks |
| `specification <n>` | `spec` | View specification |

### Search

| Command | Aliases | Description |
|---------|----------|-------------|
| `find <query>` | - | AI-powered code search |
| `memory <query>` | - | Search semantic memory |

### Task Management

| Command | Aliases | Description |
|---------|----------|-------------|
| `simplify [files]` | - | Simplify code based on workflow state |
| `label add <lbl...>` | - | Add labels to active task |
| `label remove <lbl...>` | `label rm` | Remove labels from active task |
| `label set <lbl...>` | - | Set all labels (replace) |
| `label clear` | - | Clear all labels |
| `label list` | `label ls` | List all labels |

### Control

| Command | Aliases | Description |
|---------|----------|-------------|
| `quick <desc>` | - | Create a quick task |
| `undo` | - | Undo to previous checkpoint |
| `redo` | - | Redo to next checkpoint |

### Session

| Command | Aliases | Description |
|---------|----------|-------------|
| `clear` | - | Clear screen |
| `help` | `?` | Show available commands |
| `exit` | `quit`, `q` | Exit interactive mode |

**Note**: Any unrecognized input will be sent to the agent as a chat message.

## Example Session

```bash
$ mehr interactive
mehrhof (idle) > start task.md
✓ Task started successfully
Next: Use plan to enter planning phase

mehrhof (idle) > plan
✓ Planning phase started
[planning] → Agent: Creating specifications...

mehrhof (planning) > status
Task Status:
  ID:      abc123
  Title:   Add user authentication
  State:   planning
  Specs:   2

mehrhof (planning) > How will the auth work?
[Agent streams response...]

mehrhof (planning) > implement
✓ Implementation phase started

mehrhof (implementing) > exit
Saving chat history...
Goodbye!
```

## Cancellation

Press **Ctrl+C** at any time to immediately cancel the current operation:

- **Agent operations** (chat, plan, implement, review, find, simplify) are terminated immediately
- **The agent process is killed** - no further output is generated
- **You stay in the REPL** - type another command or `exit` to leave
- **Workflow state is preserved** - you can retry the cancelled operation

This is useful when:
- An agent is taking too long
- You realize you want to change your approach
- The agent is going in the wrong direction

```bash
mehrhof (idle) > plan
Entering planning phase...
^C
Operation stopped. Type 'exit' to quit.
mehrhof (idle) > plan   # Retry is safe
```

## Keyboard Shortcuts

- **Ctrl+C**: Cancel current operation (stays in interactive mode)
- **Up/Down Arrows**: Navigate command history
- **Tab**: Auto-complete commands
- **Ctrl+D** or type `exit`: Exit interactive mode

## Flags

| Flag | Description |
|------|-------------|
| `--no-history` | Disable command history (doesn't save to ~/.mehr_history) |

## Chat History

Chat history is automatically saved to `~/.valksor/.mehr_history` for command history navigation.
Session transcripts are preserved in the workspace for context continuity.

## Related Commands

- `mehr start <ref>` - Start a task (one-time, non-interactive)
- `mehr plan` - Enter planning phase (one-time)
- `mehr implement` - Execute specifications (one-time)
- `mehr review` - Review code (one-time)
- `mehr status` - Show task status

## Web UI Equivalent

Interactive mode is also available in the Web UI at `/interactive` when running `mehr serve`.

The Web UI provides:
- Chat interface with real-time streaming
- Command buttons for workflow actions
- State indicators
- Side panel with quick actions

See [Web UI: Interactive Mode](../web-ui/interactive.md) for details.
