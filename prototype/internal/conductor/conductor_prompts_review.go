package conductor

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// buildReviewFixPrompt creates the prompt for implementing fixes from a code review.
// This is a focused prompt that tells the agent to address specific review feedback
// rather than implementing specifications.
func buildReviewFixPrompt(workspace *storage.Workspace, workingDir, title, sourceContent, reviewContent, notes, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert software engineer. Your task is to fix the issues identified in the following code review.

Current timestamp: %s
Working directory: %s

## Task
%s

## Original Requirements
%s

## Review Feedback to Address
%s
`, currentTime, workingDir, title, sourceContent, reviewContent)

	if notes != "" {
		prompt += fmt.Sprintf(`
## Additional Notes
%s
`, notes)
	}

	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Constraints
- Focus ONLY on fixing the issues identified in the review
- Do not make unrelated changes or "improvements" outside the review feedback
- Follow existing code patterns in the codebase
- Address each review item explicitly
- If a review comment is unclear, make your best interpretation and document your reasoning

## Instructions
For each fix you implement:
1. Use yaml:file blocks with path, operation (create/update/delete), and content
2. File paths must be relative to the working directory
3. Follow existing code style and patterns
4. Include necessary imports
5. Add appropriate error handling

Output each file change in a yaml:file block.

## Example Output
` + "```yaml:file\n" + `path: internal/api/handler.go
operation: update
content: |
  package api

  // ... fixed code addressing review feedback ...
` + "```" + `

## Required Output Format
Your response MUST include:
1. Brief explanation of what you're fixing and why
2. yaml:file blocks for each file modified to address the review feedback
`

	// Browser tools (optional)
	if browserSection := buildBrowserToolsSectionForStep(workspace, "implementing"); browserSection != "" {
		prompt += browserSection
	}

	return prompt
}

// buildReviewPrompt creates the prompt for code review.
// Note: workspace parameter kept for API consistency with other prompt builders,
// even though tests pass nil (production code uses buildReviewPromptWithLint directly).
//
//nolint:unparam // workspace is nil in tests but needed for API consistency
func buildReviewPrompt(workspace *storage.Workspace, title, sourceContent, specsContent string) string {
	return buildReviewPromptWithLint(workspace, title, sourceContent, specsContent, "", "")
}

// buildReviewPromptWithLint creates the prompt for code review including lint results.
// If lintResults is empty, it falls back to the standard review prompt.
func buildReviewPromptWithLint(workspace *storage.Workspace, title, sourceContent, specsContent, lintResults, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")
	prompt := fmt.Sprintf(`You are an expert software engineer specializing in code review and quality assurance.

Current timestamp: %s

## Task
%s

## Original Requirements
%s

## Specifications
%s
`, currentTime, title, sourceContent, specsContent)

	// Include lint results if available
	if lintResults != "" {
		prompt += fmt.Sprintf(`
%s
`, lintResults)
	}

	// Inject custom instructions before static instructions
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	// Add browser tools section if enabled (step-specific)
	if browserSection := buildBrowserToolsSectionForStep(workspace, "reviewing"); browserSection != "" {
		prompt += browserSection
	}

	// Add chain-of-thought guidance for review
	prompt += `
## Approach
Before providing your review:
1. Trace through the code paths systematically
2. Check each specification requirement against the implementation
3. Consider edge cases the implementation might miss
4. Look for patterns that commonly cause bugs

Document your analysis before listing issues.
`

	prompt += `
## Constraints
- If the approach has flaws, explain the specific problem and suggest an alternative
- If you find issues, provide concrete fixes not just descriptions
- Prefer existing patterns in the codebase over introducing new ones

## Instructions
Review the implementation for:
1. Correctness - Does it meet the specifications?
2. Code quality - Is it clean, readable, and maintainable?
3. Security - Are there any vulnerabilities?
4. Performance - Are there any obvious bottlenecks?
5. Best practices - Does it follow language/framework conventions?
6. Testing - Are there tests? Do they cover the requirements?
7. Integration - Does it integrate well with existing code?`

	// Add lint-specific instruction if results were provided
	if lintResults != "" {
		prompt += `
8. Lint issues - Address all issues found by automated linters above`
	}

	prompt += `

## Review Process

**Round 1: Analysis**
- Trace code paths systematically
- Identify all issues (critical, major, minor)
- Document findings with file locations

**Round 2: Fixes**
- Provide yaml:file blocks for ALL issues found
- Verify each fix addresses the root cause
- Check fixes don't introduce new issues

**Round 3: Verification**
- Re-check the implementation with fixes applied
- Confirm all specification requirements are met
- Give final approval only when satisfied

If you find issues, you MUST provide fixes - don't just describe problems.

## Handling Large Codebases

If unable to review all changes due to context limits:
1. Focus on critical paths and security-sensitive code first
2. Prioritize files with the most significant changes
3. Note which files were not fully reviewed and why
4. Request a follow-up review for remaining files if needed

## Example Review Output

### Analysis
Traced through the authentication flow:
1. Login handler receives credentials (auth.go:45)
2. Validates against database (user_repo.go:89)
3. Issues JWT token (token.go:23) - ISSUE: No expiration set

### Issues Found

**Critical:**
1. JWT tokens have no expiration - security vulnerability
   File: internal/auth/token.go:23
   Fix: Add exp claim with 24h default

**Minor:**
1. Missing error log when validation fails
   File: internal/auth/auth.go:52

### Fixes
` + "```yaml:file\n" + `path: internal/auth/token.go
operation: update
content: |
  func GenerateToken(userID string) (string, error) {
    claims := jwt.MapClaims{
      "sub": userID,
      "iat": time.Now().Unix(),
      "exp": time.Now().Add(24 * time.Hour).Unix(), // Added expiration
    }
    // ...
  }
` + "```" + `

## Required Output Format
Your response MUST include:
1. Analysis section documenting your review process
2. Issues found with severity (critical/major/minor)
3. yaml:file blocks fixing ALL issues found
4. Final verdict: APPROVED or NEEDS_CHANGES`

	return prompt
}

