# mehr simplify

Simplify content based on the current workflow state.

## Synopsis

```bash
mehr simplify [flags] [files...]
```

## Description

The `simplify` command automatically determines what to simplify based on where you are in the workflow:

- **Pre-plan (no specs)**: Simplifies task input/description for clarity
- **After planning**: Simplifies specification files for better maintainability
- **After implementing**: Simplifies code changes while preserving functionality

The command uses the configured AI agent to analyze and refine your content, making it clearer and more maintainable without changing functionality.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--agent` | string | "" | Use a specific agent for simplification |
| `--no-checkpoint` | bool | false | Skip creating a checkpoint before simplifying (not recommended) |
| `--verbose` | bool | false | Show detailed simplification process |
| `--standalone` | bool | false | Simplify without active task (see Standalone Mode) |
| `--branch` | string | "" | Compare current branch vs base (standalone only) |
| `--range` | string | "" | Compare commit range (standalone only) |
| `--context` | int | 3 | Lines of context in diff (standalone only) |

## Examples

### Simplify task input

```bash
# Start a task and simplify the description
mehr start "Add user auth with JWT"
mehr simplify

# View the simplified description
mehr note
```

Output:

```
Simplifying task input...
Agent simplifying input...
Task input simplified

  Task input simplified
```

### Simplify specifications

```bash
# After planning, simplify the specs for clarity
mehr plan
mehr simplify

# Check the simplified specifications
cat .mehrhof/work/<task-id>/specifications/specification-1.md
```

Output:

```
Simplifying planning output...
Found 1 specification(s) to simplify
Agent simplifying specifications...
Simplified 1 specification(s)

  Specifications simplified: 1
```

### Simplify code

```bash
# After implementing, simplify the code
mehr implement
mehr simplify

# View what changed
git diff
```

Output:

```
Simplifying implementation output...
Found 3 file(s) to simplify
Agent simplifying code...
Simplified internal/auth/handler.go
Simplified internal/auth/middleware.go
Simplified 3 file(s)

  Files simplified: 3
```

### Use specific agent

```bash
mehr simplify --agent claude-opus
```

### Skip checkpoint (not recommended)

```bash
mehr simplify --no-checkpoint
```

## What Gets Simplified

### Pre-Plan: Task Input

Simplifies the original task description to be clearer and more actionable.

**Before:**
```
Add user auth with JWT tokens and OAuth providers
also handle refresh tokens and session management
```

**After:**
```
Implement JWT-based user authentication with OAuth integration.
Support refresh token rotation and secure session management.
```

### After Planning: Specifications

Simplifies specification files while preserving all technical details.

**Before:**
```markdown
## Plan
1. Add auth package
2. Implement JWT logic
3. Add middleware
4. Handle refresh tokens
## Unknowns
1. Should we use a library?
```

**After:**
```markdown
## Plan
1. Add `internal/auth/` package for authentication
2. Implement JWT token generation and validation in `internal/auth/jwt.go`
3. Add authentication middleware to `internal/middleware/auth.go`
4. Implement refresh token rotation with secure storage
## Complete Condition
- manual: Test login/logout flow with valid and invalid tokens
- run: go test ./internal/auth/...
```

### After Implementation: Code

Simplifies code changes while preserving exact functionality.

**Before:**
```go
func GetToken(u User) (string, error) {
    t, e := db.GetToken(u.ID)
    if e != nil {
        return "", e
    }
    return t, nil
}
```

**After:**
```go
// GetToken retrieves the active JWT token for the user.
// Returns empty string if token has expired.
func GetToken(user User) (string, error) {
    token, err := db.GetToken(user.ID)
    if err != nil {
        return "", fmt.Errorf("get token for user %s: %w", user.ID, err)
    }
    return token, nil
}
```

## Safety

### Automatic Checkpoints

By default, `mehr simplify` creates a git checkpoint before modifying any files. This provides a safety net:

```bash
# Simplify with checkpoint (default)
mehr simplify

# If something goes wrong, undo
mehr undo

