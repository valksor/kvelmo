# Web UI

The Mehrhof Web UI provides a comfortable browser-based interface for the plan → create → review workflow. It offers the same functionality as the CLI with a visual, interactive experience—no command-line expertise required.

## When to Use the Web UI

| Use Case                       | Web UI                        | CLI                             |
|--------------------------------|-------------------------------|---------------------------------|
| **First-time users**           | ✅ Visual discovery            | ⚠️ Requires memorizing commands |
| **Non-technical contributors** | ✅ Familiar browser experience | ⚠️ Terminal can be intimidating |
| **Screen sharing**             | ✅ Easy to follow              | ❌ Hard to see terminal          |
| **Task monitoring**            | ✅ Real-time dashboard         | ⚠️ Manual status checks         |
| **Configuration**              | ✅ Forms with validation       | ⚠️ Editing YAML files           |
| **Automation**                 | ❌ Not available               | ✅ Scriptable                    |
| **Git workflows**              | ⚠️ Possible                   | ✅ Native integration            |
| **Remote access**              | ✅ Any browser                 | ⚠️ SSH required                 |

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
- **Workflow State Diagram** - Visual representation of current step and available actions
- **Active Task Card** - Title, state, branch, progress
- **Quick Actions** - Plan, Create, Review, Finish buttons
- **Live Output** - Real-time streaming of progress
- **Task History** - Browse and resume past tasks

### Workflow Actions

Each workflow phase has a dedicated button:

| Action      | Button Location  | Description                                  |
|-------------|------------------|----------------------------------------------|
| Create Task | Dashboard        | Start a new task from file, URL, or provider |
| Plan        | Active Task Card | Generate a structured plan                   |
| Create      | Active Task Card | Execute the plan to produce changes          |
| Review      | Active Task Card | Run quality checks                           |
| Finish      | Active Task Card | Complete and merge the task                  |
| Continue    | Dashboard        | Execute next logical step                    |
| Undo/Redo   | Dashboard        | Navigate checkpoint history                  |

See individual workflow pages for details:
- [Creating Tasks](creating-tasks.md)
- [Planning](planning.md)
- [Implementing](implementing.md)
- [Reviewing](reviewing.md)
- [Finishing](finishing.md)
- [Continuing](continuing.md)
- [Undo & Redo](undo-redo.md)

### Additional Features

- **[Chat](interactive.md)** - Conversational interface for real-time communication (Workflow dropdown)
- **[Settings](settings.md)** - Configure workflow, providers, git, and other options (Admin dropdown)
- **[Tools](browser.md)** - Browser automation for testing and authentication (More dropdown)
- **Task History** - Search, filter, and resume past tasks
- **Notes** - Add context at any point in the workflow
- **[Parallel Tasks](parallel-tasks.md)** - Run and monitor multiple tasks simultaneously

> **For developers:** Programmatic access to all features is available via the [REST API](/reference/rest-api.md).

## Remote Access

Access the Web UI from anywhere:
- [Remote Access](remote-access.md) - SSH tunnels, Cloudflare, Tailscale
- [Authentication](authentication.md) - User management and security

## Comparison with CLI

Both interfaces provide the same core functionality. The choice depends on your workflow:

| Feature           | Web UI            | CLI                |
|-------------------|-------------------|--------------------|
| Task management   | Buttons and forms | Commands and flags |
| Status monitoring | Always visible    | Run `mehr status`  |
| Configuration     | Visual forms      | Edit YAML files    |
| Automation        | Not available     | Scriptable         |
| Learning curve    | Lower             | Higher             |

For a detailed comparison, see [Web UI vs CLI](/guides/web-ui-vs-cli.md).

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

See [CLI: serve](/cli/serve.md) for all server options.

## Next Steps

- [**Getting Started**](getting-started.md) - Complete walkthrough
- [**Creating Tasks**](creating-tasks.md) - Task creation workflows
- [**Settings**](settings.md) - Configuration guide
- [**REST API**](api.md) - Programmatic access
