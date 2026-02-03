# Task File Format

Task files are Markdown documents that describe what you want Mehrhof to accomplish. They support optional YAML frontmatter for metadata.

## Basic Structure

```markdown
---
title: Feature Title
priority: high
---

# Feature Title

Description of what needs to be done.

## Requirements

- Requirement 1
- Requirement 2
```

## Frontmatter Fields

| Field         | Type   | Description                                              |
|---------------|--------|----------------------------------------------------------|
| `title`       | string | Task title (overrides `# Heading` in body)               |
| `priority`    | string | Priority level: `critical`, `high`, `normal`, `low`      |
| `labels`      | array  | Tags for categorization                                  |
| `key`         | string | External key for branch naming (e.g., `FEAT-123`)        |
| `type`        | string | Task type: `feature`, `fix`, `chore`, `refactor`, `docs` |
| `agent`       | string | Agent name or alias to use                               |
| `agent_args`  | array  | CLI arguments for the agent                              |
| `agent_env`   | object | Environment variables for the agent                      |
| `agent_steps` | object | Per-step agent configuration                             |
| `budget`      | object | Task budget configuration (cost/tokens)                  |

## Frontmatter Examples

### Minimal

```yaml
---
title: Add Health Endpoint
---
```

### With Priority and Labels

```yaml
---
title: User Authentication
priority: high
labels:
  - security
  - backend
  - auth
type: feature
---
```

### With External Key

```yaml
---
title: Fix Login Bug
key: BUG-456
type: fix
---
```

When `key` is provided, branches are named: `task/<id>--<key-slug>`

### With Agent Configuration

```yaml
---
title: Complex Feature
agent: opus
agent_args: ["--max-turns", "20"]
agent_env:
  MAX_TOKENS: "16384"
---
```

### With Per-Step Agents

```yaml
---
title: OAuth2 Implementation
agent: claude
agent_steps:
  planning:
    agent: opus
    args: ["--max-turns", "15"]
  implementing:
    agent: sonnet-fast
---
```

See [AI Agents](/agents/index.md#per-step-agent-configuration) for details on agent configuration.

### With Budget Limits

```yaml
---
title: Webhook Integration
budget:
  max_cost: 5.00
  max_tokens: 50000
  currency: USD
  on_limit: pause
  warning_at: 0.8
---
```

## Body Content

The body of a task file is standard markdown. Structure it to clearly communicate requirements:

```markdown
# Add Search Feature

Implement full-text search for the product catalog.

## Requirements

- Search by product name and description
- Support fuzzy matching
- Return top 10 results by relevance

## Constraints

- Use existing database (PostgreSQL)
- Response time < 200ms

## Example Usage

```bash
curl "/api/search?q=laptop"
```
```

### Best Practices

- **Be specific** - Include constraints, examples, and edge cases
- **Use headings** - Structure with `##` for sections
- **Include examples** - Show expected inputs/outputs
- **Mention constraints** - Performance, compatibility, dependencies

## Multiple Files (Directory Source)

For complex tasks, use a directory:

```
auth-feature/
├── requirements.md
├── api-spec.yaml
└── mockups/
    └── login-page.png
```

Start with:

```bash
mehr start ./auth-feature/
```

All files are read and provided as context to the AI.

## See Also

- [Tasks Concept](/concepts/tasks.md) - Task lifecycle and states
- [mehr start](/cli/start.md) - Starting tasks
- [Specification Format](specification-format.md) - AI-generated specification files
