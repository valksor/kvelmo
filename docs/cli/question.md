# mehr question

Ask the AI agent a question during planning, implementation, or review without changing the workflow state.

## Synopsis

```bash
mehr question [message]
mehr ask [message]
mehr q [message]
```

## Description

The `question` command sends a question to the currently active AI agent and receives a response. Unlike `note` (which only saves text), the `question` command runs the AI agent to get an intelligent response based on:

- Current task title and state
- Latest specification content
- Recent session history
- Recent file changes (optional)

The key difference from `note` is that `question` **returns an AI response** while keeping the workflow in the same state. You can continue planning or implementation after getting an answer.

## Arguments

| Argument  | Description               |
|-----------|---------------------------|
| `message` | Question to ask the agent |

## When to Use

### During Planning

Ask why the agent chose a particular approach:

```bash
mehr plan
mehr question "Why did you choose GraphQL over REST?"
# Agent responds with reasoning
mehr plan  # Continues planning
```

### During Implementation

Get clarification on implementation choices:

```bash
mehr implement
# (interrupt with Ctrl+C after some output)
mehr question "What's the reason for using channels here?"
mehr implement  # Continues implementation
```

### During Review

Understand review findings:

```bash
mehr review
mehr question "Can you explain this security issue in more detail?"
mehr review  # Continues review
```

## What Happens

1. **Validation**
   - Active task must exist
   - State must be `planning`, `implementing`, or `reviewing`

2. **Context Building**
   - Task title and current state
   - Latest specification content
   - Recent session Q&A history
   - Optional git diff for recent changes

3. **Agent Execution**
   - Agent runs with streaming response
   - Response displayed in real-time
   - Tokens consumed (unlike `note`)

4. **Session Recording**
   - User question saved to session
   - Agent response saved to session
   - Q&A available in future contexts

5. **State Preserved**
   - Workflow state unchanged
   - Continue with `plan`, `implement`, or `review`

## Examples

### Clarify Approach Choice

```bash
$ mehr question "Why use PostgreSQL instead of MySQL?"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  QUESTION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Why use PostgreSQL instead of MySQL?

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  RESPONSE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

I chose PostgreSQL for several reasons:

1. **Advanced Features** - PostgreSQL offers:
   - Better JSON support with JSONB
   - Full-text search built-in
   - Advanced indexing (GIN, GiST)
   - Array types and hstore

2. **Consistency** - Your codebase already uses PostgreSQL for:
   - User data (internal/users/postgres.go)
   - Sessions (internal/session/postgres.go)
   - Analytics (internal/analytics/postgres.go)

3. **Transactional DDL** - Schema migrations are transactional,
   reducing deployment risk.

4. **License** - PostgreSQL is more permissive for potential
   future proprietary use cases.

Would you like me to reconsider this choice?
```

### Get Implementation Details

```bash
$ mehr question "How does the authentication middleware work?"

The authentication middleware follows a standard pattern:

1. **JWT Extraction** - Token extracted from `Authorization` header
2. **Validation** - Signature verified using JWT secret
3. **Claims Extraction** - User ID and roles extracted
4. **Context Injection** - User info added to request context

See `internal/server/middleware/auth.go:45-82` for the full
implementation.
```

### Ask About Testing Strategy

```bash
$ mehr question "What's the testing approach for this service?"

The testing strategy includes:

1. **Unit Tests** - `internal/service/*_test.go`
   - Mock dependencies
   - Table-driven tests
   - ~80% coverage target

2. **Integration Tests** - `test/integration/`
   - Real database (test containers)
   - HTTP layer tests
   - End-to-end flows

3. **Property Tests** - Uses `go-testlimits`
   - Random input generation
   - Invariant checking
```

## Question vs Note

| Feature               | `question`  | `note`   |
|-----------------------|-------------|----------|
| **AI Response**       | ✅ Yes       | ❌ No     |
| **Tokens Consumed**   | ✅ Yes       | ❌ No     |
| **Context Awareness** | ✅ Full      | ❌ None   |
| **State Change**      | ❌ No        | ❌ No     |
| **Use Case**          | Get answers | Add info |

## Interactive Mode

```bash
$ mehr question

Active Task: abc12345 (state: implementing)

Enter your question (press Ctrl+D to finish):

> How does the cache invalidation work?

The cache uses a write-through invalidation strategy...

> What about distributed scenarios?

Good question! For distributed scenarios...

```

## Aliases

Three aliases are available:

| Alias           | Command        |
|-----------------|----------------|
| `mehr ask`      | Ask a question |
| `mehr q`        | Quick question |
| `mehr question` | Full command   |

All work identically:

```bash
mehr ask "Why this approach?"
mehr q "Why this approach?"
mehr question "Why this approach?"
```

## Back-Questions

If the agent needs clarification, it may ask a back-question:

```bash
$ mehr question "Should I add caching?"

Agent: That depends on a few factors:

1. What's the expected read/write ratio?
2. What's the data volatility?
3. What's your latency budget?

Can you provide more context about your use case?
```

When the agent asks a back-question:
1. The workflow transitions to `waiting` state
2. Use `answer` or `note` to respond
3. Use `plan`/`implement`/`review` to continue

## Session History

Questions and answers are saved to the session file:

```
~/.valksor/mehrhof/workspaces/<project-id>/sessions/<id>/<timestamp>.jsonl
```

This provides:
- Full conversation history
- Context for future questions
- Audit trail of decisions

## Error Cases

### No Active Task

```bash
$ mehr question "Why this approach?"

Error: no active task
Use 'mehr start <ref>' to begin a task
```

### Invalid State

```bash
$ mehr question "Why this approach?"

Error: cannot ask questions in 'idle' state
Questions are only available during planning, implementing, or reviewing
```

### No Conductor

```bash
$ mehr question "Why this approach?"

Error: conductor not initialized
Ensure the task is in an active workflow state
```

## Workflow Integration

### Planning Workflow

```bash
mehr start file:task.md
mehr plan
mehr question "Can you explain this specification choice?"
# Continue planning based on answer
mehr plan
```

### Implementation Workflow

```bash
mehr implement
# See something unclear
mehr question "What's the pattern being used here?"
# Continue with clearer understanding
mehr implement
```

### Review Workflow

```bash
mehr review
mehr question "Is this a critical security issue?"
# Decide based on severity
mehr finish
```

## Best Practices

1. **Be specific** - Clear questions get better answers
2. **Ask about reasoning** - Understand "why", not just "what"
3. **Clarify trade-offs** - Agent can explain alternatives
4. **Check understanding** - Verify assumptions before continuing
5. **Document decisions** - Important Q&A is saved to sessions

## Web UI

Prefer a visual interface? See [Web UI: Quick Question](/web-ui/questions.md).

## See Also

- [note](note.md) - Add notes without AI response
- [plan](plan.md) - Create specifications with AI
- [implement](implement.md) - Generate code with AI
- [review](review.md) - Review code with AI
- [Workflow](/concepts/workflow.md) - Understanding states
