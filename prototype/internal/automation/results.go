package automation

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// ResultCommenter is a subset of provider.Commenter interface.
type ResultCommenter interface {
	AddComment(ctx context.Context, workUnitID string, body string) (any, error)
}

// ResultLabelManager is a subset of provider.LabelManager interface.
type ResultLabelManager interface {
	AddLabels(ctx context.Context, workUnitID string, labels []string) error
	RemoveLabels(ctx context.Context, workUnitID string, labels []string) error
}

// ResultHandler posts results back to providers.
type ResultHandler struct {
	providerGetter ProviderGetter
	config         *storage.AutomationLabelConfig
}

// NewResultHandler creates a new result handler.
func NewResultHandler(providerGetter ProviderGetter, cfg *storage.AutomationLabelConfig) *ResultHandler {
	if cfg == nil {
		cfg = &storage.AutomationLabelConfig{
			MehrhofGenerated: "mehrhof-generated",
			InProgress:       "mehrhof-processing",
			Failed:           "mehrhof-failed",
		}
	}

	return &ResultHandler{
		providerGetter: providerGetter,
		config:         cfg,
	}
}

// HandleJobStart is called when a job starts processing.
func (h *ResultHandler) HandleJobStart(ctx context.Context, job *WebhookJob) error {
	if h.config.InProgress == "" {
		return nil
	}

	// Add in-progress label.
	return h.addLabel(ctx, job, h.config.InProgress)
}

// HandleJobSuccess is called when a job completes successfully.
func (h *ResultHandler) HandleJobSuccess(ctx context.Context, job *WebhookJob) error {
	// Remove in-progress label.
	if h.config.InProgress != "" {
		if err := h.removeLabel(ctx, job, h.config.InProgress); err != nil {
			slog.Warn("failed to remove in-progress label", "error", err)
		}
	}

	// Post success comment.
	comment := h.buildSuccessComment(job)
	if comment != "" {
		if err := h.postComment(ctx, job, comment); err != nil {
			slog.Warn("failed to post success comment", "error", err)
		}
	}

	return nil
}

// HandleJobFailure is called when a job fails.
func (h *ResultHandler) HandleJobFailure(ctx context.Context, job *WebhookJob, err error) error {
	// Remove in-progress label.
	if h.config.InProgress != "" {
		if err := h.removeLabel(ctx, job, h.config.InProgress); err != nil {
			slog.Warn("failed to remove in-progress label", "error", err)
		}
	}

	// Add failed label.
	if h.config.Failed != "" {
		if err := h.addLabel(ctx, job, h.config.Failed); err != nil {
			slog.Warn("failed to add failed label", "error", err)
		}
	}

	// Post failure comment.
	comment := h.buildFailureComment(job, err)
	if err := h.postComment(ctx, job, comment); err != nil {
		slog.Warn("failed to post failure comment", "error", err)
	}

	return nil
}

// AddMehrhofLabel adds the mehrhof-generated label to a PR/issue.
func (h *ResultHandler) AddMehrhofLabel(ctx context.Context, job *WebhookJob) error {
	if h.config.MehrhofGenerated == "" {
		return nil
	}

	return h.addLabel(ctx, job, h.config.MehrhofGenerated)
}

// buildSuccessComment creates a success comment for a job.
func (h *ResultHandler) buildSuccessComment(job *WebhookJob) string {
	var sb strings.Builder

	sb.WriteString("### Mehr Automation: Success\n\n")

	switch job.WorkflowType {
	case WorkflowTypeIssueFix:
		sb.WriteString("I've automatically fixed this issue and created a pull request.\n\n")
		if job.Result != nil && job.Result.PRURL != "" {
			sb.WriteString(fmt.Sprintf("**Pull Request:** %s\n\n", job.Result.PRURL))
		}
		sb.WriteString("Please review the changes and merge if everything looks good.")

	case WorkflowTypePRReview:
		sb.WriteString("I've completed the review of this pull request.\n\n")
		if job.Result != nil && job.Result.CommentsPosted > 0 {
			sb.WriteString(fmt.Sprintf("Posted %d review comments.\n", job.Result.CommentsPosted))
		} else {
			sb.WriteString("No issues found in the review.")
		}

	case WorkflowTypeCommand:
		sb.WriteString("Command executed successfully.")
	}

	return sb.String()
}

