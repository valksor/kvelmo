# Configuration Guide

Mehrhof uses a simple configuration system with sensible defaults.

## How Claude Integration Works

Mehrhof calls Claude CLI as a subprocess:

- **Claude's settings** (API keys, model) are handled by Claude CLI itself
- **Mehrhof's settings** control workflow behavior and are passed to agents
- Environment variables in workspace config are passed to Claude when it runs

## Configuration Methods

| Method           | Use Case              | Location                            |
|------------------|-----------------------|-------------------------------------|
| CLI flags        | Per-command overrides | `mehr --verbose plan`               |
| Workspace config | Project settings      | `.mehrhof/config.yaml` (in project) |
| Environment file | Secrets (gitignored)  | `.mehrhof/.env` (in project)        |
| User settings    | Personal preferences  | `~/.valksor/mehrhof/settings.json`  |

## File Locations

**In project** (`.mehrhof/`):
- `config.yaml` - Workspace configuration (safe to commit)
- `.env` - Project-specific secrets (gitignored)

**In home directory** (`~/.valksor/mehrhof/workspaces/<project-id>/`):
- `work/` - Task data (specifications, sessions, notes, etc.)
- `.active_task` - Current task state

## Workspace Configuration

**Location:** `.mehrhof/config.yaml` (in project)

```yaml
git:
  auto_commit: true
  commit_prefix: "[{key}]"
  branch_pattern: "{type}/{key}--{slug}"

agent:
  default: claude
  timeout: 300

sandbox:
  enabled: false
  network: true

providers:
  default: file

workflow:
  auto_init: true
  session_retention_days: 30

env:
  CLAUDE_ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
```

## Configuration Sections

| Section         | Description                        | Details                                 |
|-----------------|------------------------------------|-----------------------------------------|
| `git`           | Version control integration        | [Git Configuration](git.md)             |
| `agent`         | AI agent behavior and aliases      | [Agents Guide](../agents/index.md)      |
| `sandbox`       | Agent execution sandboxing         | [Sandbox Guide](../advanced/sandbox.md) |
| `browser`       | Browser automation                 | [Browser Configuration](browser.md)     |
| `storage`       | Task data and caches               | [Storage Configuration](storage.md)     |
| `specification` | Spec storage location and naming   | [Storage Configuration](storage.md)     |
| `review`        | Review storage location and naming | [Storage Configuration](storage.md)     |
| `context`       | Hierarchical task context          | [Context Configuration](context.md)     |
| `budget`        | Cost and token budgets             | (see below)                             |
| `quality`       | Code quality and linters           | (see below)                             |

### providers

```yaml
providers:
  default: file  # Allow "mehr start task.md" without prefix
```

### workflow

```yaml
workflow:
  auto_init: true                  # Auto-initialize workspace
  session_retention_days: 30       # Keep sessions for N days
  delete_work_on_finish: false     # Delete work dirs after finish
  delete_work_on_abandon: true     # Delete work dirs on abandon
```

### budget

```yaml
budget:
  per_task:
    max_tokens: 100000
    max_cost: 10.00
    currency: USD
    on_limit: warn   # warn | pause | stop
    warning_at: 0.8

  monthly:
    max_cost: 100.00
    currency: USD
    warning_at: 0.8
```

### quality

Code quality and linter configuration for the review phase.

**Default behavior (safer):** Built-in linters are NOT auto-enabled. You must explicitly configure linters.

```yaml
quality:
  enabled: true                     # Master switch for quality checks
  use_defaults: false               # Don't auto-enable built-in linters (default)
  linters:
    golangci-lint:
      enabled: true                 # Explicitly enable Go linter
    phpstan:                         # Custom linter
      enabled: true
      command: ["vendor/bin/phpstan", "analyse", "--error-format=json"]
      extensions: [".php"]
```

**Opt-in to auto-detection (old behavior):**

```yaml
quality:
  enabled: true
  use_defaults: true               # Auto-enable built-in linters based on project files
```

**Built-in linters:**

| Linter          | Language              | Auto-Detection                                             |
|-----------------|-----------------------|------------------------------------------------------------|
| `golangci-lint` | Go                    | `go.mod` exists                                            |
| `eslint`        | JavaScript/TypeScript | `package.json` exists                                      |
| `ruff`          | Python                | `pyproject.toml`, `setup.py`, or `requirements.txt` exists |
| `php-cs-fixer`  | PHP                   | `composer.json` exists                                     |

> **Note:** Built-in linters are only auto-detected when `use_defaults: true`. With `use_defaults: false` (default), linters must be explicitly enabled.

**Custom linters:**

Define custom linters using the `command` field:

```yaml
quality:
  enabled: true
  use_defaults: false               # Safer: only run explicitly configured linters
  linters:
    phpstan:
      enabled: true
      command: ["vendor/bin/phpstan", "analyse", "--error-format=json"]
      extensions: [".php"]

    psalm:
      enabled: true
      command: ["vendor/bin/psalm", "--output-format=json"]
      extensions: [".php"]

    mypy:
      enabled: true
      command: ["python", "-m", "mypy", "--json"]
      extensions: [".py"]
```

