package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

var (
	// ErrNoProviderForEvent is returned when no provider can be resolved for an event.
	ErrNoProviderForEvent = errors.New("no provider for event")

	// ErrWorkflowFailed is returned when a workflow step fails.
	ErrWorkflowFailed = errors.New("workflow step failed")
)

// ConductorFactory creates conductors for isolated execution.
type ConductorFactory func(ctx context.Context, workDir string, opts conductor.Options) (*conductor.Conductor, error)

// ProviderGetter returns an instantiated provider by name.
type ProviderGetter func(name string) (any, error)

// Executor runs workflows for webhook jobs.
type Executor struct {
	conductorFactory ConductorFactory
	git              *vcs.Git
	config           *storage.AutomationSettings
	workspace        *storage.Workspace
	providerGetter   ProviderGetter
	workspaceRoot    string
	useWorktrees     bool
}

// ExecutorConfig holds configuration for the Executor.
type ExecutorConfig struct {
	ConductorFactory ConductorFactory
	Git              *vcs.Git
	Config           *storage.AutomationSettings
	Workspace        *storage.Workspace
	ProviderGetter   ProviderGetter
	WorkspaceRoot    string
}

// NewExecutor creates a new workflow executor.
func NewExecutor(cfg ExecutorConfig) *Executor {
	useWorktrees := true
	if cfg.Config != nil {
		// Check if any provider has worktrees disabled.
		for _, p := range cfg.Config.Providers {
			if !p.UseWorktrees {
				useWorktrees = false

				break
			}
		}
	}

	return &Executor{
		conductorFactory: cfg.ConductorFactory,
		git:              cfg.Git,
		config:           cfg.Config,
		workspace:        cfg.Workspace,
		providerGetter:   cfg.ProviderGetter,
		workspaceRoot:    cfg.WorkspaceRoot,
		useWorktrees:     useWorktrees,
	}
}

// Execute runs the appropriate workflow for a job.
func (e *Executor) Execute(ctx context.Context, job *WebhookJob) error {
	start := time.Now()

	slog.Info("executing job",
		"job_id", job.ID,
		"workflow", job.WorkflowType,
		"provider", job.Event.Provider,
		"repo", job.Event.Repository.FullName,
	)

	var err error
	switch job.WorkflowType {
	case WorkflowTypeIssueFix:
		err = e.executeIssueFix(ctx, job)
	case WorkflowTypePRReview:
		err = e.executePRReview(ctx, job)
	case WorkflowTypeCommand:
		err = e.executeCommand(ctx, job)
	default:
		err = fmt.Errorf("unknown workflow type: %s", job.WorkflowType)
	}

	duration := time.Since(start)
	if err != nil {
		slog.Error("job execution failed",
			"job_id", job.ID,
			"duration", duration,
			"error", err,
		)

		return err
	}

	slog.Info("job execution completed",
		"job_id", job.ID,
		"duration", duration,
	)

	return nil
}

