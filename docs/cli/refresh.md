# refresh

Check PR status and update local state.

## Usage

```bash
kvelmo refresh
```

## Description

Checks the current pull request status and provides guidance on next steps.

If the PR is merged, it suggests running `kvelmo finish`.
If the PR is still open, it checks if the branch needs rebasing.

## Output

```
Task: add-user-auth
Branch: feature/add-user-auth
PR: https://github.com/org/repo/pull/123 (merged)

PR has been merged.

Run: kvelmo finish
```

## PR Status Actions

| PR Status | Suggested Action                        |
|-----------|-----------------------------------------|
| merged    | `kvelmo finish`                         |
| closed    | `kvelmo finish --force`                 |
| open      | Wait for review, or check rebase status |

## Examples

### PR Merged

```bash
kvelmo refresh
```

Output suggests running `kvelmo finish`.

### PR Behind Base

```bash
kvelmo refresh
```

Shows commits behind count if branch needs rebasing.

## Related

- [finish](/cli/finish.md) — Clean up after merge
- [submit](/cli/submit.md) — Create the PR
- [status](/cli/status.md) — Show current task state
