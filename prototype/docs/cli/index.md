# CLI Overview

Mehrhof provides a command-line interface for structured creation and review.

## Basic Usage

```bash
mehr <command> [flags]
```

## Global Flags

These flags work with any command:

| Flag         | Short | Description                   |
|--------------|-------|-------------------------------|
| `--verbose`  | `-v`  | Enable verbose output         |
| `--quiet`    | `-q`  | Suppress non-essential output |
| `--no-color` |       | Disable colored output        |

## Commands

### Task Management

| Command                     | Description                                     |
|-----------------------------|-------------------------------------------------|
| [start](cli/start.md)       | Register a new task from file or directory      |
| [status](cli/status.md)     | Show task status                                |
| [continue](cli/continue.md) | Show status and suggested next actions          |
| [sync](cli/sync.md)         | Sync task from provider and generate delta spec |
| [abandon](cli/abandon.md)   | Abandon task without merging                    |

### Quick Tasks

| Command                     | Description                                          |
|-----------------------------|------------------------------------------------------|
| [quick](cli/quick.md)       | Create a task quickly in a queue                     |
| [optimize](cli/optimize.md) | AI optimize a task based on notes                    |
| [export](cli/export.md)     | Export queue task to markdown file                   |
| [submit](cli/submit.md)     | Submit a task to external provider (queue or source) |
| [delete](cli/delete.md)     | Delete a queue task                                  |

### Workflow

| Command                       | Description                                        |
|-------------------------------|----------------------------------------------------|
| [workflow](cli/workflow.md)   | Display the workflow state machine diagram         |
| [plan](cli/plan.md)           | Create implementation specifications               |
| [implement](cli/implement.md) | Implement the specifications                       |
| [review](cli/review.md)       | Run code review                                    |
| [note](cli/note.md)           | Add notes to the task                              |
| [finish](cli/finish.md)       | Complete task and merge                            |
| [auto](cli/auto.md)           | Full automation: start → plan → implement → finish |
| [guide](cli/guide.md)         | Get context-aware next actions                     |

### History

| Command             | Description                   |
|---------------------|-------------------------------|
| [undo](cli/undo.md) | Revert to previous checkpoint |
| [redo](cli/redo.md) | Restore undone checkpoint     |

### Project Planning

| Command                                   | Description                                   |
|-------------------------------------------|-----------------------------------------------|
| [project plan](cli/project.md#plan)       | Create task breakdown from source             |
| [project sync](cli/project-sync.md)       | Sync project/epic from external provider      |
| [project tasks](cli/project.md#tasks)     | List tasks in a project queue                 |
| [project edit](cli/project.md#edit)       | Edit task properties (deps, priority, status) |
| [project reorder](cli/project.md#reorder) | Reorder tasks in queue                        |
| [project submit](cli/project.md#submit)   | Submit tasks to provider                      |
| [project start](cli/project.md#start)     | Start implementing tasks from queue           |

### Web UI

| Command                     | Description                        |
|-----------------------------|------------------------------------|
| [serve](cli/serve.md)       | Start web UI server                |
| [register](cli/register.md) | Register project for remote access |
| unregister                  | Remove project from registry       |

### Utility

| Command                       | Description                                 |
|-------------------------------|---------------------------------------------|
| [init](cli/init.md)           | Initialize task workspace                   |
| [find](cli/find.md)           | AI-powered code search with focused results |
| [agents](cli/agents.md)       | List available AI agents                    |
| [config](cli/config.md)       | Validate configuration files                |
| [plugins](cli/plugins.md)     | Manage extension plugins                    |
| [templates](cli/templates.md) | Manage task templates                       |
| [cost](cli/cost.md)           | Show token usage and costs                  |
| [budget](cli/budget.md)       | Manage cost and token budgets               |
| [license](cli/license.md)     | Display license information                 |
| [label](cli/label.md)         | Manage task labels                          |
| [list](cli/list.md)           | List all tasks in workspace                 |
| [version](cli/version.md)     | Print version information                   |
| [mcp](cli/mcp.md)             | Start MCP server for AI agent integration   |

### Provider Authentication

| Command                        | Description                |
|--------------------------------|----------------------------|
| [github login](cli/login.md)   | Authenticate with GitHub   |
| [gitlab login](cli/login.md)   | Authenticate with GitLab   |
| [notion login](cli/login.md)   | Authenticate with Notion   |
| [jira login](cli/login.md)     | Authenticate with Jira     |
| [linear login](cli/login.md)   | Authenticate with Linear   |
| [wrike login](cli/login.md)    | Authenticate with Wrike    |
| [youtrack login](cli/login.md) | Authenticate with YouTrack |

## Command Shortcuts

Mehrhof supports Symfony-style command abbreviation for faster typing.

### Prefix Matching

Type any unique prefix of a command name:

```bash
mehr gu          # → mehr guide
mehr imp         # → mehr implement
mehr fin         # → mehr finish
mehr pl          # → ambiguous: plan, plugins (shows selection)
```

If the prefix matches multiple commands, you'll be prompted to select one interactively. Use arrow keys to navigate, Enter to select, or choose `[Cancel]` to abort.

### Colon Notation for Subcommands

Use colons to abbreviate nested commands:

```bash
mehr config:v    # → mehr config validate
mehr config:i    # → mehr config init
mehr a:l         # → mehr agents list
mehr p:l         # → mehr plugins list
```

### Non-Interactive Mode

In scripts or pipes, ambiguous commands show an error listing the matches instead of prompting:

```bash
echo "" | mehr s
# Command "s" is ambiguous. Did you mean one of these?
#   start - Start a new task...
#   status - Show full task details
#   sync - Sync task from provider...
```

## Command Help

Get help for any command:

```bash
mehr --help
mehr <command> --help
```

### Contextual Help

The help output adapts to your current workspace state. Commands are grouped into:

- **Available in Context** - Commands that can be run in the current context
- **Other Commands** - Commands that require additional context (with reasons why)

Example output without an active task:

```
Available in Context:
  start       Start a new task
  auto        Full automation
  list        List all tasks
  ...

Other Commands:
  status      Show task state (needs active task)
  implement   Implement specifications (needs specifications)
  undo        Revert checkpoint (needs git task)
  ...
```

## Workflow Summary

```bash
mehr start file:task.md
mehr plan
mehr implement
mehr finish
```

## Output Formats

By default, output is human-readable text. Some commands support JSON output via the `--json` flag for programmatic access:

- `mehr cost --json` - Token usage and cost data
- `mehr list --json` - Task listing
- `mehr status --json` - Detailed task status

## Exit Codes

| Code | Meaning           |
|------|-------------------|
| 0    | Success           |
| 1    | General error     |
| 2    | Invalid arguments |

## Configuration

See [Configuration Guide](/configuration/index.md) for workspace settings and CLI flags.

## Examples

### Quick Task (Auto Mode)

```bash
mehr auto file:feature.md
```

### Quick Task (Manual)

```bash
mehr start file:feature.md && mehr plan && mehr implement && mehr finish
```

### Interactive Development

```bash
mehr start file:feature.md
mehr plan
mehr note "Add error handling for edge cases"
mehr implement
mehr finish
```

### Exploring Without a Task

```bash
mehr plan --standalone "How should I implement caching?"
```
