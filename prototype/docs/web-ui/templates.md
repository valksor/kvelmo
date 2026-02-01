# Templates

Templates provide pre-configured frontmatter, agent selection, and workflow settings for common task types. Use templates to ensure consistency across similar tasks and reduce repetitive configuration.

## Overview

Templates include:
- **Task type** - Classification (bug-fix, feature, refactor, etc.)
- **Agent selection** - Which AI agent to use
- **Git configuration** - Branch naming, commit prefixes
- **Workflow options** - Quality checks, skip options

## Available Templates

| Template   | Description             | Best For               |
|------------|-------------------------|------------------------|
| `bug-fix`  | Stricter validation     | Bug fixes, issues      |
| `feature`  | New feature development | New capabilities       |
| `refactor` | Quality-focused         | Code improvements      |
| `docs`     | Skips quality checks    | Documentation changes  |
| `test`     | Test-focused            | Adding/improving tests |
| `chore`    | Maintenance             | Routine tasks          |

## Accessing in the Web UI

Templates are available when:

1. **Creating a task** - Choose from template dropdown
2. **From the dashboard** - Click "Create Task" → Select template
3. **Via API** - `POST /api/v1/templates/apply`

## Using Templates

### Applying When Creating a Task

1. Click **"Create Task"** from the dashboard
2. In the template dropdown, select a template:
   - `bug-fix` - For bug fixes
   - `feature` - For new features
   - `refactor` - For refactoring
   - `docs` - For documentation
   - `test` - For tests
   - `chore` - For maintenance
3. The template frontmatter is applied automatically
4. Fill in your task details
5. Click **"Create"**

### Template Details

Click on any template to see its configuration:

**Template: bug-fix**
```
Type: fix
Agent: claude
Git Branch: fix/{key}--{slug}
Commit Prefix: [fix/{key}]
Quality Checks: Enabled
```

**Template: docs**
```
Type: docs
Agent: claude-haiku (faster, cheaper)
Git Branch: docs/{key}--{slug}
Commit Prefix: [docs]
Quality Checks: Disabled
```

### What Templates Do

When you apply a template, it sets:

**Frontmatter:**
```yaml
---
type: fix
agent: claude
---
```

**Git Behavior:**
- Branch naming pattern
- Commit message prefix

**Workflow:**
- Whether to run quality checks
- Which agent to use

## Template Reference

### bug-fix

For bug fix tasks requiring stricter validation.

```yaml
type: fix
agent: claude
git:
  branch_pattern: "fix/{key}--{slug}"
  commit_prefix: "[fix/{key}]"
workflow:
  skip_quality: false
```

**Use when:** Fixing reported issues, bugs, crashes.

### feature

For new feature development.

```yaml
type: feature
agent: claude
git:
  branch_pattern: "feature/{key}--{slug}"
  commit_prefix: "[{key}]"
workflow:
  skip_quality: false
```

**Use when:** Adding new capabilities, user-facing changes.

### refactor

For code refactoring and quality improvements.

```yaml
type: refactor
agent: claude
git:
  branch_pattern: "refactor/{key}--{slug}"
  commit_prefix: "[refactor]"
workflow:
  skip_quality: false
```

**Use when:** Improving code structure, removing duplication.

### docs

For documentation changes (quality checks skipped).

```yaml
type: docs
agent: claude-haiku
git:
  branch_pattern: "docs/{key}--{slug}"
  commit_prefix: "[docs]"
workflow:
  skip_quality: true
```

**Use when:** Updating README, adding guides, fixing typos.

### test

For adding or improving tests.

```yaml
type: test
agent: claude
git:
  branch_pattern: "test/{key}--{slug}"
  commit_prefix: "[test]"
workflow:
  skip_quality: false
```

**Use when:** Adding unit tests, improving coverage.

### chore

For maintenance tasks and chores.

```yaml
type: chore
agent: claude
git:
  branch_pattern: "chore/{key}--{slug}"
  commit_prefix: "[chore]"
workflow:
  skip_quality: true
```

**Use when:** Dependencies, configuration, cleanup.

## Creating Custom Templates

You can create custom templates in `.mehrhof/templates/`:

### Custom Template File

Create `.mehrhof/templates/security.yaml`:

```yaml
name: security
description: Template for security fixes
frontmatter:
  type: security
  priority: critical
agent: claude
git:
  branch_pattern: "security/{key}--{slug}"
  commit_prefix: "[security/{key}]"
workflow:
  skip_quality: false
```

Custom templates appear in the dropdown alongside built-in templates.

## Template Merging

When applying a template to a file with existing frontmatter:

- Template values **take precedence** over existing values
- Existing values not in template are preserved
- The merge is shallow (nested maps are replaced, not merged)

**Example:**

**Before:**
```yaml
---
title: Fix login bug
key: BUG-123
---
```

**After applying `bug-fix`:**
```yaml
---
title: Fix login bug
key: BUG-123
type: fix
agent: claude
---
```

## Common Workflows

### Bug Fix with Template

```
1. Create task from template: bug-fix
2. Enter: "Fix null pointer in auth"
3. Template sets type=fix, quality checks enabled
4. Branch named: fix/BUG-123--fix-null-pointer
5. Commits prefixed: [fix/BUG-123]
```

### Documentation Update

```
1. Create task from template: docs
2. Enter: "Update API documentation"
3. Template sets type=docs, uses faster agent
4. Quality checks skipped (faster completion)
```

### Custom Security Template

```
1. Create .mehrhof/templates/security.yaml
2. Set priority=critical, specific agent
3. Apply when creating security tasks
4. Consistent security workflow
```

## CLI Equivalent

See [`mehr templates`](../cli/templates.md) for CLI usage.

| CLI Command                                  | Web UI Action         |
|----------------------------------------------|-----------------------|
| `mehr templates`                             | List templates        |
| `mehr templates show bug-fix`                | Show template details |
| `mehr templates apply bug-fix task.md`       | Apply to file         |
| `mehr start --template bug-fix file:task.md` | Create with template  |

## Template Storage

Built-in templates are bundled with Mehrhof.

Custom templates are stored in:
```
.mehrhof/templates/
  security.yaml
  performance.yaml
  (your custom templates)
```
