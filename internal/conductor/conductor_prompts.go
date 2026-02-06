package conductor

import (
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
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
	prompt += getSpecValidationInstructions()

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
` + getUnknownsSection(useDefaults) + `

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

// buildSimplePlanningPrompt creates a minimal prompt for straightforward tasks.
// It produces the same output format as buildPlanningPrompt but with ~50% fewer tokens
// by omitting verbose guidance, examples, and optional context sections.
//
// Use this for tasks detected as "simple" by DetectTaskComplexity:
// - Short descriptions
// - Single file changes
// - Action keywords like "update", "bump", "fix typo".
func buildSimplePlanningPrompt(workingDir, title, sourceContent, notes string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert software engineer. Create a concise specification for this straightforward task.

Current timestamp: %s
Working directory: %s

## Task
%s

## Source Content
%s
`, currentTime, workingDir, title, sourceContent)

	if notes != "" {
		prompt += fmt.Sprintf(`
## Notes
%s
`, notes)
	}

	prompt += `
## Instructions
Create a brief specification with these sections:

1. **Request** - Task description in your own words
2. **Plan** - Numbered implementation steps
3. **Context** - Relevant files (path:lines: description)
4. **Unknowns** - Questions (numbered) or "0. None"
5. **Complete Condition** - How to verify (manual + run command)
6. **Status** - "planned" + current timestamp

## Output Format
` + "```markdown\n" + `## Request
<brief task description>

## Plan
1. <step>
2. <step>

## Context
path/to/file:lines: <description>

## Unknowns
0. None

## Complete Condition
- manual: <verification step>
- run: <validation command>

## Status
planned YYYY-MM-DD HH:MM
` + "```\n" + `
Output your specification in this exact structure.`

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
	prompt += getErrorRecoverySection()

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

` + getQualityGateInstructions()

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

// computeUnknownsSection creates the Unknowns section based on whether to use defaults or ask user.
// Called once at init time; use getUnknownsSection() for cached access.
func computeUnknownsSection(useDefaults bool) string {
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
1. STOP and ask the user using the AskUserQuestion tool before proceeding
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
