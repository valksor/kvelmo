# mehr start

Start a new task from a file, directory, or provider.

## Synopsis

```bash
mehr start <scheme:reference> [flags]
```

## Description

The `start` command registers a new task and prepares the workspace. It:

1. Generates a unique task ID
2. Creates a git branch (unless `--no-branch`)
3. Copies source content to the work directory
4. Sets the task as active

This command does **not** run planning. Use `mehr plan` for that.

## Provider Schemes

Specify a provider scheme prefix to use different task sources:

```bash
mehr start file:task.md
mehr start dir:./tasks/
mehr start github:5
mehr start github:owner/repo#123
```

**Default Provider:** The `file:` provider is the default, so you can omit the scheme for markdown files:

```bash
mehr start task.md
```

To change the default provider, configure it in `.mehrhof/config.yaml`:

```yaml
providers:
  default: dir
```

## Arguments

| Argument           | Description                                                     |
|--------------------|-----------------------------------------------------------------|
| `scheme:reference` | Provider scheme and path (e.g., `file:task.md`, `dir:./tasks/`) |

## Flags

| Flag                | Short | Type   | Default                | Description                                          |
|---------------------|-------|--------|------------------------|------------------------------------------------------|
| `--agent`           | `-A`  | string | auto                   | Agent to use (overrides all steps)                   |
| `--agent-plan`      |       | string |                        | Agent for planning step                              |
| `--agent-implement` |       | string |                        | Agent for implementation step                        |
| `--agent-review`    |       | string |                        | Agent for review step                                |
| `--no-branch`       |       | bool   | false                  | Skip creating a git branch                           |
| `--worktree`        | `-w`  | bool   | false                  | Create a separate git worktree                       |
| `--parallel`        | `-p`  | int    | 1                      | Max parallel tasks (requires `--worktree` if > 1)    |
| `--stash`           |       | bool   | false                  | Stash uncommitted changes before creating branch     |
| `--key`             | `-k`  | string | auto                   | External key for branch/commit naming                |
| `--title`           |       | string | auto                   | Task title override                                  |
| `--slug`            |       | string | auto                   | Branch slug override                                 |
| `--commit-prefix`   |       | string | `[{key}]`              | Commit prefix template                               |
| `--branch-pattern`  |       | string | `{type}/{key}--{slug}` | Branch pattern template                              |
| `--template`        |       | string |                        | Template to apply (bug-fix, feature, refactor, etc.) |

### Context Flags (Hierarchical Tasks)

