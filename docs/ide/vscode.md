# VS Code Extension

The Mehrhof VS Code extension integrates structured creation workflows directly into your editor.

## Prerequisites

- VS Code 1.108.0 or later
- Mehrhof CLI installed (`go install github.com/valksor/go-mehrhof/cmd/mehr@latest`)

### Build Requirements (for development)

- Node.js 24 or later
- [Bun](https://bun.sh/) package manager

## Installation

### From VSIX (Recommended)

Download the latest `.vsix` from [GitHub Releases](https://github.com/valksor/go-mehrhof/releases) and install:

```bash
code --install-extension mehrhof-*.vsix
```

### From Source

1. Install Bun if not already installed:
   ```bash
   curl -fsSL https://bun.sh/install | bash
   ```

2. Navigate to the extension directory:
   ```bash
   cd ide/vscode
   ```

3. Install dependencies and build:
   ```bash
   bun install
   bun run compile
   ```

4. Package and install:
   ```bash
   bun run package
   code --install-extension mehrhof-*.vsix
   ```

## Getting Started

1. **Open a Mehrhof project** - Open a folder containing a `.mehrhof/` directory
2. **Open the Mehrhof panel** - Click the Mehrhof icon in the Activity Bar
3. **Start the server** - Click "Start Server" in the Interactive panel
4. **Start a task** - Type `start github:123` or click "Start Task"

## Features

### Interactive Panel

The main interface for interacting with Mehrhof. Located in the sidebar.

**Components:**
- **Header**: Server controls and connection status
- **Task Info**: Current task title and workflow state
- **Messages**: Chat history with agents
- **Input Field**: Type commands or chat messages
- **Action Buttons**: Quick access to workflow commands

**Commands:**
- `start <ref>` - Start a new task
- `plan` - Execute planning phase
- `implement` - Execute implementation phase
- `review` - Execute code review
- `finish` - Complete the task
- `abandon` - Abandon the task
- `undo` / `redo` - Navigate checkpoints
- `status` - Show task status
- `cost` - Show token usage

**Chat:**
Type any message that's not a command to chat with the agent.

### Task Tree View

Browse all tasks in the current project.

- **Active task** shown at top with highlight
- **State icons** indicate task status (done, failed, in progress)
- **Click to refresh** using the refresh button

### Status Bar

Quick view of workflow state in the VS Code status bar.

- Shows: `Mehrhof: [State] - [Task Title]`
- Click to show action picker
- Color-coded for active states

### Output Channel

View detailed logs in the "Mehrhof" output channel.

- Agent messages
- State transitions
- Errors and warnings

## Configuration

Access settings via `File > Preferences > Settings` and search "Mehrhof".

| Setting                         | Type    | Default | Description                                             |
|---------------------------------|---------|---------|---------------------------------------------------------|
| `mehrhof.serverUrl`             | string  | `""`    | Manual server URL. Leave empty to auto-start server.    |
| `mehrhof.mehrExecutable`        | string  | `""`    | Path to `mehr` binary. Auto-detects from PATH if empty. |
| `mehrhof.showNotifications`     | boolean | `true`  | Show notification messages for events.                  |
| `mehrhof.defaultAgent`          | string  | `""`    | Default agent for workflow steps.                       |
| `mehrhof.autoReconnect`         | boolean | `true`  | Automatically reconnect on disconnect.                  |
| `mehrhof.reconnectDelaySeconds` | number  | `5`     | Seconds to wait between reconnect attempts.             |
| `mehrhof.maxReconnectAttempts`  | number  | `10`    | Maximum number of reconnection attempts.                |

## Commands

All commands available via Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`):

### Server Commands
- **Mehrhof: Start Server** - Launch the Mehrhof server
- **Mehrhof: Stop Server** - Stop the running server

### Connection Commands
- **Mehrhof: Connect** - Connect to server (auto-starts if needed)
- **Mehrhof: Disconnect** - Disconnect from server

### Workflow Commands
- **Mehrhof: Start Task...** - Prompt for task reference and start
- **Mehrhof: Plan** - Execute planning phase
- **Mehrhof: Implement** - Execute implementation phase
- **Mehrhof: Review** - Execute code review phase
- **Mehrhof: Continue** - Continue paused workflow
- **Mehrhof: Finish** - Complete and merge the task
- **Mehrhof: Abandon** - Abandon current task (with confirmation)

### Navigation Commands
- **Mehrhof: Undo** - Revert to previous checkpoint
- **Mehrhof: Redo** - Restore next checkpoint

### Information Commands
- **Mehrhof: Status** - Show current task status
- **Mehrhof: Refresh** - Refresh state from server

## Workflow States

The extension displays the current workflow state:

| State        | Icon | Color  | Description                  |
|--------------|------|--------|------------------------------|
| idle         | ○    | Gray   | No active task               |
| planning     | ✎    | Blue   | AI generating specifications |
| implementing | < >  | Orange | AI implementing code         |
| reviewing    | 👁   | Purple | Code review in progress      |
| waiting      | ?    | Yellow | Waiting for user input       |
| done         | ✓    | Green  | Task completed               |
| failed       | ✗    | Red    | Task failed                  |

## Keyboard Shortcuts

In the Interactive panel input field:
- `Enter` - Send message/command
- `↑` / `↓` - Navigate command history

## Troubleshooting

### Server won't start

1. Check that `mehr` is installed: `which mehr`
2. Set the path manually in settings: `mehrhof.mehrExecutable`
3. Check the Output channel for error messages

### Connection keeps dropping

1. Enable auto-reconnect: `mehrhof.autoReconnect: true`
2. Increase reconnect attempts: `mehrhof.maxReconnectAttempts: 20`
3. Check server logs for issues

### Commands not working

1. Ensure you're connected (check status bar)
2. Verify you have an active task for workflow commands
3. Check the Output channel for error details

## Architecture

The extension communicates with the Mehrhof server via:
- **REST API** - For commands and state queries
- **Server-Sent Events (SSE)** - For real-time updates

```
┌─────────────────────────────────────┐
│         VS Code Extension           │
│  ├── Interactive Panel (Webview)    │
│  ├── Task Tree View                 │
│  ├── Status Bar Widget              │
│  └── Output Channel                 │
└──────────────┬──────────────────────┘
               │ HTTP (REST + SSE)
┌──────────────▼──────────────────────┐
│     Mehrhof Server (mehr serve)     │
└─────────────────────────────────────┘
```

## See Also

- [JetBrains Plugin](jetbrains.md) - IntelliJ IDEA integration
- [Web UI](../web-ui/interactive.md) - Browser-based interface
- [CLI Reference](../cli/index.md) - Command-line interface
