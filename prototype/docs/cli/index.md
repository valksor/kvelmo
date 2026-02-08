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

| Command                     | Description                                     | Web UI                                        |
|-----------------------------|-------------------------------------------------|-----------------------------------------------|
| [start](cli/start.md)       | Register a new task from file or directory      | [Creating Tasks](/web-ui/creating-tasks.md)   |
| [status](cli/status.md)     | Show task status                                | [Dashboard](/web-ui/dashboard.md)             |
| [continue](cli/continue.md) | Show status and suggested next actions          | [Continuing](/web-ui/continuing.md)           |
| [sync](cli/sync.md)         | Sync task from provider and generate delta spec | [Syncing](/web-ui/sync.md)                    |
| [abandon](cli/abandon.md)   | Abandon task without merging                    | [Getting Started](/web-ui/getting-started.md) |

### Quick Tasks

| Command                     | Description                                          | Web UI                                  |
|-----------------------------|------------------------------------------------------|-----------------------------------------|
| [quick](cli/quick.md)       | Create a task quickly in a queue                     | [Quick Tasks](/web-ui/quick-tasks.md)   |
| [optimize](cli/optimize.md) | AI optimize a task based on notes                    | [Quick Tasks](/web-ui/quick-tasks.md)   |
| [export](cli/export.md)     | Export queue task to markdown file                   | [Quick Tasks](/web-ui/quick-tasks.md)   |
| [submit](cli/submit.md)     | Submit a task to external provider (queue or source) | [Quick Tasks](/web-ui/quick-tasks.md)   |
| [delete](cli/delete.md)     | Delete a queue task                                  | [Task History](/web-ui/task-history.md) |

### Workflow

| Command                       | Description                                        | Web UI                                    |
|-------------------------------|----------------------------------------------------|-------------------------------------------|
| [workflow](cli/workflow.md)   | Display the workflow state machine diagram         | —                                         |
| [plan](cli/plan.md)           | Create implementation specifications               | [Planning](/web-ui/planning.md)           |
| [implement](cli/implement.md) | Implement the specifications                       | [Implementing](/web-ui/implementing.md)   |
| [review](cli/review.md)       | Run code review                                    | [Reviewing](/web-ui/reviewing.md)         |
| [note](cli/note.md)           | Add notes to the task                              | [Notes](/web-ui/notes.md)                 |
| [finish](cli/finish.md)       | Complete task and merge                            | [Finishing](/web-ui/finishing.md)         |
| [auto](cli/auto.md)           | Full automation: start → plan → implement → finish | [Auto Mode](/web-ui/auto.md)              |
| [guide](cli/guide.md)         | Get context-aware next actions                     | [Dashboard](/web-ui/dashboard.md)         |

### History

| Command             | Description                   | Web UI                              |
|---------------------|-------------------------------|-------------------------------------|
| [undo](cli/undo.md) | Revert to previous checkpoint | [Undo & Redo](/web-ui/undo-redo.md) |
| [redo](cli/redo.md) | Restore undone checkpoint     | [Undo & Redo](/web-ui/undo-redo.md) |

### Project Planning

