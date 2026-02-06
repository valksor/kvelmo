# Working Within Company Guidelines

Mehrhof is designed for enterprise settings where changes need oversight, auditability, and alignment with internal policies. This guide explains how to configure Mehrhof to match your organization's requirements.

---

## Overview

Every company has different standards for how changes are reviewed, approved, and tracked. Mehrhof provides configuration options to enforce these standards consistently:

- **Review requirements** — Ensure all changes go through quality checks
- **Approval workflows** — Require human sign-off before merging
- **Audit trails** — Track who created what and when
- **Guardrails** — Limit what can be created or modified
- **Integration** — Connect with existing tools (Git, issue trackers)

---

## Setting Up Review Requirements

### Require Review Before Finish

Ensure the review step is always completed before changes can be merged:

```yaml
# .mehrhof/config.yaml
workflow:
  require_review: true
```

With this setting, attempting to finish a task without running review will fail.

### Custom Quality Checks

Configure which checks run during the review phase:

```yaml
workflow:
  quality_checks:
    - lint
    - test
    - security
```

---

## Configuring Approval Workflows

### Branch Protection

Mehrhof integrates with Git branch protection rules. Configure your repository to require:
- Pull request reviews before merging
- Status checks to pass
- Signed commits

### Manual Approval Steps

For sensitive changes, you can configure required manual approval:

```yaml
workflow:
  require_manual_approval: true
  approvers:
    - team-lead
    - security-reviewer
```

---

## Audit Trails and Logging

### What Mehrhof Tracks

Every task automatically records:
- Who created the task and when
- What was requested (original task description)
- What plan was generated
- What changes were made
- When each workflow step occurred
- Any notes or adjustments made during the process

### Accessing Audit Information

```bash
# View task history with timestamps
mehr list --format detailed

# Export task history for compliance
mehr export --task <id> --format json
```

### Saving Plans in Version Control

Store plans alongside your code for full traceability:

```yaml
specification:
  save_in_project: true
  project_dir: "plans"
  filename_pattern: "PLAN-{n}.md"

review:
  save_in_project: true
  filename_pattern: "review-{n}.txt"
```

---

## Restricting What Can Be Created

### File Path Restrictions

Limit which files can be modified:

```yaml
# TODO: Configuration format TBD
restrictions:
  allowed_paths:
    - "src/**"
    - "docs/**"
  blocked_paths:
    - ".env*"
    - "secrets/**"
```

### Sensitive File Protection

Prevent modifications to critical files:

```yaml
restrictions:
  protected_files:
    - "package-lock.json"
    - "go.sum"
    - ".github/workflows/**"
```

---

## Integration with Existing Tools

### Git Configuration

```yaml
git:
  auto_commit: true
  commit_prefix: "[{key}]"
  branch_pattern: "{type}/{key}--{slug}"
  target_branch: "main"
  signed_commits: true
```

### Issue Tracker Integration

Connect to your existing providers:

```yaml
providers:
  default: jira
  jira:
    url: "https://yourcompany.atlassian.net"
    project: "PROJ"
```

Supported integrations: GitHub, GitLab, Bitbucket, Jira, Linear, Asana, ClickUp, Azure DevOps, Notion, Trello, Wrike, YouTrack.

---

## Team Configuration

### Shared Configuration

Store configuration in version control so all team members use the same settings:

```
project/
├── .mehrhof/
│   ├── config.yaml      # Shared team settings
│   └── .env             # Local secrets (gitignored)
```

### Per-User Overrides

Users can override team settings locally:

```bash
# ~/.valksor/mehrhof/config.yaml
# Personal preferences that don't affect the team
```

---

## Security Considerations

### Credential Management

- Never store credentials in `config.yaml`
- Use `.env` files (gitignored) for API tokens
- Use environment variables in CI/CD

<!-- DISABLED: remote serve temporarily unavailable
### Access Control

- Configure authentication for remote Web UI access
- Use SSH tunnels for secure remote access
- Restrict who can run certain commands

See [Authentication](/web-ui/authentication.md) for Web UI access control.
-->

---

## Next Steps

- [Configuration Reference](/configuration/index.md) — All configuration options
- [Providers](/providers/index.md) — Connect to issue trackers
- [Web UI: Settings](/web-ui/settings.md) — Configure via browser
