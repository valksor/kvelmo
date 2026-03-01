# Your First Task

A complete walkthrough of creating and completing your first task with kvelmo.

## Prerequisites

- kvelmo installed (`kvelmo version` works)
- An agent CLI installed (e.g., Claude)
- A project to work on

## Step 1: Start the Server

Open a terminal and start kvelmo:

```bash
cd /path/to/your/project
kvelmo serve
```

You should see:
```
Global socket listening at ~/.valksor/kvelmo/global.sock
Web UI available at http://localhost:6337
```

## Step 2: Create a Task File

Create a simple task. In your project directory:

```bash
cat > task.md << 'EOF'
---
title: Add hello endpoint
---

Add a GET /hello endpoint that returns "Hello, World!".

Requirements:
- Return JSON response
- Status code 200
EOF
```

## Step 3: Start the Task

In a new terminal:

```bash
kvelmo start --from file:task.md
```

Output:
```
Task started: Add hello endpoint
Branch created: feature/add-hello-endpoint
State: loaded
```

## Step 4: Plan

Generate a specification:

```bash
kvelmo plan
```

Watch the agent analyze your codebase and create a plan. When done:
```
State: planned
Specification: .kvelmo/specifications/specification.md
```

Review the specification:
```bash
cat .kvelmo/specifications/specification.md
```

## Step 5: Implement

Execute the plan:

```bash
kvelmo implement
```

Watch the agent write code. When done:
```
State: implemented
Files modified: src/routes/hello.js
```

Check the changes:
```bash
git diff
```

## Step 6: Review

Review the implementation:

```bash
kvelmo review
```

If satisfied, you're ready to submit. If not:
```bash
kvelmo undo  # Go back to planned state
# Adjust task description or plan
kvelmo implement  # Try again
```

## Step 7: Submit

Create a PR:

```bash
kvelmo submit
```

Output:
```
PR created: https://github.com/your/repo/pull/123
State: submitted
```

## Using the Web UI

All of the above can be done in the Web UI:

1. Open http://localhost:6337
2. Click **New Task**
3. Enter title: "Add hello endpoint"
4. Enter description (same as above)
5. Click **Start**
6. Click **Plan**
7. Review the specification
8. Click **Implement**
9. Review changes in the Changes panel
10. Click **Submit**

## Common Issues

### "No agent found"

Install an agent CLI:
```bash
# Check if Claude is installed
claude --version
```

### "Branch already exists"

Delete the old branch:
```bash
git branch -D feature/add-hello-endpoint
```

### "Specification doesn't look right"

Add more context to your task description and re-plan.

## Next Steps

- [Web UI Guide](/web-ui/getting-started.md) — Learn the visual interface
- [CLI Reference](/cli/index.md) — Explore all commands
- [Workflow Concepts](/concepts/workflow.md) — Understand the philosophy
