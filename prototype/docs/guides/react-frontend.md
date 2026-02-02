# Tutorial: Stacked Features and Fast Iteration

This guide walks through building dependent features with Mehrhof's stack system — creating branches that build on each other, managing their lifecycle as PRs get reviewed and merged, and using auto mode and worktrees for speed. We use a React component library as the example, but stacking works with any codebase.

**What you'll learn:**
- Full automation with `mehr auto`
- Stacking dependent features (`--depends-on`)
- Visualizing and managing stacks (`mehr stack`)
- Cleaning up code before PR (`mehr simplify --standalone`)
- Working in isolated worktrees (`--worktree`)
- Syncing and rebasing stacks after merges (`mehr stack sync`, `mehr stack rebase`)

**Prerequisites:** Mehrhof installed, agent CLI installed, a project with Git initialized and a remote configured.

---

## The Scenario

You're building a React component library in layers:
1. **Theme system** — design tokens and theme provider (foundation)
2. **Button component** — uses the theme system (depends on #1)
3. **Form components** — uses buttons and theme (depends on #1 and #2)

Each layer builds on the previous one. You want to work on all three without waiting for PRs to be merged.

---

## Step 1: Auto-Run the First Feature

The theme system is straightforward — let Mehrhof handle it end-to-end:

```bash
mehr auto theme-system.md
```

`mehr auto` runs the complete workflow without interaction: start → plan → implement → quality check → finish. If quality checks fail (lint errors, test failures), it feeds the errors back to the agent and retries automatically.

```
[1/5] Starting task...
[2/5] Planning...         done (2 specifications)
[3/5] Implementing...     done (6 files changed)
[4/5] Quality check...    passed
[5/5] Finishing...        done

Pull request created: #10
  Branch: feat/abc12--theme-system → main
  URL:    https://github.com/yourorg/components/pull/10
```

Note the task ID from the output — you'll need it for stacking. Check with `mehr list` if you missed it.

---

## Step 2: Stack the Next Feature

Now start the button component, branching from the theme branch (not main):

```bash
mehr start button-component.md --depends-on abc12345
```

The `--depends-on` flag tells Mehrhof this task's branch should start from the theme task's branch. This means the button code can import from the theme system immediately, even though that PR hasn't merged yet.

```
Task started: button-component
  Branch:   feat/def67--button-component
  Based on: feat/abc12--theme-system (stacked)
```

---

## Step 3: Work Through the Button Component

This time, step through manually since you want more control:

```bash
mehr plan
```

Review the specs, then implement:

```bash
mehr implement
```

---

## Step 4: Clean Up Before Creating the PR

Before finishing, use standalone simplify to clean up the diff:

```bash
mehr simplify --standalone --branch
```

This runs an AI pass over the diff between your branch and its base, looking for:
- Unnecessary complexity
- Redundant code
- Inconsistent patterns
- Overly verbose implementations

It creates a checkpoint first, so you can `mehr undo` if you don't like the simplification.

---

## Step 5: Finish as a Draft PR

The theme PR is still in review, so create a draft:

```bash
mehr finish --draft
```

```
Pull request created: #11 (draft)
  Branch: feat/def67--button-component → main
  URL:    https://github.com/yourorg/components/pull/11
  Stack:  depends on #10 (theme-system)
```

---

## Step 6: Visualize the Stack

```bash
mehr stack --graph
```

```
● feat/abc12--theme-system (#10, pending review)
└── ◐ feat/def67--button-component (#11, draft)
```

Or generate a Mermaid diagram for documentation:

```bash
mehr stack --mermaid
```

---

## Step 7: Do Other Work in a Worktree

While waiting for reviews, fix something unrelated without leaving your current branch:

```bash
mehr auto hotfix.md --worktree
```

The `--worktree` flag creates an isolated git worktree — a separate checkout in a different directory. Your current branch stays untouched. When the fix is done, the worktree is cleaned up automatically.

---

## Step 8: Theme PR Gets Merged — Rebase the Stack

Your team merges the theme PR. Now the button branch needs to rebase onto the updated main:

```bash
mehr stack sync
```

Mehrhof checks PR status from your provider:

```
Syncing stack...
  feat/abc12--theme-system: merged ✓
  feat/def67--button-component: needs rebase ⟳
```

Now rebase:

```bash
mehr stack rebase
```

```
Rebasing feat/def67--button-component onto main...
  Rebase successful.
  Push required: git push --force-with-lease
```

The button branch is now based on main (which includes the merged theme code). Force-push to update the PR.

---

## Step 9: Continue the Stack

Start the form components, stacking on the button:

```bash
mehr start form-components.md --depends-on def67890
mehr plan
mehr implement
mehr simplify --standalone --branch
mehr finish --draft
```

```bash
mehr stack --graph
```

```
✓ feat/abc12--theme-system (#10, merged)
└── ● feat/def67--button-component (#11, pending review)
    └── ◐ feat/ghi89--form-components (#12, draft)
```

---

## Tips

- **When to use auto vs manual**: Use `mehr auto` for well-defined, self-contained tasks where you trust the specs. Step through manually (`plan` → `implement` → `review` → `finish`) when you want to influence decisions or the task is exploratory.

- **Auto mode retries**: `mehr auto` retries quality failures up to 3 times by default. Increase with `--max-retries 5` for complex tasks, or skip with `--no-quality` if you'll review manually.

- **Parallel tasks**: For independent tasks, run them simultaneously:
  ```bash
  mehr start a.md b.md c.md --parallel=3 --worktree
  ```
  Each gets its own worktree and runs concurrently.

- **Stack conflicts**: If `mehr stack rebase` hits a conflict, it aborts cleanly — no partial rebase state. Resolve the conflict manually, then re-run.

- **Long-lived stacks**: For stacks that span days, run `mehr stack sync` regularly to detect merged parents early.

## See Also

- [stack command](../cli/stack.md) — Full stack management reference
- [auto command](../cli/auto.md) — Automated workflow details
- [simplify command](../cli/simplify.md) — Standalone and in-workflow simplification
- [Stacked Features](../concepts/stacked-features.md) — Concept explanation
