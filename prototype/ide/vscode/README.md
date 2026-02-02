# Mehrhof VS Code Extension

Structured creation environment for software development, integrated directly into VS Code.

## Features

- **Interactive Panel**: Chat with AI agents and execute workflow commands
- **Task Tree View**: Browse and manage tasks
- **Status Bar**: Quick view of current workflow state
- **Server Management**: Start/stop the Mehrhof server from VS Code
- **Real-time Updates**: Live streaming of agent output via SSE

## Installation

### From Source

1. Clone the repository and navigate to the extension:
   ```bash
   cd ide/vscode
   ```

2. Install dependencies and build:
   ```bash
   bun install
   bun run compile
   ```

3. Package and install:
   ```bash
   bun run package
   code --install-extension mehrhof-*.vsix
   ```

## Usage

1. Open a project with a `.mehrhof/` directory
2. Click the Mehrhof icon in the Activity Bar
3. Click "Start Server" to launch the Mehrhof server
4. Use the Interactive panel to:
   - Start tasks: `start github:123` or `start file:path/to/task.md`
   - Execute workflow: `plan`, `implement`, `review`, `finish`
   - Chat with agents
   - Undo/redo checkpoints

## Configuration

Open VS Code Settings and search for "Mehrhof":

| Setting | Description | Default |
|---------|-------------|---------|
| `mehrhof.serverUrl` | Manual server URL (leave empty to auto-start) | `""` |
| `mehrhof.mehrExecutable` | Path to `mehr` binary | Auto-detect |
| `mehrhof.showNotifications` | Show notification messages | `true` |
| `mehrhof.defaultAgent` | Default agent for workflow steps | `""` |
| `mehrhof.autoReconnect` | Auto-reconnect on disconnect | `true` |
| `mehrhof.reconnectDelaySeconds` | Delay between reconnect attempts | `5` |
| `mehrhof.maxReconnectAttempts` | Maximum reconnect attempts | `10` |

## Commands

All commands are available via Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`):

| Command | Description |
|---------|-------------|
| `Mehrhof: Start Server` | Start the Mehrhof server |
| `Mehrhof: Stop Server` | Stop the Mehrhof server |
| `Mehrhof: Connect` | Connect to server |
| `Mehrhof: Disconnect` | Disconnect from server |
| `Mehrhof: Start Task...` | Start a new task |
| `Mehrhof: Plan` | Execute planning phase |
| `Mehrhof: Implement` | Execute implementation phase |
| `Mehrhof: Review` | Execute review phase |
| `Mehrhof: Continue` | Continue workflow |
| `Mehrhof: Finish` | Complete the task |
| `Mehrhof: Abandon` | Abandon current task |
| `Mehrhof: Undo` | Revert to previous checkpoint |
| `Mehrhof: Redo` | Restore next checkpoint |
| `Mehrhof: Status` | Show task status |
| `Mehrhof: Refresh` | Refresh state |

## Authentication & Security

The extension supports authenticated connections to remote Mehrhof servers:

- **Session cookies** are automatically extracted from server responses and sent on subsequent requests
- **CSRF tokens** are included on state-changing requests (POST/PUT/DELETE) via the `X-Csrf-Token` header
- **SSE connections** include session cookies for authenticated event streaming

In localhost mode (default), authentication and CSRF are not enforced. When connecting to a remote server with `mehrhof.serverUrl`, you may need to configure authentication via the server's CLI (`mehr serve auth add`).

## Development

```bash
# Install dependencies
bun install

# Build
bun run compile

# Watch mode
bun run watch

# Lint
bun run lint

# Package
bun run package
```

## Requirements

- VS Code 1.85.0 or later
- Mehrhof CLI (`mehr`) installed
- Node.js 18+ (for extension runtime)

## License

MIT
