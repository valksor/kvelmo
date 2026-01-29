# Stacked Features

The stack page allows you to manage dependent features and their git branches through the Web UI.

## Accessing the Stack Page

Navigate to **Stacked Features** in the sidebar, or go directly to `/stack`:

```
┌──────────────────────────────────────────────────────────────┐
│  Sidebar                                                      │
├──────────────────────────────────────────────────────────────┤
│  Dashboard                                                   │
│  Tasks                                                       │
│  Project Planning                                            │
│  Quick Tasks                                                 │
│  Memory                                                      │
│  Links                                                       │
│  ► Stacked Features  ← Click here                            │
│  Settings                                                    │
└──────────────────────────────────────────────────────────────┘
```

## Stack Overview

The stack page shows all feature stacks with their tasks and status:

```
┌──────────────────────────────────────────────────────────────┐
│  Stacked Features                   [Sync PR Status]         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Stack: auth-system                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  3 tasks · Root: issue-100                              │ │
│  │                                                          │ │
│  │  ✓ issue-100  feature/auth-system       merged          │ │
│  │  ⟳ issue-101  feature/auth-oauth        needs-rebase    │ │
│  │  ● issue-102  feature/auth-oauth-google active          │ │
│  │                                                          │ │
│  │  [⚠ Needs Rebase]                    [Rebase All]       │ │
│  │                                                          │ │
│  │  Created: 2024-01-15 10:00 · Updated: 2024-01-18 14:30  │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Task States

Each task displays its current state with a visual indicator:

| Icon | State | Color | Description |
|------|-------|-------|-------------|
| ✓ | merged | Green | PR merged to target |
| ⟳ | needs-rebase | Yellow | Parent merged, needs rebasing |
| ✗ | conflict | Red | Rebase failed due to conflicts |
| ◯ | pending-review | Blue | PR open, awaiting review |
| ◉ | approved | Green | PR approved, ready to merge |
| ● | active | Gray | Being worked on |
| ○ | abandoned | Gray | PR closed without merge |

## Syncing PR Status

Click **"Sync PR Status"** to fetch the latest PR status from your provider:

```
┌──────────────────────────────────────────────────────────────┐
│  [Sync PR Status]  ← Click this button                       │
│                     ↓                                        │
│  Syncing... ⟳                                               │
│                     ↓                                        │
│  ✓ Synced: 2 tasks updated                                  │
└──────────────────────────────────────────────────────────────┘
```

The sync operation:
1. Fetches PR status for all stacked tasks
2. Updates states (pending-review → merged, etc.)
3. Marks children as needs-rebase when parents merge

## Rebasing Stacks

When a parent feature merges, its children need rebasing. Click **"Rebase All"** on a stack card:

```
┌──────────────────────────────────────────────────────────────┐
│  Stack: auth-system                                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  [⚠ Needs Rebase]                    [Rebase All]  ←   │ │
│  │                                                          │ │
│  │  Click to rebase all tasks in this stack                │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### Rebase Progress

Watch the rebase progress in real-time:

```
┌──────────────────────────────────────────────────────────────┐
│  Rebasing stack auth-system...                               │
│                                                              │
│  ✓ issue-101: rebased onto main                             │
│  ⟳ issue-102: rebasing onto feature/auth-oauth...          │
│                                                              │
│  ▶ In progress...                                            │
└──────────────────────────────────────────────────────────────┘
```

### Rebase Success

When rebase completes successfully:

```
┌──────────────────────────────────────────────────────────────┐
│  ✓ Rebase Complete                                          │
│                                                              │
│  Rebased 2 tasks:                                           │
│    • issue-101: main ← feature/auth-system                  │
│    • issue-102: feature/auth-oauth ← feature/auth-oauth-google │
│                                                              │
│  Next steps:                                                 │
│    1. Push updated branches                                  │
│    2. Update PRs if needed                                   │
└──────────────────────────────────────────────────────────────┘
```

### Rebase Conflict

If a conflict occurs, the rebase aborts and shows the conflict:

```
┌──────────────────────────────────────────────────────────────┐
│  ✗ Rebase Failed - Conflict                                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Task: issue-102 (feature/auth-oauth-google)                 │
│  Rebasing onto: feature/auth-oauth                           │
│                                                              │
│  Conflict hint:                                              │
│    Merge conflict in internal/auth/oauth.go                  │
│                                                              │
│  Resolution:                                                 │
│    1. Open terminal in your project                          │
│    2. Checkout the branch: git checkout feature/auth-oauth-google │
│    3. Start rebase: git rebase feature/auth-oauth            │
│    4. Resolve conflicts manually                             │
│    5. Continue rebase: git rebase --continue                 │
│    6. Return here and sync status                            │
│                                                              │
│  [Copy Resolution Steps]                                     │
└──────────────────────────────────────────────────────────────┘
```

## Task Details

Click on a task to see more details:

```
┌──────────────────────────────────────────────────────────────┐
│  Task: issue-101                                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Branch: feature/auth-oauth                                  │
│  State: needs-rebase                                         │
│  PR: #101 (https://github.com/org/repo/pull/101)            │
│                                                              │
│  Depends on: issue-100 (merged)                             │
│  Blocks: issue-102                                           │
│                                                              │
│  Part of stack: auth-system                                  │
│                                                              │
│  [View PR] [Rebase This Task]                               │
└──────────────────────────────────────────────────────────────┘
```

## Empty State

When no stacks exist:

```
┌──────────────────────────────────────────────────────────────┐
│  No Stacked Features                                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Create dependent features by using:                         │
│                                                              │
│    mehr start <task> --depends-on <parent>                  │
│                                                              │
│  Or start a task while on a feature branch to be prompted   │
│  about creating a dependency.                                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Status Badges

Stacks display status badges for quick visibility:

| Badge | Meaning |
|-------|---------|
| `⚠ Needs Rebase` | One or more tasks need rebasing |
| `✗ Conflict` | One or more tasks have conflicts |
| `✓ All Merged` | All tasks in stack are merged |

## Help Section

The page includes a help section explaining stacked features:

```
┌──────────────────────────────────────────────────────────────┐
│  About Stacked Features                                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Stacked features allow you to work on Feature B while       │
│  Feature A is waiting on code review.                        │
│                                                              │
│  • Create a dependent feature:                               │
│      mehr start feature-b --depends-on feature-a            │
│                                                              │
│  • Sync PR status:                                           │
│      Click "Sync PR Status" to fetch latest PR states        │
│                                                              │
│  • Rebase after parent merges:                               │
│      When a parent PR merges, children are marked            │
│      "needs-rebase". Click "Rebase All" to rebase them.      │
│                                                              │
│  Use `mehr stack` in the CLI for more options including      │
│  --graph and --mermaid visualization.                        │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## API Endpoints

The stack page uses these API endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/stack` | List all stacks |
| POST | `/api/v1/stack/sync` | Sync PR status |
| POST | `/api/v1/stack/rebase` | Rebase stacks/tasks |

## CLI Equivalent

```bash
# List stacks
mehr stack

# Sync PR status
mehr stack sync

# Rebase all stacks
mehr stack rebase

# Rebase specific stack
mehr stack rebase --stack stack-abc123

# Rebase specific task
mehr stack rebase --task issue-101
```

See [CLI: stack](../cli/stack.md) for all options.

## See Also

- [Stacked Features Concept](../concepts/stacked-features.md) - Architecture and design
- [CLI: stack](../cli/stack.md) - CLI equivalent
- [Settings](settings.md) - Configure stack behavior
