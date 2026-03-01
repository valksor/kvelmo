# kvelmo review

Start the review phase for implemented changes.

## Usage

```bash
kvelmo review
```

## Prerequisites

- Task must be in `implemented` state
- Run `kvelmo implement` first

## Examples

```bash
# Start review
kvelmo review
```

## What Happens

1. State transitions to `reviewing`
2. Security scanning runs (if configured)
3. You review the changes
4. Decide to submit or undo

## Reviewing Changes

View the changes:
```bash
git diff
```

## Approving

If satisfied, submit:
```bash
kvelmo submit
```

## Rejecting

If changes aren't right:
```bash
kvelmo undo
```

Then adjust and re-implement.

Also in Web UI: [Review Phase](/web-ui/reviewing.md).

## Related

- [implement](/cli/implement.md) — Implement before reviewing
- [submit](/cli/submit.md) — Submit after approval
- [undo](/cli/undo.md) — Revert if needed
