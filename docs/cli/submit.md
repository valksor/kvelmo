# kvelmo submit

Create a PR and submit to the provider.

## Usage

```bash
kvelmo submit
```

## Prerequisites

- Task must be in `reviewing` state
- Run `kvelmo review` first
- All review checklist items must be checked (if configured)
- Transition must be approved via `kvelmo approve submit` (if approval is required)
- Documentation requirements must be met (if configured)

## Examples

```bash
# Submit PR
kvelmo submit
```

## What Happens

1. Review checklist and approval gates are verified
2. Changelog entry is appended (if `storage.changelog_path` is set)
3. Repo PR template is detected and auto-filled (if present)
4. Changes are pushed to remote
5. A PR is created with the configured title pattern
6. Task status is synced to the ticket system (if `status_sync` is enabled)
7. State transitions to `submitted`

## Output

```
PR created: https://github.com/owner/repo/pull/123
State: submitted
```

## Provider Integration

For GitHub/GitLab tasks:
- PR is linked to the original issue
- Labels may be applied
- Assignees may be set

For file tasks:
- PR is created with task title
- Description includes task details

## After Submission

The task is complete. You can:
- Start a new task
- Monitor the PR in your provider

Also in Web UI: [Review Phase](/web-ui/reviewing.md).

## Related

- [review](/cli/review.md) — Review before submitting
- [approve](/cli/approve.md) — Approve gated transitions
- [checklist](/cli/checklist.md) — Manage review checklist
- [Workflow](/concepts/workflow.md) — Complete workflow
