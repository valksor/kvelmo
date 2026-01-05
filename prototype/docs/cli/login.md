# Provider Login Commands

The `mehr <provider> login` commands provide an interactive way to configure authentication tokens for various providers.

## Supported Providers

| Provider | Command | Environment Variable |
|----------|---------|----------------------|
| GitHub | `mehr github login` | `GITHUB_TOKEN` |
| GitLab | `mehr gitlab login` | `GITLAB_TOKEN` |
| Notion | `mehr notion login` | `NOTION_TOKEN` |
| Jira | `mehr jira login` | `JIRA_TOKEN` |
| Linear | `mehr linear login` | `LINEAR_API_KEY` |
| Wrike | `mehr wrike login` | `WRIKE_TOKEN` |
| YouTrack | `mehr youtrack login` | `YOUTRACK_TOKEN` |

## Usage

```bash
mehr github login
mehr gitlab login
mehr jira login
```

## What Happens When You Run Login

1. **Check for existing tokens**: The command checks if a token is already configured in:
   - `.mehrhof/.env` file
   - `config.yaml`

2. **Prompt for override**: If a token exists, you'll be asked if you want to replace it.

3. **Enter your token**: Paste your token when prompted. **Token input is hidden** and displayed as asterisks (`****`) for security.

4. **Save to .env**: The token is saved to `.mehrhof/.env` with secure permissions (`0o600`).

5. **Update config.yaml**: A reference to the environment variable is added to `config.yaml` using `${VAR}` syntax.

## Example Session

```bash
$ mehr github login

Get a token at: https://github.com/settings/tokens
Token will be saved to .mehrhof/.env and referenced in config.yaml
? Enter your GitHub API token (leave empty to cancel): ********

Token saved to /project/.mehrhof/.env
Token reference added to config.yaml
```

> **Note**: Your token input is masked with asterisks (`****`) for security. The token is not visible in the terminal while typing or pasting.

### When Token Already Exists

```bash
$ mehr github login
Token already configured via .mehrhof/.env file: ghp_...abcd
Override? [y/N]: y

Get a token at: https://github.com/settings/tokens
Token will be saved to .mehrhof/.env and referenced in config.yaml
? Enter your GitHub API token (leave empty to cancel): ********

Token saved to /project/.mehrhof/.env
Token reference added to config.yaml
```

## Token Storage

Tokens are stored using a two-file approach:

### 1. `.mehrhof/.env` (gitignored)
Stores the actual token values with secure permissions (`0o600`).

```bash
# .mehrhof/.env
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
GITLAB_TOKEN=glpat_xxxxxxxxxxxxxxxxxxxx
NOTION_TOKEN=secret_xxxxxxxxxxxxxxxxxxxxxx
```

### 2. `.mehrhof/config.yaml` (safe to commit)
References the environment variables using `${VAR}` syntax.

```yaml
# .mehrhof/config.yaml
github:
  token: ${GITHUB_TOKEN}

gitlab:
  token: ${GITLAB_TOKEN}

notion:
  token: ${NOTION_TOKEN}
```

**Benefits:**
- ✅ **Single source of truth**: `config.yaml` shows all token references
- ✅ **Security**: Actual secrets never committed to git
- ✅ **Clarity**: Easy to see what's configured
- ✅ **Flexibility**: Can override with system environment variables

## Provider Aliases

Some providers have short aliases:

| Alias | Resolves To |
|-------|-------------|
| `gh` | github |
| `gl` | gitlab |
| `nt` | notion |
| `yt` | youtrack |

```bash
mehr github login
mehr gh login
```

## Migration from Old Format

If you have plaintext tokens in `config.yaml` from an older version, migrate them:

```bash
mehr migrate-tokens
```

This command:
1. Reads your `config.yaml`
2. Finds plaintext tokens (not using `${VAR}` syntax)
3. Moves token values to `.mehrhof/.env`
4. Updates `config.yaml` to use `${VAR}` references

## Getting Tokens

### GitHub
Visit: https://github.com/settings/tokens
- Create a **Classic** Personal Access Token
- Required scopes: `repo`, `read:org`

### GitLab
Visit: https://gitlab.com/-/user_settings/personal_access_tokens
- Create a personal access token
- Required scopes: `api`, `read_repository`

### Notion
Visit: https://www.notion.so/my-integrations
- Create a new integration
- Copy the "Internal Integration Token"

### Jira
Visit: https://id.atlassian.com/manage-profile/security/api-tokens
- Create an API token
- For Jira Cloud, use your email as the username

### Linear
Visit: https://linear.app/settings/api
- Create a personal API token
- Requires `read` scope

### Wrike
Visit: https://www.wrike.com/workspace.htm
- Generate an API token in your profile settings

### YouTrack
Visit: https://www.jetbrains.com/help/youtrack/manage-user-token.html
- Create a permanent token in your profile

