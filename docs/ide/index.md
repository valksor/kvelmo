# IDE Integrations

Mehrhof provides native IDE plugins that integrate directly with your development environment, giving you access to AI-powered task automation without leaving your editor.

## Available Integrations

| IDE                     | Status      | Installation            |
|-------------------------|-------------|-------------------------|
| [JetBrains](#jetbrains) | ✅ Available | [Install](jetbrains.md) |
| [VS Code](#vscode)      | ✅ Available | [Install](vscode.md)    |

## How It Works

IDE plugins connect to a running Mehrhof server (`mehr serve`) via the REST API. They provide:

- **Tool Window/Sidebar** - Task list, specifications, agent output
- **Status Bar** - Current workflow state at a glance
- **Menu Actions** - Plan, Implement, Review, Finish from the IDE
- **Real-time Updates** - Live streaming of agent output via SSE

```
┌─────────────────────────────────────────────────────────┐
│ IDE Plugin                                              │
│  ├── REST API calls to localhost:3000/api/v1/*          │
│  └── SSE stream for real-time updates                   │
└─────────────────────────────────────────────────────────┘
         ↓ HTTP
┌─────────────────────────────────────────────────────────┐
│ mehr serve (Web UI Server)                              │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

All IDE integrations require:

1. **Mehrhof installed** - `go install github.com/valksor/go-mehrhof/cmd/mehr@latest`
2. **Server running** - `mehr serve` in your project directory
3. **IDE plugin installed** - See integration-specific docs

## JetBrains

Supports IntelliJ IDEA, GoLand, WebStorm, PyCharm, and other JetBrains IDEs (2025.1+).

**Features:**
- Tool window with task list and agent output
- Status bar widget showing current state
- Tools menu with all workflow actions
- Settings panel for server configuration

[Full JetBrains documentation →](jetbrains.md)

## VS Code

VS Code extension providing full Mehrhof integration.

**Features:**
- Interactive sidebar panel with chat and workflow controls
- Task tree view for browsing tasks
- Status bar widget showing current state
- Command palette integration for all workflow commands
- Output channel for agent logs

[Full VS Code documentation →](vscode.md)
