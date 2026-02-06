package conductor

import (
	"fmt"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

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
