package conductor

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

const (
	// maxContextLength is the maximum length for context summaries.
	maxContextLength = 2000
)

// buildHierarchySection formats hierarchical task context (parent and siblings) for prompts.
func buildHierarchySection(hierarchy *HierarchicalContext) string {
	if hierarchy == nil {
		return ""
	}

	var sections []string

	// Add parent context
	if hierarchy.Parent != nil {
		parentDesc := hierarchy.Parent.Description
		if len(parentDesc) > 500 {
			parentDesc = parentDesc[:500] + "..."
		}

		sections = append(sections, fmt.Sprintf(`### Parent Task Context
**Title:** %s
**Status:** %s
**Description:**
%s

This is a subtask of the parent task above. Consider how your work fits into the broader context.
`, hierarchy.Parent.Title, hierarchy.Parent.Status, parentDesc))
	}

	// Add siblings context
	if len(hierarchy.Siblings) > 0 {
		var siblingList []string
		for _, s := range hierarchy.Siblings {
			siblingList = append(siblingList, fmt.Sprintf("- **%s** (Status: %s)", s.Title, s.Status))
		}

		sections = append(sections, fmt.Sprintf(`### Related Subtasks
%s

Consider how your implementation relates to these sibling tasks. Avoid duplicating work and ensure consistency.
`, strings.Join(siblingList, "\n")))
	}

	if len(sections) == 0 {
		return ""
	}

	return fmt.Sprintf(`
## Hierarchical Context
%s
`, strings.Join(sections, "\n"))
}

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
func buildPlanningPrompt(workspace *storage.Workspace, workingDir, title, sourceContent, notes, existingSpecs, customInstructions string, useDefaults bool, hierarchy *HierarchicalContext) string {
	currentTime := time.Now().Format("2006-01-02 15:04")
	prompt := fmt.Sprintf(`You are an expert software engineer specializing in architecture and system design. Analyze this task and create a detailed implementation specification.

Current timestamp: %s
Working directory: %s

## Task
%s

## Source Content
%s
`, currentTime, workingDir, title, sourceContent)

	// Add hierarchical context if available
	if hierarchySection := buildHierarchySection(hierarchy); hierarchySection != "" {
		prompt += hierarchySection
	}

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
func buildImplementationPrompt(workspace *storage.Workspace, workingDir, title, sourceContent, specsContent, notes, customInstructions, specStatusSummary, specTrackingSummary string, hierarchy *HierarchicalContext) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	// Phase 1: Context (Task, Requirements, Specifications)
	prompt := fmt.Sprintf(`You are an expert software engineer. Implement the following task according to the specifications.

Current timestamp: %s
Working directory: %s

## Task
%s

## Original Requirements
%s

## Specifications
%s
`, currentTime, workingDir, title, sourceContent, specsContent)

	// Add hierarchical context if available
	if hierarchySection := buildHierarchySection(hierarchy); hierarchySection != "" {
		prompt += hierarchySection
	}

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
- MUST update specification YAML frontmatter status to "done" when implementing a spec
`

	// Phase 4: What to do (Instructions)
	prompt += `
## Instructions
Implement this task according to the specifications. For each file you create or modify:
1. Use yaml:file blocks with path, operation (create/update/delete), and content
2. File paths must be relative to the working directory (e.g., "internal/foo.go" NOT "/full/path/foo.go")
3. Follow existing code style and patterns
4. Include necessary imports
5. Add appropriate error handling
6. Write clean, maintainable code

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

Focus on specifications with status "draft" (not yet "done"). Work on specifications in priority order.
`, specTrackingSummary)
	}

	// Phase 6: Reference Material (Error Recovery, Quality Gates, Verification)
	prompt += buildErrorRecoverySection()

	prompt += `
## Testing and Verification

After implementing your changes:
1. **Update specification status** - Edit the specification file and change the YAML frontmatter status field from "draft" to "done"
2. **Verify complete condition** - Run all validation steps from spec
3. **Review your implementation** - Check that it meets the specifications
4. **Self-correction** - If you find any issues, provide additional yaml:file blocks to fix them

IMPORTANT: When updating a specification to "done", use yaml:file to edit the spec file:
` + "```yaml:file\n" + `path: .mehrhof/work/{task-id}/specifications/specification-1.md
operation: update
content: |
  ---
  title: Specification 1
  status: done
  created_at: 2026-01-28T14:36:22Z
  updated_at: 2026-01-28T14:40:00Z
  implemented_files:
      - hello.md
  ---

  # Specification 1
  ... (rest of content unchanged)
` + "```" + `

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
2. yaml:file block to update the specification's YAML frontmatter status to "done" (see example above)`

	return prompt
}

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
		var detailsBuilder strings.Builder
		for _, msg := range response.Messages {
			detailsBuilder.WriteString(msg + "\n\n")
		}
		content += detailsBuilder.String()
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
		// Truncate to maxContextLength chars for token efficiency
		if len(msg) > maxContextLength {
			return msg[:maxContextLength] + "\n[truncated...]"
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

// shouldOptimizePrompt checks if prompt optimization is enabled for a given step.
// Checks step-specific setting first, then falls back to global setting.
// Returns false if no config is provided.
func shouldOptimizePrompt(cfg *storage.WorkspaceConfig, step string) bool {
	if cfg == nil {
		return false
	}

	// Step-specific setting takes precedence
	// If step is explicitly configured (even with false), use that value
	if stepCfg, ok := cfg.Agent.Steps[step]; ok {
		return stepCfg.OptimizePrompts
	}

	// Fall back to global setting
	return cfg.Agent.OptimizePrompts
}

// buildSimplifyInputPrompt creates a prompt to simplify task input.
func buildSimplifyInputPrompt(title, sourceContent, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert technical writer. Simplify and refine the following task description to make it clearer and more actionable.

Current timestamp: %s

## Task Title
%s

## Current Task Description
%s
`, currentTime, title, sourceContent)

	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Your Mission
Simplify the task description while preserving all requirements:

1. **Clarify the goal** - Make the objective crystal clear
2. **Remove ambiguity** - Eliminate vague language
3. **Improve structure** - Organize information logically
4. **Be specific** - Use precise technical language
5. **Stay concise** - Remove fluff without losing meaning

## Key Principles

1. **One goal per sentence** - Don't combine multiple requirements
2. **Use active voice** - "Implement X" not "X should be implemented"
3. **Define terms** - Introduce abbreviations before using them
4. **Prioritize** - Put the most important requirements first
5. **Be complete** - Don't remove any technical requirements

## Output Format
Return only the simplified task description. No markdown formatting, no sections - just the clear, refined task text.`

	return prompt
}

// buildSimplifyPlanningPrompt creates a prompt to simplify planning output.
func buildSimplifyPlanningPrompt(title, sourceContent, notes, specContent, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert technical writer and software architect. Simplify and refine the following planning specifications to make them clearer and more maintainable.

Current timestamp: %s

## Task
%s

## Original Requirements
%s
`, currentTime, title, sourceContent)

	if notes != "" {
		prompt += fmt.Sprintf(`
## Additional Notes
%s
`, notes)
	}

	prompt += fmt.Sprintf(`
## Current Specifications
%s
`, specContent)

	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Your Mission
Simplify the specifications while preserving all technical details and requirements. Your goal is to enhance clarity by:

1. **Reducing complexity** - Break down convoluted steps into clear, actionable items
2. **Improving names** - Use consistent, descriptive terminology
3. **Enhancing structure** - Organize information logically
4. **Maintaining balance** - Don't oversimplify to the point of losing important details
5. **Preserving functionality** - Every requirement must remain intact

## Key Principles

1. **Be precise, not brief** - Clarity trumps brevity
2. **Use active voice** - "Implement feature X" not "Feature X is implemented"
3. **One action per step** - Don't combine multiple operations
4. **Define terms** - Introduce abbreviations before using them
5. **Avoid weasel words** - No "should", "could", "might" - use definitive language

## Output Format
Return the complete simplified specifications using this format:

--- specification-N.md ---
[content of specification file]
--- end ---

Where N is the specification number.

Remember: You are simplifying for clarity, not removing content. Every technical requirement must be preserved.`

	return prompt
}

// buildSimplifyImplementingPrompt creates a prompt to simplify implemented code.
func buildSimplifyImplementingPrompt(title, sourceContent string, files map[string]string, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert code reviewer and refactoring specialist. Simplify and refine the recently implemented code to improve clarity and maintainability while preserving exact functionality.

Current timestamp: %s

## Task
%s

## Original Requirements
%s

## Implemented Files
The following files were recently modified:
`, currentTime, title, sourceContent)

	var promptSb strings.Builder
	for filePath, content := range files {
		promptSb.WriteString(fmt.Sprintf(`
### %s
%s
`, filePath, content))
	}
	prompt += promptSb.String()

	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Your Mission
Simplify the code while preserving exact functionality. Focus on:

1. **Improving names** - Better variable, function, and type names
2. **Reducing complexity** - Break down complex functions
3. **Enhancing readability** - Clear logic flow and structure
4. **Removing redundancy** - Eliminate duplicate code
5. **Applying patterns** - Use existing project patterns

## Key Principles

1. **Preserve behavior** - No functional changes allowed
2. **Follow conventions** - Match existing code style
3. **Be idiomatic** - Use language best practices
4. **Add clarity** - Better comments where needed
5. **Stay focused** - Only simplify recently modified code

## Output Format
Return the complete simplified code for each file using this format:

--- path/to/file.ext ---
[simplified code content]
--- end ---

Remember: You must preserve exact functionality.`

	return prompt
}

// buildPRReviewPrompt creates a prompt for PR/MR review.
// Supports incremental reviews by including previous state if available.
func buildPRReviewPrompt(pr *provider.PullRequest, diff *provider.PullRequestDiff, prevState *PRReviewState, scope string, workspace *storage.Workspace) string {
	currentTime := time.Now().Format("2006-01-02 15:04")
	isRerun := prevState != nil

	prompt := fmt.Sprintf(`You are an expert software engineer specializing in code review. Analyze this pull request for issues.

Current timestamp: %s

## Pull Request
**Number:** #%d
**Title:** %s
**Branch:** %s → %s
**URL:** %s
`, currentTime, pr.Number, pr.Title, diff.BaseBranch, diff.HeadBranch, pr.URL)

	// If this is a re-run, include context
	if isRerun {
		prompt += `
## Previous Review Context
This is a follow-up review after changes were made. Focus on:
1. **New issues** in the latest changes
2. **Previously reported issues** that still exist (don't re-comment)

### Previously Reported Issues (Do NOT re-comment on these)
`
		var prevIssuesSB strings.Builder
		for _, issue := range prevState.Issues {
			if issue.Status == "open" || issue.Status == "" {
				lineInfo := fmt.Sprintf("%s:%d", issue.File, issue.Line)
				if issue.Line == 0 {
					lineInfo = issue.File
				}
				prevIssuesSB.WriteString(fmt.Sprintf("- [%s] %s - %s\n", issue.Severity, lineInfo, issue.Message))
			}
		}
		prompt += prevIssuesSB.String()
		prompt += "\n"
	}

	// Load custom instructions from config if workspace is available
	var customInstructions string
	if workspace != nil {
		cfg, err := workspace.LoadConfig()
		if err == nil && cfg.Agent.Instructions != "" {
			customInstructions = cfg.Agent.Instructions
		}
	}

	// Inject custom instructions before static instructions
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Project Guidelines
%s
`, customInstructions)
	}

	prompt += `
## Approach
Before providing your review:
1. Trace through the code paths systematically
2. Check for common issues and anti-patterns
3. Consider edge cases the implementation might miss
4. Look for patterns that commonly cause bugs

Document your analysis before listing issues.

`

	// Add diff based on scope
	prompt += formatDiffForReview(diff, scope)

	prompt += `
## Review Instructions

Analyze the changes for:
1. **Code Quality** - Style, readability, maintainability
2. **Correctness** - Logic errors, edge cases, potential bugs
3. **Security** - Vulnerabilities, unsafe patterns
4. **Performance** - Inefficiencies, resource leaks

**Important:`
	if isRerun {
		prompt += `
- Do NOT re-report issues that were mentioned in "Previously Reported Issues"
- Only report NEW issues that weren't in the previous review
- If a previously reported issue is still present, don't mention it again
`
	} else {
		prompt += `
- Report all issues found in the code
`
	}

	prompt += `
## Output Format

Return your review in this structure:

` + "```markdown" + `
## Summary
[Brief overview of the review - 2-3 sentences]

## Overall Assessment
[One of: approved, changes_requested, comment]

## Issues
### Security
[CRITICAL] [file.go:42] Issue description...
[HIGH] [file.go:123] Issue description...

### Correctness
[HIGH] [file.go:45] Issue description...
[MEDIUM] [util.go:10] Issue description...

### Performance
[MEDIUM] [file.go:78] Issue description...

### Style
[LOW] [file.go:20] Issue description...

` + "```" + `

**Issue Format Guidelines:**
- Use format: [SEVERITY] [file:line] Description
- Severity levels: CRITICAL, HIGH, MEDIUM, LOW
- Categories: Security, Correctness, Performance, Style
- Include file path and line number for each issue
- Keep descriptions concise but actionable

**CRITICAL:** Be thorough but concise. Focus on actionable feedback.

## Required Output Format
Your response MUST include:
1. Summary section with brief overview
2. Overall assessment (approved/changes_requested/comment)
3. Issues section with all findings (if any)
`

	return prompt
}

// formatDiffForReview formats the PR diff based on scope.
func formatDiffForReview(diff *provider.PullRequestDiff, scope string) string {
	var sb strings.Builder

	sb.WriteString("## Current Changes\n\n")

	switch scope {
	case "files-changed":
		sb.WriteString("**Files changed:**\n\n")
		for _, file := range diff.Files {
			var status string
			switch file.Mode {
			case "added":
				status = "added"
			case "deleted":
				status = "deleted"
			case "renamed":
				status = "renamed"
			default:
				status = "modified"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, status))
		}

	case "compact":
		sb.WriteString("**Diff Summary:**\n\n")
		for _, file := range diff.Files {
			sb.WriteString(fmt.Sprintf("### %s\n", file.Path))
			sb.WriteString(fmt.Sprintf("- Mode: %s\n", file.Mode))
			sb.WriteString(fmt.Sprintf("- Additions: +%d\n", file.Additions))
			sb.WriteString(fmt.Sprintf("- Deletions: -%d\n", file.Deletions))
			// Show first few lines of patch
			lines := strings.Split(file.Patch, "\n")
			if len(lines) > 10 {
				sb.WriteString("```\n" + strings.Join(lines[:10], "\n") + "\n...\n```\n\n")
			} else if len(lines) > 0 {
				sb.WriteString("```\n" + file.Patch + "\n```\n\n")
			}
			sb.WriteString("\n")
		}

	default: // "full"
		sb.WriteString("**Full Diff:**\n\n")
		if len(diff.Files) > 0 {
			sb.WriteString(fmt.Sprintf("```diff\n%s\n```\n\n", diff.Patch))
		} else {
			sb.WriteString("*No changes*\n\n")
		}
	}

	return sb.String()
}

// buildQuestionPrompt creates a prompt for asking the agent a question during implementation.
// Includes task context, specifications, and recent conversation history.
func buildQuestionPrompt(title, question, specificationContent, sessionHistory string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are in the middle of implementing a task.

Current timestamp: %s

## Task
%s

`, currentTime, title)

	if specificationContent != "" {
		prompt += fmt.Sprintf(`## Current Specification
%s

`, specificationContent)
	}

	if sessionHistory != "" {
		prompt += fmt.Sprintf(`## Recent Conversation
%s

`, sessionHistory)
	}

	prompt += fmt.Sprintf(`## User Question
%s

Please answer the user's question based on the current implementation context.
Be concise and helpful. If you need more information to provide a good answer, ask a follow-up question.
`, question)

	return prompt
}

