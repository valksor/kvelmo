# Settings

Configure your workspace through the Web UI settings page without editing configuration files directly.

## Settings Mode

Mehrhof offers two settings views to match your needs:

| Mode                 | What You See                                       | Best For                             |
|----------------------|----------------------------------------------------|--------------------------------------|
| **Simple** (default) | Agent selection, Git basics, Budget                | Most users — essential settings only |
| **Advanced**         | All settings including providers, memory, security | Power users, custom configurations   |

Toggle between modes using the **Simple/Advanced** switch in the top-right corner of the Settings page. Your preference is saved automatically.

> **Tip:** In Simple mode, you'll see a "Switch to Advanced" button if you need more options — click it to toggle instantly.

## Accessing Settings

From the navigation bar, click **Settings**. In global mode, you must first select a project.

The Settings page is organized into two sections:

| Section    | What it contains                                                                                              |
|------------|---------------------------------------------------------------------------------------------------------------|
| **Work**   | Git, agent defaults, workflow, project storage, and provider connections                                      |
| **System** | Agent aliases, budget, stack options, update patterns, memory, security, sandbox, and other advanced features |

Each settings block is collapsible. Use the chevron on the right side of each block header to expand or collapse details.

## Work Section

The **Work** section controls day-to-day workflow behavior.

### Git Settings

Configure version control integration:

| Setting            | Description                                                   |
|--------------------|---------------------------------------------------------------|
| **Commit Prefix**  | Pattern for commit messages (use `{key}`, `{type}`, `{slug}`) |
| **Branch Pattern** | Pattern for branch names                                      |
| **Default Branch** | Override branch detection (e.g., main, develop)               |
| **Auto Commit**    | Commit automatically after implementation                     |
| **Sign Commits**   | GPG sign commits                                              |
| **Stash on Start** | Auto-stash changes before creating branch                     |
| **Auto Pop Stash** | Pop stash after branch creation                               |

### Appearance Settings

Configure visual and language preferences:

| Setting      | Description                                                         |
|--------------|---------------------------------------------------------------------|
| **Language** | UI language preference (currently English, more languages coming)   |

See [Language & Translations](/web-ui/language.md) for customizing terminology and translation overrides.

### Display Settings

Configure how dates and times are shown:

| Setting      | Description                                                             |
|--------------|-------------------------------------------------------------------------|
| **Timezone** | IANA timezone for display (e.g., `Europe/Riga`, `America/New_York`)     |

Enter your timezone directly or select from the suggestions dropdown. All dates display in **dd.mm.yyyy** format.

### Agent Settings

Configure AI agent behavior:

| Setting              | Description                                 |
|----------------------|---------------------------------------------|
| **Default Agent**    | Agent to use when not specified             |
| **Timeout**          | Maximum time for agent execution (seconds)  |
| **Max Retries**      | Retry count on transient failures           |
| **Instructions**     | Global instructions included in all prompts |
| **Optimize Prompts** | Optimize prompts for token efficiency       |

### Agent Aliases

Create custom aliases that wrap base agents with additional configuration.
Location: **Settings → System**.

| Field           | Description                                     |
|-----------------|-------------------------------------------------|
| **Extends**     | Base agent to wrap (e.g., claude, codex)        |
| **Binary Path** | Custom path to agent binary (overrides default) |
| **Description** | Human-readable description                      |
| **Args**        | Space-separated CLI arguments                   |

Click **Add Alias** to create a new alias. Click the trash icon to remove one.

### Workflow Settings

Configure task lifecycle:

| Setting                    | Description                          |
|----------------------------|--------------------------------------|
| **Auto Init**              | Auto-initialize new workspaces       |
| **Session Retention**      | Days to keep session history         |
| **Delete Work on Finish**  | Clean up work directory after finish |
| **Delete Work on Abandon** | Clean up work directory on abandon   |

### Budget Settings

Control spending limits:
Location: **Settings → System**.

**Per Task:**
- **Max Cost** — Maximum USD per task
- **Max Tokens** — Maximum tokens per task
- **On Limit** — Action when limit reached (warn, pause, stop)
- **Warning At** — Percentage threshold for warnings

**Monthly:**
- **Enable Monthly Budget** — Track spending across workspace
- **Max Cost** — Monthly spending limit
- **Currency** — Display currency
- **Warning At** — Percentage threshold for warnings

### Project & Storage

Configure directory layout:

| Setting               | Description                                       |
|-----------------------|---------------------------------------------------|
| **Code Directory**    | Separate code directory (relative or absolute)    |
| **Project Directory** | Where to store work files                         |
| **Save in Project**   | Store work in project directory instead of global |

### Stack Settings

Configure stacked feature branches:
Location: **Settings → System**.

| Setting                | Description                                              |
|------------------------|----------------------------------------------------------|
| **Auto Rebase**        | When to auto-rebase child branches (disabled, on_finish) |
| **Block on Conflicts** | Block auto-rebase if conflicts detected                  |

### Updates & Patterns

Location: **Settings → System**.

| Setting                   | Description                  |
|---------------------------|------------------------------|
| **Enable Update Checks**  | Check for new versions       |
| **Check Interval**        | Hours between update checks  |
| **Specification Pattern** | Filename pattern for specs   |
| **Review Pattern**        | Filename pattern for reviews |

## Provider Settings (Work)

Configure authentication and settings for external task providers.

### Default Provider

