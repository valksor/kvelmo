package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

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
		c.publishAgentEvent(event)
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

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

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
		c.publishAgentEvent(event)
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
		c.eventBus.PublishRaw(eventbus.Event{
			Type: events.TypeSpecUpdated,
			Data: map[string]any{"task_id": taskID, "spec_number": spec.Number},
		})
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

	// Ensure any existing session is saved before creating a new one
	c.ensureSessionSaved(taskID)

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
		c.publishAgentEvent(event)
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
