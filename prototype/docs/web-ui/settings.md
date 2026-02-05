# Settings

Configure your workspace through the Web UI settings page without editing YAML files directly.

## Accessing Settings

From the navigation bar, open the **Admin** dropdown and click **Settings**.

```
┌──────────────────────────────────────────────────────────────┐
│  Workspace Settings                                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [General] [Project] [Git] [Agent] [Workflow] [Browser] ...  │
│  ─────────────────────────────────────────────────────────   │
│                                                              │
│  Settings apply to: current-project                          │
│                                                              │
│                                        [Reset] [Save]        │
└──────────────────────────────────────────────────────────────┘
```

## Settings Sections

### Project Settings

Configure the project layout when your task hub and code target are in different directories:

```
┌──────────────────────────────────────────────────────────────┐
│  Project Settings                                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Code Directory                                              │
│  [../reporting-engine ]                                      │
│                                                              │
│  Path to code target directory (relative or absolute).       │
│  Leave empty if the project hub is the code target.          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting            | Description                                                          | Default |
|--------------------|----------------------------------------------------------------------|---------|
| **Code Directory** | Path to the code target directory (relative to project root or absolute) | `""`    |

When set, the project hub (`.mehrhof/`, tasks, queues) stays in the current directory while agents edit code, run git operations, and execute linters in the specified code directory. Supports relative paths and environment variables (`${HOME}/code`).

**Example use case:** You have a Mehrhof hub in `/projects/task-hub/` but the actual code lives in `/projects/reporting-engine/`. Set **Code Directory** to `../reporting-engine` so agents operate on the correct codebase.

**CLI equivalent:** `project.code_dir` in `.mehrhof/config.yaml`. See [CLI: config](/cli/config.md#project-settings).

### Git Settings

Configure version control integration:

```
┌──────────────────────────────────────────────────────────────┐
│  Git Settings                                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Auto-commit               [✓ Enabled]                       │
│  Sign commits             [✗ Disabled]                       │
│  Target branch            [main              ]               │
│  Branch pattern           [{type}/{key}--{slug}]             │
│  Commit prefix            [{key}]                            │
│  Stash on start           [✗ Disabled]                       │
│  Auto-pop stash           [✓ Enabled]                        │
│                                                              │
│  Branch Pattern Variables:                                   │
│    {type}    - Task type (feature, fix, etc.)                │
│    {key}     - External key (JIRA-123, etc.)                 │
│    {slug}    - URL-safe title slug                           │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting            | Description                          | Default                |
|--------------------|--------------------------------------|------------------------|
| **Auto-commit**    | Automatically commit changes         | `true`                 |
| **Sign commits**   | GPG sign commits                     | `false`                |
| **Target branch**  | Branch to merge into                 | `main`                 |
| **Branch pattern** | Template for branch names            | `{type}/{key}--{slug}` |
| **Commit prefix**  | Template for commit messages         | `[{key}]`              |
| **Stash on start** | Auto-stash before creating branch    | `false`                |
| **Auto-pop stash** | Auto-pop stash after branch creation | `true`                 |

### Agent Settings

Configure AI agent behavior:

```
┌──────────────────────────────────────────────────────────────┐
│  Agent Settings                                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Default agent            [claude            ]               │
│  Timeout (seconds)       [300              ]                 │
│  Max retries             [3                ]                 │
│                                                              │
│  Step-specific agents:                                       │
│    Planning:           [claude           ]                   │
│    Implementing:       [claude-sonnet    ]                   │
│    Reviewing:          [claude           ]                   │
│                                                              │
│  Available agents:                                           │
│    • claude - Claude AI via Claude CLI                       │
│    • claude-opus - Claude 3 Opus (slower, smarter)           │
│    • claude-sonnet - Claude 3.5 Sonnet (faster)              │
│                                                              │
│  [Add Custom Agent Alias]                                    │
└──────────────────────────────────────────────────────────────┘
```

| Setting           | Description                   | Default  |
|-------------------|-------------------------------|----------|
| **Default agent** | Agent to use for all steps    | `claude` |
| **Timeout**       | Maximum seconds per operation | `300`    |
| **Max retries**   | Retry attempts on failure     | `3`      |

### Workflow Settings

Configure task lifecycle behavior:

