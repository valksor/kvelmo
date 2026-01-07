# mcp

Start an MCP (Model Context Protocol) server for AI agent integration.

## Usage

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

| Tool | Description |
|------|-------------|
| `mehr_status` | Show workspace state |
| `mehr_list` | List all tasks |
| `mehr_guide` | Get suggested next actions |
| `mehr_browser_*` | Browser automation (goto, screenshot, click, etc.) |
| `mehr_version` | Print version information |
| `mehr_config` | Validate and show configuration |
| `mehr_providers` | List available providers |
| `mehr_templates` | List task templates |
| `mehr_agents` | List available AI agents |
| `mehr_cost` | Show token usage and costs |

### Workspace Data

Direct access to workspace state:

| Tool | Description |
|------|-------------|
| `workspace_get_active_task` | Get current active task info |
| `workspace_list_tasks` | List all tasks with metadata |
| `workspace_get_specs` | Get specifications for a task |
| `workspace_get_sessions` | Get session history |
| `workspace_get_notes` | Get task notes |

### Registry Queries

Agent and provider registry information:

| Tool | Description |
|------|-------------|
| `agents_list` | List all registered agents |
| `agents_get_default` | Get default agent |
| `providers_list` | List all registered providers |
| `providers_resolve` | Resolve provider from reference |

## Configuration

MCP server settings in `.mehrhof/config.yaml`:

```yaml
mcp:
  enabled: true
  tools:
    - mehr_status
    - mehr_browser_goto
    - workspace_get_active_task
```

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

- **Command tools**: `mehr_<command>` (e.g., `mehr_status`, `mehr_browser_goto`)
- **Data tools**: `<category>_<action>` (e.g., `workspace_get_active_task`, `agents_list`)

## Workspace Detection

MCP tools automatically detect the workspace from the current working directory:

1. Check if `.git/worktrees` exists → use worktree root
2. Check if `.git` exists → use repo root
3. Otherwise → use current directory

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
    "name": "mehr_status",
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

## See Also

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Browser Automation](../guides/browser-automation.md)
- [Configuration](../configuration/index.md)