// buildReviewPromptWithLintAndSecurity creates the prompt for code review including lint results and security findings.
func buildReviewPromptWithLintAndSecurity(workspace *storage.Workspace, title, sourceContent, specsContent, lintResults, securityFindings, customInstructions string) string {
	// Start with the standard review prompt with lint
	prompt := buildReviewPromptWithLint(workspace, title, sourceContent, specsContent, lintResults, customInstructions)

	// Add security findings if available
	if securityFindings != "" {
		// Insert security findings before the "Approach" section
		securitySection := fmt.Sprintf(`

## Security Scan Results
%s

Please review these security findings and provide guidance on how to address them.
`, securityFindings)

		// Insert before "## Approach" section
		approachIndex := strings.Index(prompt, "\n## Approach")
		if approachIndex != -1 {
			prompt = prompt[:approachIndex] + securitySection + prompt[approachIndex:]
		} else {
			prompt += securitySection
		}
	}

	return prompt
}

// buildFinishPrompt creates the prompt for commit message generation.
func buildFinishPrompt(ticketID, title string, specPaths []string, specSnapshot, diffStat, stagedFiles, stagedDiff string) string {
	// Format spec list
	specList := "- None"
	if len(specPaths) > 0 {
		var specs []string
		for _, path := range specPaths {
			specs = append(specs, "- "+path)
		}
		specList = strings.Join(specs, "\n")
	}

	prompt := fmt.Sprintf(`You are an expert software engineer. Construct a final squash commit message.

Ticket: %s
Title: %s

Specifications:
%s

Spec Snapshot:
%s

Diff Stat:
%s

Staged Files:
%s

Staged Diff:
%s

## Commit Message Guidelines

**Conventional Commits Types:**
- feat: New feature or functionality
- fix: Bug fix
- refactor: Code change that neither fixes a bug nor adds a feature
- docs: Documentation only changes
- test: Adding or updating tests
- chore: Maintenance tasks (build, config, dependencies)

**Format Rules:**
- Keep first line under 72 characters
- Use imperative mood ("Add" not "Added", "Fix" not "Fixed")
- Summary explains WHAT was done; the diff shows HOW
- Reference ticket ID in the format: (%s): <summary>

## Commit Message Format
(%s): <summary under 72 chars, imperative mood>
- `+"`<file>`"+`: <change description>
- `+"`<file>`"+`: <change description>
...

## Rules
- First line: (%s): <summary under 72 chars, imperative mood>
- Bullet lines: one per changed file, format: - `+"`<file>`"+`: <change>
- Skip /tickets/%s/ paths from bullet list (spec files are internal)
- No extra commentary, explanations, or markdown formatting
- RESPOND WITH ONLY THE COMMIT MESSAGE, NOTHING ELSE

## Example Commit Messages

**Good:**
`+"```"+`
(AUTH-123): Add JWT token expiration for security
- internal/auth/token.go: Add exp claim with 24h default
- internal/config/auth.go: Add token_expiry config option
- internal/auth/token_test.go: Add expiration validation tests
`+"```"+`

**Bad:**
`+"```"+`
updated files
`+"```"+`

`+"```"+`
(AUTH-123): Updated the token.go file to add expiration and also fixed some other things
- token.go: changes
`+"```"+`

## Instructions
1. Analyze the specs to understand what was implemented
2. Determine the appropriate conventional commit type (feat, fix, refactor, etc.)
3. Create a concise summary (action verb + scope, under 72 chars)
4. List each changed file with a brief description
5. Output ONLY the commit message in the exact format above
`,
		ticketID,
		title,
		specList,
		specSnapshot,
		diffStat,
		stagedFiles,
		stagedDiff,
		ticketID, // Format Rules reference
		ticketID, // Format line
		ticketID, // Rules line
		ticketID, // Skip paths
	)

	return prompt
}