```
┌──────────────────────────────────────────────────────────────┐
│  Workflow Settings                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Auto-init               [✓ Enabled]                         │
│  Session retention       [7                ] days            │
│  Delete work on finish   [✗ Disabled]                        │
│  Delete work on abandon  [✗ Disabled]                        │
│                                                              │
│  Quality checks on finish:                                   │
│    Run tests             [✓ Enabled]                         │
│    Run quality           [✓ Enabled]                         │
│    Fail on error         [✓ Enabled]                         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting                    | Description                      | Default |
|----------------------------|----------------------------------|---------|
| **Auto-init**              | Auto-initialize new workspaces   | `true`  |
| **Session retention**      | Days to keep session history     | `7`     |
| **Delete work on finish**  | Clean up after successful finish | `false` |
| **Delete work on abandon** | Clean up after abandon           | `false` |

### Browser Settings

Configure browser automation:

```
┌──────────────────────────────────────────────────────────────┐
│  Browser Settings                                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Enabled                 [✓ Enabled]                         │
│  Headless                [✗ Disabled]                        │
│  Port                    [9222             ]                 │
│  Timeout (seconds)       [30               ]                 │
│  Screenshot directory    [./screenshots    ]                 │
│                                                              │
│  Cookies:                                                    │
│    Profile                [default          ]                │
│    Persist cookies       [✓ Enabled]                         │
│                                                              │
│  [Test Browser Connection]                                   │
└──────────────────────────────────────────────────────────────┘
```

| Setting                  | Description               | Default         |
|--------------------------|---------------------------|-----------------|
| **Enabled**              | Enable browser automation | `true`          |
| **Headless**             | Run Chrome without UI     | `false`         |
| **Port**                 | Chrome debugging port     | `9222`          |
| **Timeout**              | Seconds before timeout    | `30`            |
| **Screenshot directory** | Where to save screenshots | `./screenshots` |

### Integrations

Configure authentication tokens for external task providers. Each provider section is collapsible and includes helpful guidance on obtaining tokens.

```
┌──────────────────────────────────────────────────────────────┐
│  Integrations                                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ▶ GitHub                              [Configured]          │
│  ▶ GitLab                                                    │
│  ▶ Jira                                [Configured]          │
│  ▶ Linear                                                    │
│  ▶ Notion                                                    │
│  ▶ Wrike                                                     │
│  ▶ YouTrack                                                  │
│  ▶ Bitbucket                                                 │
│  ▶ Asana                                                     │
│  ▶ ClickUp                                                   │
│  ▶ Trello                                                    │
│  ▶ Azure DevOps                                              │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

Click any provider to expand its configuration:

```
┌──────────────────────────────────────────────────────────────┐
│  ▼ GitHub                              [Configured]          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Personal Access Token                                       │
│  [••••••••••••••••••••••••]                                  │
│                                                              │
│  Get token → • Settings → Developer settings →               │
│  Personal access tokens • Required: repo, read:user          │
│                                                              │
│  Organization (optional)                                     │
│  [my-org                 ]                                   │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Token Help**: Each provider shows:
- **Get token →** - Link to the token generation page
- **Navigation steps** - How to find the token settings
- **Required scopes** - Permissions needed for the token

**Supported Providers:**

| Provider     | Token Type            | Additional Config     |
|--------------|-----------------------|-----------------------|
| GitHub       | Personal Access Token | Organization          |
| GitLab       | Personal Access Token | Instance URL          |
| Jira         | API Token             | Base URL, Email       |
| Linear       | API Key               | Team                  |
| Notion       | Integration Token     | Database ID           |
| Wrike        | Permanent Token       | Folder ID             |
| YouTrack     | Permanent Token       | Instance URL          |
| Bitbucket    | App Password          | Username              |
| Asana        | Personal Access Token | Workspace GID         |
| ClickUp      | API Token             | Team ID               |
| Trello       | API Key + Token       | (both required)       |
| Azure DevOps | Personal Access Token | Organization, Project |

### Specification Settings

Configure where specifications are saved:

```
┌──────────────────────────────────────────────────────────────┐
│  Specification Settings                                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Save in project         [✗ Disabled]                        │
│  Project directory       [tickets          ]                 │
│  Filename pattern        [specification-{n}.md]              │
│                                                              │
│  Pattern Variables:                                          │
│    {n}    - Specification number (1, 2, 3, ...)              │
│                                                              │
│  Example: "SPEC-{n}.md" → SPEC-1.md, SPEC-2.md, ...          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting               | Description                                                  | Default                |
|-----------------------|--------------------------------------------------------------|------------------------|
| **Save in project**   | Save specifications to project directory for version control | `false`                |
| **Project directory** | Directory name (e.g., "tickets")                             | `""` (home dir only)   |
| **Filename pattern**  | Template with `{n}` placeholder                              | `specification-{n}.md` |

When enabled, specifications are saved to both:
- Home directory (authoritative): `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/specifications/`
- Project directory (committable): `<project-dir>/<task-id>/`

### Review Settings

Configure where code reviews are saved:

```
┌──────────────────────────────────────────────────────────────┐
│  Review Settings                                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Save in project         [✗ Disabled]                        │
│  Filename pattern        [review-{n}.txt   ]                 │
│                                                              │
│  Example: "CODERABBIT-{n}.txt" → CODERABBIT-1.txt, ...       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting              | Description                                      | Default          |
|----------------------|--------------------------------------------------|------------------|
| **Save in project**  | Save reviews alongside specifications in project | `false`          |
| **Filename pattern** | Template with `{n}` placeholder                  | `review-{n}.txt` |

Reviews use the same project directory as specifications.

### Provider Settings

Configure default provider behavior:

```
┌──────────────────────────────────────────────────────────────┐
│  Provider Settings                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Default provider         [file             ]                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### License Information

