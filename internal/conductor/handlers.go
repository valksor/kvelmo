package conductor

import (
	"context"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/progress"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// ErrPendingQuestion is returned when the agent asks a question.
// Using errors.New() instead of fmt.Errorf() ensures errors.Is() works reliably.
var ErrPendingQuestion = errors.New("agent has a pending question")

const (
	// Retry configuration.
	maxRetries          = 3
	initialBackoffDelay = 1 * time.Second
	maxBackoffDelay     = 10 * time.Second
	backoffMultiplier   = 2.0
)

// RunPlanning executes the planning phase (creates SPEC files).
func (c *Conductor) RunPlanning(ctx context.Context) error {
	c.publishProgress("Starting planning phase...", 0)

	taskID := c.activeTask.ID

	// Create progress tracker for this phase
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		statusLine = progress.NewStatusLine("Planning")
		defer statusLine.Done()
	}

	// Get agent for planning step
	planningAgent, err := c.GetAgentForStep(ctx, workflow.StepPlanning)
	if err != nil {
		return fmt.Errorf("get planning agent: %w", err)
	}

	// Create session for this planning run
	session, filename, err := c.workspace.CreateSession(taskID, "planning", planningAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	// Get source content for the prompt
	sourceContent, err := c.workspace.GetSourceContent(taskID)
	if err != nil {
		return fmt.Errorf("get source content: %w", err)
	}

	// NOTE: Errors from workspace reads below are ignored intentionally.
	// Missing notes/specs are valid states (new task), so empty results are acceptable.
	var notes string
	notes, _ = c.workspace.ReadNotes(taskID)

	// Get existing specifications (for iterative planning)
	existingSpecifications, _ := c.workspace.GatherSpecificationsContent(taskID)
	if existingSpecifications != "" {
		specifications, _ := c.workspace.ListSpecifications(taskID)
		c.publishProgress(fmt.Sprintf("Found %d existing specification(s), including in context...", len(specifications)), 5)
	}

	// Check for pending context from previous planning session
	var pendingContext string
	if c.workspace.HasPendingQuestion(taskID) {
		pq, err := c.workspace.LoadPendingQuestion(taskID)
		if err == nil && pq != nil {
			// Use summary by default, full context if flag is set
			if c.opts.IncludeFullContext && pq.FullContext != "" {
				pendingContext = pq.FullContext
			} else if pq.ContextSummary != "" {
				pendingContext = pq.ContextSummary
			}
		}
		// Clear the pending question (answer should be in notes via note command)
		_ = c.workspace.ClearPendingQuestion(taskID)
	}

	// Load historical Q&A context from sessions (for when pending question was already answered)
	historicalContext := c.buildHistoricalContext(taskID, c.opts.IncludeFullContext)

	// Build planning prompt with custom instructions
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "planning")
	prompt := buildPlanningPrompt(c.workspace, c.taskWork.Metadata.Title, sourceContent, notes, existingSpecifications, customInstructions, c.opts.UseDefaults)
	if pendingContext != "" {
		prompt += "\n\n## Previous Analysis (before question)\nThe following is context from your previous planning session. Use this to avoid re-exploring:\n\n" + pendingContext
	}
	if historicalContext != "" {
		prompt += "\n\n## Previous Q&A History\nThe following questions were asked and answered in previous planning sessions:\n\n" + historicalContext
	}

	// Check for orchestration configuration
	if c.isOrchestrationEnabledForPhase("planning") {
		return c.runOrchestratedPlanning(ctx, taskID, prompt, statusLine)
	}

	// Run agent with streaming, accumulate output for transcript
	c.publishProgress("Agent analyzing task...", 20)
	var transcriptBuilder strings.Builder
	response, err := planningAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		// Always publish to event bus
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
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
			c.logError(fmt.Errorf("save active task after planning error: %w", err))
		}
		_ = c.machine.Dispatch(ctx, workflow.EventError)

		return fmt.Errorf("agent planning: %w", err)
	}

	// Record usage stats
	if response.Usage != nil {
		if err := c.workspace.AddUsage(taskID, "planning",
			response.Usage.InputTokens,
			response.Usage.OutputTokens,
			response.Usage.CachedTokens,
			response.Usage.CostUSD,
		); err != nil {
			c.logError(fmt.Errorf("record planning usage: %w", err))
		}
	}

	// Save full transcript for archive (--full-context recovery)
	if transcript := transcriptBuilder.String(); transcript != "" {
		transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-planning.log"
		if err := c.workspace.SaveTranscript(taskID, transcriptFile, transcript); err != nil {
			c.logError(fmt.Errorf("save planning transcript: %w", err))
		}
	}

	// Record exchanges to session for context recovery
	if c.currentSession != nil {
		now := time.Now()
		// Record the prompt (truncated for summary)
		promptSummary := prompt
		if len(promptSummary) > 500 {
			promptSummary = promptSummary[:500] + "..."
		}
		c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
			Role:      "user",
			Content:   promptSummary,
			Timestamp: now,
		})
		// Record agent response summary
		responseSummary := extractContextSummary(response)
		if responseSummary != "" {
			c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
				Role:      "agent",
				Content:   responseSummary,
				Timestamp: now,
			})
		}
	}

	// If agent asked a question, handle based on mode
	if response.Question != nil {
		if c.opts.SkipAgentQuestions {
			// In auto mode, skip questions and proceed with agent's best guess
			c.publishProgress("Skipping agent question (auto mode)...", 50)
			// Log the skipped question for audit trail
			c.logError(fmt.Errorf("skipped agent question: %s", response.Question.Text))
			// Continue with whatever specs were generated (if any)
		} else {
			// Normal mode: save question and return, waiting for user answer
			pendingQuestion := &storage.PendingQuestion{
				Question: response.Question.Text,
				Phase:    "planning",
				AskedAt:  time.Now(),
				// Save context to avoid re-exploration on next plan call
				ContextSummary: extractContextSummary(response),
				FullContext:    buildFullContext(response),
				ExploredFiles:  extractExploredFiles(response),
			}
			for _, opt := range response.Question.Options {
				pendingQuestion.Options = append(pendingQuestion.Options, storage.QuestionOption{
					Label:       opt.Label,
					Description: opt.Description,
				})
			}
			if err := c.workspace.SavePendingQuestion(taskID, pendingQuestion); err != nil {
				c.logError(fmt.Errorf("save pending question: %w", err))
			}
			// Record question in session exchanges for later recovery
			if c.currentSession != nil {
				c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
					Role:      "agent",
					Content:   "QUESTION: " + response.Question.Text,
					Timestamp: time.Now(),
				})
				// Save session before returning (question state)
				c.saveCurrentSession(taskID)
			}
			// Dispatch EventWait to properly transition FSM to StateWaiting
			_ = c.machine.Dispatch(ctx, workflow.EventWait)
			c.activeTask.State = string(workflow.StateWaiting)
			if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
				c.logError(fmt.Errorf("save active task after pending question: %w", err))
			}

			return ErrPendingQuestion
		}
	}

	c.publishProgress("Creating specifications...", 70)

	// Create specification from response
	nextNum, err := c.workspace.NextSpecificationNumber(taskID)
	if err != nil {
		return fmt.Errorf("get next specification number: %w", err)
	}

	// Format specification content
	specContent := formatSpecificationContent(nextNum, response)

	if err := c.workspace.SaveSpecification(taskID, nextNum, specContent); err != nil {
		return fmt.Errorf("save specification: %w", err)
	}

	// Create checkpoint if git is available
	commitMsg := c.generateCommitMessage(ctx, fmt.Sprintf("planning (spec-%d)", nextNum))
	c.createCheckpointIfNeeded(ctx, taskID, commitMsg)

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	_ = c.machine.Dispatch(ctx, workflow.EventPlanDone)

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Planning complete", 100)

	return nil
}