| Command                                   | Description                                   | Web UI                                            |
|-------------------------------------------|-----------------------------------------------|---------------------------------------------------|
| [project plan](cli/project.md#plan)       | Create task breakdown from source             | [Project Planning](/web-ui/project-planning.md)   |
| [project sync](cli/project-sync.md)       | Sync project/epic from external provider      | [Project Planning](/web-ui/project-planning.md)   |
| [project tasks](cli/project.md#tasks)     | List tasks in a project queue                 | [Project Planning](/web-ui/project-planning.md)   |
| [project edit](cli/project.md#edit)       | Edit task properties (deps, priority, status) | [Project Planning](/web-ui/project-planning.md)   |
| [project reorder](cli/project.md#reorder) | Reorder tasks in queue                        | [Project Planning](/web-ui/project-planning.md)   |
| [project submit](cli/project.md#submit)   | Submit tasks to provider                      | [Project Planning](/web-ui/project-planning.md)   |
| [project start](cli/project.md#start)     | Start implementing tasks from queue           | [Project Planning](/web-ui/project-planning.md)   |

### Web UI

| Command                     | Description                        | Web UI                                        |
|-----------------------------|------------------------------------|-----------------------------------------------|
| [serve](cli/serve.md)       | Start web UI server                | —                                             |
| [register](cli/register.md) | Register project for global access | [Settings](/web-ui/settings.md)               |
| unregister                  | Remove project from registry       | [Settings](/web-ui/settings.md)               |

### Utility

| Command                       | Description                                 | Web UI                                          |
|-------------------------------|---------------------------------------------|-------------------------------------------------|
| [init](cli/init.md)           | Initialize task workspace                   | —                                               |
| [find](cli/find.md)           | AI-powered code search with focused results | [Find](/web-ui/find.md)                         |
| [agents](cli/agents.md)       | List available AI agents                    | [Settings](/web-ui/settings.md)                 |
| [config](cli/config.md)       | Validate configuration files                | [Settings](/web-ui/settings.md)                 |
| [plugins](cli/plugins.md)     | Manage extension plugins                    | [Settings](/web-ui/settings.md)                 |
| [templates](cli/templates.md) | Manage task templates                       | [Templates](/web-ui/templates.md)               |
| [cost](cli/cost.md)           | Show token usage and costs                  | [Dashboard](/web-ui/dashboard.md)               |
| [budget](cli/budget.md)       | Manage cost and token budgets               | [Settings](/web-ui/settings.md)                 |
| [license](cli/license.md)     | Display license information                 | [Settings](/web-ui/settings.md)                 |
| [label](cli/label.md)         | Manage task labels                          | [Labels](/web-ui/labels.md)                     |
| [list](cli/list.md)           | List all tasks in workspace                 | [Task History](/web-ui/task-history.md)         |
| [version](cli/version.md)     | Print version information                   | —                                               |
| [mcp](cli/mcp.md)             | Start MCP server for AI agent integration   | —                                               |

### Provider Authentication

| Command                        | Description                | Web UI                          |
|--------------------------------|----------------------------|---------------------------------|
| [github login](cli/login.md)   | Authenticate with GitHub   | [Settings](/web-ui/settings.md) |
| [gitlab login](cli/login.md)   | Authenticate with GitLab   | [Settings](/web-ui/settings.md) |
| [notion login](cli/login.md)   | Authenticate with Notion   | [Settings](/web-ui/settings.md) |
| [jira login](cli/login.md)     | Authenticate with Jira     | [Settings](/web-ui/settings.md) |
| [linear login](cli/login.md)   | Authenticate with Linear   | [Settings](/web-ui/settings.md) |
| [wrike login](cli/login.md)    | Authenticate with Wrike    | [Settings](/web-ui/settings.md) |
| [youtrack login](cli/login.md) | Authenticate with YouTrack | [Settings](/web-ui/settings.md) |

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

---

## Power User Tips

### Rapid Command Entry

Combine prefix matching with colon notation for maximum speed:

| Full Command         | Shorthand   | Description                |
|---------------------|-------------|----------------------------|
| `mehr config validate` | `mehr c:v`  | Validate configuration    |
| `mehr agents list`    | `mehr a:l`  | List available agents     |
| `mehr plugins list`   | `mehr p:l`  | List installed plugins    |
| `mehr project plan`   | `mehr pr:p` | Plan project tasks        |
| `mehr project start`  | `mehr pr:s` | Start project queue       |
| `mehr budget status`  | `mehr b:s`  | Check budget status       |
| `mehr budget resume`  | `mehr b:r`  | Resume after budget pause |

### Skip Confirmation Prompts

Most destructive commands support `--yes` or `-y` to skip confirmation:

```bash
mehr abandon -y          # Skip confirmation
mehr finish --yes        # Skip confirmation
mehr undo -y             # Skip confirmation
mehr reset --yes         # Skip confirmation
```

### Quick Context Check

Three commands answer "Where am I?" with different levels of detail:

| Command    | Output Level | Best For                              |
|------------|--------------|---------------------------------------|
| `mehr gu`  | Minimal      | Quick "what's next?" check            |
| `mehr c`   | Summary      | Resume after break, optional `--auto` |
| `mehr st`  | Full         | Deep inspection, debugging            |

See [Choosing the Right Context Command](#choosing-the-right-context-command) for detailed comparison.

### Worktree Workflow

When using worktrees for parallel tasks, all commands auto-detect context:

```bash
cd ../project-worktrees/a1b2c3d4
mehr guide     # Auto-detects task from worktree path
mehr status    # Shows worktree-specific status
mehr continue  # Resumes the right task
```

### JSON Output for Scripting

Commands with `--json` output enable scripting:

```bash
# Get task ID programmatically
TASK_ID=$(mehr status --json | jq -r '.task_id')

# Check state in scripts
STATE=$(mehr status --json | jq -r '.state')
if [ "$STATE" = "implementing" ]; then
  mehr finish
fi

# Export cost data
mehr cost --json > cost-report.json
```

---

## Choosing the Right Context Command

These three commands help you understand your current context, each optimized for different scenarios:

| Aspect            | `guide`                     | `continue`                      | `status`                      |
|-------------------|-----------------------------|---------------------------------|-------------------------------|
| **Purpose**       | "What command next?"        | "Resume work"                   | "Show everything"             |
| **Output**        | Minimal (state + 1-3 tips)  | Summary (metadata + tips)       | Full (specs, checkpoints, tokens) |
| **Best for**      | Quick orientation           | Returning after a break         | Debugging, inspection         |
| **Special**       | Answers pending questions   | `--auto` runs next step         | `--json` for scripting        |
| **Speed**         | Fastest                     | Fast                            | Comprehensive                 |

### Decision Tree

```
Need to know what to do?
  └─ Just the next command? → mehr guide
  └─ See status + decide?   → mehr continue
  └─ Full details needed?   → mehr status
  └─ Want AI to proceed?    → mehr continue --auto
```
