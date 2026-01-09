package conductor

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// buildPlanningPrompt creates the prompt for specification generation.
//
// The workspace parameter is used to:
//   - Load browser automation configuration (if enabled)
//   - Load custom agent instructions from workspace config
//
// If workspace is nil, browser tools and custom instructions are omitted.
//
// If useDefaults is true, the agent will provide best-guess default answers for unknowns
// without asking the user. If false (default), the agent will ask the user for clarification.
func buildPlanningPrompt(workspace *storage.Workspace, title, sourceContent, notes, existingSpecs, customInstructions string, useDefaults bool) string {
	currentTime := time.Now().Format("2006-01-02 15:04")
	prompt := fmt.Sprintf(`You are an expert software engineer specializing in architecture and system design. Analyze this task and create a detailed implementation specification.

Current timestamp: %s

## Task
%s

## Source Content
%s
`, currentTime, title, sourceContent)

	if existingSpecs != "" {
		prompt += fmt.Sprintf(`
## Previous Specifications
IMPORTANT: The following specifications already exist from previous planning iterations.
DO NOT start from scratch. Build upon these, refine them, or address any gaps:

%s

Your new specification should acknowledge what was already planned and either:
1. Refine/improve the existing specification
2. Add missing details
3. Address any gaps or questions that arose
`, existingSpecs)
	}

	if notes != "" {
		prompt += fmt.Sprintf(`
## Additional Notes
%s
`, notes)
	}

	// Inject custom instructions before static instructions
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	// Add browser tools section if enabled (step-specific)
	if browserSection := buildBrowserToolsSectionForStep(workspace, "planning"); browserSection != "" {
		prompt += browserSection
	}

	// Add specification validation instructions
	prompt += buildSpecValidationInstructions()

	// Add chain-of-thought guidance for planning
	prompt += `
## Approach
Before writing your specification:
1. Identify existing patterns in the codebase that apply to this task
2. Consider integration points and dependencies
3. Think through edge cases and failure scenarios
4. Evaluate trade-offs if multiple approaches exist

Briefly explain your key architectural decisions before the specification.
`

	prompt += `
## Constraints
- If the approach has flaws, explain the specific problem and suggest an alternative
- If requirements are ambiguous, list your assumptions explicitly
- Prefer existing patterns in the codebase over introducing new ones
- If you're uncertain about something, say so rather than guessing
- Keep specifications focused - one spec file unless tasks are clearly independent
- No unit tests unless explicitly requested

## Instructions
Create a detailed specification that includes:
1. **Request** - The task requirement in your own words
2. **Plan** - Numbered implementation steps (1. <step>, 2. <step>, ...)
3. **Context** - Files to reference with format: path/to/file:line-range: description
4. **Unknowns** - Numbered questions (1-10) or "0. None" if none
5. **Complete Condition** - Validation steps:
   - manual: <describe manual verification step>
   - run: <command that validates implementation>
6. **Status** - Current state: "planned" + timestamp

## SPEC Template Structure
` + "```markdown\n" + `## Request
<task description in your own words>

## Plan
1. <first implementation step>
2. <second implementation step>
...

## Context
path/to/file:line-range: <description of relevant code>
path/to/file:line-range: <description of relevant code>
...

## Unknowns
` + buildUnknownsSection(useDefaults) + `

## Complete Condition
- manual: <describe manual verification step>
- run: <command to validate implementation>

## Status
planned YYYY-MM-DD HH:MM
` + "```\n" + `

## Example Specification
` + "```markdown\n" + `## Request
Add rate limiting to the /api/users endpoint to prevent abuse.

## Plan
1. Add rate limiter middleware using golang.org/x/time/rate
2. Configure limits in config.yaml (100 req/min default)
3. Return 429 status with Retry-After header when exceeded
4. Add rate limit headers to all responses (X-RateLimit-*)

## Context
internal/api/middleware/auth.go:15-45: existing middleware pattern to follow
internal/config/config.go:89-102: config loading pattern
cmd/server/main.go:34: middleware registration point

## Unknowns
1. Should rate limits be per-user or per-IP?
   Default: Per-IP for unauthenticated, per-user for authenticated
2. Should limits be configurable per-endpoint?
   Default: No, use global limit initially

## Complete Condition
- manual: Send 101 requests in 1 minute, verify 429 response on 101st
- run: go test ./internal/api/middleware/... -run TestRateLimit

## Status
planned 2024-01-15 14:30
` + "```\n" + `

Output your specification in the exact structure above.

## Required Output Format
Your response MUST include:
1. Brief reasoning about your approach (from the ## Approach section)
2. A specification with ALL sections: Request, Plan, Context, Unknowns, Complete Condition, Status`

	return prompt
}

