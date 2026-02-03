# Troubleshooting

Solutions for common issues with Mehrhof.

## Quick Fixes

| Problem                         | Quick Fix                                              |
|---------------------------------|--------------------------------------------------------|
| Command not found               | `export PATH="$PATH:$(go env GOPATH)/bin"`             |
| Claude not working              | Ensure Claude CLI is installed: `claude --version`     |
| No active task                  | `mehr start file:task.md`                              |
| Bad implementation              | `mehr undo`                                            |
| Implement says done but no code | `mehr update` (older versions had permission mode bug) |
| Merge conflict                  | Resolve manually, `git add .`, `git commit`            |
| Timeout                         | Increase `agent.timeout` in `.mehrhof/config.yaml`     |
| Start fresh                     | `mehr abandon --yes && mehr start file:task.md`        |
| Web UI won't load               | Check the URL in terminal output, try `http://127.0.0.1:PORT` |
| Buttons not responding          | Refresh browser, check server terminal for errors      |
| Task stuck in state             | Click "Reset State" on dashboard, or `mehr reset`      |

---

## Web UI Issues

### Server Won't Start

**Cause:** Port already in use or configuration error.

Check if another process is using the port:

```bash
# Check port (default 8080)
lsof -i :8080
```

Try a different port:

```bash
mehr serve --port 9090
```

### Browser Won't Load the UI

**Possible causes:**

1. **Wrong URL** — Check the terminal output for the correct URL and port
2. **Server not running** — Ensure `mehr serve` is still running in another terminal
3. **`--open` didn't work** — Copy the URL from the terminal and paste it manually
4. **Firewall or proxy** — Try `http://127.0.0.1:PORT` instead of `localhost:PORT`

### Buttons Not Responding / Timeout

**Cause:** The AI agent is still processing, or the SSE connection dropped.

1. **Check server terminal** — Look for error messages or activity
2. **Refresh the page** — The UI reconnects via SSE automatically
3. **Use Reset** — If the task is stuck, click "Reset State" on the dashboard (or run `mehr reset`)

### Provider Authentication Failed

**Cause:** Missing or expired tokens for GitHub, Jira, Linear, etc.

1. Check that the required environment variable is set (e.g., `GITHUB_TOKEN`, `JIRA_TOKEN`)
2. Verify the token has the required permissions (repo access, issue read)
3. Check `.mehrhof/config.yaml` for provider-specific settings

See [Providers](/providers/index.md) for token setup per provider.

### Remote Access Not Working

**Cause:** Server not bound to accessible address.

```bash
# Bind to all interfaces (for remote access)
mehr serve --host 0.0.0.0

# With authentication (required for remote)
mehr serve --host 0.0.0.0 --auth
```

See [Remote Access](/web-ui/remote-access.md) for full setup.

---

## Installation Issues

### "command not found: mehr"

**Cause:** Binary not in PATH.

```bash
ls $(go env GOPATH)/bin/mehr

export PATH="$PATH:$(go env GOPATH)/bin"

echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
```

### "go: command not found"

**Cause:** Go not installed.

```bash
brew install go

sudo apt install golang-go

go version
```

### Build Fails

**Cause:** Missing dependencies or old Go version.

```bash
go version

go mod tidy

make build
```

---

## Claude CLI Issues

### Claude Not Found

**Cause:** Claude CLI not installed or not in PATH.

```bash
which claude
claude --version

```

### Claude Authentication Issues

**Cause:** Claude CLI not properly configured.

Mehrhof calls Claude CLI, so authentication is handled by Claude:

```bash
claude "Hello"

```

### "Implementation complete" but no code written

**Cause:** Older Mehrhof versions didn't pass `--permission-mode acceptEdits` to Claude during implementation. Claude would default to plan mode, plan the work, exit plan mode, and Mehrhof interpreted this as completion.

**Fix:** Update to latest Mehrhof version which automatically sets the correct permission mode.

```bash
mehr update
```

### "Rate limited"

**Cause:** Your AI provider (e.g., Anthropic) is throttling requests due to API rate limits. This is not a Mehrhof issue — it comes from the provider's usage tiers.

**Fixes:**

1. **Wait and retry** — Rate limits are temporary. Wait a minute and run the command again:

```bash
mehr continue
```

2. **Use budget configuration** to pace requests:

```yaml
# .mehrhof/config.yaml
budget:
  warning_at: 0.8
  on_limit: pause
```

3. **Check your provider's rate limits** — Anthropic and other providers have usage tiers. Higher tiers allow more requests per minute. See your provider's dashboard for current limits.

### CSRF Token Errors

**Cause:** Making POST/PUT/DELETE requests to an authenticated server without a valid CSRF token.

```
Error: CSRF token invalid or missing (HTTP 403)
```

**Fixes:**