View license and dependency information:

```
┌──────────────────────────────────────────────────────────────┐
│  License Information                                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Project License: BSD 3-Clause                               │
│  Copyright (c) 2025+, Dāvis Zālītis (k0d3r1s)                │
│  Copyright (c) 2025+, SIA Valksor                            │
│                                                              │
│  [View Full License Text]                                    │
│                                                              │
│  Dependency Licenses:                                        │
│    MIT                    12 packages                        │
│    BSD-3-Clause           8 packages                         │
│    BSD-style              5 packages                         │
│    Unknown                1 package                          │
│                                                              │
│  [Export Attribution] [Filter Unknown]                       │
│                                                              │
│  Open Source Licenses Used:                                  │
│    • google.golang.org/protobuf    BSD-3-Clause              │
│    • github.com/stretchr/testify    MIT                      │
│    • github.com/valksor/go-toolkit  BSD-3-Clause             │
│    • gopkg.in/yaml.v3               MIT                      │
│    • golang.org/x/crypto            BSD-style                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Features:**
- **View Project License** - Full BSD 3-Clause license text
- **Dependency List** - All Go module dependencies with licenses
- **SPRF Detection** - Standardized license identifiers (MIT, Apache-2.0, etc.)
- **Export Attribution** - Generate machine-readable attribution data
- **Filter Unknown** - Show only packages with undetectable licenses

**License Types:**

| Type       | Examples               |
|------------|------------------------|
| Permissive | MIT, BSD, Apache-2.0   |
| Reciprocal | GPL-2.0, MPL-2.0       |
| Restricted | AGPL-3.0, GPL-3.0      |
| Unknown    | Custom or undetectable |

When distributing software that includes Mehrhof, ensure you:
1. Preserve license notices
2. Acknowledge dependencies
3. Comply with reciprocal license terms

## Saving Changes

1. Make your desired changes
2. Click **"Save"** at the bottom of the page

Changes are written to `.mehrhof/config.yaml` and take effect immediately.

```
┌──────────────────────────────────────────────────────────────┐
│  ✓ Settings Saved                                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Configuration updated successfully.                         │
│                                                              │
│  File: .mehrhof/config.yaml                                  │
│                                                              │
│  [Continue]                                                  │
└──────────────────────────────────────────────────────────────┘
```

## Resetting Settings

Click **"Reset"** to discard unsaved changes and reload the current configuration.

## Global Mode Settings

When running in global mode (`mehr serve --global`), a project picker appears:

```
┌──────────────────────────────────────────────────────────────┐
│  Project: [my-app ▼]                                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Available projects:                                         │
│    • my-app (3 tasks)                                        │
│    • api-service (1 task)                                    │
│    • docs (0 tasks)                                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Security note:** In global mode, sensitive fields (API tokens, passwords) are hidden for security.

## Provider Health

Monitor connection status and rate limits for configured providers:

```
┌──────────────────────────────────────────────────────────────┐
│  Provider Health                                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  GitHub                                                      │
│    Status: ● Connected                                       │
│    Rate limit: 4872/5000 remaining                           │
│    Reset: 2025-01-15 16:00:00 UTC                            │
│    Last sync: 5 minutes ago                                  │
│                                                              │
│  Jira                                                        │
│    Status: ● Connected                                       │
│    Cloud: PROD.atlassian.net                                 │
│    Last sync: 1 hour ago                                     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Settings File

All settings are stored in `.mehrhof/config.yaml`:

```yaml
project:
  code_dir: ""  # Path to code target (relative or absolute); empty = hub is code target

git:
  auto_commit: true
  sign_commits: false
  target_branch: main
  branch_pattern: "{type}/{key}--{slug}"
  commit_prefix: "[{key}]"
  stash_on_start: false
  auto_pop_stash: true

agent:
  default: claude
  timeout: 300
  max_retries: 3
  steps:
    planning: { name: claude }
    implementing: { name: claude-sonnet }
    reviewing: { name: claude }

workflow:
  auto_init: true
  session_retention_days: 7
  delete_work_on_finish: false
  delete_work_on_abandon: false

specification:
  save_in_project: false
  project_dir: ""
  filename_pattern: "specification-{n}.md"

review:
  save_in_project: false
  filename_pattern: "review-{n}.txt"

browser:
  enabled: true
  headless: false
  port: 9222
  timeout: 30
  screenshot_dir: "./screenshots"
```

## Next Steps

- [**Dashboard**](dashboard.md) - Return to main view
- [**CLI: config**](/cli/config.md) - Manage config from CLI
- [**Configuration**](/configuration/index.md) - Advanced configuration guide
