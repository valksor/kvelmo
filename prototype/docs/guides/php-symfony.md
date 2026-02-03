# Tutorial: Planning a Multi-Task Feature

This guide walks through Mehrhof's project planning workflow — breaking a large feature into multiple tasks with dependencies, refining them with AI, and auto-executing the queue. We use a Symfony authentication system as the example, but this workflow applies to any multi-part feature in any codebase.

**What you'll learn:**
- Breaking large features into tasks (`mehr project plan`)
- Managing a task queue (`mehr project tasks`, `mehr project edit`, `mehr project reorder`)
- Adding discovered work with quick tasks (`mehr quick`)
- AI-improving task descriptions (`mehr optimize`)
- Auto-executing through a task queue (`mehr project start --auto`)
- Using directory-based task sources (`dir:`)

**Prerequisites:** Mehrhof installed, agent CLI installed, a project with Git initialized.

---

## The Scenario

You need to build an authentication system for a Symfony application: login, registration, password reset, and role-based access. This is too big for a single task — it should be broken into pieces with clear dependencies.

---

## Step 1: Create a Specs Directory

Create a directory with your requirements. Each file describes a part of the feature:

```
specs/auth-system/
├── README.md          # Overview and architecture decisions
├── login.md           # Login page and session management
├── registration.md    # User registration with email verification
└── password-reset.md  # Password reset flow
```

The `README.md` is the main spec — it should describe the overall approach, shared constraints, and how the pieces fit together.

---

## Step 2: Generate a Task Breakdown

```bash
mehr project plan dir:./specs/auth-system/ --title "Auth System"
```

Mehrhof reads all files in the directory and the AI generates a structured task breakdown:

```
Project planned: "Auth System"
  Queue:  auth-system (4 tasks)
  Source: dir:./specs/auth-system/

  Tasks:
    task-1  [ready]    Set up User entity and authentication config
    task-2  [blocked]  Implement login with session management
    task-3  [blocked]  Implement registration with email verification
    task-4  [blocked]  Implement password reset flow

  Dependencies:
    task-2 → depends on task-1
    task-3 → depends on task-1
    task-4 → depends on task-1, task-3
```

The AI identified that login, registration, and password reset all depend on the User entity setup, and that password reset depends on registration (because it needs email infrastructure).

---

## Step 3: Review the Task Queue

```bash
mehr project tasks auth-system --show-deps
```

This shows all tasks with their status, priority, dependencies, and descriptions. Review them to make sure the breakdown makes sense.

---

## Step 4: Adjust Tasks

Maybe you want registration to be higher priority than login:

```bash
mehr project edit task-3 --priority 1 --labels "security,backend,email"
```

Or add a dependency you noticed the AI missed:

```bash
mehr project edit task-4 --depends-on task-2
```

---

## Step 5: Let AI Reorder

```bash
mehr project reorder auth-system --auto
```

The AI analyzes dependencies, priorities, and task complexity to suggest an optimal execution order:

```
Suggested order:
  1. task-1  Set up User entity and authentication config
     Reason: Foundation — all other tasks depend on this
  2. task-3  Implement registration with email verification
     Reason: Priority 1, and task-4 depends on it
  3. task-2  Implement login with session management
     Reason: Independent of task-3, can follow registration
  4. task-4  Implement password reset flow
     Reason: Depends on both task-1 and task-3
```

---

## Step 6: Discover Extra Work

While reviewing the specs, you realize you need CSRF protection middleware. This wasn't in the original specs:

```bash
mehr quick "Add CSRF protection middleware for all auth forms"
```

An interactive menu appears. Choose `[d]iscuss` to add context:

```
mehr (quick) > This needs to apply to login, registration, and password reset forms
mehr (quick) > Use Symfony's built-in CSRF token validation
mehr (quick) > Should run before the form submission handlers
mehr (quick) > exit
```

Now optimize the task — AI improves the title, expands the description with your notes, and suggests labels:

```bash
mehr optimize --task=quick-tasks/task-1
```

```
Optimized: "Add CSRF protection middleware for all auth forms"
  → Title:    "Add CSRF token validation to authentication forms"
  → Labels:   security, middleware, auth
  → Description expanded with implementation details from notes
```

---

## Step 7: Execute the Queue

When you're satisfied with the task breakdown, execute it:

```bash
mehr project start auth-system --auto
```

Mehrhof auto-chains through each task in order: start → plan → implement → quality check → finish. If a quality check fails, the agent gets feedback and retries.

```
[1/4] Starting: Set up User entity and authentication config
  Planning...     done
  Implementing... done
  Quality check... passed
  Finishing...    done (merged to main)

[2/4] Starting: Implement registration with email verification
  Planning...     done
  Implementing... done
  Quality check... failed (1 lint error)
  Retrying...     done
  Quality check... passed
  Finishing...    done (merged to main)

[3/4] Starting: Implement login with session management
  ...
```

Each task creates its own branch, implements the feature, runs quality checks, and merges — all hands-off.

---

## Tips

- **Large spec directories**: For directories with many files (50+), use `research:` instead of `dir:`. It builds a manifest without reading every file, and the agent selectively reads what it needs:
  ```bash
  mehr project plan research:./docs/product-specs/ --title "Q1 Roadmap"
  ```

- **Submit to external tools**: Push the task queue to your team's project management tool:
  ```bash
  mehr project submit auth-system --provider jira --create-epic --labels "q1,backend"
  mehr project submit auth-system --provider wrike --dry-run  # Preview without creating
  ```

- **Sync from external**: Pull existing project structures into Mehrhof:
  ```bash
  mehr project sync jira:AUTH-EPIC --max-depth 2
  ```

- **Don't auto-chain everything**: Use `--auto` when you trust the specs. For exploratory work, step through manually (`mehr project start auth-system` without `--auto`) and review each task's output before proceeding.

## See Also

- [project command](/cli/project.md) — Full project command reference
- [quick command](/cli/quick.md) — Quick task creation
- [optimize command](/cli/optimize.md) — AI task refinement
- [Directory Provider](/providers/directory.md) — Using directory-based task sources
