# Web UI

The Mehrhof Web UI provides a browser-based interface for managing AI-powered tasks. It offers the same functionality as the CLI with a visual, interactive experience.

## When to Use the Web UI

| Use Case | Web UI | CLI |
|----------|--------|-----|
| **First-time users** | ✅ Visual discovery | ⚠️ Requires memorizing commands |
| **Screen sharing** | ✅ Easy to follow | ❌ Hard to see terminal |
| **Task monitoring** | ✅ Real-time dashboard | ⚠️ Manual status checks |
| **Configuration** | ✅ Forms with validation | ⚠️ Editing YAML files |
| **Automation** | ❌ Not available | ✅ Scriptable |
| **Git workflows** | ⚠️ Possible | ✅ Native integration |
| **Remote access** | ✅ Any browser | ⚠️ SSH required |

## Getting Started

1. **Start the server:**
   ```bash
   mehr serve --open
   ```

2. **Open your browser** to `http://localhost:XXXX`

3. **Create your first task** by clicking "Create Task"

For a complete walkthrough, see [Getting Started](getting-started.md).

## Core Features

### Dashboard

The main dashboard provides:
- **Workflow State Diagram** - Visual representation of current state
- **Active Task Card** - Title, state, branch, progress
- **Quick Actions** - Plan, Implement, Review, Finish buttons
- **Agent Output** - Real-time streaming of AI responses
- **Task History** - Browse and resume past tasks

### Workflow Actions

Each workflow phase has a dedicated button:

| Action | Button Location | Description |
|--------|----------------|-------------|
| Create Task | Dashboard | Start a new task from file, URL, or provider |
| Plan | Active Task Card | Generate implementation specifications |
| Implement | Active Task Card | Execute specifications and write code |
| Review | Active Task Card | Run automated code review |
| Finish | Active Task Card | Complete and merge the task |
| Continue | Dashboard | Auto-execute next logical step |
| Undo/Redo | Dashboard | Navigate checkpoint history |

See individual workflow pages for details:
- [Creating Tasks](creating-tasks.md)
- [Planning](planning.md)
- [Implementing](implementing.md)
- [Reviewing](reviewing.md)
- [Finishing](finishing.md)
- [Continuing](continuing.md)
- [Undo & Redo](undo-redo.md)

### Additional Features

- **Interactive Mode** - Chat-based interface for real-time agent communication
- **Settings** - Configure agents, providers, git, and workflow options
- **Browser Control** - Automate Chrome for testing and authentication
- **Task History** - Search, filter, and resume past tasks
- **Notes** - Add context and answer agent questions
- [REST API](api.md) - Programmatic access to all features

## Remote Access

Access the Web UI from anywhere:
- [Remote Access](remote-access.md) - SSH tunnels, Cloudflare, Tailscale
- [Authentication](authentication.md) - User management and security

## Comparison with CLI

Both interfaces provide the same core functionality. The choice depends on your workflow:

| Feature | Web UI | CLI |
|---------|--------|-----|
| Task management | Buttons and forms | Commands and flags |
| Status monitoring | Always visible | Run `mehr status` |
| Configuration | Visual forms | Edit YAML files |
| Automation | Not available | Scriptable |
| Learning curve | Lower | Higher |

For a detailed comparison, see [Web UI vs CLI](../guides/web-ui-vs-cli.md).

## Starting the Server

```bash
# Basic usage
mehr serve

# Open browser automatically
mehr serve --open

# Specific port
mehr serve --port 3000

# Global mode (all projects)
mehr serve --global
```

See [CLI: serve](../cli/serve.md) for all server options.

## Next Steps

- [**Getting Started**](getting-started.md) - Complete walkthrough
- [**Creating Tasks**](creating-tasks.md) - Task creation workflows
- [**Settings**](settings.md) - Configuration guide
- [**REST API**](api.md) - Programmatic access