When working on a subtask (e.g., a GitHub issue that's a child of another issue), these flags control whether to include parent and sibling context in prompts:

| Flag                 | Type | Default | Description                                  |
|----------------------|------|---------|----------------------------------------------|
| `--with-parent`      | bool | config  | Include parent task context in prompts       |
| `--without-parent`   | bool | config  | Exclude parent task context from prompts     |
| `--with-siblings`    | bool | config  | Include sibling subtask context in prompts   |
| `--without-siblings` | bool | config  | Exclude sibling subtask context from prompts |
| `--max-siblings`     | int  | config  | Maximum number of sibling tasks to include   |

**Note:** These flags override the `context` section in `.mehrhof/config.yaml`. See [Context Configuration](../configuration/context.md) for details.

#### Example: Working on Subtasks

```bash
# Include parent context but limit siblings
mehr start github:456 --with-parent --max-siblings 3

# Exclude all hierarchical context
mehr start github:456 --without-parent --without-siblings
```

### Naming Template Variables

| Variable    | Description                                | Example                  |
|-------------|--------------------------------------------|--------------------------|
| `{key}`     | External key from filename/frontmatter/CLI | `FEATURE-123`            |
| `{task_id}` | Internal task ID                           | `a1b2c3d4`               |
| `{type}`    | Task type from filename prefix             | `feature`, `fix`, `task` |
| `{slug}`    | URL-safe slugified title                   | `add-user-auth`          |

**Key resolution priority:** CLI `--key` > frontmatter `key:` > filename > task ID

**Title resolution priority:** CLI `--title` > frontmatter `title:` > first `#` heading > filename

**Slug resolution priority:** CLI `--slug` > frontmatter `slug:` > generated from title

## Examples

### Start from a File (Default Provider)

The `file:` provider is the default, so you can omit it:

```bash
mehr start task.md
mehr start file:task.md
```

Output:

```
Task registered: a1b2c3d4
  Title: Add user authentication
  Key: task
  Branch: task/task--add-user-auth
```

### Start with Ticket ID (Auto-Extracted)

```bash
mehr start file:FEATURE-123.md
```

The key and type are extracted from the filename:

```
Task registered: a1b2c3d4
  Title: Add user authentication
  Key: FEATURE-123
  Branch: feature/FEATURE-123--add-user-auth
```

### Override Key via CLI

```bash
mehr start --key JIRA-456 file:task.md
```

```
Task registered: a1b2c3d4
  Key: JIRA-456
  Branch: task/JIRA-456--add-user-auth
```

### Override Title and Slug via CLI

```bash
mehr start --title "Fix critical login bug" file:task.md

mehr start --slug "login-fix" file:task.md

mehr start file:test.md --slug "test"
```

**Note:** When no title is found in the file (no frontmatter `title:` and no `#` heading), the filename (without extension) is used as the title.

### Start from a Directory

```bash
mehr start dir:./tasks/feature/
```

All files in the directory become part of the task context.

### Start from GitHub Issue

```bash
mehr start github:42
mehr start github:owner/repo#123
mehr start gh:42
```

Repository is auto-detected from `git remote origin`. Requires `GITHUB_TOKEN` or `MEHR_GITHUB_TOKEN`.

```
Task registered: a1b2c3d4
  Title: Fix authentication bug
  Key: 42
  Type: fix (from labels)
  Branch: issue/42--fix-authentication-bug
```

**Features:**

- Fetches issue title, body, labels, and comments
- Extracts linked issues (`#123` references)
- Infers task type from labels (`bug` → `fix`, `enhancement` → `feature`)
- Default branch pattern: `issue/{key}-{slug}`

### Start Without Branch

```bash
mehr start --no-branch task.md
```

Work happens on the current branch. Useful for quick experiments.

### Start with Worktree (Parallel Tasks)

```bash
mehr start --worktree task.md
mehr start --worktree dir:tasks/
```

Creates a separate working directory for complete isolation:

```
Task registered: a1b2c3d4
Worktree: ../project-worktrees/a1b2c3d4
```

**Why use worktrees?**

- Run multiple tasks simultaneously in separate terminals
- Each worktree has its own branch - no switching needed
- Commands auto-detect which task based on your current directory

**Parallel workflow:**

```bash
mehr start --worktree feature-a.md
cd ../project-worktrees/a1b2c3d4
mehr plan && mehr implement

mehr start --worktree feature-b.md
cd ../project-worktrees/e5f6g7h8
mehr plan && mehr implement

mehr list
```

**Note:** New tasks must be started from the main repository, not from within a worktree.

### Start Multiple Tasks in Parallel

```bash
mehr start file:a.md file:b.md file:c.md --parallel=3 --worktree
```

Starts multiple tasks simultaneously in parallel goroutines:

```
Starting 3 tasks in parallel (max workers: 3)
Registered: abc123 (file:a.md)
Registered: def456 (file:b.md)
Registered: ghi789 (file:c.md)
Running tasks in background...
```

**Requirements:**

- `--worktree` is required when `--parallel > 1` to avoid file conflicts
- Each task gets its own conductor instance and worktree
- Tasks run the full workflow: start → plan → implement

**Monitoring parallel tasks:**

```bash
# List running parallel tasks (in-memory goroutines)
mehr list --running

# Send a note to a specific running task
mehr note --running=abc123 "Consider edge case X"
```

**Output from `mehr list --running`:**

```
ID       REFERENCE       STATUS     TASK ID     DURATION   WORKTREE
abc123   file:a.md       running    task-001    5m30s      ../worktrees/abc123
def456   file:b.md       running    task-002    5m28s      ../worktrees/def456
ghi789   file:c.md       completed  task-003    4m15s      ../worktrees/ghi789

2 running, 3 total
```

### Start with Stash (Uncommitted Changes)

If you have uncommitted changes, use `--stash` to automatically stash them before creating the branch:

```bash
mehr start --stash task.md
```

This is useful when:
- You have work-in-progress changes that aren't ready to commit
- You want to start a new task without losing your current work
- You need to context-switch to a different task

**Behavior:**
1. Stashes uncommitted changes **including untracked files** (displays stash reference, e.g., `stash@{0}`)
2. Creates new branch
3. Restores stashed changes into the new branch (regular mode, if `auto_pop_stash: true`)
4. For `--worktree` mode: Stash stays in main repo (not popped into isolated worktree)

**Error Handling:**
- Stash restoration failures are now **fatal** (prevents silent data loss)
- If stash pop fails, the operation will error, and you can manually recover using the displayed stash reference

**Configuration:**
You can enable stash-on-start by default in `.mehrhof/config.yaml`:

```yaml
git:
  stash_on_start: true  # Auto-stash changes before creating task branch
  auto_pop_stash: true  # Auto-pop stash after branch creation (default: true)
  # Set to false to preserve stash for manual restoration
```

When `stash_on_start` is enabled in config, the `--stash` flag is not required.

If `auto_pop_stash` is set to `false`, the stash will be preserved, and you'll need to manually run `git stash pop` to restore the changes.

**Equivalent to:**
```bash
git stash push -u -m "mehrhof: stash before task 2026-01-05T10:30:00"
mehr start task.md
git stash pop  # Only if auto_pop_stash: true (not for worktrees)
```

### Specify Agent

```bash
mehr start --agent claude task.md
```

### Specify Per-Step Agents

Use different agents for different workflow phases:

```bash
mehr start --agent-plan claude-opus --agent-implement claude task.md

mehr start --agent-plan work-account task.md
```

See [AI Agents](../agents/index.md#per-step-agent-configuration) for details.

## Task File Format

Task files are markdown with optional YAML frontmatter:

```markdown
---
title: Add user authentication
key: AUTH-001
type: feature
---

# Task Title

Brief description of what needs to be done.

## Requirements

- Requirement 1
- Requirement 2

## Constraints

- Use existing patterns
- Maintain backward compatibility
```

### Frontmatter Fields

| Field   | Description                  | Example                   |
|---------|------------------------------|---------------------------|
| `title` | Task title (overrides H1)    | `Add user authentication` |
| `key`   | External key for naming      | `AUTH-001`, `JIRA-123`    |
| `type`  | Task type for branch pattern | `feature`, `fix`, `docs`  |
| `slug`  | Branch slug override         | `add-auth`, `login-fix`   |

## What Happens

1. **ID Generation** - 8-character unique identifier

2. **Naming Resolution**
   - External key resolved from CLI > frontmatter > filename
   - Type extracted from filename prefix (e.g., `FEATURE-` → `feature`)
   - Slug generated from title

3. **Branch Creation** (unless `--no-branch`)
   - Branch name: `{type}/{key}--{slug}` (e.g., `feature/FEATURE-123--add-auth`)
   - Base branch: current HEAD

4. **Work Directory**
   - Created at `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/`
   - Source content copied (read-only)
   - `work.yaml` metadata file created

5. **Git Operations**
   - New branch created
   - Initial commit (optional)
   - Switched to task branch

## After Starting

Check your task:

```bash
mehr status
```

Continue to planning:

```bash
mehr plan
```

## Existing Work Directories

If you've previously finished a task (and chose to save the work directory), running `mehr start` again will detect the existing work:

```bash
mehr start task.md
```

Output:

```
Found 1 existing work director(ies) from previous tasks:
  - a1b2c3d4: Add user authentication

Options:
  [d]elete and archive - Archive old work, start fresh
  [c]ontinue with existing - Reuse directory, reset to idle state

Your choice [d/c]:
```

### Options

- **Delete and archive**: Moves the old work directory to `~/.valksor/mehrhof/workspaces/<project-id>/archive/<taskID>-timestamp/` and creates a fresh task
- **Continue with existing**: Reuses the existing work directory, fetches updated content from the provider, resets state to idle, and keeps all existing files (specs, sessions, checkpoints)

### Continue Workflow

When continuing with an existing work directory:

1. Updated content is fetched from the provider (e.g., Wrike, GitHub issue)
2. Source files are updated in the existing work directory
3. Task state is reset to `idle`
4. All existing specifications, sessions, and checkpoints are preserved
5. Task is set as active and ready for planning

This is useful when:
- A task has been updated with new requirements
- You want to revisit a previously completed task
- You need to continue work on a task that was finished but not merged

## See Also

- [plan](cli/plan.md) - Create specifications
- [status](cli/status.md) - View task status
- [Tasks Concept](../concepts/tasks.md) - Understanding tasks