// buildImplementationPrompt creates the prompt for implementation.
//
// The workspace parameter is used to:
//   - Load browser automation configuration (if enabled)
//   - Load custom agent instructions from workspace config
//   - Build specification status and tracking summaries
//
// If workspace is nil, these features are disabled.
func buildImplementationPrompt(workspace *storage.Workspace, title, sourceContent, specsContent, notes, customInstructions, specStatusSummary, specTrackingSummary string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	// Phase 1: Context (Task, Requirements, Specifications)
	prompt := fmt.Sprintf(`You are an expert software engineer. Implement the following task according to the specifications.

Current timestamp: %s

## Task
%s

## Original Requirements
%s

## Specifications
%s
`, currentTime, title, sourceContent, specsContent)

	if notes != "" {
		prompt += fmt.Sprintf(`
## Additional Notes
%s
`, notes)
	}

	// Phase 2: User Priorities (Custom Instructions)
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	// Phase 3: Behavior (Constraints)
	prompt += `
## Constraints
- EXECUTE, don't plan - implement the specification even if it's large
- Stay focused - avoid unrelated changes or "improvements" outside the spec
- Follow existing code patterns in the codebase
- No unit tests unless explicitly requested in the specification
- Focus on non-completed specifications first
- MUST update spec Status to "completed YYYY-MM-DD HH:MM" when done
`

	// Phase 4: What to do (Instructions)
	prompt += `
## Instructions
Implement this task according to the specifications. For each file you create or modify:
1. Use yaml:file blocks with path, operation (create/update/delete), and content
2. Follow existing code style and patterns
3. Include necessary imports
4. Add appropriate error handling
5. Write clean, maintainable code

Output each file change in a yaml:file block.
`

	// Phase 5: Progress (Spec Status/Tracking)
	if specStatusSummary != "" {
		prompt += fmt.Sprintf(`
## Spec Status Overview
%s
`, specStatusSummary)
	}

	if specTrackingSummary != "" {
		prompt += fmt.Sprintf(`
## Specification Tracking
%s

Focus on specifications not yet marked as "completed". Work on specifications in priority order.
`, specTrackingSummary)
	}

	// Phase 6: Reference Material (Error Recovery, Quality Gates, Verification)
	prompt += buildErrorRecoverySection()

	prompt += `
## Testing and Verification

After implementing your changes:
1. **Update spec status** - Change Status to "completed YYYY-MM-DD HH:MM"
2. **Verify complete condition** - Run all validation steps from spec
3. **Review your implementation** - Check that it meets the specifications
4. **Self-correction** - If you find any issues, provide additional yaml:file blocks to fix them

` + buildQualityGateInstructions()

	// Phase 7: Browser Tools (optional, at end)
	if browserSection := buildBrowserToolsSectionForStep(workspace, "implementing"); browserSection != "" {
		prompt += browserSection
	}

	// Example at end for reference
	prompt += `
## Example Output
` + "```yaml:file\n" + `path: internal/api/middleware/ratelimit.go
operation: create
content: |
  package middleware

  import (
    "net/http"
    "golang.org/x/time/rate"
  )

  // RateLimit creates a rate limiting middleware.
  func RateLimit(limit rate.Limit, burst int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(limit, burst)
    return func(next http.Handler) http.Handler {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
          http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
          return
        }
        next.ServeHTTP(w, r)
      })
    }
  }
` + "```" + `

## Required Output Format
Your response MUST include:
1. yaml:file blocks for each file created or modified
2. Updated specification status to "completed YYYY-MM-DD HH:MM"`

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

// buildErrorRecoverySection creates a prompt section with error recovery strategies.
// Returns instructions for handling common failure scenarios.
func buildErrorRecoverySection() string {
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

// buildSpecValidationInstructions creates a prompt section with specification quality checklist.
// Returns instructions for validating specification completeness and quality.
func buildSpecValidationInstructions() string {
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

// buildQualityGateInstructions creates a prompt section with pre-review quality checklist.
// Returns instructions for self-verification before completing implementation.
func buildQualityGateInstructions() string {
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

// formatSpecificationContent formats a specification file from agent response.
func formatSpecificationContent(num int, response *agent.Response) string {
	content := fmt.Sprintf("# Specification %d\n\n", num)

	if response.Summary != "" {
		content += "## Summary\n\n" + response.Summary + "\n\n"
	}

	if len(response.Messages) > 0 {
		content += "## Details\n\n"
		var contentSb156 strings.Builder
		for _, msg := range response.Messages {
			contentSb156.WriteString(msg + "\n\n")
		}
		content += contentSb156.String()
	}

	return content
}

// extractContextSummary extracts a brief summary from the agent response.
// Uses the Summary field if available, otherwise truncates the first message.
func extractContextSummary(response *agent.Response) string {
	if response.Summary != "" {
		return response.Summary
	}
	if len(response.Messages) > 0 {
		msg := response.Messages[0]
		// Truncate to ~2000 chars for token efficiency
		if len(msg) > 2000 {
			return msg[:2000] + "\n[truncated...]"
		}

		return msg
	}

	return ""
}

// buildFullContext combines all agent output into a single context string.
// This includes the summary and all messages.
func buildFullContext(response *agent.Response) string {
	var parts []string
	if response.Summary != "" {
		parts = append(parts, "## Summary\n"+response.Summary)
	}
	if len(response.Messages) > 0 {
		parts = append(parts, "## Messages\n"+strings.Join(response.Messages, "\n\n"))
	}

	return strings.Join(parts, "\n\n")
}

// extractExploredFiles extracts file paths from the agent response.
// Returns paths from file changes and attempts to find file references in messages.
func extractExploredFiles(response *agent.Response) []string {
	seen := make(map[string]bool)
	var files []string

	// Add files from file changes
	for _, fc := range response.Files {
		if !seen[fc.Path] {
			seen[fc.Path] = true
			files = append(files, fc.Path)
		}
	}

	return files
}

// buildUnknownsSection creates the Unknowns section based on whether to use defaults or ask user.
func buildUnknownsSection(useDefaults bool) string {
	if useDefaults {
		return `0. None
OR
1. <question>?
   <your recommended default answer>
2. <question>?
   <your recommended default answer>
...

Note: Provide your best-guess default answers for any unknowns. Do not wait for user input.`
	}

	return `If you have questions about requirements or approach:
1. STOP and ask the user using the ask_question tool before proceeding
2. Do not guess - get clarification for important decisions
3. Only proceed to specification when all critical unknowns are resolved

If no questions, write:
0. None`
}

// buildCombinedInstructions combines global and step-specific instructions from workspace config.
// Global instructions are included first, then step-specific instructions are appended.
// Returns empty string if no instructions are configured.
func buildCombinedInstructions(cfg *storage.WorkspaceConfig, step string) string {
	if cfg == nil {
		return ""
	}

	var parts []string

	// Global instructions (apply to all steps)
	if global := strings.TrimSpace(cfg.Agent.Instructions); global != "" {
		parts = append(parts, global)
	}

	// Step-specific instructions (combined with global)
	if stepCfg, ok := cfg.Agent.Steps[step]; ok {
		if stepInstr := strings.TrimSpace(stepCfg.Instructions); stepInstr != "" {
			parts = append(parts, stepInstr)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}