// buildSpecStatusSummary creates a summary of specification completion status.
// Parses spec files and extracts their Status sections to show progress.
func buildSpecStatusSummary(workspace *storage.Workspace, taskID string) string {
	// Get list of spec numbers
	specNumbers, err := workspace.ListSpecifications(taskID)
	if err != nil {
		return fmt.Sprintf("- Error listing specifications: %v", err)
	}

	if len(specNumbers) == 0 {
		return "- No specifications yet"
	}

	var summary []string
	statusPattern := regexp.MustCompile(`(?im)^## Status\s*\n*(.+)$`)

	for _, num := range specNumbers {
		// Read spec file
		specContent, err := workspace.LoadSpecification(taskID, num)
		if err != nil {
			summary = append(summary, fmt.Sprintf("- specification-%d.md: error reading file", num))

			continue
		}

		// Extract status
		matches := statusPattern.FindStringSubmatch(specContent)
		if len(matches) > 1 {
			status := strings.TrimSpace(matches[1])
			summary = append(summary, fmt.Sprintf("- specification-%d.md: %s", num, status))
		} else {
			// No status section found (backward compatibility)
			summary = append(summary, fmt.Sprintf("- specification-%d.md: unknown (no status section)", num))
		}
	}

	return strings.Join(summary, "\n")
}

// buildBrowserToolsSection creates a prompt section describing available browser automation tools.
//
// The workspace parameter is used to load the workspace configuration and check if browser
// automation is enabled. Returns empty string if:
//   - workspace is nil
//   - config loading fails
//   - browser is not configured
//   - browser is disabled
//
// Errors loading config are logged to stderr but do not cause the function to fail,
// as browser tools are an optional enhancement.
func buildBrowserToolsSection(workspace *storage.Workspace) string {
	if workspace == nil {
		return ""
	}
	cfg, err := workspace.LoadConfig()
	if err != nil {
		// Config load failed - log warning but don't fail prompt building
		// This is a non-critical error (browser tools are optional)
		fmt.Fprintf(os.Stderr, "Warning: failed to load workspace config for browser tools: %v\n", err)

		return ""
	}
	if cfg.Browser == nil {
		return "" // Not configured - silent (expected)
	}
	if !cfg.Browser.Enabled {
		return "" // Disabled - silent (expected)
	}

	return `
## Browser Automation

Browser automation is ENABLED. You can control Chrome for web-based tasks:

### Available Browser Tools:
- browser_open_url - Open URLs in new tabs
- browser_screenshot - Capture screenshots (full page or viewport)
- browser_click - Click elements by CSS selector
- browser_type - Type text into input fields
- browser_evaluate - Execute JavaScript in page context
- browser_query - Query DOM elements
- browser_get_console_logs - Retrieve console.log output
- browser_get_network_requests - Monitor HTTP requests
- browser_detect_auth - Detect if page requires login
- browser_wait_for_login - Pause for manual user login

### When to Use:
- Testing web applications during implementation
- Verifying frontend features
- Handling authentication flows
- Validating API responses in browser console
`
}

// buildBrowserToolsSectionForStep creates a step-specific browser tools section.
// Different workflow steps have different browser usage patterns:
//   - planning: Focus on research and understanding
//   - implementing: Focus on testing changes
//   - reviewing: Focus on verification
func buildBrowserToolsSectionForStep(workspace *storage.Workspace, step string) string {
	if workspace == nil {
		return ""
	}
	cfg, err := workspace.LoadConfig()
	if err != nil {
		slog.Debug("failed to load workspace config for browser tools", "step", step, "error", err)

		return ""
	}
	if cfg.Browser == nil || !cfg.Browser.Enabled {
		return ""
	}

	baseTools := `
### Available Browser Tools:
- browser_open_url - Open URLs in new tabs
- browser_screenshot - Capture screenshots
- browser_click - Click elements by CSS selector
- browser_type - Type text into input fields
- browser_evaluate - Execute JavaScript
- browser_query - Query DOM elements
- browser_get_console_logs - Retrieve console output
- browser_get_network_requests - Monitor HTTP requests
`

	switch step {
	case "planning":
		return `
## Browser Automation (Planning)

Browser is ENABLED. Use it during planning to:
- Research APIs and explore documentation
- Understand existing web interfaces
- Capture screenshots for specification references
- Investigate how similar features work
` + baseTools

	case "implementing":
		return `
## Browser Automation (Implementation)

Browser is ENABLED. Use it during implementation to:
- Test your changes in a real browser
- Verify UI components render correctly
- Debug frontend issues interactively
- Validate API responses
` + baseTools

	case "reviewing":
		return `
## Browser Automation (Review)

Browser is ENABLED. Use it during review to:
- Verify the implementation works end-to-end
- Test edge cases manually
- Capture evidence of issues found
- Validate fixes before approval
` + baseTools

	default:
		return buildBrowserToolsSection(workspace)
	}
}

