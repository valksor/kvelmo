# Settings

Configure your workspace through the Web UI settings page without editing YAML files directly.

## Accessing Settings

Click the **Settings** icon (gear) in the top-right of the dashboard, or navigate to `/settings`.

```
┌──────────────────────────────────────────────────────────────┐
│  Workspace Settings                                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [Git] [Agent] [Workflow] [Browser] [Providers]              │
│  ─────────────────────────────────────────────────────────  │
│                                                              │
│  Settings apply to: current-project                          │
│                                                              │
│                                        [Reset] [Save]       │
└──────────────────────────────────────────────────────────────┘
```

## Settings Sections

### Git Settings

Configure version control integration:

```
┌──────────────────────────────────────────────────────────────┐
│  Git Settings                                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Auto-commit               [✓ Enabled]                       │
│  Sign commits             [✗ Disabled]                      │
│  Target branch            [main              ]                │
│  Branch pattern           [{type}/{key}--{slug}]             │
│  Commit prefix            [{key}]                            │
│  Stash on start           [✗ Disabled]                       │
│  Auto-pop stash           [✓ Enabled]                        │
│                                                              │
│  Branch Pattern Variables:                                    │
│    {type}    - Task type (feature, fix, etc.)               │
│    {key}     - External key (JIRA-123, etc.)                │
│    {slug}    - URL-safe title slug                           │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting | Description | Default |
|----------|-------------|---------|
| **Auto-commit** | Automatically commit changes | `true` |
| **Sign commits** | GPG sign commits | `false` |
| **Target branch** | Branch to merge into | `main` |
| **Branch pattern** | Template for branch names | `{type}/{key}--{slug}` |
| **Commit prefix** | Template for commit messages | `[{key}]` |
| **Stash on start** | Auto-stash before creating branch | `false` |
| **Auto-pop stash** | Auto-pop stash after branch creation | `true` |

### Agent Settings

Configure AI agent behavior:

```
┌──────────────────────────────────────────────────────────────┐
│  Agent Settings                                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Default agent            [claude            ]                │
│  Timeout (seconds)       [300              ]                │
│  Max retries             [3                ]                │
│                                                              │
│  Step-specific agents:                                       │
│    Planning:           [claude           ]                   │
│    Implementing:       [claude-sonnet    ]                   │
│    Reviewing:          [claude           ]                   │
│                                                              │
│  Available agents:                                           │
│    • claude - Claude AI via Claude CLI                       │
│    • claude-opus - Claude 3 Opus (slower, smarter)          │
│    • claude-sonnet - Claude 3.5 Sonnet (faster)             │
│                                                              │
│  [Add Custom Agent Alias]                                    │
└──────────────────────────────────────────────────────────────┘
```

| Setting | Description | Default |
|----------|-------------|---------|
| **Default agent** | Agent to use for all steps | `claude` |
| **Timeout** | Maximum seconds per operation | `300` |
| **Max retries** | Retry attempts on failure | `3` |

### Workflow Settings

Configure task lifecycle behavior:

```
┌──────────────────────────────────────────────────────────────┐
│  Workflow Settings                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Auto-init               [✓ Enabled]                        │
│  Session retention       [7                ] days            │
│  Delete work on finish   [✗ Disabled]                       │
│  Delete work on abandon  [✗ Disabled]                       │
│                                                              │
│  Quality checks on finish:                                   │
│    Run tests             [✓ Enabled]                        │
│    Run quality           [✓ Enabled]                        │
│    Fail on error         [✓ Enabled]                        │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

| Setting | Description | Default |
|----------|-------------|---------|
| **Auto-init** | Auto-initialize new workspaces | `true` |
| **Session retention** | Days to keep session history | `7` |
| **Delete work on finish** | Clean up after successful finish | `false` |
| **Delete work on abandon** | Clean up after abandon | `false` |

### Browser Settings

Configure browser automation:

```
┌──────────────────────────────────────────────────────────────┐
│  Browser Settings                                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Enabled                 [✓ Enabled]                        │
│  Headless                [✗ Disabled]                       │
│  Port                    [9222             ]                │
│  Timeout (seconds)       [30               ]                │
│  Screenshot directory    [./screenshots    ]                │
│                                                              │
│  Cookies:                                                    │
│    Profile                [default          ]                │
│    Persist cookies       [✓ Enabled]                        │
│                                                              │
│  [Test Browser Connection]                                   │
└──────────────────────────────────────────────────────────────┘
```

| Setting | Description | Default |
|----------|-------------|---------|
| **Enabled** | Enable browser automation | `true` |
| **Headless** | Run Chrome without UI | `false` |
| **Port** | Chrome debugging port | `9222` |
| **Timeout** | Seconds before timeout | `30` |
| **Screenshot directory** | Where to save screenshots | `./screenshots` |

### Provider Settings

Configure external task provider connections:

```
┌──────────────────────────────────────────────────────────────┐
│  Provider Settings                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Default provider         [file             ]                │
│                                                              │
│  Configured Providers:                                      │
│    • GitHub        ✓ Connected     [Configure]  [Test]     │
│    • GitLab        ✗ Not setup    [Configure]              │
│    • Jira          ✓ Connected     [Configure]  [Test]     │
│    • Linear        ✗ Not setup    [Configure]              │
│                                                              │
│  [Add Provider Configuration]                                │
└──────────────────────────────────────────────────────────────┘
```

Click **"Configure"** to set up provider credentials.

## Saving Changes

1. Make your desired changes
2. Click **"Save"** at the bottom of the page

Changes are written to `.mehrhof/config.yaml` and take effect immediately.

```
┌──────────────────────────────────────────────────────────────┐
│  ✓ Settings Saved                                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Configuration updated successfully.                          │
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
│    Status: ● Connected                                      │
│    Rate limit: 4872/5000 remaining                            │
│    Reset: 2025-01-15 16:00:00 UTC                           │
│    Last sync: 5 minutes ago                                  │
│                                                              │
│  Jira                                                        │
│    Status: ● Connected                                      │
│    Cloud: PROD.atlassian.net                                │
│    Last sync: 1 hour ago                                    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Settings File

All settings are stored in `.mehrhof/config.yaml`:

```yaml
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

browser:
  enabled: true
  headless: false
  port: 9222
  timeout: 30
  screenshot_dir: "./screenshots"
```

## Next Steps

- [**Dashboard**](dashboard.md) - Return to main view
- [**CLI: config**](../cli/config.md) - Manage config from CLI
- [**Configuration**](../configuration/index.md) - Advanced configuration guide
