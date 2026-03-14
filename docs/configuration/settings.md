# Configuration Reference

kvelmo uses a two-tier configuration system: **global** settings for user preferences and **project** settings for repository-specific overrides.

## Configuration Files

| Scope   | Location                        | Priority                  |
|---------|---------------------------------|---------------------------|
| Global  | `~/.valksor/kvelmo/kvelmo.yaml` | Lower                     |
| Project | `.valksor/kvelmo.yaml`          | Higher (overrides global) |

Project settings override global settings when both are present.

### Sensitive Values

API tokens and credentials are stored separately in `.env` files:

| Scope   | Location                 |
|---------|--------------------------|
| Global  | `~/.valksor/kvelmo/.env` |
| Project | `.valksor/.env`          |

## Environment Variables

These environment variables can be set instead of (or in addition to) configuration files:

| Variable       | Description                                          |
|----------------|------------------------------------------------------|
| `GITHUB_TOKEN` | GitHub personal access token (requires `repo` scope) |
| `GITLAB_TOKEN` | GitLab personal access token (requires `api` scope)  |
| `WRIKE_TOKEN`  | Wrike API token                                      |
| `LINEAR_TOKEN` | Linear API token (see format notes below)            |

### Linear Token Format

Linear accepts two types of tokens with different configuration formats:

| Token Type         | Prefix     | Configuration                  |
|--------------------|------------|--------------------------------|
| Personal API Key   | `lin_api_` | Use directly, no prefix needed |
| OAuth Access Token | (varies)   | Include `Bearer ` prefix       |

**Examples:**

```bash
# Personal API key - use as-is
LINEAR_TOKEN=lin_api_xxxxxxxxxxxxxxxx

# OAuth token - include Bearer prefix
LINEAR_TOKEN="Bearer oauth_token_here"
```

Personal API keys (starting with `lin_api_`) are used directly in the Authorization header. OAuth access tokens require the `Bearer ` prefix to be included in your configuration.

## Full Configuration Example

```yaml
# ~/.valksor/kvelmo/kvelmo.yaml (global) or .valksor/kvelmo.yaml (project)

agent:
  default: claude          # Default AI agent: claude | codex
  allowed:                 # Agents permitted for this project
    - claude
    - codex

providers:
  default: github          # Default task provider: github | gitlab | wrike | file
  github:
    owner: myorg           # Default repository owner (auto-detected from git remote)
    allow_ticket_comment: true  # Post status comments on issues/PRs
  gitlab:
    base_url: https://gitlab.com  # GitLab instance URL
  wrike:
    include_parent_context: true    # Include parent task in AI prompts
    include_sibling_context: true   # Include sibling tasks in AI prompts

git:
  base_branch: main                       # Base branch for feature branches (auto-detected if omitted)
  branch_pattern: "feature/{key}--{slug}"  # Branch naming pattern
  commit_prefix: "[{key}]"                # Commit message prefix
  create_branch: true                     # Auto-create branch on task start
  auto_commit: true                       # Auto-commit after implementation (default: true)
  sign_commits: false                     # GPG sign commits
  allow_pr_comment: true                  # Override default (false) to enable PR comments

workers:
  max: 3                   # Maximum concurrent workers (1-10)

storage:
  save_in_project: false   # Store specs/plans in .valksor/ instead of ~/.valksor/kvelmo/

workflow:
  use_worktree_isolation: true  # Create isolated git worktree for each task
  external_review:
    mode: ask                   # When to run external review: ask | always | never
    command: coderabbit         # CLI review tool command (default: coderabbit)

watchdog:
  enabled: true            # Monitor for memory leaks
  interval_sec: 30         # Sampling interval (10-300 seconds)
  window_size: 10          # Samples required to confirm leak (5-60)
  threshold_mb: 200        # Heap growth threshold (min 50MB)

# Custom agents extend base agents with additional configuration
custom_agents:
  my-fast-claude:
    extends: claude
    description: "Claude with faster response settings"
    args: ["--fast"]
    env:
      ANTHROPIC_TIMEOUT: "30"
```

## Settings Reference

### Agent Settings

| Key             | Type     | Default               | Description                          |
|-----------------|----------|-----------------------|--------------------------------------|
| `agent.default` | string   | `claude`              | Default AI agent when none specified |
| `agent.allowed` | string[] | `["claude", "codex"]` | Agents permitted for this project    |

### Provider Settings

| Key                                        | Type   | Default              | Description                         |
|--------------------------------------------|--------|----------------------|-------------------------------------|
| `providers.default`                        | string | `github`             | Default task provider               |
| `providers.github.owner`                   | string | (auto-detect)        | Default repository owner            |
| `providers.github.allow_ticket_comment`    | bool   | `false`              | Post status comments on issues/PRs  |
| `providers.gitlab.base_url`                | string | `https://gitlab.com` | GitLab instance URL                 |
| `providers.wrike.include_parent_context`   | bool   | `true`               | Fetch parent task for AI context    |
| `providers.wrike.include_sibling_context`  | bool   | `true`               | Fetch sibling tasks for AI context  |
| `providers.linear.team`                    | string | (none)               | Default Linear team prefix          |
| `providers.linear.include_parent_context`  | bool   | `true`               | Fetch parent issue for AI context   |
| `providers.linear.include_sibling_context` | bool   | `true`               | Fetch sibling issues for AI context |
| `providers.linear.allow_ticket_comment`    | bool   | `false`              | Post status comments on issues      |

