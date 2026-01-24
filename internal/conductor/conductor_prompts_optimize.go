package conductor

import (
	"fmt"
	"time"
)

// buildOptimizerPrompt creates a prompt for the optimizer agent.
// The optimizer's job is to refine a prompt for clarity, effectiveness, and conciseness.
func buildOptimizerPrompt(phase, originalPrompt string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert at refining AI prompts for optimal results.

Current timestamp: %s

## Your Task
You are optimizing a prompt for the "%s" phase of a software development workflow.
Your goal is to make the prompt MORE EFFECTIVE while keeping its essence intact.

## Original Prompt
%s

## Optimization Guidelines

1. **Clarity** - Make instructions explicit and unambiguous
2. **Structure** - Organize information logically (context -> task -> constraints -> output format)
3. **Conciseness** - Remove redundancy without losing critical information
4. **Precision** - Use specific technical language instead of vague terms
5. **Completeness** - Ensure all necessary context is preserved

## What NOT to Change

- Do NOT change the fundamental task or requirements
- Do NOT add new constraints or requirements
- Do NOT change the expected output format
- Do NOT alter code snippets, file paths, or technical references

## Output Format

Return ONLY the optimized prompt text. No explanations, no markdown formatting, no preamble.
Just the refined prompt that will be sent to the working agent.
`, currentTime, phase, originalPrompt)

	return prompt
}
