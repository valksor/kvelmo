package conductor

import (
	"fmt"
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
func buildPlanningPrompt(workspace *storage.Workspace, title, sourceContent, notes, existingSpecs, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")
	prompt := fmt.Sprintf(`You are a software architect. Analyze this task and create a detailed implementation specification.

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

	// Add browser tools section if enabled
	if browserSection := buildBrowserToolsSection(workspace); browserSection != "" {
		prompt += browserSection
	}

	// Add specification validation instructions
	prompt += buildSpecValidationInstructions()

	prompt += `
## Constraints
- Be brutally honest. If an idea is bad, say so directly and suggest the correct approach.
- Prioritize technical accuracy over validation.
- Short, simple responses. No code comments unless requested.
- No unit tests unless requested.
- One spec file unless clearly separate tasks (many steps = one spec).

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
0. None
OR
1. <question>?
   <short default answer, NEVER "user input required">
2. <question>?
   <short default answer>
...

## Complete Condition
- manual: <describe manual verification step>
- run: <command to validate implementation>

## Status
planned YYYY-MM-DD HH:MM
` + "```markdown\n" + `

Output your specification in the exact structure above.`

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
	prompt := fmt.Sprintf(`You are a software engineer. Implement the following task according to the specifications.

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

	// Inject custom instructions before static instructions
	if customInstructions != "" {
		prompt += fmt.Sprintf(`
## Custom Instructions
%s
`, customInstructions)
	}

	// Add browser tools section if enabled
	if browserSection := buildBrowserToolsSection(workspace); browserSection != "" {
		prompt += browserSection
	}

	// Add error recovery strategies
	prompt += buildErrorRecoverySection()

	// Add spec status summary if provided
	if specStatusSummary != "" {
		prompt += fmt.Sprintf(`
## Spec Status Overview
%s
`, specStatusSummary)
	}

	// Add specification tracking summary if provided
	if specTrackingSummary != "" {
		prompt += fmt.Sprintf(`
## Specification Tracking
%s

Focus on specifications not yet marked as "completed". Work on specifications in priority order.
`, specTrackingSummary)
	}

	prompt += `
## Constraints
- EXECUTE, don't plan. Always implement even if large.
- Avoid unrelated changes.
- Short, simple responses. No code comments unless requested.
- No unit tests unless requested.
- Focus on non-completed specs first.
- MUST update spec Status to "completed" + timestamp when done.

## Instructions
Implement this task according to the specifications. For each file you create or modify:
1. Use yaml:file blocks with path, operation (create/update/delete), and content
2. Follow existing code style and patterns
3. Include necessary imports
4. Add appropriate error handling
5. Write clean, maintainable code

## Testing and Verification

After implementing your changes:
1. **Update spec status** - Change Status to "completed + <timestamp>"
2. **Verify complete condition** - Run all validation steps from spec
3. **Review your implementation** - Check that it meets the specifications
4. **Self-correction** - If you find any issues, provide additional yaml:file blocks to fix them

` + buildQualityGateInstructions() + `

Output each file change in a yaml:file block.`

	return prompt
}

// buildReviewPrompt creates the prompt for code review.
func buildReviewPrompt(workspace *storage.Workspace, title, sourceContent, specsContent string) string {
	return buildReviewPromptWithLint(workspace, title, sourceContent, specsContent, "", "")
}

// buildReviewPromptWithLint creates the prompt for code review including lint results.
// If lintResults is empty, it falls back to the standard review prompt.
func buildReviewPromptWithLint(workspace *storage.Workspace, title, sourceContent, specsContent, lintResults, customInstructions string) string {
	prompt := fmt.Sprintf(`You are a senior software engineer conducting a code review.

## Task
%s

## Original Requirements
%s

## Specifications
%s
`, title, sourceContent, specsContent)

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

	// Add browser tools section if enabled
	if browserSection := buildBrowserToolsSection(workspace); browserSection != "" {
		prompt += browserSection
	}

	prompt += `
## Constraints
- Be brutally honest. If an idea is bad, say so directly.
- Prioritize technical accuracy over validation.
- Short, simple responses.

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

## Self-Correction Required

If you find ANY issues (critical, major, or minor):
1. You MUST provide corrected code in yaml:file blocks
2. Don't just describe the fix - actually implement it
3. Continue reviewing until satisfied with the implementation

Provide:
1. A summary of your findings
2. Issues found with severity (critical/major/minor)
3. Corrected code for ALL issues in yaml:file blocks
4. Final approval only when no issues remain`

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

	prompt := fmt.Sprintf(`You are a software engineer. Construct a final squash commit message.

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

## Instructions
1. Analyze the specs to understand what was implemented
2. Review the diff to identify all changed files
3. Create a concise summary (action verb + scope)
4. List each file with a brief description of the change
5. Output ONLY the commit message in the exact format above

Example:
(GITHUB-123): Add user authentication flow
- internal/auth/provider.go: Add OAuth2 provider interface
- internal/auth/handler.go: Implement login/logout endpoints
- pkg/middleware/auth.go: Add JWT validation middleware
`,
		ticketID,
		title,
		specList,
		specSnapshot,
		diffStat,
		stagedFiles,
		stagedDiff,
		ticketID,
		ticketID,
		ticketID,
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