// RunImplementation executes the implementation phase.
func (c *Conductor) RunImplementation(ctx context.Context) error {
	c.publishProgress("Starting implementation phase...", 0)

	taskID := c.activeTask.ID

	// Create progress tracker for this phase
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		statusLine = progress.NewStatusLine("Implementing")
		defer statusLine.Done()
	}

	// Get agent for implementing step
	implementingAgent, err := c.GetAgentForStep(ctx, workflow.StepImplementing)
	if err != nil {
		return fmt.Errorf("get implementing agent: %w", err)
	}

	// Create session for this implementation run
	session, filename, err := c.workspace.CreateSession(taskID, "implementation", implementingAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	// Get latest specification content only (use the most refined version)
	specContent, specNum, err := c.workspace.GetLatestSpecificationContent(taskID)
	if err != nil {
		return fmt.Errorf("get latest specification: %w", err)
	}
	if specContent == "" {
		return errors.New("no specifications found - run 'task plan' first")
	}

	c.publishProgress(fmt.Sprintf("Using specification-%d for implementation...", specNum), 5)

	// Get source content for context
	sourceContent, err := c.workspace.GetSourceContent(taskID)
	if err != nil {
		return fmt.Errorf("get source content: %w", err)
	}

	// Get notes (missing notes is acceptable, returns empty string)
	notes, _ := c.workspace.ReadNotes(taskID)

	// Build implementation prompt with latest spec and custom instructions
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "implementing")
	specStatusSummary := buildSpecStatusSummary(c.workspace, taskID)
	specTrackingSummary := buildSpecificationTrackingSummary(c.workspace, taskID)
	prompt := buildImplementationPrompt(c.workspace, c.taskWork.Metadata.Title, sourceContent, specContent, notes, customInstructions, specStatusSummary, specTrackingSummary)

	// Run agent with streaming, accumulate output for transcript
	c.publishProgress("Agent implementing...", 20)
	var transcriptBuilder strings.Builder

	// Retry loop for recoverable errors
	var response *agent.Response

	// Store original prompt to avoid accumulation on retries
	originalPrompt := prompt
	var lastErr error

	for attempt := range maxRetries {
		// Check for context cancellation before retry
		if ctx.Err() != nil {
			return fmt.Errorf("implementation cancelled: %w", ctx.Err())
		}

		// Use original prompt + latest error (not accumulated)
		currentPrompt := originalPrompt
		if lastErr != nil {
			currentPrompt = fmt.Sprintf(`%s

## Previous Error
The previous implementation attempt failed with: %v

Please retry the implementation, taking into account this error.
`, originalPrompt, lastErr)
		}

		response, err = implementingAgent.RunWithCallback(ctx, currentPrompt, func(event agent.Event) error {
			// Always publish to event bus
			c.eventBus.PublishRaw(events.Event{
				Type: events.TypeAgentMessage,
				Data: map[string]any{"event": event},
			})
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

		// If successful or not a recoverable error, break the loop
		if err == nil || !isRecoverableError(err) {
			break
		}

		// Store error for next retry
		lastErr = err

		// Recoverable error - retry with exponential backoff
		if attempt < maxRetries-1 {
			// Calculate exponential backoff with jitter
			backoff := time.Duration(float64(initialBackoffDelay) * math.Pow(backoffMultiplier, float64(attempt)))
			if backoff > maxBackoffDelay {
				backoff = maxBackoffDelay
			}

			// Add jitter (±20%)
			// #nosec G404 - math/rand is sufficient for non-critical jitter
			jitter := time.Duration(float64(backoff) * 0.2 * (2.0*mrand.Float64() - 1.0))
			backoff = backoff + jitter

			c.publishProgress(fmt.Sprintf("Recoverable error, retrying in %.1fs (attempt %d/%d)...",
				backoff.Seconds(), attempt+2, maxRetries), 20)

			// Wait before retry (check ctx cancellation)
			select {
			case <-time.After(backoff):
				// Proceed with retry
			case <-ctx.Done():
				return fmt.Errorf("implementation cancelled during backoff: %w", ctx.Err())
			}

			// Clear transcript builder for retry
			transcriptBuilder.Reset()
		}
	}

	if err != nil {
		if statusLine != nil {
			statusLine.Done()
		}
		c.activeTask.State = "idle"
		if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
			c.logError(fmt.Errorf("save active task after implementation error: %w", err))
		}
		_ = c.machine.Dispatch(ctx, workflow.EventError)

		return fmt.Errorf("agent implementation: %w", err)
	}

	// Record usage stats
	if response.Usage != nil {
		if err := c.workspace.AddUsage(taskID, "implementing",
			response.Usage.InputTokens,
			response.Usage.OutputTokens,
			response.Usage.CachedTokens,
			response.Usage.CostUSD,
		); err != nil {
			c.logError(fmt.Errorf("record implementation usage: %w", err))
		}
	}

	// Save full transcript for archive
	if transcript := transcriptBuilder.String(); transcript != "" {
		transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-implementation.log"
		if err := c.workspace.SaveTranscript(taskID, transcriptFile, transcript); err != nil {
			c.logError(fmt.Errorf("save implementation transcript: %w", err))
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
		// Record files changed
		if len(response.Files) > 0 {
			var fileList []string
			for _, f := range response.Files {
				fileList = append(fileList, f.Path)
			}
			c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
				Role:      "agent",
				Content:   fmt.Sprintf("Modified %d files: %s", len(response.Files), strings.Join(fileList, ", ")),
				Timestamp: now,
			})
		}
	}

	c.publishProgress("Applying changes...", 70)

	// Apply file changes
	if !c.opts.DryRun && len(response.Files) > 0 {
		if err := applyFiles(ctx, c, response.Files); err != nil {
			return fmt.Errorf("apply files: %w", err)
		}

		// Track implemented files to the latest specification
		specifications, specErr := c.workspace.ListSpecifications(taskID)
		if specErr != nil {
			// Non-critical: specification tracking is best-effort
			c.logError(fmt.Errorf("failed to list specifications for file tracking: %w", specErr))
		} else if len(specifications) > 0 {
			specificationNum := specifications[len(specifications)-1] // Latest specification
			specification, parseErr := c.workspace.ParseSpecification(taskID, specificationNum)
			if parseErr != nil {
				c.logError(fmt.Errorf("failed to parse specification-%d for file tracking: %w", specificationNum, parseErr))
			} else {
				// Check if previously tracked files still exist (handle reverts/undo)
				var validFiles []string
				for _, filePath := range specification.ImplementedFiles {
					if _, err := os.Stat(filePath); err == nil {
						// File still exists
						validFiles = append(validFiles, filePath)
					}
				}

				// Update specification if files were deleted
				if len(validFiles) != len(specification.ImplementedFiles) {
					originalCount := len(specification.ImplementedFiles)
					specification.ImplementedFiles = validFiles
					c.logVerbosef("Cleared %d deleted files from specification-%d tracking",
						originalCount-len(validFiles), specificationNum)
				}

				// Add new files from this implementation
				var filePaths []string
				for _, fc := range response.Files {
					filePaths = append(filePaths, fc.Path)
				}
				specification.ImplementedFiles = append(specification.ImplementedFiles, filePaths...)
				specification.UpdatedAt = time.Now()
				if saveErr := c.workspace.SaveSpecificationWithMeta(taskID, specification); saveErr != nil {
					c.logError(fmt.Errorf("failed to save file tracking to specification-%d: %w", specificationNum, saveErr))
				} else {
					c.logVerbosef("Tracked %d files to specification-%d", len(filePaths), specificationNum)
				}
			}
		}
	}

	// Create checkpoint if git is available
	commitMsg := c.generateCommitMessage(ctx, "implementation")
	if event := c.createCheckpointIfNeeded(ctx, taskID, commitMsg); event != nil {
		c.eventBus.PublishRaw(*event)
	}

	// Run security scans if configured
	c.publishProgress("Running security scans...", 90)
	if scanErr := c.RunSecurityScan(ctx); scanErr != nil {
		c.logError(fmt.Errorf("security scan: %w", scanErr))
		// Don't fail implementation on scan errors
	}

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	_ = c.machine.Dispatch(ctx, workflow.EventImplementDone)

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Implementation complete", 100)

	return nil
}

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

	// Run agent, accumulate output for transcript
	c.publishProgress("Agent reviewing...", 20)
	var transcriptBuilder strings.Builder
	response, err := reviewAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		// Always publish to event bus
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
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
		_ = c.machine.Dispatch(ctx, workflow.EventError)

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
	_ = c.machine.Dispatch(ctx, workflow.EventReviewDone)

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