// executeIssueFix runs the full issue-to-PR workflow.
func (e *Executor) executeIssueFix(ctx context.Context, job *WebhookJob) error {
	if job.Event.Issue == nil {
		return errors.New("no issue in event")
	}

	// Create worktree if configured.
	workDir := e.workspaceRoot
	var cleanupWorktree func()
	if e.useWorktrees {
		var err error
		workDir, cleanupWorktree, err = e.createWorktree(ctx, job)
		if err != nil {
			return fmt.Errorf("create worktree: %w", err)
		}
		job.WorktreePath = workDir
		defer func() {
			if cleanupWorktree != nil {
				cleanupWorktree()
			}
		}()
	}

	// Build provider reference.
	ref := job.Event.ProviderReference()
	if ref == "" {
		return ErrNoProviderForEvent
	}

	// Create conductor for this worktree.
	cond, err := e.conductorFactory(ctx, workDir, conductor.Options{
		UseWorktree: false, // Worktree already created
		// NoBranch defaults to false = branches created
	})
	if err != nil {
		return fmt.Errorf("create conductor: %w", err)
	}

	// Initialize conductor.
	if err := cond.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
	}

	// Step 1: Start task.
	slog.Info("starting task", "ref", ref)
	if err := cond.Start(ctx, ref); err != nil {
		return fmt.Errorf("start task: %w", err)
	}

	// Step 2: Plan.
	// This is where invalid issues will be detected.
	slog.Info("running plan step")
	if err := cond.Plan(ctx); err != nil {
		// Plan failure might indicate invalid issue.
		return e.handlePlanFailure(ctx, job, cond, err)
	}

	// Check if plan determined issue is invalid/duplicate/etc.
	if e.isPlanInvalid(cond) {
		return e.handleInvalidIssue(ctx, job, cond)
	}

	// Step 3: Implement.
	slog.Info("running implement step")
	if err := cond.Implement(ctx); err != nil {
		return fmt.Errorf("implement: %w", err)
	}

	// Step 4: Review.
	slog.Info("running review step")
	if err := cond.Review(ctx); err != nil {
		// Review failures are informational, continue to create PR.
		slog.Warn("review step had issues", "error", err)
	}

	// Step 5: Quality check (via review output).
	// Quality checks are handled as part of review.

	// Step 6: Finish with PR creation.
	slog.Info("finishing with PR creation")
	finishOpts := conductor.FinishOptions{
		DraftPR: false,
	}

	// Add mehrhof-generated label.
	if e.config != nil && e.config.Labels.MehrhofGenerated != "" {
		finishOpts.PRTitle = "" // Will use default
	}

	if err := cond.Finish(ctx, finishOpts); err != nil {
		return fmt.Errorf("finish: %w", err)
	}

	// Record result.
	job.Result = &JobResult{
		Success:  true,
		PRNumber: e.getPRNumber(cond),
		PRURL:    e.getPRURL(cond),
	}

	return nil
}

// PRReviewer is a subset of provider.PRReviewer interface.
type PRReviewer interface {
	SubmitReview(ctx context.Context, opts SubmitReviewOptions) (*ReviewSubmission, error)
}

// SubmitReviewOptions mirrors provider.SubmitReviewOptions.
type SubmitReviewOptions struct {
	PRNumber int
	Event    string // "APPROVE", "REQUEST_CHANGES", "COMMENT"
	Summary  string
	Comments []ReviewComment
}

// ReviewComment mirrors provider.ReviewComment.
type ReviewComment struct {
	Path string
	Line int
	Body string
}

// ReviewSubmission mirrors provider.ReviewSubmission.
type ReviewSubmission struct {
	ID             string
	URL            string
	CommentsPosted int
}

// executePRReview runs PR review workflow.
func (e *Executor) executePRReview(ctx context.Context, job *WebhookJob) error {
	if job.Event.PullRequest == nil {
		return errors.New("no pull request in event")
	}

	// For PR review, we don't need a full worktree.
	// We can use the existing conductor to run a standalone review.
	workDir := e.workspaceRoot

	cond, err := e.conductorFactory(ctx, workDir, conductor.Options{})
	if err != nil {
		return fmt.Errorf("create conductor: %w", err)
	}

	if err := cond.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
	}

	// Run standalone review on the PR.
	prRef := job.Event.ProviderReference()
	slog.Info("running PR review", "ref", prRef, "pr", job.Event.PullRequest.Number)

	// Use the conductor's standalone review capability.
	// Review the changes from the PR's head branch to base branch.
	reviewOpts := conductor.StandaloneReviewOptions{
		StandaloneDiffOptions: conductor.StandaloneDiffOptions{
			Mode:       conductor.DiffModeBranch,
			BaseBranch: job.Event.PullRequest.BaseBranch,
		},
	}

	result, err := cond.ReviewStandalone(ctx, reviewOpts)
	if err != nil {
		return fmt.Errorf("review: %w", err)
	}

	// Submit review to the provider if available.
	commentsPosted := 0
	var submissionError string
	if e.providerGetter != nil {
		submission, submitErr := e.submitReviewToProvider(ctx, job, result)
		if submitErr != nil {
			slog.Warn("failed to submit review to provider", "error", submitErr)
			submissionError = submitErr.Error()
		} else if submission != nil {
			commentsPosted = submission.CommentsPosted
		}
	}

	job.Result = &JobResult{
		Success:         true,
		PRNumber:        job.Event.PullRequest.Number,
		CommentsPosted:  commentsPosted,
		SubmissionError: submissionError,
	}

	return nil
}

