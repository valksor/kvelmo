package github

import (
	"context"

	"github.com/google/go-github/v67/github"
	"github.com/valksor/go-toolkit/workunit"
)

// UpdateStatus changes the state of a GitHub issue.
func (p *Provider) UpdateStatus(ctx context.Context, workUnitID string, status workunit.Status) error {
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return err
	}

	owner := ref.Owner
	repo := ref.Repo
	if owner == "" {
		owner = p.owner
	}
	if repo == "" {
		repo = p.repo
	}

	if owner == "" || repo == "" {
		return ErrRepoNotConfigured
	}

	p.client.SetOwnerRepo(owner, repo)

	// Map provider status to GitHub state
	githubState := mapStatusToGitHubState(status)

	_, _, err = p.client.EditIssue(ctx, ref.IssueNumber, &github.IssueRequest{
		State: &githubState,
	})

	return err
}

// mapStatusToGitHubState converts provider status to GitHub API state.
func mapStatusToGitHubState(status workunit.Status) string {
	switch status {
	case workunit.StatusOpen, workunit.StatusInProgress, workunit.StatusReview:
		return "open"
	case workunit.StatusClosed, workunit.StatusDone:
		return "closed"
	default:
		return "open"
	}
}

// EditIssue wraps the GitHub API call for editing an issue.
func (c *Client) EditIssue(ctx context.Context, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	return c.gh.Issues.Edit(ctx, c.owner, c.repo, number, issue)
}
