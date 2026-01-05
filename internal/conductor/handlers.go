package conductor

import (
	"context"
	"errors"
	"fmt"
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
	prompt := buildPlanningPrompt(c.taskWork.Metadata.Title, sourceContent, notes, existingSpecifications, customInstructions)
	if pendingContext != "" {
		prompt += "\n\n## Previous Analysis (before question)\nThe following is context from your previous planning session. Use this to avoid re-exploring:\n\n" + pendingContext
	}
	if historicalContext != "" {
		prompt += "\n\n## Previous Q&A History\nThe following questions were asked and answered in previous planning sessions:\n\n" + historicalContext
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
	prompt := buildImplementationPrompt(c.taskWork.Metadata.Title, sourceContent, specContent, notes, customInstructions)

	// Run agent with streaming, accumulate output for transcript
	c.publishProgress("Agent implementing...", 20)
	var transcriptBuilder strings.Builder
	response, err := implementingAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
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
	}

	// Create checkpoint if git is available
	commitMsg := c.generateCommitMessage(ctx, "implementation")
	if event := c.createCheckpointIfNeeded(ctx, taskID, commitMsg); event != nil {
		c.eventBus.PublishRaw(*event)
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

	// Build review prompt with lint results and custom instructions
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "reviewing")
	prompt := buildReviewPromptWithLint(c.taskWork.Metadata.Title, sourceContent, specContent, lintResults, customInstructions)

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
