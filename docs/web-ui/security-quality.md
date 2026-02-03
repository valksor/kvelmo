# Security & Quality

Mehrhof integrates security scanning and code quality tools that run automatically during implementation and review phases. This ensures your code meets security standards and quality benchmarks before merging.

## Overview

**Security Scanning:**
- **SAST** (Static Application Security Testing) - Code vulnerability scanning
- **Secret Detection** - Scan for leaked credentials and secrets
- **Dependency Scanning** - Check for known vulnerabilities in dependencies

**Quality Tools:**
- **Code Simplification** - Refactor code for clarity
- **Linting** - Run project linters and formatters
- **Testing** - Execute test suites

## Accessing in the Web UI

Security and quality features are available from:

| Feature               | Location                               |
|-----------------------|----------------------------------------|
| **Run Scan**          | Dashboard → Quick Actions → "Run Scan" |
| **Simplify**          | Dashboard → Quick Actions → "Simplify" |
| **Quality Settings**  | Settings → Quality                     |
| **Security Settings** | Settings → Security                    |

Scans run automatically during implementation and review if enabled.

## Security Scanning

### Running a Manual Scan

1. Go to the **Dashboard**
2. Click **"Run Scan"** in Quick Actions
3. Configure options:
   - **Directory** - Path to scan (default: project root)
   - **Scanners** - Select which scanners to run
   - **Format** - Output format (text, JSON, SARIF)
4. Click **"Start Scan"**

Results appear in the agent output area.

### Scan Results

Results show:

| Section            | Description                       |
|--------------------|-----------------------------------|
| **Findings**       | Issues discovered by each scanner |
| **Severity**       | critical, high, medium, low       |
| **Location**       | File and line number              |
| **Description**    | What the issue is                 |
| **Recommendation** | How to fix it                     |

### Severity Levels

| Level      | Description                                         |
|------------|-----------------------------------------------------|
| `critical` | Immediate security risk (default failure threshold) |
| `high`     | Significant security issue                          |
| `medium`   | Potential security concern                          |
| `low`      | Minor issue or informational                        |

### Available Scanners

#### Gosec (SAST)

[Go Security Checker](https://github.com/securego/gosec) inspects Go source code for security problems.

**Detects:**
- SQL injection
- Command injection
- Weak cryptographic algorithms
- Unhandled errors
- Timing attacks

#### Gitleaks (Secrets)

[Gitleaks](https://github.com/gitleaks/gitleaks) scans for secrets, credentials, and sensitive data.

**Detects:**
- API keys
- Passwords
- Tokens
- Certificates
- SSH keys

#### Govulncheck (Dependencies)

[Govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) checks for known vulnerabilities in Go dependencies.

**Detects:**
- CVEs in dependencies
- Vulnerable function calls
- Standard library vulnerabilities

### Automatic Tool Downloads

Mehrhof automatically downloads missing security tools to `~/.valksor/mehrhof/tools/` and caches them for future use. No manual installation required.

**Tool Priority:**
1. PATH - If installed globally
2. Cache - Previously downloaded tools
3. Auto-download - From GitHub releases
4. Skip - With warning if unavailable

## Code Simplification

### Simplifying Code

The simplify command automatically determines what to simplify based on workflow state:

- **Pre-plan**: Simplifies task input/description for clarity
- **After planning**: Simplifies specification files
- **After implementing**: Simplifies code changes while preserving functionality

1. Go to the **Dashboard**
2. Click **"Simplify"** in Quick Actions
3. The AI analyzes and refines content
4. Review changes with `git diff`

### What Gets Simplified

**Pre-Plan: Task Input**
```
Before: "add user auth with JWT and OAuth providers also handle refresh tokens and session management"

After: "Implement JWT-based user authentication with OAuth integration.
Support refresh token rotation and secure session management."
```

**After Implementation: Code**
```go
// Before
func GetToken(u User) (string, error) {
    t, e := db.GetToken(u.ID)
    if e != nil {
        return "", e
    }
    return t, nil
}

// After
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

### Safety

Simplification creates a **git checkpoint** before modifying files, so you can always undo changes using the dashboard's Undo button or the CLI.

## Configuration

### Security Configuration

Enable and configure security scanning in **Settings → Security**:

```yaml
security:
  enabled: true
  run_on:
    planning: false
    implementing: true
    reviewing: true
  fail_on:
    level: critical
    block_finish: true
  scanners:
    sast:
      enabled: true
      tools:
        - name: gosec
          severity: medium
    secrets:
      enabled: true
      tools:
        - name: gitleaks
    dependencies:
      enabled: true
      tools:
        - name: govulncheck
```

### Quality Configuration

Configure quality checks in **Settings → Quality**:

```yaml
quality:
  targets:
    - name: lint
      command: make lint
    - name: test
      command: make test
  simplify:
    instructions: |
      Follow project coding standards:
      - Use descriptive names
      - Keep functions under 50 lines
      - Add docstrings to public APIs
```

## Report Formats

### SARIF Reports

SARIF (Static Analysis Results Interchange Format) is a standard format for security tool output.

1. Run scan with **Format: SARIF**
2. Download the report file
3. Open in VS Code with [SARIF Viewer](https://marketplace.visualstudio.com/items?itemName=MS-SarifVSCode.sarif-viewer)
4. View findings with navigation to code locations

### JSON Reports

Machine-readable JSON output for integration with other tools.

### Text Reports

Human-readable text output shown in the dashboard.

## Common Workflows

### Full Security Pipeline

```
1. Enable security in Settings
2. Set fail_on.level to "high"
3. Enable block_finish
4. Implement task → Security scans run automatically
5. If issues found → Auto-retry or manual fix
6. Review → Security scans run again
7. Finish blocked if high+ severity findings
```

### Simplification Workflow

```
1. Plan task
2. Click "Simplify" to refine specifications
3. Implement based on clear specs
4. Click "Simplify" to clean up code
5. Review simplified code
6. Finish with cleaner codebase
```

### CI/CD Integration

```bash
# Run security scan in CI
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{"directory": "./src", "format": "sarif"}'
```

---

## Also Available via CLI

Run security scans and code simplification from the command line for scripting or CI/CD integration.

| Command | What It Does |
|---------|--------------|
| `mehr scan` | Run all enabled security scanners |
| `mehr scan --sarif` | Generate SARIF format report |
| `mehr simplify` | Simplify code based on workflow state |

See [CLI: scan](/cli/scan.md) for scanner selection and output options, and [CLI: simplify](/cli/simplify.md) for simplification modes.

## Troubleshooting

### Scanner Not Available

By default, Mehrhof auto-downloads tools. If download fails:

1. Check network connectivity
2. Install manually using `go install` for the relevant tool
3. Add to PATH

### Too Many Findings

Adjust severity thresholds:
- Go to **Settings → Security**
- Increase scanner severity level
- Filter out lower-priority findings

### Simplification Made Things Worse

Simplification creates checkpoints, so you can undo:
1. Click **"Undo"** from dashboard
2. Or use CLI: `mehr undo`
3. Add note with better instructions
4. Try to simplify again