// computeErrorRecoverySection creates a prompt section with error recovery strategies.
// Returns instructions for handling common failure scenarios.
// Called once at init time; use getErrorRecoverySection() for cached access.
func computeErrorRecoverySection() string {
	return `
## Error Recovery Strategies

If you encounter errors:

### Context Overflow:
- Focus on highest-priority specifications first
- Implement in priority order
- If unable to complete all, implement critical ones
- Update status to "completed" for finished specifications

### Parse Failures:
- Ask user to provide file contents
- Create missing files with best-guess structure
- If unclear, ask specific format questions

### Authentication Errors:
- Check environment variables and config
- Document required credentials in specification's Unknowns
- Implement with placeholder credentials for testing

### Dependency Errors:
- Check dependency management files (go.mod, package.json)
- Add dependencies using appropriate commands
- Document in specification if user needs to install

### Compilation Errors:
- Fix syntax errors first, then type errors
- Use yaml:file blocks for corrected code
- Simplify if error persists

### Test Failures:
- Check if tests are outdated or implementation incorrect
- Fix implementation OR update tests
- Document failing tests for manual review
`
}

// computeSpecValidationInstructions creates a prompt section with specification quality checklist.
// Returns instructions for validating specification completeness and quality.
// Called once at init time; use getSpecValidationInstructions() for cached access.
func computeSpecValidationInstructions() string {
	return `
## Specification Quality Checklist

Verify your specification includes:

### Required Sections:
✓ Request - Clear restatement of requirements
✓ Plan - Numbered, actionable steps (minimum 2 steps)
✓ Context - File references with line ranges
✓ Unknowns - All questions OR "0. None"
✓ Complete Condition - Manual AND run validation steps
✓ Status - Current state with timestamp

### Quality Criteria:
- Plan steps are actionable and numbered
- Unknowns have default answers (NEVER "user input required")
- At least 1 manual AND 1 run validation
- Clear enough for another engineer to implement

Note: Specifications will be validated for required sections during planning.
If missing sections, revise before outputting.
`
}

// buildSpecificationTrackingSummary creates a summary of specification implementation status.
//
// The workspace parameter is used to:
//   - List all specifications for the task
//   - Parse specification metadata (status, implemented files)
//
// Returns a formatted summary showing which specifications are completed and pending.
// If workspace is nil or error occurs, returns an error message.
func buildSpecificationTrackingSummary(workspace *storage.Workspace, taskID string) string {
	if workspace == nil {
		return "- No workspace available"
	}

	specifications, err := workspace.ListSpecificationsWithStatus(taskID)
	if err != nil {
		return fmt.Sprintf("- Error listing specifications: %v", err)
	}

	if len(specifications) == 0 {
		return "- No specifications yet"
	}

	var summary []string
	for _, specification := range specifications {
		status := specification.Status
		if status == "" {
			// Empty status means "not yet started" - use constant from storage
			status = storage.SpecificationStatusDraft
		}

		line := fmt.Sprintf("- Specification %d: %s", specification.Number, status)
		if len(specification.ImplementedFiles) > 0 {
			line += fmt.Sprintf(" (%d files)", len(specification.ImplementedFiles))
		}
		summary = append(summary, line)
	}

	return strings.Join(summary, "\n")
}

// computeQualityGateInstructions creates a prompt section with pre-review quality checklist.
// Returns instructions for self-verification before completing implementation.
// Called once at init time; use getQualityGateInstructions() for cached access.
func computeQualityGateInstructions() string {
	return `
## Pre-Review Quality Checklist

Before completing implementation:

### Code Quality:
✓ Compiles without errors
✓ No syntax errors
✓ Error handling present
✓ Descriptive names
✓ Follows existing style

### Functional Completeness:
✓ All specification requirements addressed
✓ Edge cases handled
✓ Helpful error messages
✓ Sensible defaults

### Testing:
✓ Code is testable
✓ Critical paths covered (if requested)
✓ Manual testing steps documented

### Verification:
1. Review yaml:file blocks above
2. Check each file change
3. Verify specification status updated to "completed + timestamp"
4. Confirm all validation steps included

If issues found, provide additional yaml:file blocks.
Only respond "IMPLEMENTATION_COMPLETE" when all checks pass.
`
}
