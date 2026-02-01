# Glossary

Plain-language definitions for Mehrhof concepts.

---

## Core Concepts

### Task

A description of what you want to create or change. Tasks can be written in plain language—no technical expertise required. You might describe a feature you want, a bug you've noticed, or a change you need.

**Example**: "Add a dark mode toggle to the settings page"

### Plan

A structured breakdown of how the work will be done. Before any changes are made, Mehrhof creates a plan that outlines the steps. You review this plan and approve it before moving forward.

**Why it matters**: Plans give you visibility into what will happen before it happens. No surprises.

### Change Set

The actual modifications produced by the workflow—code files, documentation, configuration. After the plan is executed, you get a change set to review.

**What you do**: Review the changes, ask questions, request adjustments, or approve.

### Review

Human verification before changes become permanent. Nothing ships without your explicit approval. You can undo, adjust, add notes, and iterate until you're satisfied.

**Key point**: Mehrhof assists, but you decide.

### Checkpoint

A save point you can return to if something goes wrong. Mehrhof automatically creates checkpoints at each workflow step. Use undo/redo to navigate between them.

**Like a safety net**: Made a mistake? Roll back.

### Workflow

The structured process: task → plan → create → review → finish. Each step is distinct and requires your input before moving to the next.

---

## Technical Terms

### Agent CLI

The external AI tool Mehrhof uses to handle text transformations. You install your agent CLI separately (Claude is recommended and primary-supported; Codex and custom agents also available)—Mehrhof orchestrates it but does not include or provide access to it.

**Important**: Mehrhof wraps your local agent installation. It does not provide AI access.

### Provider

Where tasks come from. Mehrhof can pull tasks from files, GitHub issues, Jira tickets, Linear issues, Notion pages, and more. Each source type is called a "provider" (e.g., the GitHub provider, the Jira provider).

### Branch

A separate version of your project where changes are made. Mehrhof automatically creates branches for each task, keeping your main project safe until changes are reviewed and approved.

### Merge

Combining your reviewed changes back into the main project. This happens when you finish a task and approve the changes.

---

## Workflow States

| State | What's Happening |
|-------|------------------|
| **Idle** | Task is registered, waiting for your next action |
| **Planning** | Creating a structured plan from your task description |
| **Creating** | Executing the plan to produce changes |
| **Reviewing** | Quality checks running on the changes |
| **Done** | Task completed and changes merged |

---

## Common Actions

| Action | What It Does |
|--------|--------------|
| **Start** | Begin a new task from a description |
| **Plan** | Create a structured plan |
| **Implement** | Execute the plan to create changes |
| **Review** | Run quality checks |
| **Finish** | Complete and merge changes |
| **Undo** | Go back to a previous checkpoint |
| **Redo** | Go forward to a later checkpoint |
| **Note** | Add context for the workflow |
| **Abandon** | Cancel task without keeping changes |

---

## See Also

- [Workflow Concepts](concepts/workflow.md) — Detailed explanation of the process
- [Web UI Guide](web-ui/getting-started.md) — Visual walkthrough
- [CLI Reference](cli/index.md) — All commands and flags
