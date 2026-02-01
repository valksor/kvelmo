# Automation (Web UI)

Monitor and manage automated GitHub/GitLab webhook processing through the Web UI.

## Overview

When running `mehr serve` with automation enabled, the dashboard provides real-time visibility into:

- Webhook job queue status
- Active and historical jobs
- Job details and logs
- Manual job cancellation

> **Warning: Remote Server Required**
>
> Automation only works when the server is publicly accessible from the internet. GitHub/GitLab webhooks cannot reach localhost or private networks. You must deploy to a public server, cloud instance, or use a tunneling service (ngrok, cloudflared) for development.

## Accessing Automation Status

### Dashboard Widget

The main dashboard displays an automation status widget showing:

- **Enabled/Disabled** - Current automation state
- **Workers** - Number of concurrent job processors
- **Queue depth** - Pending jobs waiting to process
- **Running** - Currently executing jobs

### Automation Page

Navigate to **Automation** in the sidebar for detailed management:

```
/automation
```

## Job Queue

### Viewing Jobs

The job queue shows all automation jobs with:

| Column    | Description                                      |
|-----------|--------------------------------------------------|
| Status    | pending, running, completed, failed, cancelled   |
| Provider  | github or gitlab                                 |
| Type      | issue_fix, pr_review, or command                 |
| Reference | Issue/PR number and repository                   |
| Created   | When the webhook was received                    |
| Duration  | Time elapsed (running) or total time (completed) |

### Filtering Jobs

Filter the job list by:

- **Status** - Show only pending, running, completed, failed, or cancelled
- **Provider** - Filter by GitHub or GitLab
- **Type** - Filter by workflow type

### Job Details

Click any job to view:

- Full webhook event payload
- Workflow execution logs
- Error details (if failed)
- Result information (PR URL, comments posted)

## Managing Jobs

### Cancel a Job

To stop a pending or running job:

1. Click the job row to expand details
2. Click **Cancel Job**
3. Confirm cancellation

Cancelled jobs are marked with `cancelled` status and cannot be resumed.

### Retry Failed Jobs

Failed jobs can be manually retried:

1. Click the failed job row
2. Review the error message
3. Click **Retry Job**

The job is re-queued with attempt count preserved.

## Real-time Updates

The automation page uses Server-Sent Events (SSE) for live updates:

- New jobs appear instantly when webhooks arrive
- Job status changes are reflected immediately
- Queue statistics update in real-time

No manual refresh needed.

## Configuration

Automation settings are displayed read-only in the UI. To modify:

1. Edit `.mehrhof/config.yaml`
2. Restart the server

See [docs/cli/automation.md](../cli/automation.md) for configuration options.

## API Access

The Web UI consumes the same REST API available for programmatic access:

| Endpoint                                   | Description  |
|--------------------------------------------|--------------|
| `GET /api/v1/automation/status`            | Queue status |
| `GET /api/v1/automation/jobs`              | List jobs    |
| `GET /api/v1/automation/jobs/{id}`         | Job details  |
| `POST /api/v1/automation/jobs/{id}/cancel` | Cancel job   |

## Workflow Status Integration

When automation creates tasks or PRs, they appear in the standard workflow views:

- **Dashboard** - Active tasks show automation origin
- **Task List** - Filter by `mehrhof-generated` label
- **History** - Completed automation jobs in history

## Troubleshooting

### No jobs appearing

1. Verify webhooks are reaching the server (check provider delivery logs)
2. Confirm automation is enabled: check status widget
3. Review access control settings - user/org may be blocked

### Jobs stuck in pending

1. Check if workers are available (workers > running jobs)
2. Review queue configuration for timeouts
3. Check for agent availability

### Jobs failing repeatedly

1. Click the job to view error details
2. Common issues:
   - Agent unavailable or rate limited
   - Repository access denied
   - Invalid issue/PR content
3. Fix the underlying issue, then retry

## Labels in GitHub/GitLab

Automation applies labels to track state:

| Label                | Meaning                      |
|----------------------|------------------------------|
| `mehrhof-processing` | Job is currently running     |
| `mehrhof-generated`  | PR was created by automation |
| `mehrhof-failed`     | Job failed after all retries |

These labels are visible in both the provider UI and Mehrhof Web UI.

## See Also

- [docs/cli/automation.md](../cli/automation.md) - CLI configuration and setup
- [docs/web-ui/dashboard.md](dashboard.md) - Dashboard overview
- [docs/web-ui/api.md](api.md) - REST API reference
