# kvelmo checklist

Manage the review checklist for the current task.

## Usage

```bash
kvelmo checklist                  # Show checklist status
kvelmo checklist --check <item>   # Mark an item as checked
kvelmo checklist --uncheck <item> # Unmark an item
```

## Prerequisites

- Task must be loaded
- Review checklist must be configured in settings

## Examples

```bash
# View current checklist status
kvelmo checklist

# Check off security review
kvelmo checklist --check security

# Check off performance review
kvelmo checklist --check performance

# Uncheck an item
kvelmo checklist --uncheck security
```

## Configuration

Define checklist items in project settings:

```yaml
workflow:
  policy:
    review_checklist:
      - security
      - performance
      - tests
      - documentation
```

All items must be checked before `kvelmo submit` is allowed.

## Output

```
Review Checklist:
  [x] security
  [x] performance
  [ ] tests
  [ ] documentation

2/4 items checked
```

## Related

- [review](/cli/review.md) — Start the review phase
- [approve](/cli/approve.md) — Approve transitions
- [submit](/cli/submit.md) — Submit after checklist is complete
