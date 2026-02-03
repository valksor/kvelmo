package gitlab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/valksor/go-mehrhof/internal/provider/token"
)

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}

// Client wraps the GitLab API client.
type Client struct {
	gl          *gl.Client
	httpClient  *http.Client
	token       string
	projectID   int64  // Numeric project ID (cached)
	projectPath string // Project path (e.g., "group/project")
	host        string // GitLab host (e.g., "gitlab.com" or custom)
}

// NewClient creates a new GitLab API client.
func NewClient(token, host, projectPath string, projectID int64) (*Client, error) {
	var options []gl.ClientOptionFunc

	// For self-hosted GitLab, set the base URL
	if host != "" && host != "https://gitlab.com" && host != "gitlab.com" {
		baseURL := strings.TrimSuffix(host, "/") + "/api/v4"
		options = append(options, gl.WithBaseURL(baseURL))
	}

	client, err := gl.NewClient(token, options...)
	if err != nil {
		return nil, fmt.Errorf("create GitLab client: %w", err)
	}

	return &Client{
		gl:          client,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		token:       token,
		projectPath: projectPath,
		projectID:   projectID,
		host:        host,
	}, nil
}

// ResolveToken resolves the GitLab API token.
// The configToken should be from config.yaml and may use ${VAR} syntax.
func ResolveToken(configToken string) (string, error) {
	return token.ResolveToken(token.Config("GITLAB", configToken))
}

// getProjectID retrieves the numeric project ID from the project path.
func (c *Client) getProjectID(ctx context.Context) (int64, error) {
	if c.projectID > 0 {
		return c.projectID, nil
	}

	if c.projectPath == "" {
		return 0, ErrProjectNotConfigured
	}

	// Get project by path
	project, _, err := c.gl.Projects.GetProject(c.projectPath, nil, gl.WithContext(ctx))
	if err != nil {
		return 0, wrapAPIError(err)
	}

	c.projectID = project.ID

	return c.projectID, nil
}

