# mehr serve --api (Automation)

Automated issue/PR handling via GitHub and GitLab webhooks.

## Synopsis

```bash
mehr serve --api [flags]
```

## Description

When running `mehr serve --api`, Mehrhof can receive webhooks from GitHub and GitLab too automatically:

- **Fix issues** - Automatically runs plan → implement → review → finish → PR/MR
- **Review external PRs/MRs** - Posts review comments with findings
- **Respond to comment commands** - Triggers workflows via `@mehrhof fix`, `@mehrhof review`, etc.

This enables CI/CD-style automation where issues get automatically fixed and PRs get automatically reviewed.

> **Warning: Remote Server Required**
>
> Automation only works when the server is publicly accessible from the internet. GitHub/GitLab webhooks cannot reach localhost or private networks. You must deploy to a public server, cloud instance, or use a tunneling service (ngrok, cloudflared) for development.

## Configuration

Automation is configured in `.mehrhof/config.yaml`:

```yaml
automation:
  enabled: true
  providers:
    github:
      enabled: true
      webhook_secret: "${MEHR_GITHUB_WEBHOOK_SECRET}"
      command_prefix: "@mehrhof"
      use_worktrees: true
      trigger_on:
        issue_opened: true
        issue_labeled: ["mehr-fix", "automate"]
        pr_opened: true
        comment_commands: true
    gitlab:
      enabled: true
      webhook_secret: "${MEHR_GITLAB_WEBHOOK_SECRET}"
      command_prefix: "@mehrhof"
      use_worktrees: true
      trigger_on:
        issue_opened: true
        issue_labeled: ["mehr-fix"]
        mr_opened: true
        comment_commands: true
  access_control:
    mode: allowlist           # "allowlist", "blocklist", or "all"
    allowlist:
      - "myorg"
      - "trusted-user"
      - "myorg/*"             # Wildcard patterns supported
    blocklist:
      - "known-spammer"
    allow_bots: false         # Block bot accounts by default
    require_org: false        # Require user to be org member
  queue:
    max_concurrent: 1         # Parallel job limit
    job_timeout: 30m
    max_attempts: 3           # Retry failed jobs
  labels:
    mehr_generated: "mehrhof-generated"
    in_progress: "mehrhof-processing"
    failed: "mehrhof-failed"
```

### Environment Variables

Store secrets in environment variables:

```bash
export MEHR_GITHUB_WEBHOOK_SECRET="your-github-webhook-secret"
export MEHR_GITLAB_WEBHOOK_SECRET="your-gitlab-webhook-secret"
```

## Webhook Limits

| Limit | Value | Description |
|-------|-------|-------------|
| Max payload size | 10 MB | Webhook requests exceeding this size are rejected with HTTP 413 |
| Rate limiting | Per-IP | When auth is enabled, standard rate limits apply to webhook endpoints |

Webhook endpoints use provider-specific authentication (webhook secrets) and are exempt from CSRF validation.

## Webhook Setup

### GitHub

1. Go to repository **Settings → Webhooks → Add webhook**
2. Set **Payload URL** to `https://your-server/api/v1/webhooks/github`
3. Set **Content type** to `application/json`
4. Set **Secret** to your `MEHR_GITHUB_WEBHOOK_SECRET`
5. Select events:
   - Issues (opened, labeled)
   - Pull requests (opened, synchronize)
   - Issue comments
   - Pull request reviews

### GitLab

1. Go to project **Settings → Webhooks**
2. Set **URL** to `https://your-server/api/v1/webhooks/gitlab`
3. Set **Secret token** to your `MEHR_GITLAB_WEBHOOK_SECRET`
4. Select triggers:
   - Issues events
   - Merge request events
   - Comments
   - Note events

## API Endpoints

| Method | Path                                  | Description                                   |
|--------|---------------------------------------|-----------------------------------------------|
| `POST` | `/api/v1/webhooks/github`             | Receive GitHub webhooks                       |
| `POST` | `/api/v1/webhooks/gitlab`             | Receive GitLab webhooks                       |
| `GET`  | `/api/v1/automation/status`           | Queue status (enabled, workers, job counts)   |
| `GET`  | `/api/v1/automation/jobs`             | List all jobs (filter with `?status=pending`) |
| `GET`  | `/api/v1/automation/jobs/{id}`        | Get specific job details                      |
| `POST` | `/api/v1/automation/jobs/{id}/cancel` | Cancel a pending/running job                  |

## Comment Commands

Users can trigger workflows by commenting on issues or PRs:

