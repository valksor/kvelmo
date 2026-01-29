# Context Configuration

Controls how hierarchical task context is included when working on subtasks.

## Overview

When working on a subtask (e.g., a GitHub issue that's a child of another issue), mehrhof can optionally include:

- **Parent task context** - The parent task's title and description
- **Sibling subtasks** - Other subtasks of the same parent

This context helps AI agents understand the bigger picture and avoid conflicts with related work.

## Configuration

**Location:** `.mehrhof/config.yaml` (in project)

```yaml
context:
  # Include parent task context when working on subtasks (default: true)
  include_parent: true

  # Include sibling subtask context (default: true)
  include_siblings: true

  # Maximum number of siblings to include (default: 5)
  max_siblings: 5

  # Truncate descriptions to this length in characters (default: 500)
  description_limit: 500
```

## Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `include_parent` | bool | `true` | Include parent task title and description in prompts |
| `include_siblings` | bool | `true` | Include list of sibling subtasks in prompts |
| `max_siblings` | int | `5` | Maximum number of siblings to include (token budget) |
| `description_limit` | int | `500` | Maximum length of descriptions before truncation |

## CLI Flags

CLI flags override workspace configuration:

| Flag | Description |
|------|-------------|
| `--with-parent` | Include parent task context (overrides config) |
| `--without-parent` | Exclude parent task context (overrides config) |
| `--with-siblings` | Include sibling subtask context (overrides config) |
| `--without-siblings` | Exclude sibling subtask context (overrides config) |
| `--max-siblings N` | Maximum sibling tasks to include (overrides config) |

### Examples

```bash
# Include parent but not siblings
mehr plan --without-siblings

# Include up to 10 siblings instead of default 5
mehr plan --max-siblings 10

# Explicitly exclude all hierarchical context
mehr plan --without-parent --without-siblings
```

## Provider Support

Hierarchical context is available for providers that support parent fetching:

| Provider | Parent Fetching | Subtask Fetching |
|----------|----------------|------------------|
| Wrike | ✓ | ✓ |
| GitHub | ✓ | ✓ |
| GitLab | ✓ | ✓ |
| JIRA | ✓ | ✓ |
| Asana | ✓ | ✓ |
| Linear | ✓ | ✓ |
| ClickUp | ✓ | ✓ |
| Azure DevOps | ✓ | ✓ |
| Bitbucket | ✓ | ✓ |
| YouTrack | ✓ | ✓ |
| Trello | ✓ | ✓ |

For providers that don't support hierarchical features, context inclusion is silently skipped.

## How It Works

1. **Subtask Detection** - mehrhof detects if the current task is a subtask by:
   - Checking for `is_subtask: true` in task metadata
   - Checking for `parent_id` in task metadata
   - Detecting `-task-` or `:task-` patterns in task IDs (for markdown providers)

2. **Parent Fetching** - If the task is a subtask and parent fetching is enabled:
   - The parent task is fetched from the provider
   - Title and description are extracted
   - Description is truncated to `description_limit` characters

3. **Sibling Fetching** - If sibling fetching is enabled:
   - All subtasks of the parent are fetched
   - The current task is filtered out
   - Results are limited to `max_siblings`
   - Siblings are included as a summary list (title + state)

4. **Prompt Inclusion** - The hierarchical context is added to planning and implementation prompts:

```markdown
## Parent Task Context

**Title:** Implement User Authentication

**Description:** Add OAuth2 authentication with support for Google
and GitHub providers. Include session management and token
refresh logic. [truncated...]

## Related Subtasks

- ● Implement OAuth2 Provider Interface (done)
- ◐ Add Session Management (in progress)
- ○ Add Token Refresh Logic (todo)
- ○ Create Login/Logout Endpoints (todo)
```

## Storage

Hierarchical context metadata is persisted in the task work file:

**Location:** `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/work.yaml`

```yaml
hierarchy:
  parent_id: "TASK-123"
  parent_title: "Parent Task Title"
  sibling_ids:
    - "SUBTASK-1"
    - "SUBTASK-2"
    - "SUBTASK-3"
```

This cached information is used by the Web UI to display hierarchy without additional API calls.

## Web UI Display

When hierarchical context is available, the dashboard shows:

### Parent Task Section

```
┌─────────────────────────────────────────────────────────────┐
│  👤 Parent Task                                             │
├─────────────────────────────────────────────────────────────┤
│  Implement User Authentication System                       │
│                                                             │
│  Add OAuth2 authentication with support for Google and...   │
│                                                             │
│  [View in provider →]                                       │
└─────────────────────────────────────────────────────────────┘
```

### Sibling Subtasks Section

```
┌─────────────────────────────────────────────────────────────┐
│  🔗 Related Subtasks                                        │
├─────────────────────────────────────────────────────────────┤
│  ● Implement OAuth2 Provider Interface                      │
│  ○ Add Token Refresh Logic                                  │
│  ○ Create Login/Logout Endpoints                            │
└─────────────────────────────────────────────────────────────┘
```

See [Web UI Dashboard](../web-ui/dashboard.md#hierarchical-context) for more details.

## Benefits

Including hierarchical context provides several benefits:

1. **Better Understanding** - Agents understand the broader goal, not just the isolated subtask
2. **Avoid Conflicts** - Agents can see what siblings are working on to avoid duplicate work
3. **Consistent Implementation** - Siblings follow similar patterns when they see the parent context
4. **Reduced Context Switching** - Developers see related work without leaving the current task

## Token Budget

Hierarchical context consumes tokens. To control usage:

```yaml
context:
  include_parent: true      # ~100-500 tokens depending on description length
  include_siblings: true     # ~50-200 tokens depending on max_siblings
  max_siblings: 5            # Each sibling is ~10-40 tokens
  description_limit: 500     # Truncate long descriptions
```

If token usage is a concern, consider:
- Setting `max_siblings: 3` or lower
- Setting `description_limit: 200` for shorter descriptions
- Using `--without-siblings` flag to exclude siblings entirely
- Using `--without-parent` flag for tasks where parent context isn't needed

## See Also

- [CLI: start](../cli/start.md#context-flags) - Starting tasks with context flags
- [CLI: plan](../cli/plan.md#context-flags) - Planning with context flags
- [Providers](../providers/index.md) - Provider-specific capabilities
