# Continuing

The continue action resumes work on a task and automatically executes the next logical step.

## What Continuing Does

When you click **"Continue"**, Mehrhof:

1. **Shows current status** - Displays task state and what's been done
2. **Suggests next action** - Recommends the logical next step
3. **Auto-executes (optional)** - With auto-mode, runs the next step automatically

## Starting Continue

Click the **"Continue"** button in the Quick Actions section. The section displays your task name, current state, suggestions for what to do next, and the **Continue** button.

## Continue Workflow

```text
                                    No specs ────▶ ┌──────────┐
                               ┌───────────────────│ Run Plan │
                               │                   └──────────┘
                               │   Specs ready ──▶ ┌───────────────┐
┌────────────────┐     ┌───────┴───────┐───────────│ Run Implement │
│ Click Continue │ ──▶ │ Current State?│           └───────────────┘
└────────────────┘     └───────┬───────┘───────────┌────────────┐
                               │   Code ready ───▶ │ Run Review │
                               │                   └────────────┘
                               └───────────────────┌────────────────┐
                                    Reviewed ────▶ │ Suggest Finish │
                                                   └────────────────┘
```

## Context-Aware Suggestions

Continue analyzes your task state and suggests what to do next:

### After Starting (No Specifications)

The **Continue Workflow** dialog shows your task name and state (Idle), notes that no specifications exist yet, and suggests creating specifications using the Plan phase. Action buttons include **Click Plan** and **Close**.

### After Planning

When specifications are ready, the dialog shows the spec count and suggests implementing them. Action buttons include **Click Implement** and **Close**.

### After Implementation

When code has been generated, the dialog shows the number of files modified and suggests reviewing changes, running automated code review, or completing and merging the task. Action buttons include **Review**, **Finish**, and **Close**.

## Auto-Execute Mode

For faster workflow, use auto-execute. The dialog explains that auto-execute will run the next logical step automatically, shows what the next step will be (e.g., "Plan"), and offers **Continue with Auto** and **Close** buttons.

With auto-execute, Mehrhof runs the next step without requiring another click.

## Use Cases

### Resuming Work

After stepping away from a task:

1. Open the dashboard
2. Click **"Continue"**
3. See what's next
4. Click the suggested action or let auto-execute handle it

### Quick Status Check

Use Continue for a faster status update than full status. The **Quick Status** panel shows task ID, title, state, branch, and what the task is ready for (e.g., "Implement (specs created)"). Relevant action buttons are displayed.

### Speed Through Workflow

For experienced users, use Continue with auto-execute to speed through the workflow:

1. Create task
2. Click Continue (auto-runs plan)
3. Click Continue (auto-runs implement)
4. Click Continue (auto-runs review)
5. Click Finish

## Continue vs Status

| Feature         | Continue             | Status                       |
|-----------------|----------------------|------------------------------|
| **Speed**       | Fast, minimal output | Full details                 |
| **Purpose**     | Action-oriented      | Information-oriented         |
| **Suggestions** | Shows next action    | Shows all available commands |
| **Best for**    | Resuming work        | Full inspection              |

## Next Steps

After using Continue:

- [**Planning**](planning.md) - Create specifications
- [**Implementing**](implementing.md) - Execute specifications
- [**Reviewing**](reviewing.md) - Run quality checks
- [**Finishing**](finishing.md) - Complete the task

---

## Also Available via CLI

Resume tasks from the command line when returning to work or automating workflows.

See [CLI: continue](/cli/continue.md) for all flags and options.