**Configuration options:**

| Option                      | Type     | Default | Description                                         |
|-----------------------------|----------|---------|-----------------------------------------------------|
| `enabled`                   | bool     | `true`  | Master switch for all quality checks                |
| `use_defaults`              | bool     | `false` | Auto-enable built-in linters based on project files |
| `linters.<name>.enabled`    | bool     | -       | Enable/disable specific linter                      |
| `linters.<name>.command`    | []string | -       | Command to run (creates custom linter)              |
| `linters.<name>.extensions` | []string | -       | File extensions to lint (optional)                  |
| `linters.<name>.args`       | []string | -       | Additional arguments (optional)                     |

### env

Environment variables filtered by agent name prefix:

```yaml
env:
  CLAUDE_ANTHROPIC_API_KEY: sk-ant-xxx  # → ANTHROPIC_API_KEY to claude
  CLAUDE_MAX_TOKENS: "4096"              # → MAX_TOKENS to claude
```

Variables are filtered by agent name prefix, stripped when passed.

## Environment File (.env)

Store secrets locally without committing to git.

**Location:** `.mehrhof/.env` (in project, gitignored)

### Provider Authentication

Provider tokens should be stored in `.mehrhof/.env` and referenced in `config.yaml` using `${VAR}` syntax:

```bash
# .mehrhof/.env
ANTHROPIC_API_KEY=sk-ant-...
WORK_API_KEY=sk-ant-...  # For work-account alias

GITHUB_TOKEN=ghp_...
GITLAB_TOKEN=glpat-...
NOTION_TOKEN=your-notion-token
JIRA_TOKEN=your-jira-token
LINEAR_API_KEY=your-linear-key
```

**Reference in config.yaml:**

```yaml
# .mehrhof/config.yaml (in project)
github:
  token: ${GITHUB_TOKEN}
  owner: myorg
  repo: myrepo

gitlab:
  token: ${GITLAB_TOKEN}
  host: https://gitlab.com
```

**How it works:**

1. `.mehrhof/.env` stores actual token values (gitignored)
2. `.mehrhof/config.yaml` references tokens using `${VAR}` syntax (safe to commit)
3. At startup, `${VAR}` references are replaced with values from `.env` or system environment
4. Provider receives the expanded token value

### Setting Up Provider Tokens

**Option 1: Using login commands (recommended)**

```bash
mehr github login    # Prompts for token, writes to .env and config.yaml
mehr gitlab login    # Prompts for token, writes to .env and config.yaml
```

**Option 2: Manual setup**

1. Add token to `.mehrhof/.env`:
   ```bash
   GITHUB_TOKEN=ghp_your_token_here
   ```

2. Add reference to `config.yaml`:
   ```yaml
   github:
     token: ${GITHUB_TOKEN}
   ```

### Usage with Agent Aliases

```yaml
# .mehrhof/config.yaml
agents:
  work-account:
    extends: claude
    env:
      ANTHROPIC_API_KEY: "${WORK_API_KEY}"  # Reads from .env
```

**Security:**
- Created with `0600` permissions (user read/write only)
- Automatically added to `.gitignore` by `mehr init`
- Never commit `.env` to version control

## User Settings

Personal preferences stored automatically.

**Location:** `~/.valksor/mehrhof/settings.json`

```json
{
  "preferred_agent": "claude",
  "target_branch": "main",
  "last_provider": "file",
  "recent_tasks": ["abc12345", "def67890"]
}
```

## CLI Flags

Global flags available on all commands:

| Flag            | Description            |
|-----------------|------------------------|
| `-v, --verbose` | Enable verbose output  |
| `--no-color`    | Disable colored output |

The `NO_COLOR` environment variable is also respected.

## Quick Reference

| What                  | How                                           |
|-----------------------|-----------------------------------------------|
| Verbose output        | `mehr --verbose <command>`                    |
| Disable color         | `mehr --no-color <command>` or `NO_COLOR=1`   |
| Set default agent     | `.mehrhof/config.yaml` → `agent.default`      |
| Custom branch pattern | `.mehrhof/config.yaml` → `git.branch_pattern` |
| Set default provider  | `.mehrhof/config.yaml` → `providers.default`  |

## Best Practices

### What to Commit

```
.mehrhof/config.yaml    # Workspace config (no secrets!)
```

### What to Gitignore

```
.mehrhof/.env           # Secrets (work/ and .active_task are in home dir)
```

### Validate Configuration

```bash
mehr config validate
```

## See Also

- [Git Configuration](git.md) - Branch patterns, commit settings
- [Browser Configuration](browser.md) - Browser automation
- [Storage Configuration](storage.md) - Task data locations
- [Agents Guide](../agents/index.md) - Agent configuration and aliases
- [CLI Reference](../cli/index.md) - All commands