| Setting              | Description                                                  |
|----------------------|--------------------------------------------------------------|
| **Default Provider** | Provider for bare task references (file, github, jira, etc.) |
| **Default Mention**  | Mention text when submitting tasks                           |

### GitHub

| Setting           | Description                    |
|-------------------|--------------------------------|
| **Token**         | Personal access token          |
| **Owner**         | Repository owner               |
| **Repository**    | Repository name                |
| **Target Branch** | Default branch for PRs         |
| **Draft PRs**     | Create PRs as draft by default |

### GitLab

| Setting          | Description                       |
|------------------|-----------------------------------|
| **Token**        | Personal access token             |
| **Host**         | GitLab host (default: gitlab.com) |
| **Project Path** | Path like `group/project`         |

### Jira

| Setting      | Description                    |
|--------------|--------------------------------|
| **Token**    | API token                      |
| **Email**    | Email for Cloud authentication |
| **Base URL** | Jira instance URL              |
| **Project**  | Default project key            |

### Linear

| Setting   | Description      |
|-----------|------------------|
| **Token** | API key          |
| **Team**  | Default team key |

### Notion

| Setting             | Description              |
|---------------------|--------------------------|
| **Token**           | Integration token        |
| **Database ID**     | Default database         |
| **Status Property** | Property name for status |

### Other Providers

Additional providers available: Bitbucket, Azure DevOps, Asana, ClickUp, Trello, Wrike, YouTrack. Expand the **Other Providers** section to configure these.

## System Features

Enable and configure optional features. These settings are found under the **System** section.

### Browser Automation

Control Chrome automation for testing and verification:

| Setting                    | Description                          |
|----------------------------|--------------------------------------|
| **Enable Browser**         | Allow AI agents to control a browser |
| **Port**                   | Chrome debugging port (0 = random)   |
| **Timeout**                | Seconds before timeout               |
| **Screenshot Directory**   | Where to save screenshots            |
| **Headless**               | Run browser without UI               |
| **Auto-load/save Cookies** | Persist browser cookies              |

### MCP (Model Context Protocol)

| Setting               | Description                             |
|-----------------------|-----------------------------------------|
| **Enable MCP Server** | Allow AI agents to call Mehrhof via MCP |
| **Rate Limit**        | Requests per second                     |
| **Burst Size**        | Maximum burst requests                  |

### Security Scanning

Configure automated security checks:

| Setting                      | Description                                        |
|------------------------------|----------------------------------------------------|
| **Enable Security Scanning** | Scan code for vulnerabilities                      |
| **Run On**                   | When to scan (planning, implementing, reviewing)   |
| **Fail On**                  | Severity level that blocks completion              |
| **Block Finish**             | Block task completion on failures                  |
| **Scanners**                 | Enable SAST, Secrets, Dependencies, License checks |

### Memory System

Configure semantic memory:

| Setting                  | Description                                |
|--------------------------|--------------------------------------------|
| **Enable Memory**        | Enable semantic search and learning        |
| **Backend**              | Vector database (ChromaDB, Pinecone, etc.) |
| **Connection String**    | Database connection                        |
| **Embedding Model**      | Hash-based (default) or ONNX (semantic)    |
| **Max Results**          | Search results limit                       |
| **Similarity Threshold** | Minimum match score (0-1)                  |

### Sandbox

Isolate agent execution:

| Setting            | Description             |
|--------------------|-------------------------|
| **Enable Sandbox** | Isolate agent execution |
| **Allow Network**  | Required for LLM APIs   |
| **Tmp Directory**  | Temporary file location |

### Quality & Linters

| Setting                   | Description                                |
|---------------------------|--------------------------------------------|
| **Enable Quality Checks** | Run linters and formatters                 |
| **Use Defaults**          | Auto-enable linters for detected languages |

### Links (Bidirectional Linking)

| Setting                | Description                   |
|------------------------|-------------------------------|
| **Enabled**            | Enable bidirectional linking  |
| **Auto Index**         | Automatically index links     |
| **Case Sensitive**     | Case-sensitive link matching  |
| **Max Context Length** | Characters of context to show |

### Hierarchical Context

| Setting               | Description                 |
|-----------------------|-----------------------------|
| **Include Parent**    | Show parent task context    |
| **Include Siblings**  | Show sibling task context   |
| **Max Siblings**      | Maximum siblings to include |
| **Description Limit** | Characters per description  |

### Labels

| Setting           | Description          |
|-------------------|----------------------|
| **Enable Labels** | Enable task labeling |

### Library (Documentation)

| Setting                  | Description                     |
|--------------------------|---------------------------------|
| **Auto Include Max**     | Max collections to auto-include |
| **Max Pages Per Prompt** | Pages per agent prompt          |
| **Max Token Budget**     | Token limit for library content |

## Global Mode

When running in global mode, a project picker appears at the top. Select a project to view and edit its settings.

**Security note:** In global mode, sensitive fields (API tokens, passwords) are masked for security.

## Saving Changes

1. Make your desired changes
2. Click **Save** at the top of the page

Changes are written to the configuration file and take effect immediately. A success notification confirms the update.

---

## Also Available via CLI

Prefer working from the terminal? See [CLI: config](/cli/config.md) for configuration options.

## Next Steps

- [**Dashboard**](/web-ui/dashboard.md) — Return to main view
- [**Language & Translations**](/web-ui/language.md) — Customize terminology and translations
- [**Tools**](/web-ui/tools.md) — Browser, Memory, Security, Stack tools
- [**Configuration Guide**](/configuration/index.md) — Advanced configuration
