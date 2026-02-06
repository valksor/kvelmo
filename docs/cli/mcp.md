# mehr mcp

Start an MCP (Model Context Protocol) server for AI agent integration.

> **Note:** This is a CLI-only feature. No web UI equivalent exists for the MCP server, as it is designed for stdio-based communication with AI agents.

## Synopsis

```bash
mehr mcp
```

## Description

The `mcp` command starts a stdio-based MCP server that exposes Mehrhof commands and workspace data as MCP tools. This enables external AI agents (like Claude Code) to interact with Mehrhof programmatically.

**Transport**: stdio (JSON-RPC 2.0 messages over stdin/stdout)

**Protocol**: [Model Context Protocol](https://modelcontextprotocol.io/)

## How It Works

1. Start the MCP server: `mehr mcp`
2. The server listens for JSON-RPC requests on stdin and writes responses to stdout
3. AI agents can discover available tools via `tools/list`
4. Agents call tools via `tools/call` with arguments

## Exposed Tools

### CLI Commands

Safe Mehrhof commands exposed as MCP tools:

| Tool               | Description                                        |
|--------------------|----------------------------------------------------|
| `status`           | Show workspace state                               |
| `list`             | List all tasks                                     |
| `guide`            | Get suggested next actions                         |
| `browser_*`        | Browser automation (goto, screenshot, click, etc.) |
| `version`          | Print version information                          |
| `config`           | Validate and show configuration                    |
| `providers`        | List available providers                           |
| `templates`        | List task templates                                |
| `agents`           | List available AI agents                           |
| `cost`             | Show token usage and costs                         |
| `scan`             | Run security scans on codebase                     |
| `find`             | Search tasks by pattern                            |

### Workspace Data

Direct access to workspace state:

| Tool                        | Description                   | Parameters                                                  |
|-----------------------------|-------------------------------|-------------------------------------------------------------|
| `workspace_get_active_task` | Get current active task info  | None                                                        |
| `workspace_list_tasks`      | List all tasks with metadata  | None                                                        |
| `workspace_get_specs`       | Get specifications for a task | `task_id` (string), `summary_only` (boolean, default false) |
| `workspace_get_sessions`    | Get session history           | `task_id` (string)                                          |
| `workspace_get_notes`       | Get task notes                | `task_id` (string)                                          |

When `task_id` is omitted, workspace tools fall back to the active task.

### Registry Queries

Agent and provider registry information:

| Tool                 | Description                     | Parameters             |
|----------------------|---------------------------------|------------------------|
| `agents_list`        | List all registered agents      | None                   |
| `agents_get_default` | Get default agent               | None                   |
| `providers_list`     | List all registered providers   | None                   |
| `providers_resolve`  | Resolve provider from reference | `reference` (string)   |

## Configuration

MCP server settings in `.mehrhof/config.yaml`:

```yaml
mcp:
  enabled: true
  tools:                          # Allowlist of tools to expose (omit for all safe tools)
    - status
    - browser_goto
    - browser_screenshot
    - workspace_get_active_task
  rate_limit:
    rate: 10                      # Requests per second
    burst: 20                     # Burst size
```

When `tools` is specified, only the listed tools are exposed to the AI agent. If omitted, all safe tools are available.

## Security Model

The MCP server exposes **safe operations** for AI agents:

- **Read-only commands**: workspace status, task lists, provider/agent information
- **Browser automation**: navigate, screenshot, click, type, evaluate (can modify browser state but not the codebase)
- **Not exposed**: Workflow-modifying commands (`start`, `plan`, `implement`, `review`, `finish`, `continue`, `undo`, `redo`, `abandon`)
- **Not exposed**: Credential operations (`provider login`)
- **Reason**: Prevent agents from disrupting workflows or accessing credentials while enabling testing and read operations

## Example: Integration with Claude Code

Configure Claude Code to use Mehrhof as an MCP server:

```json
{
  "mcpServers": {
    "mehrhof": {
      "command": "mehr",
      "args": ["mcp"],
      "cwd": "/path/to/project"
    }
  }
}
```

Claude Code can then:
- Query workspace state before making changes
- Use browser automation for testing
- List available commands and providers
- Access task notes and specifications

## Tool Naming

- **Command tools**: Named by subcommand path with underscores (e.g., `status`, `browser_goto`, `browser_screenshot`)
- **Data tools**: `<category>_<action>` (e.g., `workspace_get_active_task`, `agents_list`)

The root command name (`mehr`) is stripped — tool names start from the first subcommand.

## Workspace Detection

MCP tools automatically detect the workspace from the current working directory:

1. Check if `.git/worktrees` exists → use worktree root
2. Check if `.git` exists → use repo root
3. Otherwise, → use current directory

No workspace parameter is needed in tool arguments.

## Protocol Details

### Supported Methods

- `initialize` - Initialize MCP session
- `tools/list` - List available tools
- `tools/call` - Execute a tool
- `shutdown` - Graceful shutdown

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "status",
    "arguments": {
      "json": true
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Task: FEATURE-1\nState: implementing\n..."
      }
    ]
  }
}
```

## Troubleshooting

| Problem                     | Cause                                                     | Solution                                                                          |
|-----------------------------|-----------------------------------------------------------|-----------------------------------------------------------------------------------|
| `Server not initialized`    | Called `tools/list` or `tools/call` before `initialize`   | Send an `initialize` request first                                                |
| `Rate limit exceeded`       | Too many tool calls in a short period                     | Increase `rate_limit.rate` and `rate_limit.burst` in config                       |
| `Protocol version mismatch` | Client sends wrong protocol version                       | Use protocol version `2025-06-18`                                                 |
| `Tool not found`            | Tool name doesn't match any registered tool               | Check tool names via `tools/list` — command tools don't include the `mehr` prefix |
| Browser tools fail          | Chrome/Chromium not running or CDP not available          | Start a browser with `mehr browser` first, then use browser tools                 |
| Timeout errors              | Command execution exceeds 30s (Cobra) or 5min (all tools) | Simplify the operation or increase context deadline                               |

## See Also

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Browser Automation](browser.md)
- [Configuration](/configuration/index.md)
