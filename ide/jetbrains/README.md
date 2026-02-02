# Mehrhof IntelliJ Plugin

JetBrains IDE plugin for [Mehrhof](https://github.com/valksor/go-mehrhof) — structured creation environment.

## Features

- **Interactive Terminal**: Full interactive mode matching CLI and Web UI
- **Integrated Server Management**: Start/stop Mehrhof server directly from IDE
- **Chat with Agent**: Send messages and receive streaming responses
- **Full Workflow Control**: Start, Plan, Implement, Review, Continue, Finish, Abandon
- **Checkpoint Navigation**: Undo/Redo to previous states
- **Command History**: Navigate previous commands with up/down arrows
- **Real-time Updates**: SSE streaming for live state changes and agent output
- **Status Bar Widget**: Current workflow state, click for quick actions

## Requirements

- IntelliJ IDEA 2025.1+ (or other JetBrains IDE)
- `mehr` CLI installed (auto-detected from common paths)

## Installation

### From ZIP

1. Build the plugin: `make build`
2. In your IDE: Settings → Plugins → ⚙️ → Install Plugin from Disk
3. Select `build/distributions/mehrhof-intellij-0.1.0.zip`
4. Restart IDE

### From Source (Development)

```bash
# Run in sandbox IDE
make run
```

## Usage

1. Open the Mehrhof tool window (right sidebar)
2. Click **"Start Server"** to launch `mehr serve`
3. Use the **Interactive** tab:
   - Type commands in the input field (e.g., `start github:123`, `plan`, `implement`)
   - Or use the action buttons on the right panel
   - Chat with agent by typing messages
4. Monitor output in the messages area
5. Click **"Stop Server"** when done

### Available Commands

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
| `note <msg>` | Add a note |
| `help` | Show available commands |
| `clear` | Clear messages |

## Configuration

Settings → Tools → Mehrhof:

| Setting | Default | Description |
|---------|---------|-------------|
| mehr executable | (auto-detect) | Path to `mehr` binary. Auto-detects from `~/.local/bin`, `~/bin`, `/usr/local/bin` |
| Server URL | (empty) | Optional: manual server address. Leave empty to use Start Server button |
| Show Notifications | `true` | Balloon notifications for events |
| Auto-reconnect | `true` | Reconnect on disconnect |
| Default Agent | (empty) | Agent to use for workflow steps |

## Building

```bash
# Build plugin
make build

# Output: build/distributions/mehrhof-intellij-0.1.0.zip
```

## Development

```bash
# Compile only
make check

# Run in sandbox
make run

# Run tests
make test
```

## Project Structure

```
src/main/kotlin/com/valksor/mehrhof/
├── api/
│   ├── MehrhofApiClient.kt    # REST + Interactive API client
│   ├── EventStreamClient.kt   # SSE client for real-time events
│   └── models/                # API request/response DTOs
├── services/
│   └── MehrhofProjectService.kt  # Server process & connection management
├── actions/
│   └── WorkflowActions.kt     # IDE actions (start, plan, implement, etc.)
├── toolwindow/
│   ├── MehrhofToolWindowFactory.kt
│   ├── InteractivePanel.kt    # Main interactive terminal UI
│   ├── TaskListPanel.kt       # Task list view
│   └── OutputPanel.kt         # Agent output viewer
├── statusbar/
│   └── MehrhofStatusWidget.kt # Status bar widget
└── settings/
    ├── MehrhofSettings.kt     # Persistent settings
    └── MehrhofConfigurable.kt # Settings UI
```

## API Endpoints Used

The plugin uses the Interactive API endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/interactive/command` | POST | Execute workflow commands |
| `/api/v1/interactive/chat` | POST | Chat with agent |
| `/api/v1/interactive/answer` | POST | Answer agent question |
| `/api/v1/interactive/state` | GET | Get current state |
| `/api/v1/interactive/stop` | POST | Cancel running operation |
| `/api/v1/events` | GET (SSE) | Real-time event stream |

## Authentication & Security

The plugin supports authenticated connections to remote Mehrhof servers:

- **Session cookies** are automatically extracted from server responses and sent on subsequent requests
- **CSRF tokens** are included on POST requests via the `X-Csrf-Token` header
- **SSE connections** include session cookies for authenticated event streaming

In localhost mode (default when using "Start Server"), authentication and CSRF are not enforced. When configuring a manual Server URL in settings, you may need to configure authentication via the server's CLI (`mehr serve auth add`).

## License

Same as go-mehrhof.
