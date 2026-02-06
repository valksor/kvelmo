package conductor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/progress"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

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

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

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

	// Determine the working directory for path context in prompts
	workingDir := c.CodeDir()

	// Build implementation prompt with latest spec and custom instructions
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "implementing")
	specStatusSummary := buildSpecStatusSummary(c.workspace, taskID)
	specTrackingSummary := buildSpecificationTrackingSummary(c.workspace, taskID)

	// Fetch hierarchical context (parent and sibling tasks) if configured
	// Check CLI options first, then workspace config
	var hierarchy *HierarchicalContext
	shouldIncludeParent := false
	if c.opts.WithParent != nil {
		shouldIncludeParent = *c.opts.WithParent
	} else if workspaceCfg.Context != nil {
		shouldIncludeParent = workspaceCfg.Context.IncludeParent
	}

	if shouldIncludeParent {
		// Resolve provider and fetch hierarchical context
		resolveOpts := provider.ResolveOptions{
			DefaultProvider: c.opts.DefaultProvider,
		}
		ref := c.activeTask.Ref
		providerCfg := buildProviderConfig(workspaceCfg, parseScheme(ref))
		if p, id, err := c.providers.Resolve(ctx, ref, providerCfg, resolveOpts); err == nil {
			// Get the current work unit for hierarchy detection
			if reader, ok := p.(provider.Reader); ok {
				if workUnit, err := reader.Fetch(ctx, id); err == nil {
					hierarchy, _ = c.FetchHierarchicalContextFromConfig(ctx, p, workUnit)
				}
			}
		}
	}

	prompt := buildImplementationPrompt(c.workspace, workingDir, c.taskWork.Metadata.Title, sourceContent, specContent, notes, customInstructions, specStatusSummary, specTrackingSummary, hierarchy)

	// Optimize prompt if enabled (CLI flag or workspace config)
	shouldOptimize := c.opts.OptimizePrompts || shouldOptimizePrompt(workspaceCfg, "implementing")
	if shouldOptimize {
		prompt = c.optimizePrompt(ctx, "implementing", prompt)
	}

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
			jitter := time.Duration(float64(backoff) * 0.2 * (2.0*rand.Float64() - 1.0))
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
		if dispatchErr := c.dispatchWithRetry(ctx, workflow.EventError); dispatchErr != nil {
			c.logError(dispatchErr)
		}

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
		if err := c.checkBudgets(ctx, "implementing"); err != nil {
			return err
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

	// Sync specification statuses - mark specs with implemented files as done
	// This provides a fallback in case the agent forgets to update the status
	c.publishProgress("Syncing specification statuses...", 65)
	specifications, specErr := c.workspace.ListSpecificationsWithStatus(taskID)
	if specErr != nil {
		c.logError(fmt.Errorf("failed to list specifications for status sync: %w", specErr))
	} else {
		for _, spec := range specifications {
			if spec.Status == storage.SpecificationStatusDraft && len(spec.ImplementedFiles) > 0 {
				if updateErr := c.workspace.UpdateSpecificationStatus(taskID, spec.Number, storage.SpecificationStatusDone); updateErr != nil {
					c.logError(fmt.Errorf("failed to update specification-%d status to done: %w", spec.Number, updateErr))
				} else {
					c.logVerbosef("Auto-marked specification-%d as done (has %d implemented files)", spec.Number, len(spec.ImplementedFiles))
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
	if err := c.dispatchWithRetry(ctx, workflow.EventImplementDone); err != nil {
		return err
	}

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Implementation complete", 100)

	return nil
}

// RunReviewImplementation executes implementation of fixes from a specific review.
// Unlike RunImplementation which uses specifications, this uses review feedback as the guide.
func (c *Conductor) RunReviewImplementation(ctx context.Context, reviewNumber int) error {
	c.publishProgress(fmt.Sprintf("Starting review fix implementation (review %d)...", reviewNumber), 0)

	taskID := c.activeTask.ID

	// Create progress tracker for this phase
	var statusLine *progress.StatusLine
	if !c.opts.DryRun {
		statusLine = progress.NewStatusLine("Implementing review fixes")
		defer statusLine.Done()
	}

	// Get agent for review implementing step (falls back to implementing if not configured)
	implementingAgent, err := c.GetAgentForStep(ctx, workflow.StepReviewImplementing)
	if err != nil {
		return fmt.Errorf("get implementing agent: %w", err)
	}

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

	// Create session for this review implementation run
	session, filename, err := c.workspace.CreateSession(taskID, "review-implementation", implementingAgent.Name(), c.activeTask.State)
	if err != nil {
		c.logError(fmt.Errorf("create session: %w", err))
	} else {
		c.currentSession = session
		c.currentSessionFile = filename
	}

	// Load the review content
	reviewContent, err := c.workspace.LoadReview(taskID, reviewNumber)
	if err != nil {
		return fmt.Errorf("load review %d: %w", reviewNumber, err)
	}
	if strings.TrimSpace(reviewContent) == "" {
		return fmt.Errorf("review %d is empty, nothing to implement", reviewNumber)
	}

	c.publishProgress(fmt.Sprintf("Loaded review %d content...", reviewNumber), 5)

	// Get source content for context
	sourceContent, err := c.workspace.GetSourceContent(taskID)
	if err != nil {
		return fmt.Errorf("get source content: %w", err)
	}

	// Get notes (missing notes is acceptable, returns empty string)
	notes, _ := c.workspace.ReadNotes(taskID)

	// Determine the working directory for path context in prompts
	workingDir := c.CodeDir()

	// Build review fix prompt
	workspaceCfg, _ := c.workspace.LoadConfig()
	customInstructions := buildCombinedInstructions(workspaceCfg, "implementing")

	prompt := buildReviewFixPrompt(c.workspace, workingDir, c.taskWork.Metadata.Title, sourceContent, reviewContent, notes, customInstructions)

	// Optimize prompt if enabled
	shouldOptimize := c.opts.OptimizePrompts || shouldOptimizePrompt(workspaceCfg, "implementing")
	if shouldOptimize {
		prompt = c.optimizePrompt(ctx, "implementing", prompt)
	}

	// Run agent with streaming, accumulate output for transcript
	c.publishProgress("Agent implementing review fixes...", 20)
	var transcriptBuilder strings.Builder

	// Retry loop for recoverable errors (same pattern as RunImplementation)
	var response *agent.Response

	// Store original prompt to avoid accumulation on retries
	originalPrompt := prompt
	var lastErr error

	for attempt := range maxRetries {
		// Check for context cancellation before retry
		if ctx.Err() != nil {
			return fmt.Errorf("review implementation cancelled: %w", ctx.Err())
		}

		// Use original prompt + latest error (not accumulated)
		currentPrompt := originalPrompt
		if lastErr != nil {
			currentPrompt = fmt.Sprintf(`%s

## Previous Error
The previous review implementation attempt failed with: %v

Please retry the implementation, taking into account this error.
`, originalPrompt, lastErr)
		}

		response, err = implementingAgent.RunWithCallback(ctx, currentPrompt, func(event agent.Event) error {
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
			jitter := time.Duration(float64(backoff) * 0.2 * (2.0*rand.Float64() - 1.0))
			backoff = backoff + jitter

			c.publishProgress(fmt.Sprintf("Recoverable error, retrying in %.1fs (attempt %d/%d)...",
				backoff.Seconds(), attempt+2, maxRetries), 20)

			// Wait before retry (check ctx cancellation)
			select {
			case <-time.After(backoff):
				// Proceed with retry
			case <-ctx.Done():
				return fmt.Errorf("review implementation cancelled during backoff: %w", ctx.Err())
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
		if saveErr := c.workspace.SaveActiveTask(c.activeTask); saveErr != nil {
			c.logError(fmt.Errorf("save active task after implementation error: %w", saveErr))
		}
		if dispatchErr := c.dispatchWithRetry(ctx, workflow.EventError); dispatchErr != nil {
			c.logError(dispatchErr)
		}

		return fmt.Errorf("agent review implementation: %w", err)
	}

	// Record usage stats
	if response.Usage != nil {
		if err := c.workspace.AddUsage(taskID, "review-implementing",
			response.Usage.InputTokens,
			response.Usage.OutputTokens,
			response.Usage.CachedTokens,
			response.Usage.CostUSD,
		); err != nil {
			c.logError(fmt.Errorf("record review implementation usage: %w", err))
		}
		if err := c.checkBudgets(ctx, "implementing"); err != nil {
			return err
		}
	}

	// Save full transcript for archive
	if transcript := transcriptBuilder.String(); transcript != "" {
		transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-review-implementation.log"
		if err := c.workspace.SaveTranscript(taskID, transcriptFile, transcript); err != nil {
			c.logError(fmt.Errorf("save review implementation transcript: %w", err))
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
		if len(response.Files) > 0 {
			var fileList []string
			for _, f := range response.Files {
				fileList = append(fileList, f.Path)
			}
			c.currentSession.Exchanges = append(c.currentSession.Exchanges, storage.Exchange{
				Role:      "agent",
				Content:   fmt.Sprintf("Modified %d files to fix review issues: %s", len(response.Files), strings.Join(fileList, ", ")),
				Timestamp: now,
			})
		}
	}

	c.publishProgress("Applying review fixes...", 70)

	// Apply file changes
	if !c.opts.DryRun && len(response.Files) > 0 {
		if err := applyFiles(ctx, c, response.Files); err != nil {
			return fmt.Errorf("apply files: %w", err)
		}
	}

	// Create checkpoint if git is available
	commitMsg := c.generateCommitMessage(ctx, fmt.Sprintf("review-%d-fixes", reviewNumber))
	if event := c.createCheckpointIfNeeded(ctx, taskID, commitMsg); event != nil {
		c.eventBus.PublishRaw(*event)
	}

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	if err := c.dispatchWithRetry(ctx, workflow.EventImplementDone); err != nil {
		return err
	}

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress(fmt.Sprintf("Review %d fixes complete", reviewNumber), 100)

	return nil
}
