# mehr scan

Security scanning commands for vulnerability detection and compliance checking.

## Synopsis

```bash
mehr scan [flags]
```

## Description

Mehrhof integrates security scanners that run automatically during implementation and review phases:
- **SAST** (Static Application Security Testing) - Code vulnerability scanning
- **Secret Detection** - Scan for leaked credentials and secrets
- **Dependency Scanning** - Check for known vulnerabilities in dependencies

## Supported Languages

Scanners are **auto-detected** based on your project type:

| Language                  | SAST Scanner    | Dependency Scanner |
|---------------------------|-----------------|--------------------|
| **Cross-language**        | Semgrep         | -                  |
| **Secrets**               | Gitleaks        | -                  |
| **Go**                    | Gosec           | Govulncheck        |
| **JavaScript/TypeScript** | ESLint Security | npm audit          |
| **Python**                | Bandit          | pip-audit          |

## Configuration

Security scanning is disabled by default. Enable it in `.mehrhof/config.yaml`:

```yaml
security:
  enabled: true
  run_on:
    planning: false       # Don't run during planning
    implementing: true    # Run after implementation
    reviewing: true       # Run during review

  # Failure policy
  fail_on:
    level: critical       # or high, medium, low, any
    block_finish: true    # Block task completion

  scanners:
    sast:
      enabled: true
      tools:
        - name: gosec
          enabled: true
          severity: medium  # minimum severity to report
          confidence: medium

    secrets:
      enabled: true
      tools:
        - name: gitleaks
          enabled: true
          config_path: ""   # custom gitleaks config
          max_depth: 0       # scan depth (0 = unlimited)

    dependencies:
      enabled: true
      tools:
        - name: govulncheck
          enabled: true

  # Tool management (auto-download missing tools)
  tools:
    auto_download: true   # Automatically download missing tools (default: true)
    cache_dir: ""         # Override default cache directory (default: ~/.valksor/mehrhof/tools)
    timeout: 60           # Download timeout in seconds (default: 60)

  # Reporting
  output:
    format: sarif        # or json, text
    file: .mehrhof/security-report.json
    include_suggestions: true
```

### Tool Management

By default, mehrhof automatically downloads missing security scanning tools to `~/.valksor/mehrhof/tools/` and caches them for future use. This means you don't need to manually install gitleaks, gosec, or govulncheck - mehrhof handles this for you.

**Tool sources** (in order of priority):
1. **PATH** - If you have a tool installed in your PATH, it will be used
2. **Cache** - Previously downloaded tools from `~/.valksor/mehrhof/tools/`
3. **Auto-download** - Automatically download from GitHub releases (if enabled)
4. **Skip** - Tool is skipped with a warning if not available

**Disable auto-download**:
```yaml
security:
  tools:
    auto_download: false
```

**Custom cache location**:
```yaml
security:
  tools:
    cache_dir: /custom/path/to/tools
```

### Severity Levels

| Level      | Description                         |
|------------|-------------------------------------|
| `critical` | Critical vulnerabilities (default)  |
| `high`     | High-severity issues                |
| `medium`   | Medium-severity issues              |
| `low`      | Low-severity issues                 |
| `any`      | All findings regardless of severity |

### Report Formats

| Format  | Description                       |
|---------|-----------------------------------|
| `sarif` | SARIF 2.1.0 JSON format (default) |
| `json`  | Custom JSON format                |
| `text`  | Human-readable text output        |

## Commands

### scan

Run all enabled security scanners on the codebase:

```bash
mehr scan [--dir=PATH] [--scanners=LIST] [--format=FMT] [--output=FILE]
```

Flags:
- `--dir`, `-d` - Directory to scan (default: current directory or project root)
- `--scanners`, `-s` - Specific scanners to run (comma-separated: `sast,secrets,dependencies`)
- `--format` - Output format: `sarif`, `json`, or `text` (default: `text`)
- `--output`, `-o` - Save report to file
- `--fail-level` - Failure threshold (default: `critical`)
- `--sarif` - Generate SARIF report (alias for `--format sarif`)
- `--json` - JSON output (alias for `--format json`)

