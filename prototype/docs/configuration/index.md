# Configuration Guide

Mehrhof uses a simple configuration system with sensible defaults.

## How Claude Integration Works

Mehrhof calls Claude CLI as a subprocess:

- **Claude's settings** (API keys, model) are handled by Claude CLI itself
- **Mehrhof's settings** control workflow behavior and are passed to agents
- Environment variables in workspace config are passed to Claude when it runs

## Configuration Methods

| Method | Use Case | Location |
|--------|----------|----------|
| CLI flags | Per-command overrides | `mehr --verbose plan` |
| Workspace config | Project settings | `.mehrhof/config.yaml` (in project) |
| Environment file | Secrets (gitignored) | `.mehrhof/.env` (in project) |
| User settings | Personal preferences | `~/.valksor/mehrhof/settings.json` |

## File Locations

**In project** (`.mehrhof/`):
- `config.yaml` - Workspace configuration (safe to commit)
- `.env` - Project-specific secrets (gitignored)

**In home directory** (`~/.valksor/mehrhof/workspaces/<project-id>/`):
- `work/` - Task data (specifications, sessions, notes, etc.)
- `.active_task` - Current task state

| File | Purpose |
|------|---------|
| `.mehrhof/config.yaml` | Workspace configuration (in project) |
| `.mehrhof/.env` | Secrets (in project, gitignored) |
| `~/.valksor/mehrhof/workspaces/<project-id>/work/` | Task data |
| `~/.valksor/mehrhof/workspaces/<project-id>/.active_task` | Current task state |
| `~/.valksor/mehrhof/settings.json` | User preferences |
| `~/.valksor/mehrhof/plugins/` | Global plugins |

## Workspace Configuration

**Location:** `.mehrhof/config.yaml` (in project)

```yaml
git:
  auto_commit: true
  commit_prefix: "[{key}]"
  branch_pattern: "{type}/{key}--{slug}"
  sign_commits: false

agent:
  default: claude
  timeout: 300
  max_retries: 3

providers:
  default: file  # Allow bare references like "task.md"

workflow:
  auto_init: true
  session_retention_days: 30

env:
  CLAUDE_ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
```

### git

Controls version control integration:

| Setting | Default | Description |
|---------|---------|-------------|
| `auto_commit` | `true` | Auto-commit after operations |
| `commit_prefix` | `[{key}]` | Commit message prefix template |
| `branch_pattern` | `{type}/{key}--{slug}` | Branch naming template |
| `sign_commits` | `false` | GPG-sign commits |
| `stash_on_start` | `false` | Auto-stash changes before creating task branch |
| `auto_pop_stash` | `true` | Auto-pop stash after branch creation |

**Stash behavior:**

When `stash_on_start` is enabled, Mehrhof automatically stashes uncommitted changes (including untracked files) before creating a new task branch. The `auto_pop_stash` setting controls whether the stash is automatically restored:

- `auto_pop_stash: true` (default) - Stash is automatically restored after branch creation
- `auto_pop_stash: false` - Stash is preserved for manual restoration (use `git stash pop`)

This is useful when you have work-in-progress changes that aren't ready to commit.

```yaml
git:
  stash_on_start: true  # Auto-stash changes before creating branch
  auto_pop_stash: true  # Auto-pop stash after branch (default: true)
  # Set to false to preserve stash for manual restoration
```

