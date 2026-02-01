# PR Review State Management

## Overview

PR reviews store state incrementally in markdown comments to enable:
- Tracking issues across PR iterations
- Detecting when previously reported issues are fixed
- Acknowledging fixes in review comments

The state is embedded directly in PR comments as hidden HTML comments, allowing the review history to travel with the PR without requiring local files or external storage.

## Architecture

### State Flow

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  Initial Review │ ───> │   Re-review     │ ───> │  Re-review #2   │
│                 │      │                 │      │                 │
│  • 5 issues     │      │  • Extract      │      │  • 2 new issues │
│  • State saved  │      │    previous     │      │  • 3 fixed      │
│                 │      │    state        │      │  • Update state │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

### State Structure

```go
type PRReviewState struct {
    Provider         string        // "github", "gitlab", "bitbucket", "azuredevops"
    PRNumber         int           // PR/MR number
    CommitSHA        string        // Last reviewed commit SHA
    HeadBranch       string        // Head branch name
    LastReviewAt     time.Time     // When last review ran
    Issues           []ReviewIssue // Current issues found
    Signature        string        // HMAC signature for tamper detection
    ReviewedDiffHash string        // Hash of the reviewed diff for validation
    AgentVersion     string        // Agent or config version
}
```

### Issue Tracking

Each issue has a stable ID computed as a hash of `(file + line + normalized message)`:

```go
type ReviewIssue struct {
    ID       string // SHA256 hash for stability
    File     string // File path
    Line     int    // Line number (0 if not line-specific)
    Category string // "correctness", "security", "performance", "style"
    Severity string // "critical", "high", "medium", "low"
    Message  string // The issue description
    Status   string // "open", "fixed", "acknowledged"
}
```

## Security

### State Signing

All state is HMAC-SHA256 signed to prevent tampering:

- **Requirement**: `MEHRHOF_STATE_SECRET` environment variable (min 32 characters)
- **Signature**: Embedded in state JSON as `sig` field
- **Verification**: Fails on tampered state with error message

```bash
# Generate secure secret using mehr
mehr generate-secret

# Or export directly
export MEHRHOF_STATE_SECRET="$(mehr generate-secret)"
```

### State Encoding

State is base64-encoded to prevent XSS from AI-generated content:

```go
// Before (vulnerable):
<!-- MEHRHOF_REVIEW_STATE {"provider":"github"...} -->

// After (secure):
<!-- MEHRHOF_REVIEW_STATE eyJwcm92aWRlciI6ImdpdGh...
```

### Memory Protection

Large diffs (>10MB) are truncated before hashing to prevent memory exhaustion:

```go
const maxDiffSize = 10 * 1024 * 1024 // 10MB

func hashDiffPatch(patch string) string {
    if len(patch) > maxDiffSize {
        h := sha256.New()
        h.Write([]byte(patch[:maxDiffSize]))
        h.Write([]byte(fmt.Sprintf("...truncated...len=%d", len(patch))))
        return hex.EncodeToString(h.Sum(nil))
    }
    h := sha256.Sum256([]byte(patch))
    return hex.EncodeToString(h[:])
}
```

## Data Flow

### 1. Initial Review

```
AI Agent → ParsedReview → BuildPRReviewState → EmbedStateInComment → Post Comment
```

- AI agent generates review with issues
- State built with all issues marked as "open"
- State signed with HMAC
- Base64-encoded and embedded in comment

### 2. Re-review

```
Fetch Previous Comment → ExtractStateFromComment → VerifyState → ComputeReviewDelta → FormatReviewComment
```

- Previous comment with state found
- State extracted and signature verified
- Delta computed: new issues vs. fixed issues vs. unchanged
- New comment formatted with acknowledgments

### 3. Issue Delta Computation

```go
type ReviewDelta struct {
    NewIssues   []ReviewIssue // Issues not seen before → POST
    FixedIssues []ReviewIssue // Issues that existed before but now gone → ACKNOWLEDGE
    Unchanged   []ReviewIssue // Issues still present → SKIP
}
```