| Command           | Description                | Context      |
|-------------------|----------------------------|--------------|
| `@mehrhof fix`    | Trigger issue fix workflow | Issue        |
| `@mehrhof review` | Trigger PR review          | Pull request |
| `@mehrhof status` | Report queue status        | Any          |
| `@mehrhof help`   | List available commands    | Any          |

### Examples

```markdown
<!-- On an issue -->
@mehrhof fix

<!-- On a PR -->
@mehrhof review --detailed

<!-- Check queue status -->
@mehrhof status
```

## Workflow Behavior

### Issue Fix Workflow

When an issue is opened (or labeled with a trigger label):

1. Job is queued with `mehrhof-processing` label added
2. Worktree is created for isolation (if `use_worktrees: true`)
3. Workflow executes: `start → plan → implement → review → finish`
4. **On success**: PR created with `mehrhof-generated` label, processing label removed
5. **On failure**: Comment posted with error details, `mehrhof-failed` label added

The plan step acts as triage - invalid or unclear issues are detected here and reported.

### PR Review Workflow

When a PR/MR is opened (excluding those with `mehrhof-generated` label):

1. Job is queued
2. Code is fetched and analyzed
3. Review comments are posted:
   - **GitHub**: Formal review submission (APPROVED/REQUEST_CHANGES)
   - **GitLab**: MR notes (no formal review API)
4. Results are summarized in a comment

### Retry Behavior

Failed jobs are automatically retried up to `max_attempts` times with exponential backoff. Final failures:

- Post detailed error comment
- Add `mehrhof-failed` label
- Stop processing (no infinite loops)

## Access Control

### Modes

| Mode        | Behavior                              |
|-------------|---------------------------------------|
| `all`       | Allow everyone (subject to blocklist) |
| `allowlist` | Only allow listed users/orgs          |
| `blocklist` | Allow everyone except blocked         |

### Pattern Matching

```yaml
allowlist:
  - "myorg"           # Exact match
  - "trusted-*"       # Prefix wildcard
  - "*-bot"           # Suffix wildcard
  - "myorg/*"         # All repos in org
```

### Bot Filtering

By default, bot accounts are blocked to prevent infinite loops. Enable with:

```yaml
access_control:
  allow_bots: true
```

## Labels

| Label                | Purpose                                      |
|----------------------|----------------------------------------------|
| `mehrhof-generated`  | PRs created by automation (skip self-review) |
| `mehrhof-processing` | Job currently in progress                    |
| `mehrhof-failed`     | Job failed after all retries                 |

Labels are customizable in config. Set to empty string to disable:

```yaml
labels:
  in_progress: ""  # Disable progress label
```

## Examples

### Minimal Setup (GitHub only)

```yaml
automation:
  enabled: true
  providers:
    github:
      enabled: true
      webhook_secret: "${MEHR_GITHUB_WEBHOOK_SECRET}"
      trigger_on:
        issue_opened: true
        pr_opened: true
```

### Production Setup

```yaml
automation:
  enabled: true
  providers:
    github:
      enabled: true
      webhook_secret: "${MEHR_GITHUB_WEBHOOK_SECRET}"
      command_prefix: "@mehrhof"
      use_worktrees: true
      trigger_on:
        issue_opened: false          # Only via labels
        issue_labeled: ["mehr-fix"]  # Explicit opt-in
        pr_opened: true
        comment_commands: true
    gitlab:
      enabled: true
      webhook_secret: "${MEHR_GITLAB_WEBHOOK_SECRET}"
      command_prefix: "@mehrhof"
      use_worktrees: true
      trigger_on:
        issue_opened: false
        issue_labeled: ["mehr-fix"]
        mr_opened: true
        comment_commands: true
  access_control:
    mode: allowlist
    allowlist:
      - "mycompany"
      - "mycompany/*"
    allow_bots: false
  queue:
    max_concurrent: 2
    job_timeout: 45m
    max_attempts: 3
```

## Troubleshooting

### Webhooks not received

1. Check webhook secret matches configuration
2. Verify server is accessible from provider
3. Check provider webhook delivery logs
4. Ensure correct events are selected

### Jobs not processing

1. Check `automation.enabled: true`
2. Verify provider is enabled
3. Check access control allows the user/org
4. Review `GET /api/v1/automation/status`

### PRs not auto-reviewed

PRs with the `mehrhof-generated` label are automatically skipped to prevent self-review loops.

## Web UI

Prefer a visual interface? See [Web UI: Automation](../web-ui/automation.md).

## See Also

- [docs/cli/serve.md](serve.md) - Server configuration
- [docs/concepts/workflows.md](../concepts/workflows.md) - Workflow architecture
