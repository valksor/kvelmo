# Planning

The planning phase is where the AI analyzes your requirements and creates detailed specifications — a blueprint before any code is written.

## What Planning Does

When you click **"Plan"**, the AI:

1. **Reads your task description** - Understands requirements and constraints
2. **Analyzes your codebase** - Scans existing files to understand patterns and structure
3. **Creates specifications** - Writes detailed implementation plans
4. **May ask questions** - Requests clarification if something is unclear

**Key benefit:** You can review and adjust the plan before any code changes are made.

## Starting Planning

After creating a task, click the **"Plan"** button in the Active Task card:

The Active Task card shows your task details and action buttons. Click **Plan** to start the planning phase.

## Planning Phase Workflow

```text
┌────────────┐     ┌────────────┐     ┌────────────────┐     ┌─────────────┐     ┌─────────────────┐     ┌───────────────────────────────┐
│ Idle State │ ──▶ │ Click Plan │ ──▶ │ Planning State │ ──▶ │ AI Analyzes │ ──▶ │ Specs Created   │ ──▶ │ Back to Idle - Ready to Impl. │
└────────────┘     └────────────┘     └────────────────┘     └─────────────┘     └─────────────────┘     └───────────────────────────────┘
```

## Real-Time Progress

Watch the AI work in the **Agent Output** section:

The **Agent Output** section shows real-time progress as the AI reads your task description, analyzes the codebase structure, reviews existing patterns, and generates specification files.

## The Planning State

During planning, the task state changes to **"Planning"**:

| State        | What's Happening                   | What You Can Do                     |
|--------------|------------------------------------|-------------------------------------|
| **Planning** | AI is analyzing and creating specs | Watch progress, wait for completion |
| **Waiting**  | AI has a question                  | Answer in the Questions section     |
| **Idle**     | Planning complete, specs ready     | Review specs, proceed to implement  |

## Reviewing Specifications

After planning completes, review the generated specifications:

The **Specifications** section lists all generated specification files with their key topics. Click **View Full Content** to read the complete specifications, or **Add another specification** to create additional specs.

## Iterative Planning

You can run planning multiple times to build on existing specifications:

```text
┌────────────┐     ┌─────────────────────────┐     ┌─────────────┐     ┌────────────────────────────┐
│ Click Plan │ ──▶ │ specification-1.md      │ ──▶ │ Review spec │ ──▶ │ Add note: error handling   │
└────────────┘     └─────────────────────────┘     └─────────────┘     └──────────────┬─────────────┘
                                                                                      │
                                                                                      ▼
┌─────────────────────────────────────┐     ┌─────────────────────────┐     ┌──────────────────┐
│ Both specs used in implementation   │ ◀── │ specification-2.md      │ ◀── │ Click Plan again │
└─────────────────────────────────────┘     └─────────────────────────┘     └──────────────────┘
```

**Steps:**
1. Click **"Plan"** → Creates `specification-1.md`
2. Review the specification
3. Add a note: "Also add comprehensive error handling"
4. Click **"Plan"** again → Creates `specification-2.md`

Both specifications will be used during implementation. Each new spec builds on the previous ones.

## Adding Context Before Planning

Add notes to provide additional context before planning:

1. Click **"Add Note"** button
2. Enter your context or requirements

The **Add Note** dialog accepts markdown-formatted text. Include specific requirements, existing dependencies to use, or constraints the AI should follow during planning.

## Answering Questions

If the AI needs clarification during planning, it will enter the **"Waiting"** state and show a question:

When the AI needs clarification, it enters the **Waiting** state and displays a question with suggested options. You can click a suggested answer or provide a custom response using **Add Note**.

Use the **"Add Note"** button to answer, then click **"Plan"** again.

## What Makes a Good Task Description

The AI can only plan based on what you provide:

**Good:**
````markdown
---
title: Add user authentication
---

Add OAuth2 login using Google as the provider.

## Requirements
- Use the existing OAuth2 library we already depend on
- Store user sessions in PostgreSQL
- Add logout functionality

## Endpoints to create
- GET /auth/login - Redirect to Google OAuth
- GET /auth/callback - Handle OAuth return
- POST /auth/logout - Clear session cookie
````

**Vague:**
```
Add login
```

## Planning Best Practices

1. **Be specific** - Detailed requirements produce better specs
2. **Add notes** - Use the Notes section for additional context
3. **Review specs** - Always review before implementing
4. **Iterate** - Create multiple specs if needed
5. **Answer questions** - Provide clear responses when AI asks

## When to Re-Plan

**Re-plan if:**
- Requirements changed
- First plan missed important details
- AI misunderstood your request
- You thought of new features

**Don't re-plan if:**
- Just need to tweak implementation details (add a note instead)
- Plan looks correct but incomplete (add a note with "also add X")

## Next Steps

After planning completes:

- [**Implementing**](implementing.md) - Execute the specifications
- [**Notes**](notes.md) - Add more context before implementation
- [**Undo**](undo-redo.md) - Revert planning if needed

---

## Also Available via CLI

Run planning from the command line for terminal-based workflows or automation.

See [CLI: plan](/cli/plan.md) for all flags and options.
