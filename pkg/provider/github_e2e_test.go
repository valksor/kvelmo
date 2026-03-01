//go:build e2e

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
)

// E2E tests for GitHub provider.
// Run with: go test -tags=e2e -v ./pkg/provider/... -run TestE2E
//
// Required environment variables:
//   E2E_GITHUB_REPO  - Repository in "owner/repo" format (e.g., "ozo2003/e2e-test")
//   GITHUB_TOKEN     - Personal access token with repo scope

func getE2EConfig(t *testing.T) (owner, repo, token string) {
	t.Helper()

	repoFull := os.Getenv("E2E_GITHUB_REPO")
	if repoFull == "" {
		t.Skip("E2E_GITHUB_REPO not set")
	}

	parts := strings.SplitN(repoFull, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("E2E_GITHUB_REPO must be in owner/repo format, got: %s", repoFull)
	}
	owner, repo = parts[0], parts[1]

	token = os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("E2E_GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("GITHUB_TOKEN or E2E_GITHUB_TOKEN not set")
	}

	return owner, repo, token
}

func TestE2E_CreateAndCloseIssue(t *testing.T) {
	owner, repo, token := getE2EConfig(t)
	ctx := context.Background()

	provider := NewGitHubProvider(token)
	client := newGitHubClient(token, "")

	// Create issue
	title := fmt.Sprintf("E2E Test Issue %d", time.Now().Unix())
	body := "This is an automated test issue.\n\nIt will be closed automatically."
	labels := []string{"test", "automated"}

	issue, _, err := client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}
	issueNum := issue.GetNumber()
	t.Logf("Created issue #%d: %s", issueNum, issue.GetHTMLURL())

	// Cleanup: close and delete issue
	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			t.Logf("Skipping cleanup (E2E_SKIP_CLEANUP set)")
			return
		}
		state := "closed"
		_, _, _ = client.Issues.Edit(ctx, owner, repo, issueNum, &github.IssueRequest{State: &state})
		t.Logf("Closed issue #%d", issueNum)
	})

	// Fetch via provider
	taskID := fmt.Sprintf("%s/%s#%d", owner, repo, issueNum)
	task, err := provider.FetchTask(ctx, taskID)
	if err != nil {
		t.Fatalf("FetchTask: %v", err)
	}

	if task.Title != title {
		t.Errorf("Title = %q, want %q", task.Title, title)
	}
	if task.Description != body {
		t.Errorf("Description mismatch")
	}
	if len(task.Labels) != 2 {
		t.Errorf("Labels = %v, want 2 labels", task.Labels)
	}

	// Update status to closed
	err = provider.UpdateStatus(ctx, taskID, "closed")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// Verify closed
	task, err = provider.FetchTask(ctx, taskID)
	if err != nil {
		t.Fatalf("FetchTask after close: %v", err)
	}
	if task.Metadata("github_state") != "closed" {
		t.Errorf("State = %q, want closed", task.Metadata("github_state"))
	}
}

func TestE2E_UpdateIssue(t *testing.T) {
	owner, repo, token := getE2EConfig(t)
	ctx := context.Background()

	client := newGitHubClient(token, "")

	// Create issue
	title := fmt.Sprintf("E2E Update Test %d", time.Now().Unix())
	body := "Original body"

	issue, _, err := client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title: &title,
		Body:  &body,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}
	issueNum := issue.GetNumber()
	t.Logf("Created issue #%d", issueNum)

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		state := "closed"
		_, _, _ = client.Issues.Edit(ctx, owner, repo, issueNum, &github.IssueRequest{State: &state})
	})

	// Update title and body
	newTitle := title + " (updated)"
	newBody := "Updated body\n\nWith more content."
	_, _, err = client.Issues.Edit(ctx, owner, repo, issueNum, &github.IssueRequest{
		Title: &newTitle,
		Body:  &newBody,
	})
	if err != nil {
		t.Fatalf("Edit issue: %v", err)
	}

	// Verify update
	updated, _, err := client.Issues.Get(ctx, owner, repo, issueNum)
	if err != nil {
		t.Fatalf("Get issue: %v", err)
	}

	if updated.GetTitle() != newTitle {
		t.Errorf("Title = %q, want %q", updated.GetTitle(), newTitle)
	}
	if updated.GetBody() != newBody {
		t.Errorf("Body not updated")
	}
}

