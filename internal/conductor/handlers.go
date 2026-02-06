package conductor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/progress"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
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

// publishAgentEvent publishes an agent streaming event to the event bus
// with a flat data structure matching the React frontend's expectations.
func (c *Conductor) publishAgentEvent(event agent.Event) {
	// Determine content to send
	content := event.Text
	if content == "" && event.ToolCall != nil {
		// For tool_use events, send the tool description (matches StatusLine.OnEvent behavior)
		content = event.ToolCall.Description
		if content == "" {
			content = event.ToolCall.Name
		}
	}

	// Skip empty events to reduce noise
	if content == "" {
		return
	}

	// Include task ID for clients that scope SSE messages by task.
	taskID := ""
	if c.activeTask != nil {
		taskID = c.activeTask.ID
	}

	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeAgentMessage,
		Data: map[string]any{
			"task_id": taskID,
			"content": content,
			// Compatibility aliases for older/web clients.
			"message": content,
			"text":    content,
			"type":    string(event.Type),
		},
	})
}

// RunPlanning executes the planning phase (creates SPEC files).
func (c *Conductor) RunPlanning(ctx context.Context) error {
	slog.Debug("RunPlanning called",
		"task_id", c.activeTask.ID,
		"state", c.activeTask.State)

	// GUARD: Prevent concurrent or sequential planning calls.
	// This happens when Claude Code's plan mode behavior triggers multiple planning cycles.
	c.mu.Lock()
	if c.planningInProgress {
		c.mu.Unlock()
		slog.Warn("RunPlanning called while planning already in progress, skipping",
			"task_id", c.activeTask.ID)

		return nil // Return success - the original planning call will handle the work
	}
	c.planningInProgress = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.planningInProgress = false
		c.mu.Unlock()
	}()

	c.publishProgress("Starting planning phase...", 0)

	taskID := c.activeTask.ID

	// IDEMPOTENCY GUARD: Record starting spec count to detect duplicate spec creation.
	// If specs are created by another process/goroutine during this planning phase,
	// we skip our spec creation to prevent duplicates.
	startingSpecNum, _ := c.workspace.NextSpecificationNumber(taskID)
	startingSpecCount := startingSpecNum - 1 // NextSpecificationNumber returns next num

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
	// Disable retries for planning - spec creation is the success signal.
	// Claude Code in plan mode doesn't send completion events, so non-zero exit
	// would trigger retries and create duplicate specs.
	planningAgent = planningAgent.WithRetries(0)

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

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

	// Determine the working directory for path context in prompts
	workingDir := c.CodeDir()

	// Detect task complexity for prompt routing
	// Skip complexity detection if force flags are set
	workspaceCfg, _ := c.workspace.LoadConfig()
	hasParent := c.taskWork.Hierarchy != nil && c.taskWork.Hierarchy.ParentID != ""
	complexity := DetectTaskComplexity(
		c.taskWork.Metadata.Title,
		sourceContent,
		len(c.taskWork.Source.Files),
		c.taskWork.Metadata.TaskType,
		c.taskWork.Metadata.Labels,
		hasParent,
	)

	// Check for force flags (CLI overrides)
	if c.opts.ForceQuickPlanning {
		complexity = ComplexitySimple
		slog.Debug("forcing simple planning via --quick flag")
	} else if c.opts.ForceFullPlanning {
		complexity = ComplexityComplex
		slog.Debug("forcing full planning via --full flag")
	}

	slog.Debug("detected task complexity", "complexity", complexity, "title_len", len(c.taskWork.Metadata.Title), "files", len(c.taskWork.Source.Files))

	var prompt string

	if complexity == ComplexitySimple {
		// Simple tasks use minimal prompt - skip hierarchical context and verbose guidance
		prompt = buildSimplePlanningPrompt(workingDir, c.taskWork.Metadata.Title, sourceContent, notes)
	} else {
		// Full planning path for medium/complex tasks
		customInstructions := buildCombinedInstructions(workspaceCfg, "planning")

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

		prompt = buildPlanningPrompt(c.workspace, workingDir, c.taskWork.Metadata.Title, sourceContent, notes, existingSpecifications, customInstructions, c.opts.UseDefaults, hierarchy)
	}

	// Inject library context if auto-include is enabled
	if c.opts.LibraryAutoInclude {
		libContext, libErr := c.getLibraryContextForWorkingDir(ctx, workingDir)
		if libErr != nil {
			// Log warning for actual errors (not "no docs found")
			slog.Warn("library context injection failed", "error", libErr)
		} else if libContext != "" {
			prompt += "\n\n" + libContext
		}
	}

	if pendingContext != "" {
		prompt += "\n\n## Previous Analysis (before question)\nThe following is context from your previous planning session. Use this to avoid re-exploring:\n\n" + pendingContext
	}
	if historicalContext != "" {
		prompt += "\n\n## Previous Q&A History\nThe following questions were asked and answered in previous planning sessions:\n\n" + historicalContext
	}

	// Optimize prompt if enabled (CLI flag or workspace config)
	shouldOptimize := c.opts.OptimizePrompts || shouldOptimizePrompt(workspaceCfg, "planning")
	if shouldOptimize {
		prompt = c.optimizePrompt(ctx, "planning", prompt)
	}

	// Check for orchestration configuration
	if c.isOrchestrationEnabledForPhase("planning") {
		return c.runOrchestratedPlanning(ctx, taskID, prompt, statusLine)
	}

	// Run agent with streaming, accumulate output for transcript
	c.publishProgress("Agent analyzing task...", 20)
	var transcriptBuilder strings.Builder
	response, agentErr := planningAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
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

	// Check if spec was created - this is the real success signal.
	// Claude Code in plan mode may exit with non-zero code even when successful,
	// so we use spec creation as the source of truth.
	currentSpecNum, _ := c.workspace.NextSpecificationNumber(taskID)
	specCreated := currentSpecNum > startingSpecNum

	if agentErr != nil {
		if specCreated {
			// Spec was created despite agent error - treat as success
			slog.Info("planning succeeded - spec created despite agent error",
				"task_id", taskID,
				"spec_num", currentSpecNum-1,
				"agent_error", agentErr)
			// Continue with normal completion flow
		} else {
			// Real failure - no spec was created
			if statusLine != nil {
				statusLine.Done()
			}
			c.activeTask.State = "idle"
			if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
				c.logError(fmt.Errorf("save active task after planning error: %w", err))
			}
			if dispatchErr := c.dispatchWithRetry(ctx, workflow.EventError); dispatchErr != nil {
				c.logError(dispatchErr)
			}

			return fmt.Errorf("agent planning (no spec created): %w", agentErr)
		}
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
		if err := c.checkBudgets(ctx, "planning"); err != nil {
			return err
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
	// First check structured question, then fallback to plain-text detection
	if response.Question == nil {
		// Fallback: detect plain-text questions (when agent doesn't use AskUserQuestion tool)
		// Use the full context (summary + messages) for detection
		fullContext := buildFullContext(response)
		response.Question = agent.DetectPlainTextQuestion(fullContext)
	}
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
					Value:       opt.Value,
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
			if err := c.dispatchWithRetry(ctx, workflow.EventWait); err != nil {
				return err
			}
			c.activeTask.State = string(workflow.StateWaiting)
			if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
				c.logError(fmt.Errorf("save active task after pending question: %w", err))
			}

			return ErrPendingQuestion
		}
	}

	// IDEMPOTENCY GUARD: Check if specs were already created during this planning phase.
	// This prevents duplicate specs when RunPlanning is called multiple times
	// (e.g., due to Claude Code's plan mode behavior with ExitPlanMode).
	currentSpecNum, _ = c.workspace.NextSpecificationNumber(taskID)
	currentSpecCount := currentSpecNum - 1
	if currentSpecCount > startingSpecCount {
		slog.Warn("specs already created during planning, skipping duplicate spec creation",
			"task_id", taskID,
			"starting_count", startingSpecCount,
			"current_count", currentSpecCount)
		// Skip spec creation but continue to completion
	} else {
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

		c.eventBus.PublishRaw(eventbus.Event{
			Type: events.TypeSpecUpdated,
			Data: map[string]any{"task_id": taskID, "spec_number": nextNum},
		})

		// Create checkpoint if git is available
		commitMsg := c.generateCommitMessage(ctx, fmt.Sprintf("planning (spec-%d)", nextNum))
		c.createCheckpointIfNeeded(ctx, taskID, commitMsg)
	}

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	if err := c.dispatchWithRetry(ctx, workflow.EventPlanDone); err != nil {
		return err
	}

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Planning complete", 100)

	return nil
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
		if dispatchErr := c.dispatchWithRetry(ctx, workflow.EventError); dispatchErr != nil {
			c.logError(dispatchErr)
		}

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
		if err := c.checkBudgets(ctx, "planning"); err != nil {
			return err
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

	c.eventBus.PublishRaw(eventbus.Event{
		Type: events.TypeSpecUpdated,
		Data: map[string]any{"task_id": taskID, "spec_number": nextNum},
	})

	c.publishProgress(fmt.Sprintf("Specification %d created", nextNum), 90)

	// Update state back to idle
	c.activeTask.State = "idle"
	if err := c.workspace.SaveActiveTask(c.activeTask); err != nil {
		c.logError(fmt.Errorf("save active task: %w", err))
	}

	// Dispatch completion
	if err := c.dispatchWithRetry(ctx, workflow.EventPlanDone); err != nil {
		return err
	}

	// Save session with completion time
	c.saveCurrentSession(taskID)

	c.publishProgress("Planning complete", 100)

	return nil
}