Examples:
```bash
# Scan current directory
mehr scan

# Scan specific directory
mehr scan --dir ./src

# Run only SAST scanners
mehr scan --scanners sast

# Generate SARIF report
mehr scan --sarif --output security-report.json

# Run with custom failure threshold
mehr scan --fail-level medium
```

### Scan Output

Text format example:
```
Running security scans on: ./src

⚠ Warnings:
  - gitleaks not found in PATH or cache, downloading...
  - Downloading gitleaks 8.18.0 for linux/amd64...

## Skipped Tools

The following tools were not available and were skipped: govulncheck

Install manually or enable auto-download in config.

## gosec (2.3s)

Found 3 issue(s):

### G104: Errors unhandled
**Severity**: medium
**Location**: internal/auth/token.go:45:2
**Description**: Errors unhandled in main function

### G401: Detected dereference in map range
**Severity**: low
**Location**: internal/cache/store.go:78:10
**Description**: Dereference in map range

## gitleaks (1.8s)

Found 1 issue(s):

### SSH Private Key
**Severity**: critical
**Location**: tests/fixtures/test_key:1:1
**Description**: SSH Private Key found

---

## Summary

**Total Findings**: 4 (from 2 scanner(s))

**By Severity**:
- critical: 1
- medium: 1
- low: 1
```

## Exit Codes

| Code | Meaning                                     |
|------|---------------------------------------------|
| 0    | No findings or only below failure threshold |
| 1    | Findings at or above failure threshold      |
| 2    | Scanner errors                              |

## Scanners

### Cross-Language Scanners

#### Semgrep (SAST)

