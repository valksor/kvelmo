# mehr login

Provider login commands for configuring authentication tokens.

## Synopsis

```bash
mehr <provider> login
```

## Description

The `mehr <provider> login` commands provide an interactive way to configure authentication tokens for various providers.

## Supported Providers

| Provider     | Command                  | Environment Variable     |
|--------------|--------------------------|--------------------------|
| GitHub       | `mehr github login`      | `GITHUB_TOKEN`           |
| GitLab       | `mehr gitlab login`      | `GITLAB_TOKEN`           |
| Notion       | `mehr notion login`      | `NOTION_TOKEN`           |
| Jira         | `mehr jira login`        | `JIRA_TOKEN`             |
| Linear       | `mehr linear login`      | `LINEAR_API_KEY`         |
| Wrike        | `mehr wrike login`       | `WRIKE_TOKEN`            |
| YouTrack     | `mehr youtrack login`    | `YOUTRACK_TOKEN`         |
| Bitbucket    | `mehr bitbucket login`   | `BITBUCKET_APP_PASSWORD` |
| Asana        | `mehr asana login`       | `ASANA_TOKEN`            |
| ClickUp      | `mehr clickup login`     | `CLICKUP_TOKEN`          |
| Trello       | `mehr trello login`      | `TRELLO_TOKEN`           |
| Azure DevOps | `mehr azuredevops login` | `AZURE_DEVOPS_PAT`       |

## What Happens When You Run Login

1. **Check for existing tokens**: The command checks if a token is already configured in:
   - `.mehrhof/.env` file
   - `config.yaml`

2. **Prompt for override**: If a token exists, you'll be asked if you want to replace it.

3. **Enter your token**: Paste your token when prompted. **Token input is hidden** and displayed as asterisks (`****`) for security.

4. **Save to .env**: The token is saved to `.mehrhof/.env` with secure permissions (`0o600`).

5. **Update config.yaml**: A reference to the environment variable is added to `config.yaml` using `${VAR}` syntax.

## Example Session

When you run a login command, you'll see helpful guidance on where to get your token:

```bash
$ mehr github login

GitHub Token Setup
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📍 Get token: https://github.com/settings/tokens
📋 Steps:     Settings → Developer settings → Personal access tokens
🔑 Required:  repo, read:user
💡 Format:    Token starts with 'ghp_'
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Token will be saved to .mehrhof/.env and referenced in config.yaml
? Enter your GitHub API token (leave empty to cancel): ********

Token saved to /project/.mehrhof/.env
Token reference added to config.yaml
```

The guidance includes:
- **📍 Get token**: Direct URL to the token generation page
- **📋 Steps**: Navigation path to find the token settings
- **🔑 Required**: Permissions or scopes needed for the token
- **💡 Format**: Token prefix pattern for validation (when applicable)

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

Many providers have short aliases for faster typing:

| Alias          | Resolves To |
|----------------|-------------|
| `gh`           | github      |
| `gl`           | gitlab      |
| `nt`           | notion      |
| `yt`           | youtrack    |
| `bb`           | bitbucket   |
| `cu`           | clickup     |
| `ado`, `azure` | azuredevops |

```bash
mehr github login
mehr gh login        # Same as above

mehr azuredevops login
mehr ado login       # Same as above
```

## Getting Tokens

### GitHub
Visit: https://github.com/settings/tokens
- Create **Classic** Personal Access Token
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
- Profile → Account Security → Tokens
- Required scopes: Hub scope

### Bitbucket
Visit: https://bitbucket.org/account/settings/app-passwords
- Settings → App passwords → Create app password
- Required permissions: Repositories read/write

### Asana
Visit: https://app.asana.com/0/developer-console
- Profile → Apps → Developer Console → Personal access token
- Grants full access to your Asana account

### ClickUp
Visit: https://app.clickup.com/settings/apps
- Settings → Apps → Generate API Token
- Grants full workspace access

### Trello
Visit: https://trello.com/power-ups/admin
- Power-Ups Admin → Developer API Keys → Generate Token
- Requires both API key (`TRELLO_KEY`) and token (`TRELLO_TOKEN`)
- Required permissions: Read/write boards

### Azure DevOps
Visit: https://dev.azure.com/_usersSettings/tokens
- User Settings → Personal access tokens → New Token
- Required scopes: Work Items read/write, Code read

