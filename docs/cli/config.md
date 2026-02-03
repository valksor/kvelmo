# mehr config

Manage and validate Mehrhof configuration files.

## Synopsis

```bash
mehr config <subcommand> [flags]
```

## Description

The `config` command group provides utilities for managing Mehrhof configuration files in your project.

## Subcommands

### mehr config init

Create a new `.mehrhof/config.yaml` file with sensible defaults.

```bash
mehr config init [flags]
```

**Flags:**

| Flag        | Short | Description                                                          |
|-------------|-------|----------------------------------------------------------------------|
| `--force`   | `-f`  | Overwrite existing config without prompting (requires confirmation)  |
| `--project` |       | Project type for intelligent defaults: `go`, `node`, `python`, `php` |

**Behavior:**

- Detects project type automatically from common files (`go.mod`, `package.json`, `pyproject.toml`, `composer.json`)
- Creates `.mehrhof/config.yaml` with project-appropriate defaults
- Safe by default: warns if config exists and exits (unless `--force` is used)
- Generates helpful comments showing all available options

**Examples:**

```bash
mehr config init

mehr config init --project go

mehr config init --force
```

**Generated Config Structure:**

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

workflow:
  auto_init: true
  session_retention_days: 30

# User-defined agent aliases
agents:
  # Example: Custom agent with environment variables
  # work-account:
  #   extends: claude
  #   description: "Claude with work API key"
  #   env:
  #     ANTHROPIC_API_KEY: "${WORK_API_KEY}"

# Project layout (separate hub from code target)
project:
  code_dir: "../reporting-engine"  # relative or absolute; empty = hub is code target

# Plugins
plugins:
  enabled: []
```

**What to do next:**

```bash
mehr config validate

vim .mehrhof/config.yaml

mehr start task.md
```

### mehr config validate

Validate workspace (`.mehrhof/config.yaml`) and app (`.env`) configuration files.

```bash
mehr config validate [flags]
```

**Flags:**

| Flag               | Description                                                |
|--------------------|------------------------------------------------------------|
| `--workspace-only` | Only validate workspace config (`.mehrhof/config.yaml`)    |
| `--app-only`       | Only validate app config (`.env` files)                    |
| `--strict`         | Treat warnings as errors (exit code 1 if warnings present) |
| `--format`         | Output format: `text` (default), `json`                    |

**Exit Codes:**

| Code | Meaning                             |
|------|-------------------------------------|
| 0    | Configuration is valid              |
| 1    | One or more validation errors found |

### Project Settings

| Setting            | Type   | Description                                                          | Default |
|--------------------|--------|----------------------------------------------------------------------|---------|
| `project.code_dir` | string | Path to code target directory (relative to project root or absolute) | `""`    |

When `code_dir` is set, the project hub (`.mehrhof/`, tasks, queues) stays in the current directory while agents edit code, run git operations, and execute linters in the code target directory. Supports environment variables (`${HOME}/code`).

**Example:**

```yaml
# .mehrhof/config.yaml
project:
  code_dir: "../reporting-engine"
```

### mehr config explain

Trace agent resolution path to debug which agent is selected for a workflow step.

```bash
mehr config explain --agent <step> [flags]
```

**Flags:**

| Flag       | Description                                        |
|------------|----------------------------------------------------|
| `--agent`  | Workflow step to explain (plan, implement, review) |
| `--format` | Output format: `text` (default), `json`            |

**Examples:**

```bash
# Explain which agent is used for planning
mehr config explain --agent plan

# Explain implementing agent
mehr config explain --agent implement
```

**Output:**

```bash
$ mehr config explain --agent plan

Effective agent: claude-sonnet

Resolution path (priority order):
  1. CLI --agent-plan flag: not set
  2. CLI --agent flag: not set
  3. Task frontmatter agent_steps.planning.agent: not set
  4. Task frontmatter agent: not set
  5. Workspace config agent.steps.planning.name: claude-sonnet ✓
  6. Workspace config agent.default: claude (overridden)

Configuration:
  Model: claude-sonnet-4-20250514
  Timeout: 300s
  Args: --permission-mode plan

To override:
  mehr plan --agent-plan claude-opus
  Or set in .mehrhof/config.yaml:
    agent:
      steps:
        planning:
          name: claude-opus
```

**Use cases:**
- Debug why a specific agent is being used
- Understand agent resolution priority
- Verify workspace configuration is applied correctly
- Troubleshoot agent selection issues

## Examples

### Validate All Configuration

```bash
mehr config validate
```

**Output (valid config):**

```
Validating configuration...

Configuration is VALID
```

**Output (with issues):**

```
Validating configuration...

.mehrhof/config.yaml:
  ERROR [AGENT_ALIAS_CIRCULAR] agents.work-fast: Circular dependency detected: work-fast -> work-account -> work-fast
    Suggestion: Remove circular reference in 'extends' field

  WARNING [ENV_VAR_UNSET] agents.work-account.env.ANTHROPIC_API_KEY: Environment variable ${WORK_API_KEY} is not set
    Suggestion: Set WORK_API_KEY environment variable