[Semgrep](https://semgrep.dev) is a cross-language static analysis tool supporting 30+ languages.

**Installation:**
```bash
pip install semgrep
# or
brew install semgrep
```

**What it detects:**
- Security vulnerabilities across multiple languages
- OWASP Top 10 issues
- Custom security rules

#### Gitleaks (Secrets)

[Gitleaks](https://github.com/gitleaks/gitleaks) scans for secrets, credentials, and other sensitive data.

**Installation:**
```bash
brew install gitleaks
# or download from https://github.com/gitleaks/gitleaks/releases
```

**What it detects:**
- API keys
- Passwords and tokens
- Certificates and SSH keys
- Cloud credentials

### Go Scanners

#### Gosec (SAST)

[Gosec](https://github.com/securego/gosec) inspects Go source code for security problems.

**Installation:**
```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**What it detects:**
- SQL injection
- Command injection
- Weak cryptographic algorithms
- Unhandled errors

#### Govulncheck (Dependencies)

[Govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) checks for known vulnerabilities in Go dependencies.

**Installation:**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

**What it detects:**
- CVEs in dependencies
- Vulnerable function calls

### JavaScript/TypeScript Scanners

#### npm audit (Dependencies)

Built-in to npm, checks for known vulnerabilities in npm packages.

**Installation:** Built-in to npm >= 6.0

**Requires:** `package-lock.json`

**What it detects:**
- CVEs in npm packages
- Direct and transitive dependency vulnerabilities

#### ESLint Security (SAST)

[eslint-plugin-security](https://github.com/eslint-community/eslint-plugin-security) provides security-focused ESLint rules.

**Installation:**
```bash
npm install eslint eslint-plugin-security
```

**What it detects:**
- Dynamic code execution vulnerabilities
- Command injection via child_process
- SQL injection patterns
- Object injection vulnerabilities

### Python Scanners

#### Bandit (SAST)

[Bandit](https://bandit.readthedocs.io) is a security linter for Python code.

**Installation:**
```bash
pip install bandit
```

**What it detects:**
- Hardcoded passwords
- SQL injection
- Command injection
- Insecure cryptography

#### pip-audit (Dependencies)

[pip-audit](https://github.com/pypa/pip-audit) checks Python dependencies for known vulnerabilities.

**Installation:**
```bash
pip install pip-audit
```

**Requires:** `requirements.txt` or `pyproject.toml`

**What it detects:**
- CVEs in Python packages
- PyPI advisory vulnerabilities

## Integration with Workflow

When enabled, scanners run automatically:

1. **After Implementation** - If `run_on.implementing: true`
2. **During Review** - If `run_on.reviewing: true`

If `fail_on.block_finish: true`, tasks with blocking findings cannot finish.

## SARIF Reports

SARIF (Static Analysis Results Interchange Format) is a standard format for security tool output.

**Generate SARIF report:**
```bash
mehr scan --sarif --output security-report.json
```

**View in VS Code:**
Install the [SARIF Viewer](https://marketplace.visualstudio.com/items?itemName=MS-SarifVSCode.sarif-viewer) extension.

**Example SARIF structure:**
```json
{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "gosec",
          "version": "2.15.0"
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

## Examples

### Full Security Pipeline

```bash
# Enable security in config
cat >> .mehrhof/config.yaml <<EOF
security:
  enabled: true
  run_on:
    implementing: true
    reviewing: true
  fail_on:
    level: high
    block_finish: true
EOF

# Run task with automatic security checks
mehr start task.md
mehr plan
mehr implement    # Security scans run here automatically
mehr review        # Security scans run here automatically
mehr finish        # Blocked if high+ severity findings
```

### CI/CD Integration

```bash
#!/bin/bash
# ci-security-check.sh

set -e

# Run security scan
mehr scan --format sarif --output security-report.json

# Check exit code
if [ $? -ne 0 ]; then
  echo "Security scan failed - check security-report.json"
  exit 1
fi

echo "Security scan passed"
```

### Scan Specific Directory

```bash
# Scan only new code in src/
mehr scan --dir ./src --scanners sast,secrets
```

## Troubleshooting

### Scanner Not Installed

By default, mehrhof automatically downloads missing security tools to `~/.valksor/mehrhof/tools/` and caches them for future use. No manual installation is required.

**If you prefer manual installation**, you can:
1. Disable auto-download in config:
   ```yaml
   security:
     tools:
       auto_download: false
   ```
2. Install tools manually:
   ```bash
   # Install gosec
   go install github.com/securego/gosec/v2/cmd/gosec@latest

   # Install gitleaks
   go install github.com/zricethezav/gitleaks/v8/cmd/gitleaks@latest

   # Install govulncheck (Go 1.21+ has it built-in)
   go install golang.org/x/vuln/cmd/govulncheck@latest
   ```

### Skipped Tools

If you see "Skipped Tools" in the output, it means some security tools couldn't be downloaded or found. This is not an error - the scan continues with available tools.

Common reasons:
- **Network issues** - Download from GitHub failed
- **Unsupported platform** - No pre-built binary available for your OS/architecture
- **Auto-download disabled** - Tools not in PATH and auto-download is off

To see warnings about what went wrong:
```bash
mehr scan 2>&1 | grep -A 10 Warnings
```

### Clear Cache

To clear the tool cache and force re-download:
```bash
rm -rf ~/.valksor/mehrhof/tools/
mehr scan
```

### Too Many Findings

Adjust severity thresholds:
```yaml
security:
  scanners:
    sast:
      tools:
        - name: gosec
          severity: high    # Only report high+ severity
```

### Scanner Timeout

Increase timeout in agent config:
```yaml
agent:
  timeout: 600  # 10 minutes for large scans
```

## Web UI

Prefer a visual interface? See [Web UI: Security & Quality](../web-ui/security-quality.md).

## See Also

- [Configuration Guide](../configuration/index.md) - Security settings in config.yaml
- [Advanced: Security Scanning](../advanced/security.md) - Deep dive on security architecture
- [SARIF Specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