// buildFailureComment creates a failure comment for a job.
func (h *ResultHandler) buildFailureComment(job *WebhookJob, err error) string {
	var sb strings.Builder

	sb.WriteString("### Mehr Automation: Failed\n\n")

	switch job.WorkflowType {
	case WorkflowTypeIssueFix:
		sb.WriteString("I was unable to automatically fix this issue.\n\n")
	case WorkflowTypePRReview:
		sb.WriteString("I encountered an error while reviewing this pull request.\n\n")
	case WorkflowTypeCommand:
		sb.WriteString("The command failed to execute.\n\n")
	}

	if err != nil {
		sb.WriteString(fmt.Sprintf("**Error:** %s\n\n", err.Error()))
	}

	sb.WriteString("Please check the issue/PR details or try again later.\n\n")
	sb.WriteString("<details>\n<summary>Job Details</summary>\n\n")
	sb.WriteString(fmt.Sprintf("- Job ID: `%s`\n", job.ID))
	sb.WriteString(fmt.Sprintf("- Workflow: `%s`\n", job.WorkflowType))
	sb.WriteString(fmt.Sprintf("- Attempts: %d/%d\n", job.Attempts, job.MaxAttempts))
	sb.WriteString("</details>")

	return sb.String()
}

// postComment posts a comment to the issue or PR.
func (h *ResultHandler) postComment(ctx context.Context, job *WebhookJob, body string) error {
	if h.providerGetter == nil {
		return nil
	}

	p, err := h.providerGetter(job.Event.Provider)
	if err != nil {
		return err
	}

	commenter, ok := p.(ResultCommenter)
	if !ok {
		return nil // Provider doesn't support commenting.
	}

	// Determine issue/PR number.
	var number int
	if job.Event.Issue != nil {
		number = job.Event.Issue.Number
	} else if job.Event.PullRequest != nil {
		number = job.Event.PullRequest.Number
	} else {
		return nil
	}

	_, err = commenter.AddComment(ctx, strconv.Itoa(number), body)

	return err
}

// addLabel adds a label to the issue or PR.
func (h *ResultHandler) addLabel(ctx context.Context, job *WebhookJob, label string) error {
	if h.providerGetter == nil {
		return nil
	}

	p, err := h.providerGetter(job.Event.Provider)
	if err != nil {
		return err
	}

	labeler, ok := p.(ResultLabelManager)
	if !ok {
		return nil // Provider doesn't support labels.
	}

	// Determine issue/PR number.
	var number int
	if job.Event.Issue != nil {
		number = job.Event.Issue.Number
	} else if job.Event.PullRequest != nil {
		number = job.Event.PullRequest.Number
	} else {
		return nil
	}

	return labeler.AddLabels(ctx, strconv.Itoa(number), []string{label})
}

// removeLabel removes a label from the issue or PR.
func (h *ResultHandler) removeLabel(ctx context.Context, job *WebhookJob, label string) error {
	if h.providerGetter == nil {
		return nil
	}

	p, err := h.providerGetter(job.Event.Provider)
	if err != nil {
		return err
	}

	labeler, ok := p.(ResultLabelManager)
	if !ok {
		return nil // Provider doesn't support labels.
	}

	// Determine issue/PR number.
	var number int
	if job.Event.Issue != nil {
		number = job.Event.Issue.Number
	} else if job.Event.PullRequest != nil {
		number = job.Event.PullRequest.Number
	} else {
		return nil
	}

	return labeler.RemoveLabels(ctx, strconv.Itoa(number), []string{label})
}
