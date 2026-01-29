# Interactive Mode

Interactive mode in the Web UI provides a chat-like interface for real-time communication with the AI agent, combined with workflow control buttons.

## Access

Navigate to `/interactive` in your browser while the server is running:

```bash
# Start the server
mehr serve

# Open your browser to
http://localhost:PORT/interactive
```

## Features

### Real-Time Chat

The chat interface allows you to:
- **Send messages** directly to the AI agent
- **Receive streaming responses** in real-time
- **View conversation history** in the chat panel

### Workflow Actions

Quick action buttons let you control the workflow:

| Action | Description |
|--------|-------------|
| **Status** | Show current task status |
| **Plan** | Enter planning phase |
| **Implement** | Execute specifications |
| **Review** | Review code |
| **Continue** | Resume from waiting/paused |

### State Panel

The side panel shows:
- **Current workflow state** (Planning, Implementing, Reviewing, etc.)
- **Active task title** (if a task is running)
- **Auto-refresh** every 30 seconds or when state changes

### Commands Reference

A quick reference guide shows available commands:
- `chat <msg>` - Chat with agent
- `start <ref>` - Start a new task
- `plan` - Enter planning phase
- `implement` - Execute specifications
- `review` - Review code
- `continue` - Resume from waiting
- `answer <resp>` - Answer agent's question
- `undo` / `redo` - Navigate checkpoints

## API Endpoints

The interactive page uses these API endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/interactive` | GET | Render interactive page |
| `/api/v1/interactive/chat` | POST | Send chat message to agent |
| `/api/v1/interactive/command` | POST | Execute workflow command |
| `/api/v1/interactive/state` | GET | Get current state |
| `/api/v1/interactive/answer` | POST | Answer agent question |
| `/api/v1/interactive/stop` | POST | Stop current operation |

### Chat Request

```json
{
  "message": "Help me understand this code"
}
```

### Command Request

```json
{
  "command": "plan",
  "args": []
}
```

### State Response

```json
{
  "success": true,
  "state": "planning",
  "task_id": "abc123",
  "title": "Add user authentication"
}
```

## Comparison with CLI Interactive Mode

| Feature | Web UI | CLI |
|---------|--------|-----|
| Chat interface | Visual panel | Terminal REPL |
| Command history | N/A | Up/Down arrows |
| Tab completion | N/A | Yes |
| Copy/paste | Easy | Terminal dependent |
| Remote access | Any browser | SSH required |

For CLI interactive mode, see [`mehr interactive`](../cli/interactive.md).

## See Also

- [Web UI Overview](index.md)
- [CLI: Interactive Mode](../cli/interactive.md)
- [REST API](api.md)
