# mehr generate-secret

Generate a cryptographically secure secret for PR review state signing.

> **Note:** This is a CLI-only utility. For authentication setup in the web UI, see [Web UI Authentication](/web-ui/authentication.md).

## Synopsis

```bash
mehr generate-secret
```

## Description

The `generate-secret` command creates a cryptographically secure 32-byte (256-bit) random secret encoded in base64. This secret is used for:

- **HMAC-SHA256 signing** of PR review state embedded in comments
- **Tamper detection** - verifies state hasn't been modified
- **CI/CD integration** - provides secrets for automated workflows

The generated secret meets the minimum 32-character requirement for `MEHRHOF_STATE_SECRET`.

## Security

- Uses `crypto/rand` for cryptographically secure random bytes
- Generates 32 bytes (256 bits) of entropy
- Base64-encoded for easy use in environment variables and CI/CD configs

## Output

The command outputs instructions in multiple formats:

### Shell Export Format

```bash
# Add this to your environment or CI/CD secrets:
export MEHRHOF_STATE_SECRET="base64-encoded-secret"
```

### CI/CD Format

```bash
# Or in GitHub Actions / GitLab CI:
MEHRHOF_STATE_SECRET: base64-encoded-secret
```

## Usage

### Local Development

```bash
# Generate and copy to clipboard
mehr generate-secret

# Export directly to shell
export MEHRHOF_STATE_SECRET="$(mehr generate-secret)"

# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export MEHRHOF_STATE_SECRET="'$(mehr generate-secret)'"' >> ~/.bashrc
```

### GitHub Actions

```yaml
# .github/workflows/pr-review.yml
env:
  MEHRHOF_STATE_SECRET: ${{ secrets.MEHRHOF_STATE_SECRET }}

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run PR review
        run: |
          mehr review pr --pr-number ${{ github.event.pull_request.number }}
```

Generate and add to repository secrets:

```bash
# Generate secret
mehr generate-secret

# Add as GitHub secret via UI or CLI
gh secret set MEHRHOF_STATE_SECRET
```

### GitLab CI

```yaml
# .gitlab-ci.yml
variables:
  MEHRHOF_STATE_SECRET: $MEHRHOF_STATE_SECRET

review:
  script:
    - mehr review pr --pr-number $CI_MERGE_REQUEST_IID
```

Add to CI/CD variables in GitLab project settings.

## Requirements

The secret must:
- Be at least 32 characters long
- Be kept consistent across reviews (don't regenerate)
- Be stored securely (environment variable or secrets manager)

**Important**: Once you start using a secret, keep it consistent. Changing the secret between reviews will cause signature verification to fail.

## Flags

This command has no specific flags.

## Examples

### Basic Usage

```bash
$ mehr generate-secret

# Add this to your environment or CI/CD secrets:
export MEHRHOF_STATE_SECRET="dGVzdGluZ3NlY3JldGJhc2U2NGVuY29kZWRzdHJpbmdzaGFsbGJlMzJjaGFyc"

# Or in GitHub Actions / GitLab CI:
MEHRHOF_STATE_SECRET: dGVzdGluZ3NlY3JldGJhc2U2NGVuY29kZWRzdHJpbmdzaGFsbGJlMzJjaGFyc
```

### CI/CD Integration

```bash
# GitHub Actions - add via CLI
gh secret set MEHRHOF_STATE_SECRET --body="$(mehr generate-secret)"

# GitLab - add via CLI (requires glab)
glab variable set MEHRHOF_STATE_SECRET --value="$(mehr generate-secret)"
```

### Shell Profile

```bash
# Add to ~/.bashrc, ~/.zshrc, or equivalent
cat >> ~/.bashrc << 'EOF'
# Mehrhof PR review state secret
export MEHRHOF_STATE_SECRET="$(mehr generate-secret)"
EOF

# Source the file or restart shell
source ~/.bashrc
```

## Web UI

Prefer a visual interface? See the authentication setup in [Authentication](/web-ui/authentication.md).

## See Also

- [PR Review State Management](/guides/pr-review-state.md) - How state signing works
- [review pr](review.md) - PR review command
- [CLI Overview](index.md) - All available commands