// submitReviewToProvider submits review results to the provider.
func (e *Executor) submitReviewToProvider(ctx context.Context, job *WebhookJob, result *conductor.StandaloneReviewResult) (*ReviewSubmission, error) {
	if job.Event.PullRequest == nil {
		return nil, errors.New("no pull request associated with job for review submission")
	}

	p, err := e.providerGetter(job.Event.Provider)
	if err != nil {
		return nil, err
	}

	// Check if provider supports PR reviews.
	reviewer, ok := p.(PRReviewer)
	if !ok {
		// Provider doesn't support formal reviews, fall back to comment.
		return e.postReviewAsComment(ctx, job, result)
	}

	// Build review options.
	var event string
	switch result.Verdict {
	case "APPROVED":
		event = "APPROVE"
	case "NEEDS_CHANGES":
		event = "REQUEST_CHANGES"
	default:
		event = "COMMENT"
	}

	// Convert issues to review comments.
	var comments []ReviewComment
	for _, issue := range result.Issues {
		if issue.File != "" && issue.Line > 0 {
			comments = append(comments, ReviewComment{
				Path: issue.File,
				Line: issue.Line,
				Body: formatReviewIssue(issue),
			})
		}
	}

	opts := SubmitReviewOptions{
		PRNumber: job.Event.PullRequest.Number,
		Event:    event,
		Summary:  result.Summary,
		Comments: comments,
	}

	return reviewer.SubmitReview(ctx, opts)
}

// postReviewAsComment posts review results as a simple comment.
func (e *Executor) postReviewAsComment(ctx context.Context, job *WebhookJob, result *conductor.StandaloneReviewResult) (*ReviewSubmission, error) {
	p, err := e.providerGetter(job.Event.Provider)
	if err != nil {
		return nil, err
	}

	commenter, ok := p.(ResultCommenter)
	if !ok {
		return nil, errors.New("provider does not support commenting")
	}

	// Build comment body.
	body := buildReviewComment(result)
	prNumber := strconv.Itoa(job.Event.PullRequest.Number)
	_, err = commenter.AddComment(ctx, prNumber, body)
	if err != nil {
		return nil, err
	}

	return &ReviewSubmission{
		CommentsPosted: 1,
	}, nil
}

// buildReviewComment creates a comment body from review results.
func buildReviewComment(result *conductor.StandaloneReviewResult) string {
	var sb strings.Builder

	// Header based on verdict.
	switch result.Verdict {
	case "APPROVED":
		sb.WriteString("### ✅ Mehr Review: Approved\n\n")
	case "NEEDS_CHANGES":
		sb.WriteString("### ⚠️ Mehr Review: Changes Requested\n\n")
	default:
		sb.WriteString("### 📝 Mehr Review\n\n")
	}

	// Summary.
	if result.Summary != "" {
		sb.WriteString(result.Summary)
		sb.WriteString("\n\n")
	}

	// Issues.
	if len(result.Issues) > 0 {
		sb.WriteString("---\n\n**Issues Found:**\n\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("- **%s:%d** - %s\n", issue.File, issue.Line, issue.Message))
		}
	}

	return sb.String()
}

// formatReviewIssue formats a single review issue for a per-line comment.
func formatReviewIssue(issue conductor.ReviewIssue) string {
	var sb strings.Builder

	// Add severity/category header.
	if issue.Severity != "" || issue.Category != "" {
		sb.WriteString(fmt.Sprintf("**[%s/%s]** ", issue.Category, issue.Severity))
	}

	sb.WriteString(issue.Message)

	return sb.String()
}

