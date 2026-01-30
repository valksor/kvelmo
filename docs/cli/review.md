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

### Configuring Linters

You can control which linters run during review via `.mehrhof/config.yaml`:

**Default (safer) behavior - no auto-detection:**

```yaml
# .mehrhof/config.yaml
quality:
  enabled: true                    # Master switch for quality checks
  use_defaults: false              # Don't auto-enable built-in linters (default)
  linters:
    golangci-lint:
      enabled: true                # Explicitly enable Go linter
    # Add custom linters
    phpstan:
      enabled: true
      command: ["vendor/bin/phpstan", "analyse", "--error-format=json"]
      extensions: [".php"]
```

**Opt-in to auto-detection:**

```yaml
quality:
  enabled: true
  use_defaults: true              # Auto-enable built-in linters based on project files
```

**Built-in linters:**

| Linter | Language | Auto-Detection |
|--------|----------|----------------|
| `golangci-lint` | Go | `go.mod` exists |
| `eslint` | JavaScript/TypeScript | `package.json` exists |
| `ruff` | Python | `pyproject.toml`, `setup.py`, or `requirements.txt` exists |
| `php-cs-fixer` | PHP | `composer.json` exists |

> **Note:** With `use_defaults: false` (default), built-in linters will NOT run unless explicitly enabled. This prevents unintended code modifications (e.g., php-cs-fixer on Symfony projects with custom config paths).

See [Configuration Guide](../configuration/index.md#quality) for full details.

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
| `--standalone` |   | bool   | false        | Review without active task (see Standalone Mode) |
| `--branch` |       | string | ""           | Compare current branch vs base (standalone only) |
| `--range`  |       | string | ""           | Compare commit range (standalone only) |
| `--context` |      | int    | 3            | Lines of context in diff (standalone only) |
| `--agent`  |       | string | ""           | Agent to use for review |

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

## Standalone Review Mode

Review code changes **without an active task**. Useful for quick code reviews, reviewing uncommitted changes, or comparing branches directly.

### Synopsis

```bash
mehr review --standalone [flags] [files...]
```

### Standalone Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--standalone` | bool | false | Enable standalone mode (no active task required) |
| `--branch` | string | "" | Compare current branch against base branch (auto-detects main/master) |
| `--range` | string | "" | Compare specific commit range (e.g., `HEAD~3..HEAD`) |
| `--context` | int | 3 | Lines of context in diff |
| `--agent` | string | "" | Agent to use for review |
| `--fix` | bool | false | Apply suggested fixes (modifies files) |
| `--checkpoint` | bool | true | Create checkpoint before applying fixes (use with --fix) |

### Standalone Examples

**Review uncommitted changes (default):**
```bash
mehr review --standalone
```

**Review current branch vs main:**
```bash
mehr review --standalone --branch
```

**Review current branch vs specific base:**
```bash
mehr review --standalone --branch develop
```

**Review specific commit range:**
```bash
mehr review --standalone --range HEAD~3..HEAD
```

**Review specific files:**
```bash
mehr review --standalone src/foo.go src/bar.go
```

**Use specific agent:**
```bash
mehr review --standalone --agent opus
```

### Fix Mode

Use `--fix` to review code AND apply fixes for issues found:

**Review and fix uncommitted changes:**
```bash
mehr review --standalone --fix
```

**Review and fix branch changes:**
```bash
mehr review --standalone --fix --branch
```

**Skip checkpoint creation (not recommended):**
```bash
mehr review --standalone --fix --checkpoint=false
```

> **Safety Note**: By default, `--fix` creates a git checkpoint before modifying files. Use `mehr undo` or `git checkout .` to revert if needed.

### Standalone Output

```bash
$ mehr review --standalone

ℹ Reviewing uncommitted changes (staged + unstaged)...
Agent reviewing changes...

✓ Review complete

Verdict: NEEDS_CHANGES

Summary:
The code changes introduce a potential nil pointer dereference in the handler function.

Issues:
  [HIGH] handler.go:45 - Missing nil check before accessing user.Name
  [MEDIUM] handler.go:52 - Unused variable 'ctx' should be removed

Tokens: 1234 input, 567 output ($0.0042)
```

### When to Use Standalone Mode

- **Quick reviews**: Review changes without starting a full task
- **Pre-commit checks**: Review uncommitted changes before committing
- **Branch comparisons**: Compare feature branches against main
- **CI/CD pipelines**: Review changes in automated workflows
- **Code exploration**: Review historical changes via commit ranges

### Configuration

Set a default branch for standalone reviews in `.mehrhof/config.yaml`:

```yaml
git:
  default_branch: develop  # Used when --branch is specified without a value
```

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

| Flag                  | Description                                                            |
| --------------------- | ---------------------------------------------------------------------- |
| `--provider`          | Provider: `github`, `gitlab`, `bitbucket`, `azuredevops` (auto-detected) |
| `--pr-number`         | PR/MR number (required)                                                |
| `--format`            | Comment format: `summary` (default), `line-comments`                   |
| `--scope`             | Review scope: `full` (default), `compact`, `files-changed`             |
| `--agent-pr-review`   | Agent to use for PR review (default: `claude`)                         |
| `--token`             | Auth token (overrides config/env vars; use for CI)                     |
| `--acknowledge-fixes` | Acknowledge when previously reported issues are fixed (default: true)  |
| `--update-existing`   | Edit existing comment vs post new comment (default: true)              |

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
mehr review pr --pr-number 200 --agent-pr-review claude-opus
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