// GetIssue fetches an issue by IID (internal issue number).
func (c *Client) GetIssue(ctx context.Context, iid int64) (*gl.Issue, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	issue, _, err := c.gl.Issues.GetIssue(pid, iid, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return issue, nil
}

// GetIssueNotes fetches all notes (comments) on an issue.
func (c *Client) GetIssueNotes(ctx context.Context, iid int64) ([]*gl.Note, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	var allNotes []*gl.Note
	opts := &gl.ListIssueNotesOptions{}
	opts.Page = 1
	opts.PerPage = 100

	for {
		notes, resp, err := c.gl.Notes.ListIssueNotes(pid, iid, opts, gl.WithContext(ctx))
		if err != nil {
			return nil, wrapAPIError(err)
		}
		allNotes = append(allNotes, notes...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allNotes, nil
}

// AddNote adds a note (comment) to an issue.
func (c *Client) AddNote(ctx context.Context, iid int64, body string) (*gl.Note, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	note, _, err := c.gl.Notes.CreateIssueNote(pid, iid, &gl.CreateIssueNoteOptions{
		Body: ptr(body),
	}, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return note, nil
}

// UpdateIssue updates an issue.
func (c *Client) UpdateIssue(ctx context.Context, iid int64, opts *gl.UpdateIssueOptions) (*gl.Issue, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	issue, _, err := c.gl.Issues.UpdateIssue(pid, iid, opts, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return issue, nil
}

// ListIssues lists issues with filters.
func (c *Client) ListIssues(ctx context.Context, opts *gl.ListProjectIssuesOptions) ([]*gl.Issue, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	// Set default pagination
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}

	var allIssues []*gl.Issue
	for {
		issues, resp, err := c.gl.Issues.ListProjectIssues(pid, opts, gl.WithContext(ctx))
		if err != nil {
			return nil, wrapAPIError(err)
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, opts *gl.CreateIssueOptions) (*gl.Issue, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	issue, _, err := c.gl.Issues.CreateIssue(pid, opts, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return issue, nil
}

// SetLabels sets labels on an issue (by updating the issue).
func (c *Client) SetLabels(ctx context.Context, iid int64, labels []string) error {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return err
	}

	// Update labels requires setting the full label list
	labelOpts := gl.LabelOptions(labels)
	_, _, err = c.gl.Issues.UpdateIssue(pid, iid, &gl.UpdateIssueOptions{
		Labels: &labelOpts,
	}, gl.WithContext(ctx))

	return err
}

// AddLabels adds labels to an issue.
func (c *Client) AddLabels(ctx context.Context, iid int64, labels []string) error {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return err
	}

	// Get current issue to check existing labels
	issue, err := c.GetIssue(ctx, iid)
	if err != nil {
		return err
	}

	// Merge labels
	existingLabels := make(map[string]bool)
	for _, l := range issue.Labels {
		existingLabels[l] = true
	}

	for _, l := range labels {
		if !existingLabels[l] {
			issue.Labels = append(issue.Labels, l)
		}
	}

	// Update with new label set
	labelOpts := gl.LabelOptions(issue.Labels)
	_, _, err = c.gl.Issues.UpdateIssue(pid, iid, &gl.UpdateIssueOptions{
		Labels: &labelOpts,
	}, gl.WithContext(ctx))

	return err
}

// RemoveLabel removes a label from an issue.
func (c *Client) RemoveLabel(ctx context.Context, iid int64, label string) error {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return err
	}

	// Get current issue to check existing labels
	issue, err := c.GetIssue(ctx, iid)
	if err != nil {
		return err
	}

	// Remove the label from the list
	newLabels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		if l != label {
			newLabels = append(newLabels, l)
		}
	}

	// Update with new label set
	labelOpts := gl.LabelOptions(newLabels)
	_, _, err = c.gl.Issues.UpdateIssue(pid, iid, &gl.UpdateIssueOptions{
		Labels: &labelOpts,
	}, gl.WithContext(ctx))

	return err
}

// SetProjectPath updates the project path for the client.
func (c *Client) SetProjectPath(projectPath string) {
	c.projectPath = projectPath
	c.projectID = 0 // Reset cached ID
}

// SetProjectID sets the numeric project ID directly.
func (c *Client) SetProjectID(projectID int64) {
	c.projectID = projectID
}

// ProjectPath returns the current project path.
func (c *Client) ProjectPath() string {
	return c.projectPath
}

// ProjectID returns the cached project ID (0 if not cached).
func (c *Client) ProjectID() int64 {
	return c.projectID
}

// Host returns the GitLab host.
func (c *Client) Host() string {
	if c.host != "" {
		return c.host
	}

	return "gitlab.com"
}

// CreateMergeRequest creates a new merge request.
func (c *Client) CreateMergeRequest(ctx context.Context, title, description, sourceBranch, targetBranch string, removeSourceBranch bool) (*gl.MergeRequest, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	mr, _, err := c.gl.MergeRequests.CreateMergeRequest(pid, &gl.CreateMergeRequestOptions{
		Title:              ptr(title),
		Description:        ptr(description),
		SourceBranch:       ptr(sourceBranch),
		TargetBranch:       ptr(targetBranch),
		RemoveSourceBranch: ptr(removeSourceBranch),
	}, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return mr, nil
}

// GetDefaultBranch returns the project's default branch.
func (c *Client) GetDefaultBranch(ctx context.Context) (string, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return "", err
	}

	project, _, err := c.gl.Projects.GetProject(pid, nil, gl.WithContext(ctx))
	if err != nil {
		return "", wrapAPIError(err)
	}

	return project.DefaultBranch, nil
}

// DownloadAttachment downloads an attachment by URL with authentication.
// Only HTTPS URLs are allowed for security. The URL must be from a GitLab host.
func (c *Client) DownloadAttachment(ctx context.Context, url string) (io.ReadCloser, error) {
	// Validate URL scheme - only HTTPS allowed
	if !strings.HasPrefix(url, "https://") {
		return nil, errors.New("invalid URL scheme: only HTTPS URLs are allowed")
	}

	// Validate URL is from a trusted GitLab host
	host := c.Host()
	if host == "gitlab.com" {
		host = "https://gitlab.com"
	}
	if !strings.HasPrefix(host, "https://") && !strings.HasPrefix(host, "http://") {
		host = "https://" + host
	}
	if !strings.HasPrefix(url, host) && !strings.HasPrefix(url, "https://gitlab.com") {
		return nil, errors.New("URL host does not match configured GitLab host")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Private-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()

		return nil, fmt.Errorf("download failed: status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MR Review Support (GetMergeRequest, GetMergeRequestDiff, CreateMergeRequestNote, etc.)
// ─────────────────────────────────────────────────────────────────────────────

// GetMergeRequest fetches a merge request by IID.
func (c *Client) GetMergeRequest(ctx context.Context, iid int64) (*gl.MergeRequest, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	mr, _, err := c.gl.MergeRequests.GetMergeRequest(pid, iid, nil, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return mr, nil
}

// GetMergeRequestDiff fetches the diff for a merge request.
func (c *Client) GetMergeRequestDiff(ctx context.Context, iid int64) (string, []*gl.MergeRequestDiff, int, int, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return "", nil, 0, 0, err
	}

	var allDiffs []*gl.MergeRequestDiff
	var totalAdditions, totalDeletions int

	opts := &gl.ListMergeRequestDiffsOptions{}
	opts.PerPage = 100

	for {
		diffs, resp, err := c.gl.MergeRequests.ListMergeRequestDiffs(pid, iid, opts, gl.WithContext(ctx))
		if err != nil {
			return "", nil, 0, 0, wrapAPIError(err)
		}

		// GitLab API doesn't provide per-file additions/deletions
		// Total additions/deletions are calculated from MR diff_size field
		allDiffs = append(allDiffs, diffs...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Get MR for commit count and other info
	_, err = c.GetMergeRequest(ctx, iid)
	if err != nil {
		return "", nil, 0, 0, err
	}

	// Generate raw diff from individual file diffs
	var rawDiff strings.Builder
	for _, d := range allDiffs {
		rawDiff.WriteString(d.Diff)
	}

	return rawDiff.String(), allDiffs, totalAdditions, totalDeletions, nil
}

// CreateMergeRequestNote adds a note (comment) to a merge request.
func (c *Client) CreateMergeRequestNote(ctx context.Context, iid int64, body string) (*gl.Note, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	note, _, err := c.gl.Notes.CreateMergeRequestNote(pid, iid, &gl.CreateMergeRequestNoteOptions{
		Body: ptr(body),
	}, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return note, nil
}

// UpdateMergeRequestNote updates an existing note on a merge request.
func (c *Client) UpdateMergeRequestNote(ctx context.Context, iid int64, noteID int64, body string) (*gl.Note, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	note, _, err := c.gl.Notes.UpdateMergeRequestNote(pid, iid, noteID, &gl.UpdateMergeRequestNoteOptions{
		Body: ptr(body),
	}, gl.WithContext(ctx))
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return note, nil
}

// GetMergeRequestNotes fetches all notes on a merge request.
func (c *Client) GetMergeRequestNotes(ctx context.Context, iid int64) ([]*gl.Note, error) {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return nil, err
	}

	var allNotes []*gl.Note
	opts := &gl.ListMergeRequestNotesOptions{}
	opts.Page = 1
	opts.PerPage = 100

	for {
		notes, resp, err := c.gl.Notes.ListMergeRequestNotes(pid, iid, opts, gl.WithContext(ctx))
		if err != nil {
			return nil, wrapAPIError(err)
		}
		allNotes = append(allNotes, notes...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allNotes, nil
}

// CreateMergeRequestDiffNote creates a discussion note on a specific line of a merge request diff.
// This uses GitLab's Discussions API to create inline comments on the diff.
func (c *Client) CreateMergeRequestDiffNote(ctx context.Context, iid int64, headSHA, filePath string, newLine int, body string) error {
	pid, err := c.getProjectID(ctx)
	if err != nil {
		return err
	}

	// Create a new discussion with a position on the diff
	lineNum := int64(newLine)
	position := &gl.PositionOptions{
		BaseSHA:      ptr(headSHA),
		HeadSHA:      ptr(headSHA),
		StartSHA:     ptr(headSHA),
		PositionType: ptr("text"),
		NewPath:      ptr(filePath),
		NewLine:      &lineNum,
	}

	_, _, err = c.gl.Discussions.CreateMergeRequestDiscussion(pid, iid, &gl.CreateMergeRequestDiscussionOptions{
		Body:     ptr(body),
		Position: position,
	}, gl.WithContext(ctx))
	if err != nil {
		return wrapAPIError(err)
	}

	return nil
}