## Configuration

### Environment Variables

```bash
# Required for CI/CD usage
export MEHRHOF_STATE_SECRET="your-32-char-secret-here"
```

### Workspace Config

```yaml
# .mehrhof/config.yaml
agent:
  instructions: |
    PR review state management enabled.

review:
  acknowledge_fixes: true   # Acknowledge when issues are fixed
  update_existing: true     # Edit existing comment vs post new
```

## CLI Usage

### Basic Usage

```bash
# Auto-detect provider from git remote
mehr review pr --pr-number 123

# Explicit provider
mehr review pr --pr-number 123 --provider github

# With custom agent
mehr review pr --pr-number 123 --agent claude

# CI/CD with token
mehr review pr --pr-number 123 --token "$GITHUB_TOKEN"
```

### Options

| Flag                  | Description              | Default       |
|-----------------------|--------------------------|---------------|
| `--pr-number`         | PR/MR number (required)  | -             |
| `--provider`          | Provider name            | auto-detected |
| `--format`            | Comment format           | `summary`     |
| `--scope`             | Review scope             | `full`        |
| `--agent`             | Agent to use             | `claude`      |
| `--acknowledge-fixes` | Acknowledge fixed issues | `true`        |
| `--update-existing`   | Edit vs. new comment     | `true`        |

## Error Handling

### Missing Secret

```
Error: MEHRHOF_STATE_SECRET environment variable must be set for PR review state verification
```

**Solution**: Generate and set the secret:
```bash
# Generate a secure secret
mehr generate-secret

# Export to environment (shown in output)
export MEHRHOF_STATE_SECRET="generated-secret-here"
```

### Tampered State

```
Error: state signature verification failed - state may have been tampered with
```

**Solution**: Previous comment was modified. Post a new review comment.

### PR Closed/Merged

```
⏭️  Skipped: PR is closed - skipping review
```

**Expected**: Review is skipped for closed/merged PRs.

## Provider Support

| Provider     | Status | Notes        |
|--------------|--------|--------------|
| GitHub       | ✅      | Full support |
| GitLab       | ✅      | Full support |
| Bitbucket    | ✅      | Full support |
| Azure DevOps | ✅      | Full support |

## Implementation Details

### State Marker

```go
const StateMarker = "<!-- MEHRHOF_REVIEW_STATE"
```

### Embedding Format

```html
<!-- MEHRHOF_REVIEW_STATE eyJwcm92aWRlciI6ImdpdGh... -->
```

### Signature Algorithm

```go
// HMAC-SHA256
func SignState(stateJSON []byte) (string, error) {
    key, err := getStateSecretKey()
    if err != nil {
        return "", err
    }
    h := hmac.New(sha256.New, key)
    h.Write(stateJSON)
    return hex.EncodeToString(h.Sum(nil)), nil
}
```

### ID Generation

```go
// SHA256 hash for collision resistance
func GenerateIssueID(file, message string, line int) string {
    normalized := strings.ToLower(strings.TrimSpace(message))
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

    filepath := strings.TrimPrefix(file, "./")
    filepath = strings.TrimPrefix(filepath, "/")

    hashInput := fmt.Sprintf("%s:%d:%s", filepath, line, normalized)
    h := sha256.Sum256([]byte(hashInput))

    return hex.EncodeToString(h[:8]) // 16 hex characters (128 bits)
}
```

## Troubleshooting

### Issue: Review not updating

**Cause**: Previous comment not found or signature verification failed.

**Solution**:
1. Check bot has permission to read comments
2. Verify `MEHRHOF_STATE_SECRET` is consistent
3. Use `--update-existing=false` to post new comment

### Issue: Memory exhaustion on large PR

**Cause**: PR diff >10MB causing memory issues.

**Solution**:
- Diff is automatically truncated before hashing
- Consider reviewing individual files instead

### Issue: False tamper detection

**Cause**: `MEHRHOF_STATE_SECRET` changed between reviews.

**Solution**: Keep secret consistent across reviews.