// runOrchestratedPlanning executes the planning phase using multi-agent orchestration.
func (c *Conductor) runOrchestratedPlanning(ctx context.Context, taskID string, prompt string, statusLine *progress.StatusLine) error {
	c.publishProgress("Running planning with multi-agent orchestration...", 20)

	// Run orchestration
	result, err := c.runOrchestratedStep(ctx, "planning", c.taskWork)
	if err != nil {
		if statusLine != nil {
			statusLine.Done()
		}
		c.activeTask.State = "idle"
		if saveErr := c.workspace.SaveActiveTask(c.activeTask); saveErr != nil {
			c.logError(fmt.Errorf("save active task after planning error: %w", saveErr))
		}
		_ = c.machine.Dispatch(ctx, workflow.EventError)

		return fmt.Errorf("orchestrated planning: %w", err)
	}

	// Extract final output from orchestration result
	specContent := c.extractFinalOutput(result)

	// Record total usage from all orchestration steps
	var totalTokens int
	var totalCost float64
	for _, step := range result.StepResults {
		totalTokens += step.TokenUsage
		// Note: Cost tracking would need to be added to StepResult if not present
	}
	if totalTokens > 0 {
		if err := c.workspace.AddUsage(taskID, "planning", totalTokens, 0, 0, totalCost); err != nil {
			c.logError(fmt.Errorf("record planning usage: %w", err))
		}
	}

	// Save transcript for orchestration run
	transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-planning-orchestration.log"
	transcript := fmt.Sprintf("# Orchestration Planning Run\n\nDuration: %s\nConsensus: %.0f%%\n\nSteps Executed: %d\n\n",
		result.Duration, result.Consensus*100, len(result.StepResults))
	var stepsBuilder strings.Builder
	for stepName, stepResult := range result.StepResults {
		stepsBuilder.WriteString(fmt.Sprintf("## Step: %s (Agent: %s, Tokens: %d)\n%s\n\n",
			stepName, stepResult.AgentName, stepResult.TokenUsage, stepResult.Output))
	}
	transcript += stepsBuilder.String()
	if err := c.workspace.SaveTranscript(taskID, transcriptFile, transcript); err != nil {
		c.logError(fmt.Errorf("save orchestration transcript: %w", err))
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
		// Record orchestration summary
		orchestrationSummary := fmt.Sprintf("Orchestration completed with %d steps, %.0f%% consensus",
			len(result.StepResults), result.Consensus*100)
		c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
			Role:      "agent",
			Content:   orchestrationSummary,
			Timestamp: now,
		})
	}

	c.publishProgress("Creating specifications from orchestration result...", 70)

	// Create specification from orchestration result
	nextNum, err := c.workspace.NextSpecificationNumber(taskID)
	if err != nil {
		return fmt.Errorf("get next specification number: %w", err)
	}

	// Format specification content
	// For orchestration, we create a simpler spec format
	specContentFormatted := fmt.Sprintf("# Specification %d\n\n%s\n\n---\n\n*Generated by multi-agent orchestration (%.0f%% consensus)*\n",
		nextNum, specContent, result.Consensus*100)

	if err := c.workspace.SaveSpecification(taskID, nextNum, specContentFormatted); err != nil {
		return fmt.Errorf("save specification: %w", err)
	}

	c.publishProgress(fmt.Sprintf("Specification %d created", nextNum), 90)

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	_ = c.machine.Dispatch(ctx, workflow.EventPlanDone)

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Planning complete", 100)

	return nil
}

