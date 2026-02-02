# Security & Compliance Scanning

Integrated security scanning for SAST, secret detection, and dependency vulnerability checking.

## Overview

The security system integrates multiple scanners into the Mehrhof workflow.

### Multi-Language Support

Scanners are **auto-detected** based on project marker files:

| Marker File                          | Language              | Scanners                   |
|--------------------------------------|-----------------------|----------------------------|
| `go.mod`                             | Go                    | Gosec, Govulncheck         |
| `package.json`                       | JavaScript/TypeScript | npm audit, ESLint Security |
| `requirements.txt`, `pyproject.toml` | Python                | Bandit, pip-audit          |
| (always)                             | Cross-language        | Semgrep, Gitleaks          |

```
┌─────────────────────────────────────────────────────────────────┐
│                    Project Detection                            │
│  go.mod? package.json? requirements.txt? pyproject.toml?        │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
│ Semgrep │ │Gitleaks │ │ Gosec   │ │Govuln   │ │npm audit│ │ Bandit  │
│ (SAST)  │ │(Secrets)│ │(Go SAST)│ │(Go Deps)│ │(JS Deps)│ │(Py SAST)│
└────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘
     │           │           │           │           │           │
     └───────────┼───────────┼───────────┼───────────┼───────────┘
                 │           │           │           │
                 ▼───────────▼───────────▼───────────▼
                       ┌──────────────────┐
                       │ Scanner Registry │
                       └────────┬─────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  Report Builder  │
                       └────────┬─────────┘
                                │
                 ┌──────────────┴───────────────┐
                 ▼                              ▼
            ┌─────────┐                   ┌─────────┐
            │  SARIF  │                   │  Text   │
            │ Report  │                   │ Output  │
            └─────────┘                   └─────────┘
```

## Scanner Architecture

### Scanner Interface

All scanners implement a common interface:

```go
type Scanner interface {
    Name() string
    Scan(ctx context.Context, dir string) (*ScanResult, error)
    IsEnabled() bool
}
```

### Scan Result Structure

```go
type ScanResult struct {
    Scanner  string      // Scanner name
    Findings []Finding   // Vulnerabilities found
    Summary  Summary     // Aggregated counts
    Duration time.Duration
    Error    error       // Scanner errors
}

type Finding struct {
    ID          string         // Unique finding ID
    Scanner     string         // Source scanner
    Severity    Severity       // Critical/High/Medium/Low/Info
    Title       string         // Finding title
    Description string         // Detailed description
    Location    *Location      // File:line:column
    Code        *CodeSnippet   // Before/after code
    CVE         string         // CVE identifier
    Fix         *FixSuggestion // Remediation steps
}
```

## Scanners

### Gosec (SAST)