**Performance tip:** Setting `include_parent_context: false` and `include_sibling_context: false` for Wrike can save ~200ms per task fetch by skipping hierarchy API calls.

### Git Settings

| Key                    | Type   | Default                 | Description                           |
|------------------------|--------|-------------------------|---------------------------------------|
| `git.base_branch`      | string | (auto-detect)           | Base branch for feature branches      |
| `git.branch_pattern`   | string | `feature/{key}--{slug}` | Branch naming pattern                 |
| `git.commit_prefix`    | string | `[{key}]`               | Commit message prefix                 |
| `git.create_branch`    | bool   | `true`                  | Auto-create branch on task start      |
| `git.auto_commit`      | bool   | `true`                  | Auto-commit after implementation      |
| `git.sign_commits`     | bool   | `false`                 | GPG sign commits                      |
| `git.allow_pr_comment` | bool   | `false`                 | Post status comments on pull requests |

**Pattern Variables:**
- `{key}` - Task ID from provider (e.g., `owner/repo#123`)
- `{slug}` - URL-safe task title
- `{type}` - Task type (issue, pr, etc.)

### Worker Settings

| Key           | Type | Default | Description                       |
|---------------|------|---------|-----------------------------------|
| `workers.max` | int  | `3`     | Maximum concurrent workers (1-10) |

### Storage Settings

| Key                       | Type | Default | Description                               |
|---------------------------|------|---------|-------------------------------------------|
| `storage.save_in_project` | bool | `false` | Store data in `.valksor/` instead of home |

### Workflow Settings

| Key                                  | Type   | Default      | Description                            |
|--------------------------------------|--------|--------------|----------------------------------------|
| `workflow.use_worktree_isolation`    | bool   | `true`       | Create isolated git worktree per task  |
| `workflow.external_review.mode`      | string | `ask`        | When to run external review: ask, always, never |
| `workflow.external_review.command`   | string | `coderabbit` | CLI command for external review tool   |

### Watchdog Settings (Advanced)

| Key                     | Type | Default | Description                      |
|-------------------------|------|---------|----------------------------------|
| `watchdog.enabled`      | bool | `true`  | Enable memory leak watchdog      |
| `watchdog.interval_sec` | int  | `30`    | Heap sampling interval (10-300s) |
| `watchdog.window_size`  | int  | `10`    | Samples to confirm leak (5-60)   |
| `watchdog.threshold_mb` | int  | `200`   | Heap growth threshold (min 50MB) |

### Custom Agents

Define custom agent configurations that extend base agents:

```yaml
custom_agents:
  claude-opus:
    extends: claude
    description: "Claude with Opus model"
    args: ["--model", "opus"]
    # env:
    #   If ANTHROPIC_API_KEY is already in your shell environment, omit this entirely.
    #   The subprocess inherits the parent environment automatically.
```

> **Security:** Do not store API keys as literal values in settings files. Settings are stored in plaintext at `~/.valksor/kvelmo/kvelmo.yaml` or `.valksor/kvelmo.yaml`. If you need to override an API key for a custom agent, set it in your shell environment instead.

| Key           | Type     | Required | Description                                 |
|---------------|----------|----------|---------------------------------------------|
| `extends`     | string   | Yes      | Base agent to wrap (`claude` or `codex`)    |
| `description` | string   | No       | Human-readable description                  |
| `args`        | string[] | No       | Additional CLI arguments                    |
| `env`         | map      | No       | Environment variables (inherits from shell) |

## CLI Commands

### View Current Settings

```bash
kvelmo config show
```

### Get a Specific Value

```bash
kvelmo config get agent.default
kvelmo config get providers.wrike.include_parent_context
```

### Set a Value

```bash
# Global scope (default)
kvelmo config set agent.default codex

# Project scope
kvelmo config set --project providers.github.owner myorg

# Boolean values
kvelmo config set workflow.use_worktree_isolation true
```

### Initialize Configuration

```bash
# Create global config
kvelmo config init

# Create project config
kvelmo config init --project
```

## Web UI

The web UI provides a graphical settings editor accessible at `http://localhost:6337` when running `kvelmo serve`.

Settings are organized into sections:
- **Agent** - AI agent configuration
- **Providers** - Task source configuration (GitHub, GitLab, Wrike)
- **Git** - Branch and commit patterns
- **Workers** - Concurrent execution limits
- **Storage** - Data location preferences
- **Workflow** - Task isolation options
- **Watchdog** - Memory monitoring (advanced)

## Troubleshooting

### Settings Not Taking Effect

1. Check which scope the setting is in:
   ```bash
   kvelmo config show
   ```
   Project settings override global settings.

2. Restart the server after changing settings:
   ```bash
   # Settings are cached for performance. Restart to apply changes.
   pkill kvelmo
   kvelmo serve
   ```

### Token Issues

Tokens are stored in `.env` files, not `kvelmo.yaml`:

```bash
# Global tokens
echo "GITHUB_TOKEN=ghp_xxx" >> ~/.valksor/kvelmo/.env

# Project tokens (takes precedence)
echo "GITHUB_TOKEN=ghp_yyy" >> .valksor/.env
```

Or set as environment variables:
```bash
export GITHUB_TOKEN=ghp_xxx
```

### Permission Errors

Ensure configuration directories exist:
```bash
mkdir -p ~/.valksor/kvelmo
mkdir -p .valksor
```