// simplifyInput simplifies the task input (source content).
func (c *Conductor) simplifyInput(ctx context.Context, taskID string) error {
	c.publishProgress("Simplifying task input...", 10)

	simplifyingAgent, err := c.GetAgentForStep(ctx, workflow.StepSimplifying)
	if err != nil {
		return fmt.Errorf("get simplification agent: %w", err)
	}

	sourceContent, err := c.workspace.GetSourceContent(taskID)
	if err != nil {
		return errors.New("no source content found")
	}

	c.publishProgress("Reading task input...", 20)

	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := ""
	if workspaceCfg.Workflow.Simplify.Instructions != "" {
		customInstructions = workspaceCfg.Workflow.Simplify.Instructions
	}

	title := c.taskWork.Metadata.Title
	prompt := buildSimplifyInputPrompt(title, sourceContent, customInstructions)

	c.publishProgress("Agent simplifying input...", 40)
	var transcriptBuilder strings.Builder
	response, err := simplifyingAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("agent simplification: %w", err)
	}

	// Get simplified content from response - use Summary first, then Messages
	simplifiedContent := response.Summary
	if simplifiedContent == "" && len(response.Messages) > 0 {
		simplifiedContent = response.Messages[0]
	}

	// Write simplified content to notes with a "simplified" tag
	if err := c.workspace.AppendNote(taskID, simplifiedContent, "simplified"); err != nil {
		return fmt.Errorf("save simplified content: %w", err)
	}

	c.publishProgress("Task input simplified", 100)

	return nil
}

