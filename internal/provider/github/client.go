package github

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"

	"github.com/valksor/go-mehrhof/internal/provider/token"
	"github.com/valksor/go-toolkit/cache"
)

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}

// Client wraps the GitHub API client.
type Client struct {
	gh    *github.Client
	cache *cache.Cache
	owner string
	repo  string
}

// NewClient creates a new GitHub API client.
func NewClient(ctx context.Context, token, owner, repo string) *Client {
	return NewClientWithCache(ctx, token, owner, repo, nil)
}

// NewClientWithCache creates a new GitHub API client with an optional cache.
func NewClientWithCache(ctx context.Context, token, owner, repo string, c *cache.Cache) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh:    github.NewClient(tc),
		owner: owner,
		repo:  repo,
		cache: c,
	}
}

// SetCache sets or updates the cache for this client.
func (c *Client) SetCache(cache *cache.Cache) {
	c.cache = cache
}

// CacheKey generates a namespaced cache key for this client.
func (c *Client) CacheKey(resourceType, id string) string {
	return fmt.Sprintf("github:%s/%s:%s:%s", c.owner, c.repo, resourceType, id)
}

// ResolveToken resolves the GitHub API token.
// The configToken should be from config.yaml and may use ${VAR} syntax.
// Falls back to gh CLI auth token if config is empty.
func ResolveToken(configToken string) (string, error) {
	return token.ResolveToken(token.Config("GITHUB", configToken).
		WithCLIFallback(getGHCLIToken))
}

// getGHCLIToken attempts to get the token from the gh CLI.
func getGHCLIToken() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// GetIssue fetches an issue by number.
func (c *Client) GetIssue(ctx context.Context, number int) (*github.Issue, error) {
	key := c.CacheKey("issue", strconv.Itoa(number))

	// Try cache first
	if c.cache != nil && c.cache.Enabled() {
		if val, ok := c.cache.Get(key); ok && val != nil {
			if issue, ok := val.(*github.Issue); ok {
				return issue, nil
			}
		}
	}

	// Cache miss or disabled, fetch from API
	issue, resp, err := c.gh.Issues.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	// Store in cache
	if c.cache != nil && c.cache.Enabled() {
		c.cache.Set(key, issue, cache.DefaultIssueTTL)
	}

	return issue, nil
}

// GetIssueComments fetches all comments on an issue.
func (c *Client) GetIssueComments(ctx context.Context, number int) ([]*github.IssueComment, error) {
	key := c.CacheKey("comments", strconv.Itoa(number))

	// Try cache first
	if c.cache != nil && c.cache.Enabled() {
		if val, ok := c.cache.Get(key); ok && val != nil {
			if comments, ok := val.([]*github.IssueComment); ok {
				return comments, nil
			}
		}
	}

	// Cache miss or disabled, fetch from API
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	const maxPages = 100

	var allComments []*github.IssueComment
	truncated := true
	for range maxPages {
		comments, resp, err := c.gh.Issues.ListComments(ctx, c.owner, c.repo, number, opts)
		if err != nil {
			return nil, wrapAPIError(err)
		}

		checkRateLimit(resp)
		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			truncated = false

			break
		}
		opts.Page = resp.NextPage
	}

	// Warn if we exhausted maxPages without reaching the last page.
	if truncated && len(allComments) > 0 {
		slog.Warn("GitHub comments truncated at max pages",
			"issue", number,
			"pages_fetched", maxPages,
			"comments_fetched", len(allComments),
		)
	}

	// Store in cache
	if c.cache != nil && c.cache.Enabled() {
		c.cache.Set(key, allComments, cache.DefaultCommentsTTL)
	}

	return allComments, nil
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(ctx context.Context, number int, body string) (*github.IssueComment, error) {
	comment, resp, err := c.gh.Issues.CreateComment(ctx, c.owner, c.repo, number, &github.IssueComment{
		Body: ptr(body),
	})
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	// Invalidate comments cache for this issue
	if c.cache != nil {
		key := c.CacheKey("comments", strconv.Itoa(number))
		c.cache.Delete(key)
	}

	return comment, nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, title, body, head, base string, draft bool) (*github.PullRequest, error) {
	pr, resp, err := c.gh.PullRequests.Create(ctx, c.owner, c.repo, &github.NewPullRequest{
		Title: ptr(title),
		Body:  ptr(body),
		Head:  ptr(head),
		Base:  ptr(base),
		Draft: ptr(draft),
	})
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	return pr, nil
}

