# JetBrains Plugin

Native plugin for JetBrains IDEs (IntelliJ IDEA, GoLand, WebStorm, PyCharm, etc.) that integrates Mehrhof directly into your development workflow.

## Requirements

- JetBrains IDE **2025.1** or later
- `mehr` CLI installed (auto-detected from common paths)

## Installation

### From GitHub Releases

1. Download the latest `mehrhof-intellij-*.zip` from [Releases](https://github.com/valksor/go-mehrhof/releases)
2. In your IDE: **Settings → Plugins → ⚙️ → Install Plugin from Disk**
3. Select the downloaded ZIP file
4. Restart IDE

### From Source

```bash
cd ide/jetbrains
make build
# Plugin at: build/distributions/mehrhof-intellij-*.zip
```

## Getting Started

1. **Open the Mehrhof tool window** (right sidebar)

2. **Click "Start Server"** - the plugin launches `mehr serve` automatically

3. **Use the Interactive tab** to control the workflow:
   - Type commands like `start github:123`, `plan`, `implement`
   - Or use the action buttons on the right panel
   - Chat with the agent by typing messages

## Features

### Interactive Terminal

The main **Interactive** tab provides a full interactive experience matching the CLI and Web UI:

- **Messages Area**: Shows chat history, command output, and agent responses
- **Input Field**: Type commands or chat messages (Enter to send)
- **Command History**: Navigate with up/down arrow keys
- **Action Buttons**: Quick access to workflow commands
- **State Badge**: Shows current workflow state
- **Stop Button**: Cancel running operations

### Available Commands

Type these in the input field:

| Command | Description |
|---------|-------------|
| `start <ref>` | Start a task (e.g., `start github:123`, `start file:task.md`) |
| `plan` | Run planning phase |
| `implement` | Run implementation phase |
| `review` | Run code review |
| `continue` | Resume from waiting state |
| `finish` | Complete the task |
| `abandon` | Discard the task |
| `undo` / `redo` | Navigate checkpoints |
| `status` | Show task status |
| `cost` | Show token usage |
| `chat <msg>` | Chat with agent |
| `answer <resp>` | Answer agent question |
| `note <msg>` | Add a note to task |
| `find <query>` | AI-powered code search |
| `list` | List all tasks |
| `help` | Show available commands |
| `clear` | Clear messages |

### Tool Window Tabs

**Interactive Tab** (Primary):
- Full command/chat interface
- Messages display with user/assistant styling
- Server start/stop control
- Task info and state badge
- Action buttons panel

**Tasks Tab:**
- Recent tasks list
- Current task details
- Legacy workflow buttons

**Output Tab:**
- Real-time agent output streaming
- Workflow state change notifications

### Status Bar Widget

Shows current connection and workflow state:

- `Mehrhof: Disconnected` - Not connected to server
- `Mehrhof: Idle` - Connected, no active task
- `Mehrhof: Planning - Fix login bug` - Active task with state

Click the widget to refresh state.

### Menu Actions

**Tools → Mehrhof:**

| Action | Description |
|--------|-------------|
| Start Task... | Start a new task (prompts for reference) |
| Plan | Generate specifications |
| Implement | Implement specifications |
| Review | Review implementation |
| Continue | Resume from waiting state |
| Finish | Complete the task |
| Abandon | Discard the task |
| Undo (Checkpoint) | Revert to previous checkpoint |
| Redo (Checkpoint) | Restore to next checkpoint |
| Refresh | Refresh task state |

## Configuration

**Settings → Tools → Mehrhof:**

| Setting | Default | Description |
|---------|---------|-------------|
| mehr executable | *(auto-detect)* | Path to `mehr` binary. Auto-detects from `~/.local/bin`, `~/bin`, `/usr/local/bin` |
| Server URL | *(empty)* | Optional manual server address. Leave empty to use Start Server button |
| Show Notifications | `true` | Enable balloon notifications |
| Auto-reconnect | `true` | Reconnect on disconnect |
| Default Agent | *(empty)* | Agent for workflow steps |
| Reconnect Delay | `5` seconds | Delay between reconnect attempts |
| Max Reconnect Attempts | `10` | Stop after N failed attempts |

## Workflow Example

### Using Interactive Terminal

1. **Start server**: Click "Start Server" button
2. **Start a task**: Type `start github:123` and press Enter
3. **Plan**: Type `plan` or click Plan button
4. **Review output**: Watch agent output in messages area
5. **Answer questions**: When prompted, type `answer <your response>`
6. **Implement**: Type `implement` after planning completes
7. **Review**: Type `review` after implementation
8. **Finish**: Type `finish` to complete

### Using Action Buttons

1. Click "Start Server"
2. Click "Start Task..." → Enter task reference
3. Click "Plan" → "Implement" → "Review" → "Finish"
4. Answer questions in the messages area with `answer <response>`

### Handling Questions

When the agent asks a question:
1. The question appears in the messages area
2. Available options are shown (if any)
3. Type `answer <your response>` to reply
4. The workflow continues

### Undo/Redo

Use checkpoint navigation:
- `undo` - Revert to the previous state
- `redo` - Restore a reverted state

Or use the Undo/Redo buttons in the action panel.

## Troubleshooting

### Server Not Starting

1. Check if `mehr` is installed: `which mehr`
2. Verify mehr executable path in Settings → Tools → Mehrhof
3. Check IDE log for errors: Help → Show Log in Finder

### Plugin Not Connecting

1. Verify server started (check for "Server started on port XXXX" message)
2. Check the messages area for connection errors
3. Try stopping and restarting the server

### No Task Showing

1. Start a task: `start file:task.md` in the input field
2. Click Refresh in the toolbar
3. Check messages area for errors

### Commands Not Working

1. Verify connection status (check state badge shows connected)
2. Check messages area for error messages
3. Ensure task is in correct state for the command

## Development

### Building from Source

```bash
cd ide/jetbrains

# Build plugin
make build

# Run in sandbox IDE
make run

# Run tests
make test

# Compile only (fast check)
make check
```

### Project Structure

```
ide/jetbrains/src/main/kotlin/com/valksor/mehrhof/
├── api/
│   ├── MehrhofApiClient.kt    # REST + Interactive API client
│   ├── EventStreamClient.kt   # SSE streaming
│   └── models/                # API DTOs
├── services/
│   └── MehrhofProjectService.kt  # Server process & state management
├── actions/
│   └── WorkflowActions.kt     # IDE actions
├── toolwindow/
│   ├── MehrhofToolWindowFactory.kt
│   ├── InteractivePanel.kt    # Interactive terminal UI
│   ├── TaskListPanel.kt       # Task list view
│   └── OutputPanel.kt         # Agent output
├── statusbar/
│   └── MehrhofStatusWidget.kt
└── settings/
    ├── MehrhofSettings.kt
    └── MehrhofConfigurable.kt
```

### API Reference

The plugin uses the Interactive API endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/interactive/command` | POST | Execute commands (start, plan, implement, etc.) |
| `/api/v1/interactive/chat` | POST | Chat with agent |
| `/api/v1/interactive/answer` | POST | Answer agent question |
| `/api/v1/interactive/state` | GET | Get current state |
| `/api/v1/interactive/stop` | POST | Cancel running operation |
| `/api/v1/events` | GET (SSE) | Real-time event stream |

Additional endpoints for task info:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/status` | Server status |
| `GET /api/v1/task` | Current task |
| `GET /api/v1/tasks` | Task list |
| `GET /api/v1/guide` | Next actions guidance |
| `GET /health` | Health check |

See [REST API documentation](../web-ui/api.md) for full details.
