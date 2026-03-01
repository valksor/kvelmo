# kvelmo submit

Create a PR and submit to the provider.

## Usage

```bash
kvelmo submit
```

## Prerequisites

- Task must be in `reviewing` state
- Run `kvelmo review` first

## Examples

```bash
# Submit PR
kvelmo submit
```

## What Happens

1. Changes are pushed to remote
2. A PR is created
3. Task is linked to the PR
4. State transitions to `submitted`

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
- [Workflow](/concepts/workflow.md) — Complete workflow