// GetDefaultBranch returns the repository's default branch.
func (c *Client) GetDefaultBranch(ctx context.Context) (string, error) {
	key := c.CacheKey("metadata", "default-branch")

	// Try cache first
	if c.cache != nil && c.cache.Enabled() {
		if val, ok := c.cache.Get(key); ok && val != nil {
			if branch, ok := val.(string); ok {
				return branch, nil
			}
		}
	}

	// Cache miss or disabled, fetch from API
	repo, resp, err := c.gh.Repositories.Get(ctx, c.owner, c.repo)
	if err != nil {
		return "", wrapAPIError(err)
	}
	checkRateLimit(resp)
	branch := repo.GetDefaultBranch()

	// Store in cache
	if c.cache != nil && c.cache.Enabled() {
		c.cache.Set(key, branch, cache.DefaultMetadataTTL)
	}

	return branch, nil
}

// DownloadFile downloads a file from the repository.
func (c *Client) DownloadFile(ctx context.Context, path, ref string) ([]byte, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	content, _, resp, err := c.gh.Repositories.GetContents(ctx, c.owner, c.repo, path, opts)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	decoded, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}

	return []byte(decoded), nil
}

// GetPullRequest fetches a pull request by number.
func (c *Client) GetPullRequest(ctx context.Context, number int) (*github.PullRequest, error) {
	pr, resp, err := c.gh.PullRequests.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	return pr, nil
}

// GetPullRequestDiff fetches the diff for a pull request.
func (c *Client) GetPullRequestDiff(ctx context.Context, number int) (string, []*github.CommitFile, int, int, error) {
	opts := &github.ListOptions{PerPage: 100}

	var allFiles []*github.CommitFile
	var totalAdditions, totalDeletions int

	// Fetch files with pagination
	for {
		files, resp, err := c.gh.PullRequests.ListFiles(ctx, c.owner, c.repo, number, opts)
		if err != nil {
			return "", nil, 0, 0, wrapAPIError(err)
		}
		checkRateLimit(resp)

		for _, f := range files {
			totalAdditions += f.GetAdditions()
			totalDeletions += f.GetDeletions()
		}
		allFiles = append(allFiles, files...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Get raw diff
	diff, rawResp, err := c.gh.PullRequests.GetRaw(ctx, c.owner, c.repo, number, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", nil, 0, 0, wrapAPIError(err)
	}
	checkRateLimit(rawResp)

	return diff, allFiles, totalAdditions, totalDeletions, nil
}

// CreatePullRequestComment adds a comment to a pull request (issue comment in GitHub).
func (c *Client) CreatePullRequestComment(ctx context.Context, number int, body string) (*github.IssueComment, error) {
	comment, resp, err := c.gh.Issues.CreateComment(ctx, c.owner, c.repo, number, &github.IssueComment{
		Body: ptr(body),
	})
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	// Invalidate comments cache for this PR
	if c.cache != nil {
		key := c.CacheKey("comments", strconv.Itoa(number))
		c.cache.Delete(key)
	}

	return comment, nil
}

// UpdatePullRequestComment updates an existing comment on a pull request.
func (c *Client) UpdatePullRequestComment(ctx context.Context, number int, commentID int64, body string) (*github.IssueComment, error) {
	comment, resp, err := c.gh.Issues.EditComment(ctx, c.owner, c.repo, commentID, &github.IssueComment{
		Body: ptr(body),
	})
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	// Invalidate comments cache for this PR
	if c.cache != nil {
		key := c.CacheKey("comments", strconv.Itoa(number))
		c.cache.Delete(key)
	}

	return comment, nil
}

// GetPullRequestComments fetches all comments on a pull request.
func (c *Client) GetPullRequestComments(ctx context.Context, number int) ([]*github.IssueComment, error) {
	return c.GetIssueComments(ctx, number)
}

// CreateReview creates a formal review on a pull request.
func (c *Client) CreateReview(ctx context.Context, number int, event, body string, comments []*github.DraftReviewComment) (*github.PullRequestReview, error) {
	review := &github.PullRequestReviewRequest{
		Event: ptr(event),
	}

	// Only set body if non-empty
	if body != "" {
		review.Body = ptr(body)
	}

	// Only set comments if non-empty
	if len(comments) > 0 {
		review.Comments = comments
	}

	result, resp, err := c.gh.PullRequests.CreateReview(ctx, c.owner, c.repo, number, review)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	checkRateLimit(resp)

	return result, nil
}

// SetOwnerRepo updates the owner and repo for the client.
func (c *Client) SetOwnerRepo(owner, repo string) {
	c.owner = owner
	c.repo = repo
}

// Owner returns the current owner.
func (c *Client) Owner() string {
	return c.owner
}

// Repo returns the current repo.
func (c *Client) Repo() string {
	return c.repo
}

// checkRateLimit logs a warning when the GitHub API rate limit is running low.
func checkRateLimit(resp *github.Response) {
	if resp == nil {
		return
	}

	if resp.Rate.Remaining < 10 {
		slog.Warn("GitHub API rate limit low",
			"remaining", resp.Rate.Remaining,
			"limit", resp.Rate.Limit,
			"reset", resp.Rate.Reset.Format(time.RFC3339),
		)
	}
}