Summary: 1 error(s), 1 warning(s)
Configuration is INVALID
```

### JSON Output for CI

```bash
mehr config validate --format json
```

```json
{
  "valid": false,
  "errors": 1,
  "warnings": 1,
  "findings": [
    {
      "severity": "error",
      "code": "AGENT_ALIAS_CIRCULAR",
      "message": "Circular dependency detected: work-fast -> work-account -> work-fast",
      "path": "agents.work-fast",
      "file": ".mehrhof/config.yaml",
      "suggestion": "Remove circular reference in 'extends' field"
    },
    {
      "severity": "warning",
      "code": "ENV_VAR_UNSET",
      "message": "Environment variable ${WORK_API_KEY} is not set",
      "path": "agents.work-account.env.ANTHROPIC_API_KEY",
      "file": ".mehrhof/config.yaml",
      "suggestion": "Set WORK_API_KEY environment variable"
    }
  ]
}
```

### Strict Mode

Treat warnings as errors (useful for CI pipelines):

```bash
mehr config validate --strict
```

### Validate Only Workspace Config

```bash
mehr config validate --workspace-only
```

### Validate Only App Config

```bash
mehr config validate --app-only
```

## Validations Performed

### Workspace Config (`.mehrhof/config.yaml`)

| Check                     | Error Code               | Description                          |
|---------------------------|--------------------------|--------------------------------------|
| YAML syntax               | `YAML_SYNTAX`            | Invalid YAML structure               |
| Agent alias circular deps | `AGENT_ALIAS_CIRCULAR`   | Alias chain forms a loop             |
| Undefined agent reference | `AGENT_ALIAS_UNDEFINED`  | `extends` references unknown agent   |
| Missing extends field     | `AGENT_ALIAS_NO_EXTENDS` | Alias missing required `extends`     |
| Invalid git pattern       | `GIT_PATTERN_INVALID`    | Unknown placeholder in pattern       |
| Empty git pattern         | `GIT_PATTERN_EMPTY`      | Branch pattern is empty              |
| Invalid enum value        | `INVALID_ENUM`           | Value not in allowed set             |
| Out of range value        | `INVALID_RANGE`          | Numeric value outside bounds         |
| Unset env variable        | `ENV_VAR_UNSET`          | `${VAR}` reference not set (warning) |
| Plugin config mismatch    | `PLUGIN_NOT_FOUND`       | Config for disabled plugin (warning) |

### App Config (`.env` files)

| Check             | Error Code       | Description                          |
|-------------------|------------------|--------------------------------------|
| Load error        | `ENV_LOAD_ERROR` | Failed to parse .env file            |
| Invalid agent     | `INVALID_ENUM`   | Unknown `agent.default` value        |
| Invalid UI format | `INVALID_ENUM`   | Must be `text` or `json`             |
| Invalid progress  | `INVALID_ENUM`   | Must be `spinner`, `dots`, or `none` |
| Out of range      | `INVALID_RANGE`  | Timeout, retries, etc. out of bounds |

### Cross-Config Validation

| Check             | Error Code        | Description                                 |
|-------------------|-------------------|---------------------------------------------|
| Agent consistency | `AGENT_MISMATCH`  | Workspace default agent unknown             |
| Config conflict   | `CONFIG_CONFLICT` | Different values in app vs workspace (info) |

## Valid Git Pattern Placeholders

The following placeholders are valid in `git.branch_pattern` and `git.commit_prefix`:

| Placeholder | Description                       |
|-------------|-----------------------------------|
| `{key}`     | Task key/ID                       |
| `{task_id}` | Task ID                           |
| `{type}`    | Task type (feature, bugfix, etc.) |
| `{slug}`    | URL-safe task title               |
| `{title}`   | Task title                        |

**Example patterns:**

```yaml
git:
  branch_pattern: "{type}/{key}--{slug}" # feature/ABC123--add-login
  commit_prefix: "[{key}]" # [ABC123] Implement feature
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Validate mehr config
  run: mehr config validate --strict --format json
```

### Pre-commit Hook

```bash
#!/bin/sh
mehr config validate --strict || exit 1
```

## Troubleshooting

### "circular alias dependency detected"

Your agent aliases form a loop:

```yaml
agents:
  a:
    extends: b # Problem: a -> b -> a
  b:
    extends: a
```

Fix by breaking the cycle.

### "extends unknown agent"

The `extends` field references an agent that doesn't exist:

```yaml
agents:
  custom:
    extends: nonexistent # "nonexistent" is not a known agent
```

Fix by using a valid agent name (`claude` or another defined alias).

### "Environment variable not set"

A `${VAR}` reference in your config points to an unset variable:

```yaml
agents:
  work-account:
    env:
      ANTHROPIC_API_KEY: "${WORK_API_KEY}" # WORK_API_KEY not set
```

Fix by setting the environment variable or using a literal value.

## Web UI

Prefer a visual interface? See [Web UI: Settings](/web-ui/settings.md).

## See Also

- [Configuration Guide](/configuration/index.md) - How configuration works and file reference
- [mehr agents](agents.md) - Agent commands
- [AI Agents](/agents/index.md) - Agent configuration and aliases
