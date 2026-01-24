# mehr review

Run automated code review on current changes.

## Synopsis

```bash
mehr review [flags]
```

## Description

The `review` command runs an automated code review on the task's changes. The review process includes:

### Automated Linting

Before the AI review, Mehrhof automatically detects and runs appropriate linters based on your project:

| Project Type | Linter | Detection |
|--------------|--------|-----------|
| Go | golangci-lint | `go.mod` present |
| JavaScript/TypeScript | ESLint | `package.json` present |
| Python | Ruff | `pyproject.toml` or `requirements.txt` present |
| PHP | php-cs-fixer | `composer.json` or `.php-cs-fixer.php` present |

Lint results are included in the AI agent's review context, allowing it to address both lint issues and higher-level code quality concerns.

### AI Code Review

After linting, the AI agent analyzes:

- Code quality issues
- Potential bugs
- Style violations
- Security concerns
- Lint issues found by automated linters

Review results are saved to the work directory.

## Flags

| Flag       | Short | Type   | Default      | Description        |
| ---------- | ----- | ------ | ------------ | ------------------ |
| `--tool`   |       | string | coderabbit   | Review tool to use |
| `--output` | `-o`  | string | REVIEW-N.txt | Output file name   |
| `--optimize` |     | bool   | false        | Optimize prompt before sending to agent |

## Examples

### Basic Review

```bash
mehr review
```

Output:

```
Running code review...
Tool: CodeRabbit
Analyzing 5 changed files...

Review Status: ISSUES
Found 3 issues:
  - src/api/handler.go:45 - Missing error check
  - src/api/auth.go:23 - Potential nil pointer
  - src/api/routes.go:12 - Unused import

Review saved to: ~/.valksor/mehrhof/workspaces/<project-id>/work/a1b2c3d4/reviews/REVIEW-1.txt
```

### Custom Output File

```bash
mehr review --output security-review.txt
```

### Specify Tool

```bash
mehr review --tool coderabbit
```

### Optimize Prompt

```bash
mehr review --optimize
```

Optimize the review prompt using an optimizer agent before sending to the working agent. This can improve the quality and depth of the code review.

## Review Status

| Status   | Meaning                          |
| -------- | -------------------------------- |
| COMPLETE | No issues found                  |
| ISSUES   | Issues found that need attention |
| ERROR    | Review tool failed to run        |

## Review Output

Reviews are saved as text files:

```
Code Review Report
==================
Tool: CodeRabbit
Date: 2025-01-15 10:30:00

Files Analyzed:
- src/api/handler.go
- src/api/auth.go
- src/api/routes.go

Issues Found:
-------------

[HIGH] src/api/handler.go:45
Missing error check on database query.
Suggestion: Handle the error case.

[MEDIUM] src/api/auth.go:23
Potential nil pointer dereference.
Suggestion: Add nil check before access.

[LOW] src/api/routes.go:12
Unused import "fmt".
Suggestion: Remove unused import.
```

## When to Review

### After Implementation

```bash
mehr implement
mehr review
```

### Before Finishing

```bash
mehr review
mehr finish
```

## Acting on Review

### Fix Issues

After review, address issues and re-implement:

```bash
mehr review
mehr note "Fix the error handling in handler.go"
mehr implement
mehr review
```

### Ignore Issues

If issues are false positives or acceptable:

```bash
mehr review
mehr finish
```

## Review History

Multiple reviews are saved with incremental names:

```
~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/reviews/
├── REVIEW-1.txt    # First review
├── REVIEW-2.txt    # After fixes
└── REVIEW-3.txt    # Final review
```

## Requirements

### CodeRabbit

Ensure CodeRabbit CLI is installed and configured:

```bash

coderabbit --version
```

## Troubleshooting

### "Review tool not found"

Install the required review tool:

```bash
npm install -g @coderabbit/cli
```

### "No changes to review"

Ensure you have uncommitted changes or are on a task branch with modifications.

### "Review timeout"

Large changesets may take longer. The tool will retry automatically.

## Review Pull Requests

### mehr review pr

Review a pull request (GitHub) or merge request (GitLab) using AI agents. This is a **standalone command** that does not require an active task or workspace.

```bash
mehr review pr --pr-number <N> [flags]
```

**Use when:** You want to review a PR/MR without starting a mehrhof task. Ideal for CI/CD pipelines.

#### Provider Detection

The provider is **auto-detected from your git remote URL**:
- GitHub → `github.com`
- GitLab → `gitlab.com`
- Bitbucket → `bitbucket.org`
- Azure DevOps → `dev.azure.com`, `azure.com`

Use `--provider` to override auto-detection.

#### Flags

| Flag                | Description                                                            |
| ------------------- | ---------------------------------------------------------------------- |
| `--provider`        | Provider: `github`, `gitlab`, `bitbucket`, `azuredevops` (auto-detected) |
| `--pr-number`       | PR/MR number (required)                                                |
| `--format`          | Comment format: `summary` (default), `line-comments`                   |
| `--scope`           | Review scope: `full` (default), `compact`, `files-changed`             |
| `--agent`           | Agent to use (default: `claude`)                                       |
| `--token`           | Auth token (overrides config/env vars; use for CI)                     |
| `--acknowledge-fixes` | Acknowledge when previously reported issues are fixed (default: true)  |
| `--update-existing` | Edit existing comment vs post new comment (default: true)               |

#### Examples

**Basic PR review:**
```bash
mehr review pr --pr-number 123
```

**Specify provider:**
```bash
mehr review pr --pr-number 456 --provider gitlab
```

**CI/CD with token:**
```bash
mehr review pr --pr-number 789 --token "$GITHUB_TOKEN"
```

**Compact review scope:**
```bash
mehr review pr --pr-number 100 --scope compact
```

**Use specific agent:**
```bash
mehr review pr --pr-number 200 --agent claude-opus
```

#### Output

```bash
$ mehr review pr --pr-number 123

Reviewing PR #123 from github.com/user/repo...
Agent: claude
Scope: full
Format: summary

✅ Review completed for PR #123
   Provider: github
   Agent: claude
   Comments posted: 3
   URL: https://github.com/user/repo/pull/123#issuecomment-456
```

#### Formats

| Format        | Description                                          |
| ------------- | ---------------------------------------------------- |
| `summary`     | Single summary comment with all findings             |
| `line-comments` | Individual comments on specific lines of code       |

#### Scopes

| Scope          | Description                                          |
| -------------- | ---------------------------------------------------- |
| `full`         | Review all changes in detail                         |
| `compact`      | Concise review focusing on critical issues            |
| `files-changed` | Summary of modified files without detailed review   |

#### CI/CD Integration

**GitHub Actions:**
```yaml
name: PR Review
on:
  pull_request:
    types: [opened, synchronize]

permissions:
  pull-requests: write

steps:
  - uses: actions/checkout@v4
  - name: Run Mehrhof PR Review
    run: mehr review pr --pr-number ${{ github.event.pull_request.number }} --token "${{ secrets.GITHUB_TOKEN }}"
```

**GitLab CI:**
```yaml
review:
  stage: test
  script:
    - mehr review pr --pr-number $CI_MERGE_REQUEST_IID --provider gitlab --token "$GITLAB_TOKEN"
  only:
    - merge_requests
```

#### Skipping Reviews

If no issues are found or the PR was already reviewed:
```bash
$ mehr review pr --pr-number 123

⏭️  Skipped: No new changes since last review
```

## See Also

- [implement](cli/implement.md) - Generate code
- [finish](cli/finish.md) - Complete task
- [Workflow](../concepts/workflow.md) - Review phase
