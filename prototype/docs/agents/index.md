# AI Agents

> **⚠️ Claude Agent**
>
> Mehrhof is designed and optimized for **Claude**. Agent implementations depend on third-party APIs that may change. Manual testing recommended before production use.

Mehrhof orchestrates AI agents to help with planning and implementation. It delegates AI operations to external CLI tools.

## How It Works

Mehrhof doesn't connect to AI APIs directly. It calls agent CLIs as subprocesses:

```
User → mehr plan → Agent CLI → AI Response → Mehrhof processes output
```

## Claude as the Primary Agent

Mehrhof is designed and optimized for **Claude**. The workflow engine, approval modes, tool integration, and output parsing are all built around Claude's capabilities.

When using other agents:
- Some workflow features may not work as expected
- Approval mode handling may differ
- Output parsing may be less reliable
- Advanced features (like tool use) may not be available

**For the best experience and full feature support, use Claude. However, you may find other agents adequate for your needs.**

## Available Agents

| Agent | Description |
|-------|-------------|
| [Claude](claude.md) | Anthropic's Claude AI (Default, Primary) |
| [Codex](codex.md) | OpenAI's Codex AI (Alternative, experimental - untested implementation) |
| [Noop](noop.md) | No-operation agent for testing/CI (auto-registered when `MEHR_TEST_MODE=1`) |

## Basic Configuration

Set the default agent in `.mehrhof/config.yaml`:

```yaml
agent:
  default: claude
  timeout: 300
  max_retries: 3
```

Or specify via CLI:

```bash
mehr start --agent claude file:task.md
```

---

## Agent Aliases

Aliases let you create custom agents with specific environment variables and CLI arguments.

### Defining Aliases

**Key distinction:**
- `claude` = uses whatever model is configured in Claude Code globally
- Model aliases (`opus`, `sonnet-fast`) = override the model
- Account aliases (`work-account`) = use different API key

```yaml
# .mehrhof/config.yaml
agents:
  opus:
    extends: claude
    description: "Claude Opus for complex tasks"
    args: ["--model", "claude-opus-4-20250514"]

  sonnet-fast:
    extends: claude
    description: "Sonnet with limited turns"
    args: ["--model", "claude-sonnet-4-20250514", "--max-turns", "3"]

  work-account:
    extends: claude
    description: "Claude with work API key"
    env:
      ANTHROPIC_API_KEY: "${WORK_API_KEY}"  # Expands from .env or system env

  work-opus:
    extends: work-account  # Aliases can extend aliases
    description: "Work account with Opus"
    args: ["--model", "claude-opus-4-20250514"]
```

### Using Aliases

```bash
export WORK_API_KEY="sk-ant-..."
mehr start --agent work-account file:task.md
mehr auto --agent opus file:task.md
```

### Listing Agents

```bash
mehr agents list
```

```
NAME          TYPE      EXTENDS       AVAILABLE  DESCRIPTION
claude        built-in  -             yes        -
opus          alias     claude        yes        Claude Opus for complex tasks
work-account  alias     claude        yes        Claude with work API key
```

---

## Per-Step Agent Configuration

Different workflow steps can use different agents - use a capable agent for planning, a faster one for implementation.

### Workflow Steps

| Step | Command | Description |
|------|---------|-------------|
| `planning` | `mehr plan` | Requirement analysis, specifications |
| `implementing` | `mehr implement` | Code generation |
| `reviewing` | `mehr review` | Code review |
| `checkpointing` | (internal) | Checkpoint summaries |

### Workspace Configuration

```yaml
# .mehrhof/config.yaml
agent:
  default: claude
  steps:
    planning:
      name: claude-opus  # Expensive model for complex analysis
    implementing:
      name: claude       # Standard for code generation
    reviewing:
      name: claude-opus  # Thorough review
```

### Task Frontmatter

Override per-step agents in task files:

```yaml
---
title: Complex OAuth2 Implementation
agent: claude
agent_steps:
  planning:
    agent: claude-opus
    env:
      MAX_TOKENS: "16384"
    args: ["--max-turns", "15"]
  implementing:
    agent: sonnet-fast
---
```

### CLI Flags

```bash
mehr plan --agent-plan claude-opus
mehr implement --agent-implement sonnet-fast
mehr review --agent-review claude-opus

mehr start --agent claude file:task.md
```

### Priority Resolution

For each step, agent is resolved (highest to lowest):

1. CLI step-specific (`--agent-plan`)
2. CLI global (`--agent`)
3. Task frontmatter step (`agent_steps.planning.agent`)
4. Task frontmatter default (`agent`)
5. Workspace config step (`agent.steps.planning.name`)
6. Workspace config default (`agent.default`)
7. Auto-detect

---

## Per-Task Agent Configuration

Specify agents directly in task frontmatter:

```yaml
---
title: My Feature
agent: work-account
agent_args: ["--max-turns", "20"]
agent_env:
  ANTHROPIC_API_KEY: "${PROJECT_API_KEY}"
---
```

### Task Agent Priority

1. CLI flag (`--agent`) - always wins
2. Task frontmatter (`agent:`)
3. Workspace default (`agent.default`)
4. Auto-detect

### Agent Persistence

Once a task starts, the agent is persisted in `work.yaml`:

```bash
mehr start file:task.md
mehr plan
mehr implement
```

---

## How Agents Work

### Planning Phase (`mehr plan`)

1. Receives task source content
2. Reads any existing notes
3. Analyzes requirements
4. Generates specification files with implementation details

### Implementation Phase (`mehr implement`)

1. Reads all specification files
2. Considers notes and context
3. Generates or modifies code files
4. Provides summary of changes

### Agent Output Format

```
<<FILE:path/to/file.go>>
package main

func main() {
    // Generated code
}
<<END FILE>>

<<SUMMARY>>
Created main.go with basic structure.
<<END SUMMARY>>
```

---

## Session Logging

All agent interactions are logged:

```
.mehrhof/work/<id>/sessions/
├── 2025-01-15T10-30-00-planning.yaml
├── 2025-01-15T11-00-00-talk.yaml
└── 2025-01-15T11-30-00-implementing.yaml
```

Each session includes timestamps, message history, token usage, and cost tracking.

---

## Troubleshooting

### Claude Not Working

```bash
claude --version
claude "Hello"    # Test works
```

### Agent Timeout

Increase timeout in config:

```yaml
agent:
  timeout: 600  # 10 minutes
```

### Rate Limited

Agent retries automatically up to `max_retries` times. Wait before retrying if issues persist.

### Verbose Output

```bash
mehr plan --verbose
mehr implement --verbose
```

## See Also

- [mehr agents](../cli/agents.md) - CLI reference
- [Configuration Guide](../configuration/index.md) - Full config options