[Gosec](https://github.com/securego/gosec) inspects Go source code for security problems.

**Detection Categories**:
- SQL injection
- Command injection
- Weak cryptographic algorithms
- Hardcoded credentials
- Unhandled errors
- Timing attacks

**Configuration**:
```yaml
security:
  scanners:
    sast:
      enabled: true
      tools:
        - name: gosec
          enabled: true
          severity: medium    # minimum: low, medium, high
          confidence: medium  # minimum: low, medium, high
```

### Gitleaks (Secrets)

[Gitleaks](https://github.com/gitleaks/gitleaks) scans for secrets and credentials.

**Detection Categories**:
- API keys
- AWS keys
- GitHub tokens
- SSH private keys
- Database passwords
- JWT tokens
- SSL certificates

**Configuration**:
```yaml
security:
  scanners:
    secrets:
      enabled: true
      tools:
        - name: gitleaks
          enabled: true
          config_path: ""  # Custom gitleaks config
          max_depth: 0     # Scan depth (0 = unlimited)
```

### Govulncheck (Dependencies)

[Govulncheck](https://go.dev/blog/vuln) checks for known vulnerabilities in Go dependencies.

**Detection Categories**:
- CVEs in dependencies
- Vulnerable function calls
- Standard library vulnerabilities
- Transitive dependencies

**Configuration**:
```yaml
security:
  scanners:
    dependencies:
      enabled: true
      tools:
        - name: govulncheck
          enabled: true
```

## Integration Points

### Workflow Triggers

Scanners run at specific workflow phases:

```yaml
security:
  enabled: true
  run_on:
    planning: false       # Don't run during planning
    implementing: true    # Run after implementation
    reviewing: true       # Run during review
```

### Blocking Behavior

Control whether findings block task completion:

```yaml
security:
  fail_on:
    level: critical       # Minimum severity to block
    block_finish: true    # Block task on blocking findings
```

**Severity Levels**:
- `critical` - Blocks completion by default
- `high` - High-severity issues
- `medium` - Medium-severity issues
- `low` - Low-severity issues
- `any` - All findings block completion

### Workflow Integration

```
┌──────────────┐
│ Implementing │
│   Phase      │
└──────┬───────┘
       │
       ▼
┌───────────────────┐
│ Security Scanners │
│ (if implementing) │
└──────┬────────────┘
       │
       ▼
┌──────────────────┐
│ Store Results    │
│ in WorkUnit      │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Check Blocking   │
│  Findings        │
└──────┬───────────┘
       │
   ┌───┴────┐
   │        │
   ▼        ▼
Block    Continue
```

## Report Generation

### SARIF Format

SARIF (Static Analysis Results Interchange Format) is the default output format:

```json
{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "gosec",
          "version": "2.15.0",
          "rules": [
            {
              "id": "G104",
              "name": "Errors unhandled",
              "shortDescription": {
                "text": "Errors unhandled"
              }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "G104",
          "message": {
            "text": "Errors unhandled"
          },
          "level": "warning",
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "internal/auth/token.go"
                },
                "region": {
                  "startLine": 45
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
```

### Custom Output

```yaml
security:
  output:
    format: sarif              # or json, text
    file: .mehrhof/security-report.json
    include_suggestions: true
```

## Agent Integration

### Review Phase Augmentation

Security findings are automatically included in review prompts:

```
[Standard Review Instructions]

## Security Scan Results

### gosec Scanner (2.3s)
Found 3 issue(s):

### G104: Errors unhandled
**Severity**: medium
**Location**: internal/auth/token.go:45:2
**Description**: Errors unhandled in main function

### GWS-500: Hardcoded credential
**Severity**: critical
**Location**: config/secrets.go:12:1
**Description**: Potentially hardcoded credential

Please review these findings and provide remediation guidance.
```

## Extending the System

### Adding a New Scanner

1. Implement the `Scanner` interface:

```go
type CustomScanner struct {
    enabled bool
    config   *CustomConfig
}

func (s *CustomScanner) Name() string {
    return "custom"
}

func (s *CustomScanner) Scan(ctx context.Context, dir string) (*security.ScanResult, error) {
    // Implement scanning logic
    findings := []security.Finding{...}
    return &security.ScanResult{
        Scanner:  s.Name(),
        Findings: findings,
        Summary:  summarizeFindings(findings),
        Duration: time.Since(start),
    }, nil
}

func (s *CustomScanner) IsEnabled() bool {
    return s.enabled
}
```

2. Register in config:

```yaml
security:
  scanners:
    custom:
      enabled: true
      tools:
        - name: my-scanner
          enabled: true
```

3. Register in conductor initialization (see `internal/conductor/conductor_security.go`).

## Best Practices

### 1. Input Validation

All security scanners validate their configuration before execution to prevent:
- **Path traversal attacks**: Config paths are checked for `..` components
- **Argument injection**: Config-derived arguments are validated against allowlists
- **Command injection**: All scanner arguments use `exec.Command()` (not shell)

**Example Validations**:

```go
// Gitleaks: max depth range check
if config.MaxDepth < 0 || config.MaxDepth > 1000 {
    return fmt.Errorf("max_depth must be between 0 and 1000")
}

// Gitleaks: path traversal check
if strings.Contains(config.ConfigPath, "..") {
    return fmt.Errorf("config_path contains path traversal")
}

// Gosec: severity level validation
validSeverities := map[string]bool{"low": true, "medium": true, "high": true}
if !validSeverities[strings.ToLower(config.Severity)] {
    return fmt.Errorf("invalid severity level: %s", config.Severity)
}
```

### 2. Progressive Scanning

Start with critical-only blocking:

```yaml
security:
  fail_on:
    level: critical
    block_finish: true
```

Then gradually lower threshold as codebase improves.

### 2. Fix Workflow

```bash
# Run scan
mehr scan --format text

# Fix critical findings
# ...

# Re-scan only affected files
mehr scan --dir ./fixed-files
```

### 3. CI/CD Integration

```yaml
# .github/workflows/security.yml
name: Security Scan
on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Install Mehrhof
        run: curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash
      - name: Run Security Scan
        run: mehr scan --sarif --output security-report.json
      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: security-report.json
```

## Performance

### Scanner Execution Times

| Scanner     | Time (1000 files) | Time (10000 files) |
|-------------|-------------------|--------------------|
| Gosec       | ~30s              | ~5min              |
| Gitleaks    | ~10s              | ~1min              |
| Govulncheck | ~20s              | ~2min              |

### Optimization Tips

1. **Run specific scanners**: `mehr scan --scanners sast`
2. **Scan subdirectories**: `mehr scan --dir ./src`
3. **Cache Go build**: `go build -o /dev/null ./...` before scanning
4. **Parallel scans**: Different scanners run concurrently

## Troubleshooting

### Scanner Not Executing

Check scanner is enabled:
```yaml
security:
  scanners:
    sast:
      enabled: true  # Must be true
```

### False Positives

1. Exclude files in scanner config (if supported)
2. Use inline suppressions (e.g., `#nosec G104`)
3. Report false positives to scanner project

### Scanner Timeout

Increase agent timeout:
```yaml
agent:
  timeout: 600  # 10 minutes
```

Or limit scan scope:
```bash
mehr scan --dir ./src  # Scan only src/
```

## See Also

- [CLI: scan](../cli/scan.md) - Scan commands reference
- [Configuration Guide](../configuration/index.md) - Security settings
- [OWASP Top 10](https://owasp.org/www-project-top-ten/) - Web security risks
