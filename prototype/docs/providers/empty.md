# Empty Provider

**Schemes:** `empty:`

**Capabilities:** `read`

Creates tasks without an external source. Tasks start with an empty description that you populate using the `mehr note` command.

## Usage

```bash
# Create empty task with a simple ID
mehr start empty:FEATURE-1
mehr note "Add user authentication with OAuth2"
mehr plan

# Create empty task with descriptive title
mehr start empty:"Implement user authentication"
mehr note "Add OAuth2 support for login/logout"
mehr plan
```

## Workflow

The empty provider is designed for quick task creation without file management:

1. **Create task**: `mehr start empty:TASK-ID` or `mehr start empty:"Description"`
2. **Add notes**: `mehr note "Your task description here"`
3. **Plan**: `mehr plan` (requires notes first)
4. **Implement**: `mehr implement`

## Guard Behavior

The `mehr plan` command requires a description before proceeding. If you try to plan without adding notes, you'll see:

```
cannot plan: task has no description

Use 'mehr note' to add a task description first:
  mehr note "Implement feature X with REST API"

Then run 'mehr plan' again.
```

## Use Cases

- **Quick ad-hoc tasks**: Create tasks on-the-fly without managing markdown files
- ** prototyping**: Sketch out ideas and let the AI agent help flesh them out
- **Interrupt-driven work**: Capture incoming requests immediately, add context later
- **Learning**: Experiment with Mehrhof's workflow without setting up files

## Comparison with File Provider

| Aspect                  | Empty Provider      | File Provider              |
|-------------------------|---------------------|----------------------------|
| **Task source**         | None (in-memory)    | Markdown file              |
| **Initial description** | Empty               | From file content          |
| **Persistence**         | Notes only          | File snapshot + notes      |
| **Best for**            | Quick, ad-hoc tasks | Documented, reusable tasks |
| **Git integration**     | Full support        | Full support               |

## Examples

### Simple Task ID

```bash
# Create task
mehr start empty:AUTH-1

# Add description
mehr note "Implement JWT authentication with refresh tokens"

# Plan and implement
mehr plan
mehr implement
```

### Descriptive Title

```bash
# Create with full description as title
mehr start empty:"Add OAuth2 Google login"

# Plan directly (title serves as context)
mehr note "Support Google OAuth2 with user profile sync"
mehr plan
```

### Multiple Empty Tasks

```bash
# Create several related tasks
mehr start empty:AUTH-1
mehr note "Implement login endpoint"

mehr start empty:AUTH-2
mehr note "Implement logout endpoint"

mehr start empty:AUTH-3
mehr note "Add password reset flow"

# Work on them sequentially
mehr plan
mehr implement
```

## Branch Naming

The empty provider uses the identifier for branch naming:

```bash
mehr start empty:FEATURE-1
# Creates branch: feature/FEATURE-1--implement-x

mehr start empty:"Add user authentication"
# Creates branch: feature/add-user-authentication
```

You can override with flags:

```bash
mehr start empty:FEATURE-1 --key "AUTH" --slug "oauth-login"
# Creates branch: feature/AUTH--oauth-login
```
