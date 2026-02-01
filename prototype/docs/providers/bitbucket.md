# Bitbucket Provider

> **⚠️ Third-Party Integration**: This integration depends on external APIs that may change. Not fully tested beyond unit tests. Behavior may vary depending on the third-party service. Manual validation recommended before production use.


**Schemes:** `bitbucket:`, `bb:`

**Capabilities:** `read`, `list`, `fetch_comments`, `comment`, `update_status`, `create_work_unit`, `create_pr`, `snapshot`, `fetch_subtasks`, `fetch_pr`, `pr_comment`, `fetch_pr_comments`, `update_pr_comment`

Integrates with Bitbucket Cloud for issue tracking and pull request management.

## Usage

```bash
mehr start bitbucket:123
mehr plan bb:123

mehr start bb:workspace/repo#456

mehr start bb:https://bitbucket.org/workspace/repo/issues/123
```

## Configuration

```yaml
bitbucket:
  username: "${BITBUCKET_USERNAME}"       # Bitbucket username
  app_password: "${BITBUCKET_APP_PASSWORD}" # App password (not account password)
  workspace: "myworkspace"
  repo: "myrepo"
  branch_pattern: "issue/{key}-{slug}"
  commit_prefix: "[#{key}]"               # Commit message prefix
  target_branch: "main"                   # Auto-detected from repository if not set
  close_source_branch: false              # Delete source branch when PR is merged
```

> **Note:** If `target_branch` is not configured, Mehrhof will query the Bitbucket API for the repository's default branch. If the repository has no default branch configured, an explicit `target_branch` is required.

## Credential Resolution

1. `MEHR_BITBUCKET_USERNAME` / `MEHR_BITBUCKET_APP_PASSWORD` environment variables
2. `BITBUCKET_USERNAME` / `BITBUCKET_APP_PASSWORD` environment variables
3. Credentials from `config.yaml`

## Authentication

Create an App Password at: https://bitbucket.org/account/settings/app-passwords/

Required permissions: `Issues: Read, Write`, `Repositories: Read, Write`, `Pull requests: Write`.

## Reference Formats

| Format                 | Example                                              |
|------------------------|------------------------------------------------------|
| Scheme with issue ID   | `bitbucket:123`                                      |
| Short scheme           | `bb:456`                                             |
| Workspace/repo with ID | `bb:workspace/repo#123`                              |
| Issue URL              | `bb:https://bitbucket.org/workspace/repo/issues/123` |

## Features

- **Issue Fetching**: Retrieves title, content, state, priority, assignee
- **List Issues**: Browse repository issues with status filtering
- **Comment Support**: Fetch and add comments
- **State Updates**: Change issue state
- **PR Creation**: Create pull requests
- **PR Review**: Fetch PR details, diff, comments; add and update PR comments
- **Snapshots**: Export issue content as markdown

## Status Mapping

| Bitbucket State                     | Provider Status |
|-------------------------------------|-----------------|
| new, open                           | Open            |
| on hold                             | In Progress     |
| resolved                            | Done            |
| closed, invalid, duplicate, wontfix | Closed          |

## Priority Mapping

| Bitbucket Priority | Provider Priority |
|--------------------|-------------------|
| critical           | Critical          |
| major              | High              |
| minor              | Normal            |
| trivial            | Low               |