// executeCommand handles @mehrhof commands in comments.
func (e *Executor) executeCommand(ctx context.Context, job *WebhookJob) error {
	if job.Event.Comment == nil {
		return errors.New("no comment in event")
	}

	// Parse command from comment.
	// Format: @mehrhof <command> [args]
	cmd := e.parseCommand(job.Event.Comment.Body, job.Command)

	slog.Info("executing command",
		"command", cmd.Name,
		"args", cmd.Args,
		"issuer", job.Event.Sender.Login,
	)

	switch strings.ToLower(cmd.Name) {
	case "fix":
		// Convert to issue fix workflow.
		if job.Event.Issue != nil {
			job.WorkflowType = WorkflowTypeIssueFix

			return e.executeIssueFix(ctx, job)
		}

		return errors.New("fix command requires issue context")

	case "review":
		// Run PR review.
		if job.Event.PullRequest != nil {
			job.WorkflowType = WorkflowTypePRReview

			return e.executePRReview(ctx, job)
		}

		return errors.New("review command requires PR context")

	case "status":
		// Report status - would post a comment with current queue status.
		return e.postStatusComment(ctx, job)

	case "help":
		// Post help text.
		return e.postHelpComment(ctx, job)

	default:
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
}

// Command represents a parsed command.
type Command struct {
	Name string
	Args []string
}

// parseCommand extracts command from comment text.
func (e *Executor) parseCommand(body, presetCmd string) Command {
	if presetCmd != "" {
		return Command{Name: presetCmd}
	}

	// Find @mehrhof or command prefix.
	prefix := "@mehrhof"
	if e.config != nil {
		for _, p := range e.config.Providers {
			if p.CommandPrefix != "" {
				prefix = p.CommandPrefix

				break
			}
		}
	}

	// Find prefix in body.
	idx := strings.Index(strings.ToLower(body), strings.ToLower(prefix))
	if idx == -1 {
		return Command{Name: "unknown"}
	}

	// Extract text after prefix.
	rest := strings.TrimSpace(body[idx+len(prefix):])
	parts := strings.Fields(rest)

	if len(parts) == 0 {
		return Command{Name: "help"}
	}

	return Command{
		Name: parts[0],
		Args: parts[1:],
	}
}

// createWorktree creates an isolated worktree for a job.
func (e *Executor) createWorktree(ctx context.Context, job *WebhookJob) (string, func(), error) {
	if e.git == nil {
		return "", nil, errors.New("git not initialized")
	}

	// Create worktree directory.
	worktreeName := "mehrhof-auto-" + job.ID
	worktreePath := filepath.Join(e.workspaceRoot, ".mehrhof", "worktrees", worktreeName)

	// Ensure directory exists.
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", nil, err
	}

	// Create worktree with new branch.
	branchName := "mehrhof-auto/" + job.ID
	baseBranch := job.Event.Repository.DefaultBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	if err := e.git.CreateWorktreeNewBranch(ctx, worktreePath, branchName, baseBranch); err != nil {
		return "", nil, err
	}

	cleanup := func() {
		slog.Debug("cleaning up worktree", "path", worktreePath)
		if err := e.git.RemoveWorktree(ctx, worktreePath, true); err != nil {
			slog.Warn("failed to remove worktree", "path", worktreePath, "error", err)
		}
	}

	return worktreePath, cleanup, nil
}

// handlePlanFailure handles errors during planning.
//

func (e *Executor) handlePlanFailure(ctx context.Context, job *WebhookJob, _ *conductor.Conductor, planErr error) error {
	// Post comment about failure.
	comment := fmt.Sprintf(
		"### Mehr Automation: Plan Failed\n\n"+
			"Unable to create implementation plan for this issue.\n\n"+
			"**Error:** %s\n\n"+
			"Please review the issue description and ensure it contains sufficient detail.",
		planErr.Error(),
	)

	if err := e.postIssueComment(ctx, job, comment); err != nil {
		slog.Warn("failed to post failure comment", "error", err)
	}

	return errors.Join(ErrWorkflowFailed, planErr)
}

