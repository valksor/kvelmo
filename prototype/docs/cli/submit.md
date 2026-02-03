# mehr submit

Submit a queue task to an external provider.

## Synopsis

```bash
mehr submit --provider <name> [--task <queue>/<task-id> | --source <path/ref>] [flags]
```

## Description

The `submit` command creates a task in an external provider (GitHub, Jira, Wrike, etc.) from a queue task, or creates a queue task from a source and submits it in one step. The task is created in the external system with:

- Title and description from the queue task
- Labels applied
- Priority mapped to provider-specific values
- External ID and URL saved back to the queue task

This enables quick capture in mehrhof and submission to your project management system when ready.

## Flags

| Flag             | Short | Description                                                  |
|------------------|-------|--------------------------------------------------------------|
| `--task`         |       | Queue task ID (format: `<queue-id>/<task-id>`)               |
| `--source`       |       | Create a task from a file/dir/provider ref and submit        |
| `--provider`     |       | Provider name (github, jira, wrike, etc.) (required)         |
| `--labels`       |       | Additional labels to apply (can be specified multiple times) |
| `--note`         |       | Notes to guide task creation (repeatable)                    |
| `--title`        |       | Title override when creating from source                     |
| `--instructions` |       | Custom instructions for task creation                        |
| `--queue`        |       | Queue ID to store the created task (default: quick-tasks)    |
| `--optimize`     |       | Optimize the generated task before submitting                |
| `--dry-run`      |       | Preview without submitting                                   |

## Examples

### Basic Submission

```bash
mehr submit --task=quick-tasks/task-1 --provider github
```

Creates a GitHub issue from the task.

### With Additional Labels

```bash
mehr submit --task=quick-tasks/task-1 --provider wrike --labels urgent,bug
```

Submits to Wrike with additional labels.

### Dry Run Preview

```bash
mehr submit --task=quick-tasks/task-1 --provider github --dry-run
```

Shows what would be submitted without actually creating the issue.

### Custom Queue

```bash
mehr submit --task=backlog/task-5 --provider jira
```

### Submit From a Source File

```bash
mehr submit --provider github --source ./specs/overview.md --note "Scope to backend only"
```

Creates a quick task from the file and submits it in one step.

### Submit From a Directory

```bash
mehr submit --provider jira --source ./docs --instructions "Prefer a single actionable task" --optimize
```

Scans the directory, drafts a single task, optimizes it, and submits.

Submits a task from a custom queue.

## Supported Providers

| Provider     | Provider Name | Task Type      | Dependencies Support |
|--------------|---------------|----------------|----------------------|
| GitHub       | `github`      | Issues         | Task lists in epic   |
| GitLab       | `gitlab`      | Issues         | Task lists in epic   |
| Jira         | `jira`        | Issues         | Issue links          |
| Linear       | `linear`      | Issues         | Description-based    |
| Asana        | `asana`       | Tasks          | Native dependencies  |
| Notion       | `notion`      | Database items | Description-based    |
| Trello       | `trello`      | Cards          | Description-based    |
| Wrike        | `wrike`       | Tasks          | Native dependencies  |
| YouTrack     | `youtrack`    | Issues         | Description-based    |
| Bitbucket    | `bitbucket`   | Issues         | Description-based    |
| ClickUp      | `clickup`     | Tasks          | Native dependencies  |
| Azure DevOps | `azuredevops` | Work items     | Work item links      |

## What Happens

1. **Authentication Check**
   - Verifies you're logged in to the provider
   - Prompts for login if needed

2. **Task Creation**
   - Creates task in external provider
   - Maps mehrhof labels to provider labels
   - Maps priority to provider values

3. **Metadata Update**
   - Saves external ID to queue task
   - Saves external URL to queue task
   - Updates submission status

4. **Result Display**
   - Shows external ID
   - Shows URL to created task
   - Shows any epic/folder created (if applicable)

## Output Example

```
📤 Submitting to github
  Task: quick-tasks/task-1
  Labels: urgent

  ✓ Submitted:
    Local ID: task-1
    External ID: valksor/go-mehrhof#123
    URL: https://github.com/valksor/go-mehrhof/issues/123
```

### Dry Run Output

```
📤 Dry-run: Previewing submission to github
  Task: quick-tasks/task-1
  Labels: urgent

  Dry-run preview:
    Task ID: task-1
    Title: Fix typo in README
    Description: The word "Installation" is misspelled...

  Remove --dry-run to actually submit.
```

## Workflow Examples

### Capture and Submit

```bash
# Capture quickly
mehr quick "API returns 500 on empty user list"

# Add details
mehr note --task=quick-tasks/task-1 "nil pointer in User.FindAll"
mehr note --task=quick-tasks/task-1 "stack trace included"

# Submit to GitHub
mehr submit --task=quick-tasks/task-1 --provider github --labels bug,critical
```

### Source → Submit (Single Command)

```bash
mehr submit --provider github --source ./request.md --note "Include acceptance criteria"
```

### Batch Submission

```bash
# Capture multiple tasks
mehr quick "add user profile page"
mehr quick "implement password reset"
mehr quick "add email notifications"

# Submit them all
mehr submit --task=quick-tasks/task-1 --provider github
mehr submit --task=quick-tasks/task-2 --provider github
mehr submit --task=quick-tasks/task-3 --provider github
```

### Preview Before Submit

```bash
# Preview what will be created
mehr submit --task=quick-tasks/task-1 --provider jira --dry-run

# If satisfied, submit for real
mehr submit --task=quick-tasks/task-1 --provider jira
```

## Provider Authentication

Most providers require authentication before submission:

```bash
# GitHub
mehr github login

# Jira
mehr jira login

# Wrike
mehr wrike login
```

See [provider documentation](/providers/index.md) for provider-specific setup.

## Label Mapping

Labels are mapped as follows:

| Mehrhof Labels  | Provider Labels                     |
|-----------------|-------------------------------------|
| Task labels     | Provider labels (created if needed) |
| `--labels` flag | Additional labels                   |

## Priority Mapping

Priority is mapped to provider-specific values:

| Mehrhof Priority | GitHub    | Jira    | Wrike  |
|------------------|-----------|---------|--------|
| 1 (High)         | High/P1   | Highest | High   |
| 2 (Normal)       | Medium/P2 | High    | Medium |
| 3 (Low)          | Low/P3    | Medium  | Low    |

## Re-submission

Submitting a task that was already submitted will:

- Update the existing external task
- Add a comment if `--comment` is provided
- Not create a duplicate

## Web UI

Prefer a visual interface? See the Submit feature in [Quick Tasks](/web-ui/quick-tasks.md).

## See Also

- [quick](quick.md) - Create quick tasks
- [optimize](optimize.md) - AI optimize before submitting
- [export](export.md) - Export to file instead
- [login](login.md) - Provider authentication
- [Providers](/providers/index.md) - Provider documentation