# Or redo if you change your mind
mehr redo
```

### Skip Checkpoint (Advanced)

Use `--no-checkpoint` to skip checkpoint creation:

```bash
mehr simplify --no-checkpoint
```

**Warning:** This is not recommended. Without checkpoints, you cannot easily undo simplification changes.

## Custom Instructions

Configure project-specific simplification standards in `.mehrhof/config.yaml`:

```yaml
workflow:
    simplify:
        instructions: |
            Follow our project's coding standards:
            - Use descriptive names (no abbreviations)
            - Keep functions under 50 lines
            - Add docstrings to public APIs
            - Prefer composition over inheritance
            - Use idiomatic Go patterns
```

These instructions are appended to all simplification prompts.

## State Detection

The command automatically detects what to simplify based on:

| State | Simplifies | Condition |
|-------|-----------|-----------|
| No specifications | Task input | `work/<task-id>/specifications/` is empty |
| Specifications exist | Specifications | Found `.md` files in specifications directory |
| Implemented files exist | Code | Specification metadata lists implemented files |

Run `mehr status` to see current state and what would be simplified.

## Troubleshooting

### Nothing to simplify

```
Error: no active task
```

Start a task first:
```bash
mehr start "Your task here"
```

### No specifications found

```
Error: no specifications found to simplify
```

Run planning first:
```bash
mehr plan
```

### No implemented files

```
Error: no implemented files found - run implement first
```

Run implementation first:
```bash
mehr implement
```

## Best Practices

1. **Simplify early and often** - Simplify after planning to get clearer specs
2. **Review changes** - Always check what was simplified with `git diff`
3. **Use checkpoints** - Don't skip checkpoints unless you're sure
4. **Customize standards** - Add project-specific instructions to config.yaml
5. **Iterate** - You can simplify multiple times to progressively refine content

## Workflow Integration

### Typical Planning Workflow

```bash
mehr plan
mehr simplify           # Refine the specifications
mehr implement        # Implement based on clear specs
```

### Iterative Refinement

```bash
mehr plan
mehr implement
mehr simplify           # Simplify the code
mehr review            # Review the simplified code
mehr simplify           # Simplify review comments if needed
```

### Error Recovery

```bash
mehr implement
mehr simplify           # Oops, made it worse
mehr undo              # Revert the simplification
mehr note "Keep it simple"
mehr simplify           # Try again with better instructions
```

## Standalone Mode

Simplify code changes **without an active task**. Useful for quick refactoring, cleaning up uncommitted changes, or simplifying feature branches.

### Synopsis

```bash
mehr simplify --standalone [flags] [files...]
```

### Standalone Examples

**Simplify uncommitted changes (default):**
```bash
mehr simplify --standalone
```

**Simplify current branch vs main:**
```bash
mehr simplify --standalone --branch
```

**Simplify current branch vs specific base:**
```bash
mehr simplify --standalone --branch develop
```

**Simplify specific commit range:**
```bash
mehr simplify --standalone --range HEAD~3..HEAD
```

**Simplify specific files:**
```bash
mehr simplify --standalone src/foo.go src/bar.go
```

**Use specific agent:**
```bash
mehr simplify --standalone --agent opus
```

**Skip checkpoint (not recommended):**
```bash
mehr simplify --standalone --no-checkpoint
```

### Standalone Output

```bash
$ mehr simplify --standalone

ℹ Simplifying uncommitted changes (staged + unstaged)...
Creating checkpoint...
Agent simplifying code...

✓ Simplification complete

Summary:
Refactored handler functions to reduce complexity and improve readability.
Extracted common validation logic into a shared helper function.

Suggested Changes:
  [MODIFY] internal/handler.go
  [MODIFY] internal/validation.go

Tokens: 2345 input, 890 output ($0.0067)
```

### When to Use Standalone Mode

- **Quick refactoring**: Simplify code without starting a full task
- **Pre-commit cleanup**: Clean up changes before committing
- **Branch preparation**: Simplify feature branches before PR
- **Code maintenance**: Improve code quality in existing files

### Configuration

Set a default branch for standalone simplify in `.mehrhof/config.yaml`:

```yaml
git:
  default_branch: develop  # Used when --branch is specified without a value
```

## See Also

- [plan](plan.md) - Create specifications
- [implement](implement.md) - Generate code
- [review](review.md) - Review code
- [undo](undo.md) - Revert changes
- [status](status.md) - View task state
