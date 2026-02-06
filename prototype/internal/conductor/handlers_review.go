package conductor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/progress"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// RunReview executes the review phase.
func (c *Conductor) RunReview(ctx context.Context) error {
	c.publishProgress("Starting review phase...", 0)

	taskID := c.activeTask.ID

	// Create progress tracker for this phase
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		statusLine = progress.NewStatusLine("Reviewing")
		defer statusLine.Done()
	}

	// Get agent for reviewing step
	reviewAgent, err := c.GetAgentForStep(ctx, workflow.StepReviewing)
	if err != nil {
		return fmt.Errorf("get review agent: %w", err)
	}

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

	// Create session for this review run
	session, filename, err := c.workspace.CreateSession(taskID, "review", reviewAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	// Get source content
	sourceContent, err := c.workspace.GetSourceContent(taskID)
	if err != nil {
		return fmt.Errorf("get source content: %w", err)
	}

	// Get latest specification (review against most recent specification)
	specContent, specNum, _ := c.workspace.GetLatestSpecificationContent(taskID)
	if specContent != "" {
		c.publishProgress(fmt.Sprintf("Reviewing against specification-%d...", specNum), 5)
	}

	// Run automated linters if available
	c.publishProgress("Running automated linters...", 10)
	lintResults := c.runLinters(ctx)

	// Get security findings if available
	securityFindings := c.GetSecurityFindingsForReview()
	if securityFindings != "" {
		c.publishProgress("Including security scan findings...", 15)
	}

	// Build review prompt with lint results, security findings, and custom instructions
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "reviewing")
	prompt := buildReviewPromptWithLintAndSecurity(c.workspace, c.taskWork.Metadata.Title, sourceContent, specContent, lintResults, securityFindings, customInstructions)

	// Optimize prompt if enabled (CLI flag or workspace config)
	shouldOptimize := c.opts.OptimizePrompts || shouldOptimizePrompt(workspaceCfg, "reviewing")
	if shouldOptimize {
		prompt = c.optimizePrompt(ctx, "reviewing", prompt)
	}

	// Run agent, accumulate output for transcript
	c.publishProgress("Agent reviewing...", 20)
	var transcriptBuilder strings.Builder
	response, err := reviewAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		// Always publish to event bus
		c.publishAgentEvent(event)
		// Also track progress if not dry-run
		if statusLine != nil {
			_ = statusLine.OnEvent(event)
		}
		// Accumulate content for transcript archive
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		if statusLine != nil {
			statusLine.Done()
		}
		c.activeTask.State = "idle"
		if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
			c.logError(fmt.Errorf("save active task after review error: %w", err))
		}
		if dispatchErr := c.dispatchWithRetry(ctx, workflow.EventError); dispatchErr != nil {
			c.logError(dispatchErr)
		}

		return fmt.Errorf("agent review: %w", err)
	}

	// Record usage stats
	if response.Usage != nil {
		if err := c.workspace.AddUsage(taskID, "review",
			response.Usage.InputTokens,
			response.Usage.OutputTokens,
			response.Usage.CachedTokens,
			response.Usage.CostUSD,
		); err != nil {
			c.logError(fmt.Errorf("record review usage: %w", err))
		}
		if err := c.checkBudgets(ctx, "reviewing"); err != nil {
			return err
		}
	}

	// Save full transcript for archive
	if transcript := transcriptBuilder.String(); transcript != "" {
		transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-review.log"
		if err := c.workspace.SaveTranscript(taskID, transcriptFile, transcript); err != nil {
			c.logError(fmt.Errorf("save review transcript: %w", err))
		}
	}

	// Record exchanges to session
	if c.currentSession != nil {
		now := time.Now()
		promptSummary := prompt
		if len(promptSummary) > 500 {
			promptSummary = promptSummary[:500] + "..."
		}
		c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
			Role:      "user",
			Content:   promptSummary,
			Timestamp: now,
		})
		// Record review summary
		reviewSummary := response.Summary
		if reviewSummary == "" && len(response.Messages) > 0 {
			reviewSummary = response.Messages[0]
		}
		if reviewSummary != "" {
			if len(reviewSummary) > 500 {
				reviewSummary = reviewSummary[:500] + "..."
			}
			c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
				Role:      "agent",
				Content:   reviewSummary,
				Timestamp: now,
			})
		}
	}

	c.publishProgress("Processing review...", 70)

	// Save review as note
	reviewContent := response.Summary
	if reviewContent == "" && len(response.Messages) > 0 {
		reviewContent = response.Messages[0]
	}
	if reviewContent != "" {
		if err := c.workspace.AppendNote(taskID, "## Review Results\n\n"+reviewContent, "reviewing"); err != nil {
			c.logError(fmt.Errorf("append review note: %w", err))
		}

		// Save review as a review file (enables ListReviews, LoadReview, and implement-review workflow)
		nextReviewNum, err := c.workspace.NextReviewNumber(taskID)
		if err != nil {
			c.logError(fmt.Errorf("get next review number: %w", err))
		} else {
			var reviewBuilder strings.Builder
			reviewBuilder.WriteString(fmt.Sprintf("# Code Review - %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
			reviewBuilder.WriteString(fmt.Sprintf("Task: %s\n", taskID))
			status := "PASSED"
			if len(response.Files) > 0 || strings.Contains(strings.ToLower(reviewContent), "issue") {
				status = "ISSUES"
			}
			reviewBuilder.WriteString(fmt.Sprintf("Status: %s\n\n---\n\n", status))
			reviewBuilder.WriteString(reviewContent)

			if saveErr := c.workspace.SaveReview(taskID, nextReviewNum, reviewBuilder.String()); saveErr != nil {
				c.logError(fmt.Errorf("save review: %w", saveErr))
			}
		}
	}

	// Apply any suggested fixes if not dry-run
	if !c.opts.DryRun && len(response.Files) > 0 {
		if err := applyFiles(ctx, c, response.Files); err != nil {
			c.logError(fmt.Errorf("apply review fixes: %w", err))
		}

		// Create checkpoint for review fixes
		commitMsg := c.generateCommitMessage(ctx, "review fixes")
		c.createCheckpointIfNeeded(ctx, taskID, commitMsg)
	}

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	if err := c.dispatchWithRetry(ctx, workflow.EventReviewDone); err != nil {
		return err
	}

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Review complete", 100)

	return nil
}

// buildHistoricalContext reconstructs Q&A history from previous sessions or transcripts.
// With fullContext=true, loads from transcripts; otherwise uses session exchange summaries.
func (c *Conductor) buildHistoricalContext(taskID string, fullContext bool) string {
	var context strings.Builder

	if fullContext {
		// Load from transcripts for full context
		transcripts, err := c.workspace.ListTranscripts(taskID)
		if err != nil || len(transcripts) == 0 {
			return ""
		}
		for _, t := range transcripts {
			// Only include answer transcripts (Q&A context)
			if !strings.Contains(t, "-answer.log") {
				continue
			}
			content, err := c.workspace.LoadTranscript(taskID, t)
			if err != nil {
				continue
			}
			context.WriteString(content)
			context.WriteString("\n\n---\n\n")
		}
	} else {
		// Load from session exchanges (summaries)
		sessions, err := c.workspace.ListSessions(taskID)
		if err != nil || len(sessions) == 0 {
			return ""
		}
		for _, sess := range sessions {
			for _, ex := range sess.Exchanges {
				// Only include Q&A exchanges (marked with QUESTION: or ANSWER:)
				if strings.HasPrefix(ex.Content, "QUESTION:") || strings.HasPrefix(ex.Content, "ANSWER:") {
					context.WriteString(fmt.Sprintf("[%s] %s\n", ex.Role, ex.Content))
				}
			}
		}
	}

	return context.String()
}

// optimizePrompt runs the optimizer agent to refine a prompt before execution.
// Agent resolution follows the pattern: optimizing step -> current step -> global default.
// Returns the optimized prompt, or falls back to original if optimization fails.
func (c *Conductor) optimizePrompt(ctx context.Context, phase, prompt string) string {
	// Try to get optimizer agent with fallback chain:
	// 1. Dedicated "optimizing" agent (if configured)
	// 2. The current step's agent (e.g., "planning" agent for planning phase)
	// 3. Global default agent
	var optimizerAgent agent.Agent

	// First try: dedicated optimizing agent
	optimizerAgent, err := c.GetAgentForStep(ctx, workflow.StepOptimizing)
	if err != nil {
		// Second try: use the current step's agent
		optimizerAgent, err = c.GetAgentForStep(ctx, workflow.Step(phase))
		if err != nil {
			c.logVerbosef("Optimizer agent not available, using original prompt")

			return prompt
		}
	}

	// Build optimizer prompt
	optimizerPrompt := buildOptimizerPrompt(phase, prompt)

	c.publishProgress("Optimizing prompt...", 5)

	// Run optimizer (no streaming needed, just get the result)
	response, err := optimizerAgent.Run(ctx, optimizerPrompt)
	if err != nil {
		c.logError(fmt.Errorf("prompt optimization failed: %w", err))
		c.publishProgress("Optimization failed, using original prompt", 10)

		return prompt // Fall back to original
	}

	// Extract optimized prompt from response
	optimizedPrompt := response.Summary
	if optimizedPrompt == "" && len(response.Messages) > 0 {
		optimizedPrompt = response.Messages[0]
	}

	if optimizedPrompt == "" {
		c.publishProgress("Optimization returned empty, using original prompt", 10)

		return prompt
	}

	// Log optimization stats
	originalLen := len(prompt)
	optimizedLen := len(optimizedPrompt)
	reduction := float64(originalLen-optimizedLen) / float64(originalLen) * 100
	c.logVerbosef("Prompt optimized: %d -> %d chars (%.1f%% reduction)",
		originalLen, optimizedLen, reduction)

	c.publishProgress("Prompt optimized", 10)

	return optimizedPrompt
}

// generateCommitMessage asks the agent to generate a descriptive commit message
// based on the git changes. Returns the generated message or falls back to phase name.
func (c *Conductor) generateCommitMessage(ctx context.Context, phase string) string {
	if c.git == nil {
		return phase
	}

	// Get git change summary
	changes, err := c.git.GetChangeSummary(ctx)
	if err != nil || changes.Total == 0 {
		return phase
	}

	// Build prompt for commit message generation
	var prompt strings.Builder
	prompt.WriteString("Generate a concise git commit message (max 72 characters) describing the following changes.\n\n")
	prompt.WriteString(fmt.Sprintf("Phase: %s\n\n", phase))
	prompt.WriteString("Files changed:\n")

	if len(changes.Added) > 0 {
		prompt.WriteString(fmt.Sprintf("Added: %s\n", strings.Join(changes.Added, ", ")))
	}
	if len(changes.Modified) > 0 {
		prompt.WriteString(fmt.Sprintf("Modified: %s\n", strings.Join(changes.Modified, ", ")))
	}
	if len(changes.Deleted) > 0 {
		prompt.WriteString(fmt.Sprintf("Deleted: %s\n", strings.Join(changes.Deleted, ", ")))
	}

	prompt.WriteString(`
Return ONLY the commit message text, nothing else. Use imperative mood ("add" not "added").
Be specific about what was changed based on the file names. Focus on the most important change.`)

	// Get agent for commit message generation (can use cheaper model via config)
	commitAgent, err := c.GetAgentForStep(ctx, workflow.StepCheckpointing)
	if err != nil {
		c.logError(fmt.Errorf("get checkpointing agent for commit message: %w", err))

		return phase
	}

	// Call agent to generate commit message
	response, err := commitAgent.Run(ctx, prompt.String())
	if err != nil {
		c.logError(fmt.Errorf("generate commit message: %w", err))

		return phase
	}

	// Extract the commit message from response
	if len(response.Messages) > 0 {
		msg := strings.TrimSpace(response.Messages[0])
		if msg != "" {
			// Strip any leading [taskID] pattern that AI might have included
			// since the commit prefix is already added by the caller
			if idx := strings.Index(msg, "] "); idx != -1 && strings.HasPrefix(msg, "[") {
				msg = strings.TrimSpace(msg[idx+2:])
			}

			return msg
		}
	}

	return phase
}

// isRecoverableError checks if an error is recoverable and should trigger a retry.
// Returns true for errors like context overflow, token limits, timeouts, and rate limits.
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	// Use word boundary patterns to avoid false positives
	// Format: "pattern" or "pattern variations"
	recoverablePatterns := []struct {
		pattern      string
		wordBoundary bool
	}{
		{"context overflow", true},
		{"context length exceeded", true},
		{"token limit exceeded", true},
		{"maximum tokens exceeded", true},
		{"request timeout", false}, // False: can have suffixes
		{"connection timeout", false},
		{"rate limit", true},
		{"rate limited", true},
		{"too many requests", true},
		{"429", false}, // HTTP status can have whitespace
	}

	errMsg := strings.ToLower(err.Error())
	for _, rp := range recoverablePatterns {
		if rp.wordBoundary {
			// Match with word boundaries to avoid false positives
			// e.g., "timeout" should not match in "my_timeout_function"
			pattern := `\b` + regexp.QuoteMeta(rp.pattern) + `\b`
			matched, _ := regexp.MatchString(pattern, errMsg)
			if matched {
				return true
			}
		} else {
			// Simple substring match for patterns that may have suffixes/prefixes
			if strings.Contains(errMsg, rp.pattern) {
				return true
			}
		}
	}

	return false
}