// buildFindPrompt creates a focused prompt for AI-powered code search.
// The prompt is designed to minimize fluff and get precise results.
// The workspace parameter is reserved for future custom instructions support.
func buildFindPrompt(query, workingDir string, workspace *storage.Workspace, opts FindOptions) string {
	_ = workspace // Reserved for future custom instructions
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are a PRECISE code search tool. Your job is to find code matching: %s

Current timestamp: %s
Working directory: %s
`, query, currentTime, workingDir)

	// Add path constraint if specified
	if opts.Path != "" {
		prompt += fmt.Sprintf(`Search path: %s
`, opts.Path)
	}

	// Add file pattern constraint if specified
	if opts.Pattern != "" {
		prompt += fmt.Sprintf(`File pattern: %s
`, opts.Pattern)
	}

	prompt += `
## CRITICAL CONSTRAINTS - READ THESE FIRST
1. Use Grep tool for code searches - it is FAST and PRECISE
2. Use Glob tool to find files by pattern
3. Use Read tool ONLY for confirmed matches
4. DO NOT explain your search process
5. DO NOT explore "related" code
6. DO NOT add "helpful" context between results
7. DO NOT say "let me search" or "I'll look for"
8. DO NOT summarize what you found

## SEARCH STRATEGY
1. Extract key search terms from the query:
   - Function names (e.g., "archive_blade" → "archive_blade", "ArchiveBlade")
   - Variables, types, constants
   - Common variants (camelCase, PascalCase, snake_case)

2. Use Grep with regex patterns:
   - Search for each term variant
   - Use case-insensitive flag when appropriate
   - Include word boundaries when searching for identifiers

3. Use Glob to narrow scope if helpful:
   - "**/*.go" for Go files
   - "**/*test*.go" for test files
   - Custom patterns based on query

4. Read confirmed matches to extract relevant snippets

## OUTPUT FORMAT
For EACH match you find, output EXACTLY this format:

--- FIND ---
file: <relative/path/to/file>
line: <line_number>
snippet: <the matching line>
context: <2 lines before + matching line + 2 lines after, each on separate "context:" lines>
reason: <brief explanation of why this matches>
--- END ---

## EXAMPLE

Query: "where is archive_blade database table used"

Good approach:
1. Grep for "archive_blade" (case-insensitive)
2. Grep for "ArchiveBlade" (PascalCase)
3. Read each match to verify it's the database table
4. Output results in the exact format above

Example output:
--- FIND ---
file: internal/models/archive.go
line: 42
snippet: type ArchiveBlade struct {
context: // Archive represents a blade in the system
context: type ArchiveBlade struct {
context: 	ID      uint
context: 	Name    string
context: }
reason: Model definition for ArchiveBlade
--- END ---

--- FIND ---
file: internal/db/query.go
line: 128
snippet: db.Find(&ArchiveBlade{}).Where(...)
context: 	}
context: 	var blades []ArchiveBlade
context: 	db.Find(&ArchiveBlade{}).Where("active = ?", true).Scan(&blades)
context: 	return blades, nil
context: }
reason: Database query using ArchiveBlade
--- END ---

