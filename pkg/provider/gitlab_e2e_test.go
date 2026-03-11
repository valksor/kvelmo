//go:build e2e

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// E2E tests for GitLab provider.
// Run with: go test -tags=e2e -v ./pkg/provider/... -run TestE2E_GitLab
//
// Required environment variables:
//   E2E_GITLAB_REPO  - Project in "group/project" format
//   GITLAB_TOKEN     - Personal access token with api scope

func getGitLabE2EConfig(t *testing.T) (project, token string) {
	t.Helper()

	project = os.Getenv("E2E_GITLAB_REPO")
	if project == "" {
		t.Skip("E2E_GITLAB_REPO not set")
	}

	token = os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = os.Getenv("E2E_GITLAB_TOKEN")
	}
	if token == "" {
		t.Skip("GITLAB_TOKEN or E2E_GITLAB_TOKEN not set")
	}

	return project, token
}

func TestE2E_GitLab_CreateAndCloseIssue(t *testing.T) {
	project, token := getGitLabE2EConfig(t)
	ctx := context.Background()

	provider, err := NewGitLabProvider(token)
	if err != nil {
		t.Fatalf("NewGitLabProvider: %v", err)
	}

	client, err := newGitLabClient(token, "")
	if err != nil {
		t.Fatalf("newGitLabClient: %v", err)
	}

	// Create issue
	title := fmt.Sprintf("E2E Test Issue %d", time.Now().Unix())
	description := "This is an automated test issue.\n\nIt will be closed automatically."
	labels := gitlab.LabelOptions{"test", "automated"}

	issue, _, err := client.Issues.CreateIssue(project, &gitlab.CreateIssueOptions{
		Title:       gitlab.Ptr(title),
		Description: gitlab.Ptr(description),
		Labels:      &labels,
	}, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	issueIID := issue.IID
	t.Logf("Created issue #%d: %s", issueIID, issue.WebURL)

	// Cleanup: close issue
	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			t.Logf("Skipping cleanup (E2E_SKIP_CLEANUP set)")
			return
		}
		_, _, _ = client.Issues.UpdateIssue(project, issueIID, &gitlab.UpdateIssueOptions{
			StateEvent: gitlab.Ptr("close"),
		}, gitlab.WithContext(ctx))
		t.Logf("Closed issue #%d", issueIID)
	})

	// Fetch via provider
	taskID := fmt.Sprintf("%s#%d", project, issueIID)
	task, err := provider.FetchTask(ctx, taskID)
	if err != nil {
		t.Fatalf("FetchTask: %v", err)
	}

	if task.Title != title {
		t.Errorf("Title = %q, want %q", task.Title, title)
	}
	if task.Description != description {
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
	if task.Metadata("gitlab_state") != "closed" {
		t.Errorf("State = %q, want closed", task.Metadata("gitlab_state"))
	}
}

func TestE2E_GitLab_CreateMR(t *testing.T) {
	project, token := getGitLabE2EConfig(t)
	ctx := context.Background()

	provider, err := NewGitLabProvider(token)
	if err != nil {
		t.Fatalf("NewGitLabProvider: %v", err)
	}

	client, err := newGitLabClient(token, "")
	if err != nil {
		t.Fatalf("newGitLabClient: %v", err)
	}

	// Get default branch
	proj, _, err := client.Projects.GetProject(project, nil, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	defaultBranch := proj.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Create a new branch
	branchName := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	_, _, err = client.Branches.CreateBranch(project, &gitlab.CreateBranchOptions{
		Branch: gitlab.Ptr(branchName),
		Ref:    gitlab.Ptr(defaultBranch),
	}, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	t.Logf("Created branch: %s", branchName)

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		_, _ = client.Branches.DeleteBranch(project, branchName, gitlab.WithContext(ctx))
		t.Logf("Deleted branch: %s", branchName)
	})

	// Create a file in the branch
	content := fmt.Sprintf("# E2E Test\n\nCreated at %s\n", time.Now().Format(time.RFC3339))
	_, _, err = client.RepositoryFiles.CreateFile(project, "e2e-test.md", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr(branchName),
		Content:       gitlab.Ptr(content),
		CommitMessage: gitlab.Ptr("E2E test commit"),
	}, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	// Create MR using provider
	result, err := provider.CreatePR(ctx, PROptions{
		Title:  fmt.Sprintf("E2E Test MR %d", time.Now().Unix()),
		Body:   "Automated test MR",
		Head:   fmt.Sprintf("%s:%s", project, branchName),
		Base:   defaultBranch,
		Draft:  true,
		TaskID: fmt.Sprintf("%s#0", project),
	})
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}

	t.Logf("Created MR !%d: %s", result.Number, result.URL)

	if result.Number <= 0 {
		t.Errorf("MR number = %d, want > 0", result.Number)
	}
	if result.State != "draft" && result.State != "opened" {
		t.Errorf("MR state = %q, want draft or opened", result.State)
	}

	// Verify we can fetch it
	mrTaskID := fmt.Sprintf("%s!%d", project, result.Number)
	task, err := provider.FetchTask(ctx, mrTaskID)
	if err != nil {
		t.Fatalf("FetchTask MR: %v", err)
	}
	if !strings.Contains(task.Title, "E2E Test MR") {
		t.Errorf("MR title = %q, want to contain 'E2E Test MR'", task.Title)
	}

	// Cleanup: close MR
	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		_, _, _ = client.MergeRequests.UpdateMergeRequest(project, int64(result.Number), &gitlab.UpdateMergeRequestOptions{
			StateEvent: gitlab.Ptr("close"),
		}, gitlab.WithContext(ctx))
		t.Logf("Closed MR !%d", result.Number)
	})
}

func TestE2E_GitLab_IssueComments(t *testing.T) {
	project, token := getGitLabE2EConfig(t)
	ctx := context.Background()

	provider, err := NewGitLabProvider(token)
	if err != nil {
		t.Fatalf("NewGitLabProvider: %v", err)
	}

	client, err := newGitLabClient(token, "")
	if err != nil {
		t.Fatalf("newGitLabClient: %v", err)
	}

	// Create issue
	title := fmt.Sprintf("E2E Comment Test %d", time.Now().Unix())
	issue, _, err := client.Issues.CreateIssue(project, &gitlab.CreateIssueOptions{
		Title: gitlab.Ptr(title),
	}, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	issueIID := issue.IID

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			return
		}
		_, _, _ = client.Issues.UpdateIssue(project, issueIID, &gitlab.UpdateIssueOptions{
			StateEvent: gitlab.Ptr("close"),
		}, gitlab.WithContext(ctx))
	})

	// Add comment via provider
	taskID := fmt.Sprintf("%s#%d", project, issueIID)
	commentBody := "This is an automated test comment."
	err = provider.AddComment(ctx, taskID, commentBody)
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	// List notes to verify
	notes, _, err := client.Notes.ListIssueNotes(project, issueIID, nil, gitlab.WithContext(ctx))
	if err != nil {
		t.Fatalf("ListIssueNotes: %v", err)
	}

	if len(notes) < 1 {
		t.Errorf("Expected at least 1 note, got %d", len(notes))
	}

	found := false
	for _, n := range notes {
		if n.Body == commentBody {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Comment not found in notes")
	}
}
