# Tutorial: GitHub Issue to Pull Request

This guide walks through the complete external provider workflow — starting from a GitHub issue, generating a plan, asking the agent questions, reviewing code, and finishing with an auto-created pull request. We use a Django project as the example, but the workflow applies to any codebase.

**What you'll learn:**
- Starting tasks from GitHub issues (`mehr start github:`)
- Using task templates (`--template feature`)
- Asking the agent questions mid-workflow (`mehr question`)
- Running the code review step (`mehr review`)
- Budget controls and cost tracking (`mehr cost`)

**Prerequisites:** Mehrhof installed, agent CLI installed, a GitHub repository you have push access to.

---

## Step 1: Configure the GitHub Provider

Add your GitHub token to `.mehrhof/.env`:

```bash
GITHUB_TOKEN=ghp_your_token_here
```

No additional config is needed — Mehrhof auto-detects the repository from your git remote.

---

## Step 2: Start from a GitHub Issue

Suppose issue #42 says: *"Add a /api/health/ endpoint that returns service status for load balancer checks."*

```bash
mehr start github:42 --template feature
```

What happens:
- Mehrhof fetches the issue title, body, labels, and comments from the GitHub API
- The `--template feature` applies a standard branch pattern (`feat/{key}--{slug}`) and commit prefix
- A task branch is created and you're switched to it

```
Task started: github:42 — "Add health check endpoint"
  Branch:   feat/42--add-health-check-endpoint
  Template: feature
  Source:   https://github.com/yourorg/yourproject/issues/42
```

---

## Step 3: Generate a Plan

```bash
mehr plan
```

The agent reads the GitHub issue content and generates structured specifications. You'll see the plan in real time.

After planning completes, check what was created:

```bash
mehr status
```

Review the specification files — they live in your work directory and describe what the agent intends to build.

---

## Step 4: Ask Questions Before Implementing

Maybe you want to influence a design decision before the agent writes code:

```bash
mehr question "Should we use Django REST Framework or plain JsonResponse? We have DRF installed but this is a simple endpoint."
```

The agent responds with reasoning based on your codebase context. You can ask follow-up questions:

```bash
mehr question "Good point. Also, should the health check verify database connectivity or just return a static OK?"
```

These questions and answers become part of the task context — the agent remembers them during implementation.

---

## Step 5: Set a Budget (Optional)

If you want to limit what this task costs:

```bash
mehr budget task set --max-cost 3.00 --on-limit pause --warning-at 0.8
```

If the task hits $2.40 (80%), you'll see a warning. At $3.00, it pauses and waits for you to approve continuing with `mehr budget resume --confirm`.

---

## Step 6: Implement

```bash
mehr implement
```

The agent writes code based on the specifications and your Q&A context. A git checkpoint is created automatically.

---

## Step 7: Review

This step is often skipped in quick workflows, but for anything going to a shared repo it's worth it:

```bash
mehr review
```

The agent reviews its own implementation — checking for issues like missing error handling, test coverage gaps, naming inconsistencies, or patterns that don't match the rest of the codebase. You'll see a summary of findings.

If the review identifies issues, you can fix them:

```bash
mehr note "Fix the issues from the review — especially the missing test for the error case"
mehr implement
```

---

## Step 8: Finish — Auto-Create the Pull Request

```bash
mehr finish
```

Since the task came from GitHub, Mehrhof automatically creates a pull request:

```
Pull request created: #43
  Title:  Add health check endpoint
  Branch: feat/42--add-health-check-endpoint → main
  URL:    https://github.com/yourorg/yourproject/pull/43

Task branch deleted locally.
```

The PR description references the original issue (#42), and the issue will auto-close when the PR is merged (if your GitHub settings allow it).

---

## Step 9: Check What It Cost

```bash
mehr cost --breakdown
```

Output shows token usage per workflow step:

```
Task: Add health check endpoint (github:42)

  Step            Input     Output    Cached    Cost
  ─────────────   ───────   ───────   ───────   ──────
  Planning          4,210     1,830    2,100    $0.12
  Questions         1,450       680      890    $0.04
  Implementing      6,300     3,200    4,100    $0.21
  Reviewing         3,100     1,400    2,800    $0.09
  ─────────────   ───────   ───────   ───────   ──────
  Total            15,060     7,110    9,890    $0.46

  Cache hit rate: 65.7%
```

---

## Tips

- **Labels carry over**: Labels from the GitHub issue (e.g., `bug`, `feature`) are applied to the Mehrhof task automatically.
- **Budget defaults**: Set workspace-wide budget limits once with `mehr budget set --task-max-cost 5 --monthly-max-cost 100` instead of per-task.
- **Templates save time**: Use `--template bug-fix` for fix branches (`fix/{key}--{slug}`), `--template docs` to skip quality checks for docs-only changes, or `--template refactor` for refactoring.
- **Other providers**: This same workflow works with `mehr start jira:PROJ-123`, `mehr start linear:ENG-456`, `mehr start notion:<page-id>`, and 13+ other sources.

## See Also

- [GitHub Provider](../providers/github.md) — Full provider configuration
- [Task Templates](../cli/templates.md) — All built-in templates
- [Budget Management](../cli/budget.md) — Budget configuration and controls
- [Cost Tracking](../cli/cost.md) — Token and cost analysis