See [`mehr start --stash`](../cli/start.md#start-with-stash-uncommitted-changes) for CLI usage.

**Template variables:**

| Variable | Description | Example |
|----------|-------------|---------|
| `{key}` | External key (from filename/frontmatter) | `FEATURE-123` |
| `{task_id}` | Internal task ID | `a1b2c3d4` |
| `{type}` | Task type from filename prefix | `feature`, `fix` |
| `{slug}` | URL-safe slugified title | `add-user-auth` |

### agent

Controls AI agent behavior:

| Setting | Default | Description |
|---------|---------|-------------|
| `default` | `claude` | Default agent |
| `timeout` | `300` | Timeout in seconds |
| `max_retries` | `3` | Retry attempts |
| `instructions` | (empty) | Global instructions for all steps |
| `steps` | (empty) | Per-step agent configuration |

**Per-step configuration:**

```yaml
agent:
  default: claude
  steps:
    planning:
      name: claude-opus
      env:
        MAX_TOKENS: "16384"
      args: ["--max-turns", "15"]
    implementing:
      name: claude
    reviewing:
      name: claude-opus
```

| Step | Description |
|------|-------------|
| `planning` | Agent for `mehr plan` |
| `implementing` | Agent for `mehr implement` |
| `reviewing` | Agent for `mehr review` |
| `checkpointing` | Agent for checkpoint summaries |

**Default instructions:**

Provide custom instructions to guide AI agents during each workflow step:

```yaml
agent:
  default: claude

  # Global instructions (apply to ALL steps)
  instructions: |
    Follow existing code patterns.
    Include comprehensive error handling.
    Write clean, maintainable code.

  # Per-step instructions (combined with global)
  steps:
    planning:
      name: claude
      instructions: |
        Focus on architectural decisions.
        Consider scalability implications.
    implementing:
      name: claude-sonnet
      instructions: |
        Follow TDD - write tests alongside implementation.
        Ensure backward compatibility.
    reviewing:
      name: claude
      instructions: |
        Be thorough but constructive.
        Focus on security and performance issues.
```

**Combination behavior:**
- Global `instructions` apply to all steps
- Step-specific `instructions` are **appended** to global (not replaced)
- Empty or whitespace-only instructions are ignored
- Instructions appear as "Custom Instructions" section in agent prompts

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

### browser

Browser automation configuration for web-based testing, scraping, and authentication flows:

```yaml
browser:
  enabled: true                    # Enable browser automation
  headless: false                  # Show browser window (false = visible, true = background)
  port: 0                          # 0 = random isolated browser, 9222 = existing Chrome
  timeout: 30                      # Operation timeout in seconds
  screenshot_dir: ".mehrhof/screenshots"
  cookie_profile: "default"        # Cookie profile name (default: "default")
  cookie_auto_load: true           # Auto-load cookies on connect (default: true)
  cookie_auto_save: true           # Auto-save cookies on disconnect (default: true)
  cookie_dir: ""                   # Custom cookie directory (default: ~/.valksor/mehrhof/)
```

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable browser automation |
| `headless` | `false` | Run browser in headless mode |
| `port` | `0` | CDP port (0 = random isolated, 9222 = existing Chrome) |
| `timeout` | `30` | Operation timeout in seconds |
| `screenshot_dir` | `.mehrhof/screenshots` | Directory for screenshots |
| `cookie_profile` | `"default"` | Cookie profile name for session persistence |
| `cookie_auto_load` | `true` | Auto-load cookies on browser connect |
| `cookie_auto_save` | `true` | Auto-save cookies on browser disconnect |
| `cookie_dir` | `""` | Custom cookie storage directory (default: `~/.valksor/mehrhof/`) |

**Cookie Profiles:**

Browser sessions can be persisted using named cookie profiles, enabling:

- **Session persistence**: Stay logged in across browser sessions
- **Multiple accounts**: Use different profiles for personal vs work accounts
- **Cross-project usage**: Cookies stored globally in `~/.valksor/mehrhof/`

Example profiles:
```bash
# Use default profile
mehr browser goto https://github.com

# Use work profile
mehr browser --cookie-profile work-github goto https://github.com
```

Cookies are stored as:
```
~/.valksor/mehrhof/
  ├── cookies-default.json        # Default profile
  ├── cookies-work-github.json    # Work GitHub account
  └── cookies-client-a.json       # Client-specific profile
```

See [Browser Commands](../cli/browser.md) for complete browser automation documentation.

### storage

```yaml
storage:
  work_dir: work  # Relative to workspace data directory
```

**Storage structure:**

```
project/
├── .mehrhof/
│   ├── config.yaml    # Workspace configuration (safe to commit)
│   └── .env           # Project-specific secrets (gitignored)

~/.valksor/mehrhof/workspaces/<project-id>/
├── .active_task       # Current task state
└── work/              # Task work directories
    ├── abc123/
    │   ├── work.yaml
    │   ├── notes.md
    │   ├── source/
    │   ├── specifications/
    │   └── sessions/
    └── def456/
        └── ...
```

The `<project-id>` is automatically derived from your git remote:

| Git Remote URL | Project ID |
|----------------|------------|
| `https://github.com/user/repo` | `github.com-user-repo` |
| `git@github.com:user/project.git` | `github.com-user-project` |
| `https://gitlab.com/group/subgroup/project` | `gitlab.com-group-subgroup-project` |
| No remote (local) | `local-<hash>` |

**Migration:**

If you have an existing `.mehrhof/work/` directory or `.active_task` file in your project,
they will be automatically migrated to `~/.valksor/mehrhof/workspaces/<project-id>/` when you run any mehrhof command.
The `config.yaml` and `.env` files remain in the project.

### cache

```yaml
cache:
  enabled: true  # Enable/disable caching globally

github:
  cache:
    disabled: false  # Provider-specific override
```

### env

Environment variables filtered by agent name prefix:

```yaml
env:
  CLAUDE_ANTHROPIC_API_KEY: sk-ant-xxx  # → ANTHROPIC_API_KEY to claude
  CLAUDE_MAX_TOKENS: "4096"              # → MAX_TOKENS to claude
  OPENAI_API_KEY: sk-openai-xxx          # → API_KEY to openai agent
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
GLM_API_KEY=your-glm-key

GITHUB_TOKEN=ghp_...
GITLAB_TOKEN=glpat-...
NOTION_TOKEN=your-notion-token
JIRA_TOKEN=your-jira-token
LINEAR_API_KEY=your-linear-key
WRIKE_TOKEN=your-wrike-token
YOUTRACK_TOKEN=your-youtrack-token
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

notion:
  token: ${NOTION_TOKEN}
  database_id: abc123...

jira:
  token: ${JIRA_TOKEN}
  email: user@example.com
  base_url: https://company.atlassian.net
```

**How it works:**

1. `.mehrhof/.env` stores actual token values (gitignored)
2. `.mehrhof/config.yaml` references tokens using `${VAR}` syntax (safe to commit)
3. At startup, `${VAR}` references are replaced with values from `.env` or system environment
4. Provider receives the expanded token value

**Benefits:**

- **Single source of truth**: `config.yaml` shows all token references
- **Security**: Secrets never committed to git
- **Flexibility**: Can override with system environment variables
- **Clarity**: Easy to see what's configured without checking multiple sources

### Setting Up Provider Tokens

**Option 1: Using login commands (recommended)**

```bash
mehr github login    # Prompts for token, writes to .env and config.yaml
mehr gitlab login    # Prompts for token, writes to .env and config.yaml
```

The login command will:
1. Prompt for the API token
2. Save it to `.mehrhof/.env`
3. Add `${VAR}` reference to `config.yaml`

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

**Option 3: Migration from old format**

If you have plaintext tokens in `config.yaml`, migrate them:

```bash
mehr migrate-tokens
```

This moves token values to `.env` and updates `config.yaml` to use `${VAR}` syntax.

### Usage with Agent Aliases

```yaml
# .mehrhof/config.yaml
agents:
  glm:
    extends: claude
    env:
      ANTHROPIC_API_KEY: "${GLM_API_KEY}"  # Reads from .env
```

**Security:**
- Created with `0600` permissions (user read/write only)
- Automatically added to `.gitignore` by `mehr init`
- Never commit `.env` to version control
- `config.yaml` with `${VAR}` references is safe to commit

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

Updated automatically as you use Mehrhof.

## CLI Flags

Global flags available on all commands:

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Enable verbose output |
| `--no-color` | Disable colored output |

The `NO_COLOR` environment variable is also respected.

## Standard Environment Variables

| Variable | Description |
|----------|-------------|
| `NO_COLOR` | Disable colored output |
| `ANTHROPIC_API_KEY` | Claude API key (used by Claude CLI) |

**Note:** Provider tokens (GitHub, GitLab, Notion, etc.) should be configured in `.mehrhof/.env` and referenced in `.mehrhof/config.yaml` using `${VAR}` syntax. See [Provider Authentication](#provider-authentication) above for details.

## Quick Reference

| What | How |
|------|-----|
| Verbose output | `mehr --verbose <command>` |
| Disable color | `mehr --no-color <command>` or `NO_COLOR=1` |
| Set default agent | `.mehrhof/config.yaml` → `agent.default` |
| Custom branch pattern | `.mehrhof/config.yaml` → `git.branch_pattern` |
| Set default provider | `.mehrhof/config.yaml` → `providers.default` |

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

- [CLI Reference](../cli/index.md) - All commands
- [Agents](../agents/index.md) - Agent configuration and aliases
- [Troubleshooting](../troubleshooting/index.md) - Common issues