## WHAT NOT TO DO (VIOLATIONS)
❌ "Let me search for the archive_blade table..."
❌ "I'll start by looking at the models directory..."
❌ "I found several uses of archive_blade:"
❌ "Here's what I discovered:"
❌ Exploring related database code without being asked
❌ Adding "helpful" context about the codebase

## WHAT TO DO (CORRECT BEHAVIOR)
✅ Use Grep immediately with relevant patterns
✅ Output results in the exact format specified
✅ Include context lines for each match
✅ Report only what the query asks for

## REMINDER
Your user wants RESULTS, not a tutorial. They know how to read code.
Just find the matching code and report it in the specified format.
`

	return prompt
}

// buildStandaloneReviewPrompt creates a prompt for standalone code review.
// This reviews code changes without requiring an active task.
func buildStandaloneReviewPrompt(workingDir, diff string, mode StandaloneDiffMode, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	modeDescription := getModeDescription(mode)

	prompt := fmt.Sprintf(`You are an expert software engineer specializing in code review and quality assurance.

Current timestamp: %s
Working directory: %s

## Review Mode
%s

## Code Changes to Review
`+"```diff\n"+`%s
`+"```"+`
`, currentTime, workingDir, modeDescription, diff)

	// Add custom instructions if provided
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Approach
Before providing your review:
1. Trace through the code paths systematically
2. Look for common issues and anti-patterns
3. Consider edge cases the implementation might miss
4. Check for patterns that commonly cause bugs

Document your analysis before listing issues.

## Constraints
- Focus on actionable feedback, not style nitpicks
- If you find issues, provide concrete suggestions for fixes
- Prioritize security and correctness over style

## Instructions
Review the code changes for:
1. **Correctness** - Logic errors, edge cases, potential bugs
2. **Security** - Vulnerabilities, unsafe patterns
3. **Performance** - Inefficiencies, resource leaks
4. **Code Quality** - Readability, maintainability
5. **Best Practices** - Language/framework conventions

## Output Format
` + "```markdown" + `
## Summary
[Brief overview of the review - 2-3 sentences]

## Overall Assessment
[One of: APPROVED, NEEDS_CHANGES, COMMENT]

## Issues
### Security
[CRITICAL] [file.go:42] Issue description...
[HIGH] [file.go:123] Issue description...

### Correctness
[HIGH] [file.go:45] Issue description...
[MEDIUM] [util.go:10] Issue description...

### Performance
[MEDIUM] [file.go:78] Issue description...

### Style
[LOW] [file.go:20] Issue description...
` + "```" + `

## Issue Format Guidelines
- Use format: [SEVERITY] [file:line] Description
- Severity levels: CRITICAL, HIGH, MEDIUM, LOW
- Categories: Security, Correctness, Performance, Style
- Include file path and line number for each issue
- Keep descriptions concise but actionable

## Required Output Format
Your response MUST include:
1. Summary section with brief overview
2. Overall assessment (APPROVED/NEEDS_CHANGES/COMMENT)
3. Issues section with all findings (if any), categorized by type
`

	return prompt
}