// isPlanInvalid checks if the plan indicates an invalid issue by examining
// the generated specifications for rejection signals from the AI agent.
func (e *Executor) isPlanInvalid(cond *conductor.Conductor) bool {
	ws := cond.GetWorkspace()
	activeTask := cond.GetActiveTask()
	if ws == nil || activeTask == nil {
		return false
	}

	content, err := ws.GatherSpecificationsContent(activeTask.ID)
	if err != nil || content == "" {
		return false
	}

	lower := strings.ToLower(content)

	// Rejection phrases that indicate the issue should not be implemented.
	rejectionPhrases := []string{
		"this issue is invalid",
		"this issue is a duplicate",
		"cannot be implemented",
		"should not be implemented",
		"won't fix",
		"wont fix",
		"not actionable",
		"insufficient information to proceed",
		"duplicate of #",
		"this is not a valid issue",
		"no changes required",
		"no implementation needed",
	}

	for _, phrase := range rejectionPhrases {
		if strings.Contains(lower, phrase) {
			slog.Info("plan determined issue is invalid", "phrase", phrase, "task_id", activeTask.ID)

			return true
		}
	}

	return false
}

// handleInvalidIssue handles issues determined to be invalid.
//

func (e *Executor) handleInvalidIssue(ctx context.Context, job *WebhookJob, _ *conductor.Conductor) error {
	comment := "### Mehr Automation: Issue Analysis\n\n" +
		"After analysis, this issue appears to be invalid or cannot be automatically fixed.\n\n" +
		"Please review the issue and provide more details if needed."

	if err := e.postIssueComment(ctx, job, comment); err != nil {
		slog.Warn("failed to post invalid issue comment", "error", err)
	}

	return errors.New("issue determined to be invalid")
}

// postIssueComment posts a comment to the issue.
func (e *Executor) postIssueComment(ctx context.Context, job *WebhookJob, body string) error {
	if e.providerGetter == nil {
		return errors.New("no provider getter")
	}

	p, err := e.providerGetter(job.Event.Provider)
	if err != nil {
		return fmt.Errorf("get provider %s: %w", job.Event.Provider, err)
	}

	commenter, ok := p.(ResultCommenter)
	if !ok {
		return errors.New("provider does not support commenting")
	}

	if job.Event.Issue == nil {
		return errors.New("no issue associated with job for posting comment")
	}

	issueID := strconv.Itoa(job.Event.Issue.Number)
	_, err = commenter.AddComment(ctx, issueID, body)
	if err != nil {
		slog.Warn("failed to post issue comment",
			"job_id", job.ID,
			"issue", issueID,
			"provider", job.Event.Provider,
			"error", err,
		)

		return fmt.Errorf("post issue comment: %w", err)
	}

	return nil
}

// postStatusComment posts a status comment.
func (e *Executor) postStatusComment(ctx context.Context, job *WebhookJob) error {
	// Build status message.
	status := "### Mehr Automation Status\n\n" +
		"The automation system is running.\n\n" +
		"Use `@mehrhof help` for available commands."

	return e.postIssueComment(ctx, job, status)
}

// postHelpComment posts a help comment.
func (e *Executor) postHelpComment(ctx context.Context, job *WebhookJob) error {
	help := "### Mehr Automation Commands\n\n" +
		"Available commands:\n\n" +
		"- `@mehrhof fix` - Automatically fix this issue and create a PR\n" +
		"- `@mehrhof review` - Review the current PR\n" +
		"- `@mehrhof status` - Show automation status\n" +
		"- `@mehrhof help` - Show this help message\n"

	return e.postIssueComment(ctx, job, help)
}

// getPRNumber extracts PR number from conductor's last finish result.
func (e *Executor) getPRNumber(cond *conductor.Conductor) int {
	n, _ := cond.LastPRResult()

	return n
}

// getPRURL extracts PR URL from conductor's last finish result.
func (e *Executor) getPRURL(cond *conductor.Conductor) string {
	_, u := cond.LastPRResult()

	return u
}
