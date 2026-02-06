package conductor

import (
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

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