func TestE2E_IssueComments(t *testing.T) {
	owner, repo, token := getE2EConfig(t)
	ctx := context.Background()

	client := newGitHubClient(token, "")

	// Create issue
	title := fmt.Sprintf("E2E Comment Test %d", time.Now().Unix())
	issue, _, err := client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title: &title,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}
	issueNum := issue.GetNumber()

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		state := "closed"
		_, _, _ = client.Issues.Edit(ctx, owner, repo, issueNum, &github.IssueRequest{State: &state})
	})

	// Add comment
	commentBody := "This is an automated test comment."
	comment, _, err := client.Issues.CreateComment(ctx, owner, repo, issueNum, &github.IssueComment{
		Body: &commentBody,
	})
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	t.Logf("Created comment ID %d", comment.GetID())

	// List comments
	comments, _, err := client.Issues.ListComments(ctx, owner, repo, issueNum, nil)
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}

	if len(comments) < 1 {
		t.Errorf("Expected at least 1 comment, got %d", len(comments))
	}

	found := false
	for _, c := range comments {
		if c.GetBody() == commentBody {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Comment not found in list")
	}
}

func TestE2E_CreatePR(t *testing.T) {
	owner, repo, token := getE2EConfig(t)
	ctx := context.Background()

	provider := NewGitHubProvider(token)
	client := newGitHubClient(token, "")

	// Get default branch
	repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		t.Fatalf("Get repo: %v", err)
	}
	defaultBranch := repoInfo.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main" // GitHub's new default
	}

	// Get latest commit SHA - handle empty repo by creating initial commit
	ref, resp, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+defaultBranch)
	if err != nil {
		if resp != nil && resp.StatusCode == 409 {
			// Repo is empty, create initial README to bootstrap default branch
			t.Log("Repo is empty, creating initial commit")
			readmeContent := []byte("# E2E Test Repository\n\nThis repository is used for automated E2E testing.\n")
			initMsg := "Initial commit"
			// Explicitly specify branch when creating initial file
			_, _, err = client.Repositories.CreateFile(ctx, owner, repo, "README.md", &github.RepositoryContentFileOptions{
				Message: &initMsg,
				Content: readmeContent,
				Branch:  &defaultBranch,
			})
			if err != nil {
				t.Fatalf("CreateFile (initial): %v", err)
			}
			// Small delay to let GitHub process the commit
			time.Sleep(500 * time.Millisecond)
			// Now get the ref again
			ref, _, err = client.Git.GetRef(ctx, owner, repo, "refs/heads/"+defaultBranch)
			if err != nil {
				t.Fatalf("GetRef after init: %v", err)
			}
		} else {
			t.Fatalf("GetRef: %v", err)
		}
	}
	baseSHA := ref.Object.GetSHA()

	// Create a new branch
	branchName := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	refName := "refs/heads/" + branchName
	_, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref:    &refName,
		Object: &github.GitObject{SHA: &baseSHA},
	})
	if err != nil {
		t.Fatalf("CreateRef: %v", err)
	}
	t.Logf("Created branch: %s", branchName)

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		// Delete branch
		_, _ = client.Git.DeleteRef(ctx, owner, repo, "refs/heads/"+branchName)
		t.Logf("Deleted branch: %s", branchName)
	})

	// Create a file in the branch
	content := []byte(fmt.Sprintf("# E2E Test\n\nCreated at %s\n", time.Now().Format(time.RFC3339)))
	commitMsg := "E2E test commit"
	_, _, err = client.Repositories.CreateFile(ctx, owner, repo, "e2e-test.md", &github.RepositoryContentFileOptions{
		Message: &commitMsg,
		Content: content,
		Branch:  &branchName,
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	// Create PR using provider
	result, err := provider.CreatePR(ctx, PROptions{
		Title:  fmt.Sprintf("E2E Test PR %d", time.Now().Unix()),
		Body:   "Automated test PR",
		Head:   fmt.Sprintf("%s/%s:%s", owner, repo, branchName),
		Base:   defaultBranch,
		Draft:  true,
		TaskID: fmt.Sprintf("%s/%s#0", owner, repo),
	})
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}

	t.Logf("Created PR #%d: %s", result.Number, result.URL)

	if result.Number <= 0 {
		t.Errorf("PR number = %d, want > 0", result.Number)
	}
	if result.State != "draft" && result.State != "open" {
		t.Errorf("PR state = %q, want draft or open", result.State)
	}

	// Cleanup: close the PR
	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		state := "closed"
		_, _, _ = client.PullRequests.Edit(ctx, owner, repo, result.Number, &github.PullRequest{State: &state})
		t.Logf("Closed PR #%d", result.Number)
	})
}

func TestE2E_ListIssues(t *testing.T) {
	owner, repo, token := getE2EConfig(t)
	ctx := context.Background()

	client := newGitHubClient(token, "")

	// List open issues (just verify we can call the API)
	issues, _, err := client.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	})
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}

	t.Logf("Found %d issues/PRs in repo", len(issues))
}