1. **Web UI:** Refresh the page — the CSRF token is fetched automatically on load
2. **API clients:** Obtain the token from the login response (`csrf_token` field) or `GET /api/v1/auth/csrf`, then include it as `X-Csrf-Token` header
3. **IDE plugins:** Update to latest version — CSRF tokens are handled automatically
4. **Localhost mode:** CSRF is not enforced on localhost — if you see this error, check your `--host` flag

### Server Rate Limiting (HTTP 429)

**Cause:** Too many requests from your IP address. The server enforces per-IP rate limits when authentication is enabled.

| Endpoint Type | Limit |
|---------------|-------|
| General API | 120 req/min |
| Auth endpoints | 10 req/min |

**Fixes:**

1. **Wait and retry** — Rate limits reset after the time window
2. **Reduce request frequency** — Space out API calls
3. **Localhost mode** — Rate limiting is disabled on localhost

> **Note:** This is different from AI provider rate limits (see "Rate limited" below). Server rate limiting protects the Mehrhof server itself.

### Agent Retry Behavior

Agents automatically retry transient failures (network errors, temporary API issues) before reporting a failure. Default: 3 attempts with 5-second delay.

If you see repeated retry warnings in logs:

```
agent execution failed, retrying (attempt 1/3, retry_delay 5s)
```

**This is normal** — the agent is recovering from a transient issue. If all retries fail, the operation reports the final error.

**Configure retries** in `.mehrhof/config.yaml`:

```yaml
agent:
  retry_count: 3    # Number of attempts (default: 3)
  retry_delay: 5s   # Delay between retries (default: 5s)
```

Set `retry_count: 1` to disable retries.

### Slow Response Times

**Cause:** Large codebase or complex task.

```bash
mehr --verbose plan
```

### High Memory Usage

**Cause:** Large session logs.

```bash
find ~/.valksor/mehrhof/workspaces/*/work/*/sessions/ -mtime +7 -delete
```

Or configure retention:

```yaml
workflow:
  session_retention_days: 7
```

---

## Task Issues

### "No active task"

**Cause:** No task started or not on task branch.

```bash
mehr start file:task.md

git branch -a | grep task/
git checkout task/abc12345
```

### "Task already exists"

**Cause:** Task with same source already registered.

```bash
mehr abandon --yes

mehr start file:task.md
```

### "Cannot find task"

**Cause:** Work directory missing.

```bash
ls ~/.valksor/mehrhof/workspaces/*/work/

mehr start file:task.md
```

### "No specifications generated"

**Cause:** Agent couldn't understand requirements.

```bash
vim task.md

mehr plan
```

### "Specifications are incomplete"

**Cause:** Requirements too vague.

```bash
mehr note "Include error handling for all edge cases"

mehr plan
```

### "No specification files found"

**Cause:** Planning phase not run.

```bash
mehr plan

mehr implement
```

### "Implementation timeout"

**Cause:** Task too complex or slow API.

```yaml
# Increase timeout in config.yaml
agent:
  timeout: 900  # 15 minutes
```

Or break into smaller specifications:

```bash
mehr note "Focus on just the API handler first"
mehr plan
```

### "Generated code has errors"

**Cause:** Agent misunderstood requirements.

```bash
mehr undo

mehr note "Use the existing error handling pattern in internal/errors/"

mehr implement
```

### "Cannot undo: no checkpoints"

**Cause:** At initial state.

```bash
mehr status

mehr abandon
mehr start file:task.md
```

### "Cannot redo: nothing to redo"

**Cause:** Redo stack empty.

```bash
git reflog

git checkout abc1234 -- path/to/file
```

---

## Git Issues

### "Not a git repository"

**Cause:** Running in non-git directory.

```bash
git init

cd /path/to/repo
```

### "Working directory dirty"

**Cause:** Uncommitted changes exist.

```bash
git add .
git commit -m "WIP"

git stash

mehr finish
```

### "Merge conflict"

**Cause:** Conflicts during `mehr finish`.

```bash
git status

vim conflicting-file.go

git add conflicting-file.go

git commit
```

### "Cannot delete branch"

**Cause:** Branch is protected or checked out elsewhere.

```bash
git checkout main

git branch -D task/abc12345
```

---

## Diagnostics

### Check Version

```bash
mehr version
```

### Enable Verbose Mode

```bash
mehr --verbose <command>
```

### View Session Logs

```bash
cat ~/.valksor/mehrhof/workspaces/*/work/*/sessions/*.yaml
```

### Validate Configuration

```bash
mehr config validate
```

### Configuration Issues

**"Settings not applied"**

```bash
mehr --verbose plan

cat .mehrhof/config.yaml
```

**"Invalid configuration"**

```bash
mehr config validate

cat .mehrhof/config.yaml | python -c "import yaml,sys; yaml.safe_load(sys.stdin)"
```

---

## Report Issues

1. Note the error message
2. Check Mehrhof version: `mehr version`
3. Gather relevant logs
4. Report at [github.com/valksor/go-mehrhof/issues](https://github.com/valksor/go-mehrhof/issues)