// buildStandaloneReviewWithFixesPrompt creates a prompt for standalone code review that also applies fixes.
// This reviews code changes and fixes the issues found.
func buildStandaloneReviewWithFixesPrompt(workingDir, diff string, mode StandaloneDiffMode, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	modeDescription := getModeDescription(mode)

	prompt := fmt.Sprintf(`You are an expert software engineer specializing in code review and bug fixing.

Current timestamp: %s
Working directory: %s

## Review Mode
%s

## Code Changes to Review and Fix
`+"```diff\n"+`%s
`+"```"+`
`, currentTime, workingDir, modeDescription, diff)

	// Add custom instructions if provided
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Your Mission
Review the code changes for issues AND fix any problems you find. This is a combined review + fix operation.

## Approach
1. First, analyze the code changes carefully
2. Identify any issues (bugs, security problems, performance issues)
3. For each fixable issue, apply the fix directly to the files
4. Document what you found and what you fixed

## Review Focus Areas
1. **Correctness** - Logic errors, edge cases, potential bugs
2. **Security** - Vulnerabilities, unsafe patterns
3. **Performance** - Inefficiencies, resource leaks
4. **Code Quality** - Readability, maintainability
5. **Best Practices** - Language/framework conventions

## Constraints
- Fix issues that are clearly bugs, security issues, or correctness problems
- Do NOT refactor or simplify code that is working correctly
- Do NOT change style or formatting unless it causes issues
- Preserve the original intent and functionality
- Only modify files that have actual issues

## Output Format
First, provide a review summary in markdown:

` + "```markdown" + `
## Summary
[Brief overview of what you found and fixed - 2-3 sentences]

## Overall Assessment
[One of: APPROVED (no issues/all fixed), NEEDS_CHANGES (issues remain unfixed), COMMENT (observations only)]

## Issues Found
[SEVERITY] [file:line] Description of issue - [FIXED/NOT_FIXED]

## Changes Made
- file1.go: Fixed null pointer dereference
- file2.go: Added input validation
` + "```" + `

Then, for each file you modified, output the corrected file content using yaml:file blocks:

` + "```yaml:file\n" + `path: relative/path/to/file.ext
operation: update
content: |
  [corrected file content]
` + "```" + `

## Instructions
1. Review the code changes for issues
2. Fix any issues you find by modifying the files
3. Provide the review summary with what you found and fixed
4. Output the fixed file contents using yaml:file blocks
`

	return prompt
}

// buildStandaloneSimplifyPrompt creates a prompt for standalone code simplification.
// This simplifies code changes without requiring an active task.
func buildStandaloneSimplifyPrompt(workingDir, diff string, mode StandaloneDiffMode, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	modeDescription := getModeDescription(mode)

	prompt := fmt.Sprintf(`You are an expert code reviewer and refactoring specialist. Simplify and refine the code to improve clarity and maintainability while preserving exact functionality.

Current timestamp: %s
Working directory: %s

## Simplify Mode
%s

## Code to Simplify
`+"```diff\n"+`%s
`+"```"+`
`, currentTime, workingDir, modeDescription, diff)

	// Add custom instructions if provided
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	prompt += `
## Your Mission
Simplify the code while preserving exact functionality. Focus on:

1. **Improving names** - Better variable, function, and type names
2. **Reducing complexity** - Break down complex functions
3. **Enhancing readability** - Clear logic flow and structure
4. **Removing redundancy** - Eliminate duplicate code
5. **Applying patterns** - Use existing project patterns

## Key Principles

1. **Preserve behavior** - No functional changes allowed
2. **Follow conventions** - Match existing code style
3. **Be idiomatic** - Use language best practices
4. **Add clarity** - Better comments where needed
5. **Stay focused** - Only simplify the code shown in the diff

## Output Format
Return the simplified code for each file using yaml:file blocks:

` + "```yaml:file\n" + `path: relative/path/to/file.ext
operation: update
content: |
  [simplified code content]
` + "```" + `

## Instructions
1. Analyze the diff to understand what code needs simplification
2. Identify opportunities for improvement (naming, structure, clarity)
3. Provide yaml:file blocks with the simplified versions
4. Ensure all changes preserve exact functionality

## Required Output Format
Your response MUST include:
1. Brief explanation of the simplifications made
2. yaml:file blocks for each file that should be updated
`

	return prompt
}

// getModeDescription returns a human-readable description of the diff mode.
func getModeDescription(mode StandaloneDiffMode) string {
	switch mode {
	case DiffModeUncommitted:
		return "Reviewing uncommitted changes (staged and unstaged)"
	case DiffModeBranch:
		return "Reviewing changes in current branch compared to base branch"
	case DiffModeRange:
		return "Reviewing changes in specified commit range"
	case DiffModeFiles:
		return "Reviewing specific files"
	default:
		return "Reviewing code changes"
	}
}