// simplifyPlanning simplifies specification files.
func (c *Conductor) simplifyPlanning(ctx context.Context, taskID string) error {
	c.publishProgress("Simplifying planning output...", 10)

	simplifyingAgent, err := c.GetAgentForStep(ctx, workflow.StepSimplifying)
	if err != nil {
		return fmt.Errorf("get simplification agent: %w", err)
	}

	session, filename, err := c.workspace.CreateSession(taskID, "simplification-planning", simplifyingAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	specs, err := c.workspace.ListSpecifications(taskID)
	if err != nil || len(specs) == 0 {
		return errors.New("no specifications found to simplify")
	}

	sourceContent, _ := c.workspace.GetSourceContent(taskID)
	notes, _ := c.workspace.ReadNotes(taskID)
	specContent, _ := c.workspace.GatherSpecificationsContent(taskID)

	c.publishProgress(fmt.Sprintf("Found %d specification(s) to simplify", len(specs)), 20)

	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := ""
	if workspaceCfg.Workflow.Simplify.Instructions != "" {
		customInstructions = workspaceCfg.Workflow.Simplify.Instructions
	}

	prompt := buildSimplifyPlanningPrompt(c.taskWork.Metadata.Title,
		sourceContent, notes, specContent, customInstructions)

	c.publishProgress("Agent simplifying specifications...", 40)
	var transcriptBuilder strings.Builder
	response, err := simplifyingAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("agent simplification: %w", err)
	}

	// Get simplified content from response
	simplifiedContent := response.Summary
	if simplifiedContent == "" && len(response.Messages) > 0 {
		simplifiedContent = response.Messages[0]
	}

	simplifiedSpecs := parseSimplifiedSpecifications(simplifiedContent)
	if len(simplifiedSpecs) == 0 {
		return errors.New("no simplified specifications found")
	}

	for _, spec := range simplifiedSpecs {
		if err := c.workspace.SaveSpecification(taskID, spec.Number, spec.Content); err != nil {
			return fmt.Errorf("save specification %d: %w", spec.Number, err)
		}
	}

	if session != nil {
		session.Metadata.EndedAt = time.Now()
		if response.Usage != nil {
			session.Usage = &storage.UsageInfo{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
				CachedTokens: response.Usage.CachedTokens,
				CostUSD:      response.Usage.CostUSD,
			}
		}
		session.Exchanges = append(session.Exchanges, storage.Exchange{
			Role:      "agent",
			Timestamp: time.Now(),
			Content:   simplifiedContent,
		})
		_ = c.workspace.SaveSession(taskID, filename, session)
	}

	if response.Usage != nil {
		_ = c.workspace.AddUsage(taskID, "simplifying-planning",
			response.Usage.InputTokens, response.Usage.OutputTokens,
			response.Usage.CachedTokens, response.Usage.CostUSD)
	}

	c.publishProgress(fmt.Sprintf("Simplified %d specification(s)", len(simplifiedSpecs)), 100)

	return nil
}

