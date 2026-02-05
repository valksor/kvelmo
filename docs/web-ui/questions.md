# Quick Question

Ask the AI agent questions during planning, implementation, or review without changing the workflow state.

## What Questions Do

The **Quick Question** feature lets you have an interactive conversation with the AI agent while work is in progress. Unlike **Notes** (which only save text), **Quick Question** runs the AI agent to get intelligent responses based on:

- Current task context
- Latest specifications
- Recent session history
- Recent file changes

**Key difference:** The agent responds to your question while preserving the current workflow state.

## When to Use

### During Planning

Ask why the agent chose a particular approach:

In the Active Task card during planning, enter your question in the **Quick Question** field and click **Ask Question** to get a response from the AI agent.

### During Implementation

Get clarification on implementation choices:

During implementation, you can ask questions about the code being generated while the Agent Output streams in real-time above the question input.

### During Review

Understand review findings:

During review, you can ask questions about specific findings to understand them better before deciding how to proceed.

## Quick Question Location

The Quick Question input appears in the **Active Task** card when the state is:
- **Planning** - While creating specifications
- **Implementing** - While writing code
- **Reviewing** - While reviewing code

The **Quick Question** input appears in the Active Task card when in Planning, Implementing, or Reviewing states. A toggle lets you switch to the Notes view. Type your question and click **Ask Question** to get a response.

## How It Works

### Step 1: Ask Your Question

Type your question in the Quick Question input and click **"Ask Question"**:

Type your question in the input field and click **Ask Question**.

### Step 2: Agent Responds (Streaming)

The agent's response streams in real-time:

The **Agent Response** panel streams the AI's answer in real-time, formatted with markdown (headers, lists, code blocks). Click **Close** when done reading.

### Step 3: State Preserved

After getting your answer, the workflow state remains unchanged:

| Before Question | After Question |
|-----------------|----------------|
| Planning        | Planning       |
| Implementing    | Implementing   |
| Reviewing       | Reviewing      |

You can continue where you left off:
- **Planning** - Click **"Plan"** to continue
- **Implementing** - Click **"Implement"** to continue
- **Reviewing** - Click **"Review"** to continue

## Question Examples

### Clarify Design Choice

```
Q: Why did you use a channel for the event bus?

A: Channels provide:
1. Built-in concurrency safety
2. Goroutine-safe communication
3. Buffered async messaging
4. Idiomatic Go patterns

An alternative would be a library like watermill,
but channels keep dependencies minimal.
```

### Understand Implementation

```
Q: How does the authentication middleware work?

A: The middleware follows this flow:

1. Extract JWT from Authorization header
2. Verify signature using JWT secret
3. Extract user ID and role claims
4. Inject user info into request context

See internal/server/middleware/auth.go:45-82
```

### Ask About Testing

```
Q: What's the testing strategy?

A: The approach includes:

1. **Unit Tests** - Table-driven tests in *_test.go
2. **Integration Tests** - Real database in test/integration/
3. **Property Tests** - Using go-testlimits for invariants
```

## Questions vs Notes

| Feature             | Quick Question | Notes            |
|---------------------|----------------|------------------|
| **AI Response**     | ✅ Yes          | ❌ No             |
| **Context Aware**   | ✅ Full context | ❌ None           |
| **Tokens Consumed** | ✅ Yes          | ❌ No             |
| **State Change**    | ❌ No           | ❌ No             |
| **Use When**        | Need an answer | Want to add info |

**Quick Question** is for asking the AI and getting responses; **Add Note** is for adding context without consuming tokens. Use questions when you need answers, notes when you want to add information.

## Session History

Your questions and the agent's responses are saved to the session file:

```
~/.valksor/mehrhof/workspaces/<project-id>/sessions/<id>/<timestamp>.jsonl
```

This provides:
- Full conversation history
- Context for future questions
- Audit trail of decisions
- Recovery if interrupted

## Back-Questions

If the agent needs clarification, it may ask a back-question:

If the agent needs more information, it asks a back-question. The workflow enters **Waiting** state. Use **Add Note** to respond, then click **Continue** to resume.

When the agent asks a back-question:
1. The workflow transitions to **"Waiting"** state
2. Use **"Add Note"** to respond
3. Click **"Continue"** to resume

## When Questions Are Not Available

Quick Question is only available during active AI work:

| State            | Quick Question Available           |
|------------------|------------------------------------|
| **Planning**     | ✅ Yes                              |
| **Implementing** | ✅ Yes                              |
| **Reviewing**    | ✅ Yes                              |
| **Idle**         | ❌ No (use notes instead)           |
| **Waiting**      | ❌ No (answer agent question first) |
| **Done**         | ❌ No (task complete)               |

## Error Cases

### No Active Task

If no task is active, you'll see a prompt to create one using **Quick Task** or **From Provider** before you can ask questions.

### Invalid State

If the task is in Idle, Waiting, or Done state, questions are not available. The dialog explains that questions work only during Planning, Implementing, or Reviewing states, and suggests using **Add Note** instead.

## Best Practices

1. **Be specific** - Clear questions get better answers
2. **Ask about reasoning** - Understand "why", not just "what"
3. **Clarify trade-offs** - Agent can explain alternatives
4. **Check understanding** - Verify assumptions before continuing
5. **Use for learning** - Explore unfamiliar code patterns

## Workflow Examples

### Planning Workflow with Questions

```
1. Create task
2. Click [Plan]
3. Agent starts planning
4. Ask: "Why this approach?"
5. Get explanation
6. Click [Plan] to continue
```

### Implementation Workflow with Questions

```
1. Click [Implement]
2. Agent starts implementing
3. See something unclear
4. Ask: "What pattern is this?"
5. Get clarification
6. Click [Implement] to continue
```

### Review Workflow with Questions

```
1. Click [Review]
2. Agent finds an issue
3. Ask: "Is this critical?"
4. Get severity assessment
5. Decide: fix or accept
6. Click [Finish] or [Implement]
```

## Keyboard Shortcuts

| Key          | Action                               |
|--------------|--------------------------------------|
| `Ctrl+Enter` | Submit question (when input focused) |
| `Esc`        | Close response dialog                |

## API Endpoint

The Quick Question feature uses the following API:

```
POST /api/v1/workflow/question
```

**Request:**
```json
{
  "question": "Why did you choose this approach?"
}
```

**Response:** SSE stream with agent response

---

## Also Available via CLI

Ask questions from the command line when working in a terminal.

See [CLI: question](/cli/question.md) for all flags and options.

## Next Steps

- [**Planning**](planning.md) - Ask questions during specification creation
- [**Implementing**](implementing.md) - Ask questions during code generation
- [**Reviewing**](reviewing.md) - Ask questions during code review
- [**Notes**](notes.md) - Add context without AI response
