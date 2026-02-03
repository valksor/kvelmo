# mehr sync

Sync task data from the provider and generate a delta specification if changes are detected.

## Synopsis

```bash
mehr sync [task-id]
```

## Description

This command is useful when a task has been modified in the external system, and you want to update your local work accordingly.

The command will:

1. Fetch the latest version of the task from the provider
2. Compare it with the stored local version
3. Generate a delta specification if changes are detected
4. Save the new specification to the specifications directory

## Example

```bash
mehr sync TASK-123
```

## Output

If changes are detected:

```
→ Fetching latest task data from wrike...
→ Detecting changes...

▲ Changes detected:
  description updated, 2 new comments

→ Generating delta specification...

✓ Generated delta specification: ~/.valksor/mehrhof/workspaces/<project-id>/work/TASK-123/specifications/specification-2.md

Next steps:
  1. Review the delta specification
  2. Run `mehr plan` to create an implementation plan
  3. Run `mehr implement` to apply the changes
```

If no changes:

```
→ Fetching latest task data from wrike...
→ Detecting changes...
✓ No changes detected in the task.
```

## Generated Files

- `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/specifications/specification-N.md` - Delta specification with update instructions
- `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/source/<provider>.previous` - Backup of original source
- `~/.valksor/mehrhof/workspaces/<project-id>/work/<task-id>/source/changes.txt` - Human-readable change summary

## Change Detection

The sync command detects changes in:

- Title
- Description
- Status (e.g., Open → In Progress)
- Priority (e.g., Normal → High)
- Labels (added, removed, or changed)
- Assignees (added or removed)
- New comments
- Updated comments (comments with modified text)
- New attachments
- Removed attachments

## Workflow

After syncing:

1. **Review** the generated delta specification
2. **Plan** with `mehr plan` to create implementation steps
3. **Implement** with `mehr implement` to apply changes

## Provider Support

Currently supported providers for sync:

**Local Providers:**
- `file` - Markdown files (detects file modifications, attachment references, frontmatter metadata)
- `directory` - Directories (detects new/deleted/modified files, README changes)
- `empty` - Empty tasks (generates delta specification from note metadata changes)

**External Providers:**
- `github` - GitHub issues (full change detection including labels, assignees, milestones)
- `wrike` - Wrike tasks
- `gitlab` - GitLab issues
- `jira` - Jira tickets
- `linear` - Linear issues
- `notion` - Notion pages
- `asana` - Asana tasks
- `clickup` - ClickUp tasks
- `trello` - Trello cards

## Configuration

Provider credentials are configured via environment variables or `.mehrhof/config.yaml`:

```yaml
providers:
  wrike:
    token: YOUR_WRIKE_TOKEN
  github:
    token: YOUR_GITHUB_TOKEN
```

## Web UI

Prefer a visual interface? See the Sync from Provider feature in [Project Planning](/web-ui/project-planning.md).

## See Also

- [mehr start](start.md) - Create a new task
- [mehr continue](continue.md) - Resume work on a task
- [mehr status](status.md) - Check task status
