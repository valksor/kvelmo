# mehr migrate-tokens

Migrate provider authentication tokens from plaintext to environment variable references.

## Synopsis

```bash
mehr migrate-tokens
```

## Description

The `migrate-tokens` command converts provider tokens stored as plaintext in `.mehrhof/config.yaml` to use environment variable references (`${VAR}` syntax). The actual token values are moved to `.mehrhof/.env`.

**This migration is optional** - plaintext tokens in config.yaml will continue to work. The benefit of `${VAR}` syntax is that:
- Config.yaml becomes a single source of truth for all configuration
- Secrets are separated from configuration
- Easier to see what tokens are configured at a glance
- Aligns with 12-factor app principles

## What Gets Migrated

The following provider tokens are migrated:

| Provider      | Config Field           | Environment Variable    |
| ------------- | ---------------------- | ---------------------- |
| GitHub        | `github.token`         | `GITHUB_TOKEN`         |
| GitLab        | `gitlab.token`         | `GITLAB_TOKEN`         |
| Notion        | `notion.token`         | `NOTION_TOKEN`         |
| Jira          | `jira.token`           | `JIRA_TOKEN`           |
| Linear        | `linear.token`         | `LINEAR_API_KEY`       |
| Wrike         | `wrike.token`          | `WRIKE_TOKEN`          |
| YouTrack      | `youtrack.token`       | `YOUTRACK_TOKEN`       |
| Bitbucket     | `bitbucket.app_password` | `BITBUCKET_APP_PASSWORD` |
| Asana         | `asana.token`          | `ASANA_TOKEN`          |
| ClickUp       | `clickup.token`        | `CLICKUP_TOKEN`        |
| Azure DevOps  | `azuredevops.token`    | `AZURE_DEVOPS_TOKEN`   |
| Trello        | `trello.api_key`       | `TRELLO_API_KEY`       |
| Trello        | `trello.token`         | `TRELLO_TOKEN`         |

## Examples

### Before Migration

**config.yaml:**
```yaml
github:
  token: "ghp_abc123..."
gitlab:
  token: "glpat-xyz789..."
```

### After Migration

**config.yaml:**
```yaml
github:
  token: "${GITHUB_TOKEN}"
gitlab:
  token: "${GITLAB_TOKEN}"
```

**.mehrhof/.env:**
```bash
GITHUB_TOKEN=ghp_abc123...
GITLAB_TOKEN=glpat-xyz789...
```

## Running the Migration

```bash
$ mehr migrate-tokens

Migrated GitHub token to ${GITHUB_TOKEN}
Migrated GitLab token to ${GITLAB_TOKEN}

Migration complete!
- Tokens moved to: /path/to/project/.mehrhof/.env
- Token references updated in config.yaml

Note: If you had these tokens set as environment variables in your shell,
you may want to unset them to avoid confusion:
  unset GITHUB_TOKEN
  unset GITLAB_TOKEN
```

## When No Migration Is Needed

```bash
$ mehr migrate-tokens

No migration needed. Tokens already use ${VAR} syntax or are not configured.
```

This occurs when:
- Tokens already use `${VAR}` syntax
- No tokens are configured in config.yaml

## How It Works

1. Reads current `.mehrhof/config.yaml`
2. Identifies plaintext token values
3. Creates or updates `.mehrhof/.env` with token values
4. Updates config.yaml to use `${VAR}` references
5. Saves both files

## Environment Variable Expansion

Values using `${VAR}` syntax reference environment variables:

- **System environment variables** - Read from your shell
- **.mehrhof/.env** - Auto-loaded by mehrhof

## Security Considerations

### After Migration

1. **Add `.mehrhof/.env` to `.gitignore`**
   ```bash
   echo ".mehrhof/.env" >> .gitignore
   ```

2. **Remove token history from git**
   ```bash
   # Use git-filter-repo or BFG Repo-Cleaner to remove token history
   ```

3. **Set proper file permissions**
   ```bash
   chmod 600 .mehrhof/.env
   ```

## Reverting Migration

To revert to plaintext tokens (not recommended):

1. Read values from `.mehrhof/.env`
2. Update config.yaml with literal values
3. Delete `.mehrhof/.env`

## See Also

- [Configuration Guide](../configuration/index.md) - Configuration file reference
- [Provider Authentication](login.md) - Authenticate with providers
- [Environment Variables](../configuration/index.md) - How env vars work in config