// simplifyImplementing simplifies code files from the last implementation run.
func (c *Conductor) simplifyImplementing(ctx context.Context, taskID string) error {
	c.publishProgress("Simplifying implementation output...", 10)

	simplifyingAgent, err := c.GetAgentForStep(ctx, workflow.StepSimplifying)
	if err != nil {
		return fmt.Errorf("get simplification agent: %w", err)
	}

	session, filename, err := c.workspace.CreateSession(taskID, "simplification-implementing", simplifyingAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	specs, err := c.workspace.ListSpecifications(taskID)
	if err != nil || len(specs) == 0 {
		return errors.New("no specifications found - cannot identify implemented files")
	}

	implementedFiles := make(map[string]string)
	for _, specNum := range specs {
		spec, err := c.workspace.ParseSpecification(taskID, specNum)
		if err != nil {
			continue
		}
		for _, filePath := range spec.ImplementedFiles {
			fullPath := filepath.Join(c.workspace.Root(), filePath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				c.logError(fmt.Errorf("read file %s: %w", filePath, err))

				continue
			}
			implementedFiles[filePath] = string(content)
		}
	}

	if len(implementedFiles) == 0 {
		return errors.New("no implemented files found - run implement first")
	}

	c.publishProgress(fmt.Sprintf("Found %d file(s) to simplify", len(implementedFiles)), 20)

	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := ""
	if workspaceCfg.Workflow.Simplify.Instructions != "" {
		customInstructions = workspaceCfg.Workflow.Simplify.Instructions
	}
	sourceContent, _ := c.workspace.GetSourceContent(taskID)

	prompt := buildSimplifyImplementingPrompt(c.taskWork.Metadata.Title,
		sourceContent, implementedFiles, customInstructions)

	c.publishProgress("Agent simplifying code...", 40)
	var transcriptBuilder strings.Builder
	response, err := simplifyingAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeAgentMessage,
			Data: map[string]any{"event": event},
		})
		if event.Text != "" {
			transcriptBuilder.WriteString(event.Text)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("agent simplification: %w", err)
	}

	// Get simplified content from response
	simplifiedContent := response.Summary
	if simplifiedContent == "" && len(response.Messages) > 0 {
		simplifiedContent = response.Messages[0]
	}

	simplifiedFiles, err := parseSimplifiedCode(simplifiedContent)
	if err != nil {
		return fmt.Errorf("parse simplified code: %w", err)
	}

	for filePath, content := range simplifiedFiles {
		fullPath := filepath.Join(c.workspace.Root(), filePath)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", filePath, err)
		}
		c.publishProgress("Simplified "+filePath, 80)
	}

	if session != nil {
		session.Metadata.EndedAt = time.Now()
		if response.Usage != nil {
			session.Usage = &storage.UsageInfo{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
				CachedTokens: response.Usage.CachedTokens,
				CostUSD:      response.Usage.CostUSD,
			}
		}
		session.Exchanges = append(session.Exchanges, storage.Exchange{
			Role:      "agent",
			Timestamp: time.Now(),
			Content:   simplifiedContent,
		})
		_ = c.workspace.SaveSession(taskID, filename, session)
	}

	if response.Usage != nil {
		_ = c.workspace.AddUsage(taskID, "simplifying-implementing",
			response.Usage.InputTokens, response.Usage.OutputTokens,
			response.Usage.CachedTokens, response.Usage.CostUSD)
	}

	c.publishProgress(fmt.Sprintf("Simplified %d file(s)", len(simplifiedFiles)), 100)

	return nil
}
