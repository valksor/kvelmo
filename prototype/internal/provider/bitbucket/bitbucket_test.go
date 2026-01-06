package bitbucket

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Reference
		wantErr bool
	}{
		{
			name:  "issue ID only",
			input: "123",
			want: &Reference{
				IssueID:    123,
				IsExplicit: false,
			},
		},
		{
			name:  "issue ID with hash",
			input: "#456",
			want: &Reference{
				IssueID:    456,
				IsExplicit: false,
			},
		},
		{
			name:  "workspace/repo#issue",
			input: "myworkspace/myrepo#789",
			want: &Reference{
				Workspace:  "myworkspace",
				RepoSlug:   "myrepo",
				IssueID:    789,
				IsExplicit: true,
			},
		},
		{
			name:  "full URL",
			input: "https://bitbucket.org/myworkspace/myrepo/issues/42",
			want: &Reference{
				Workspace:  "myworkspace",
				RepoSlug:   "myrepo",
				IssueID:    42,
				IsExplicit: true,
			},
		},
		{
			name:  "URL without https",
			input: "bitbucket.org/myworkspace/myrepo/issues/100",
			want: &Reference{
				Workspace:  "myworkspace",
				RepoSlug:   "myrepo",
				IssueID:    100,
				IsExplicit: true,
			},
		},
		{
			name:    "invalid input",
			input:   "not-valid-reference",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid URL path",
			input:   "https://bitbucket.org/workspace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseReference(%q) expected error, got nil", tt.input)
				}

				return
			}

			if err != nil {
				t.Errorf("ParseReference(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got.IssueID != tt.want.IssueID {
				t.Errorf("ParseReference(%q).IssueID = %d, want %d", tt.input, got.IssueID, tt.want.IssueID)
			}
			if got.Workspace != tt.want.Workspace {
				t.Errorf("ParseReference(%q).Workspace = %q, want %q", tt.input, got.Workspace, tt.want.Workspace)
			}
			if got.RepoSlug != tt.want.RepoSlug {
				t.Errorf("ParseReference(%q).RepoSlug = %q, want %q", tt.input, got.RepoSlug, tt.want.RepoSlug)
			}
			if got.IsExplicit != tt.want.IsExplicit {
				t.Errorf("ParseReference(%q).IsExplicit = %v, want %v", tt.input, got.IsExplicit, tt.want.IsExplicit)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	info := Info()

	if info.Name != ProviderName {
		t.Errorf("Info().Name = %q, want %q", info.Name, ProviderName)
	}

	expectedSchemes := []string{"bitbucket", "bb"}
	if len(info.Schemes) != len(expectedSchemes) {
		t.Errorf("Info().Schemes = %v, want %v", info.Schemes, expectedSchemes)
	}
	for i, scheme := range expectedSchemes {
		if info.Schemes[i] != scheme {
			t.Errorf("Info().Schemes[%d] = %q, want %q", i, info.Schemes[i], scheme)
		}
	}

	// Check capabilities
	if !info.Capabilities.Has(provider.CapRead) {
		t.Error("Info().Capabilities should have CapRead")
	}
	if !info.Capabilities.Has(provider.CapList) {
		t.Error("Info().Capabilities should have CapList")
	}
	if !info.Capabilities.Has(provider.CapFetchComments) {
		t.Error("Info().Capabilities should have CapFetchComments")
	}
	if !info.Capabilities.Has(provider.CapComment) {
		t.Error("Info().Capabilities should have CapComment")
	}
	if !info.Capabilities.Has(provider.CapUpdateStatus) {
		t.Error("Info().Capabilities should have CapUpdateStatus")
	}
	if !info.Capabilities.Has(provider.CapSnapshot) {
		t.Error("Info().Capabilities should have CapSnapshot")
	}
	if !info.Capabilities.Has(provider.CapCreatePR) {
		t.Error("Info().Capabilities should have CapCreatePR")
	}
}

func TestGeneratePRTitle(t *testing.T) {
	tests := []struct {
		name     string
		taskWork *storage.TaskWork
		want     string
	}{
		{
			name:     "nil task work",
			taskWork: nil,
			want:     "Implementation",
		},
		{
			name:     "empty metadata",
			taskWork: &storage.TaskWork{},
			want:     "Implementation",
		},
		{
			name: "title only",
			taskWork: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					Title: "Add new feature",
				},
			},
			want: "Add new feature",
		},
		{
			name: "external key only",
			taskWork: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ExternalKey: "42",
				},
			},
			want: "[#42] Implementation",
		},
		{
			name: "title and external key",
			taskWork: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					Title:       "Fix login bug",
					ExternalKey: "123",
				},
			},
			want: "[#123] Fix login bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePRTitle(tt.taskWork)
			if got != tt.want {
				t.Errorf("GeneratePRTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeneratePRBody(t *testing.T) {
	tests := []struct {
		name     string
		taskWork *storage.TaskWork
		specs    []*storage.Specification
		diffStat string
		contains []string
	}{
		{
			name:     "nil task work",
			taskWork: nil,
			specs:    nil,
			diffStat: "",
			contains: []string{"## Summary", "## Test Plan"},
		},
		{
			name: "with title",
			taskWork: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					Title: "Add feature X",
				},
			},
			specs:    nil,
			diffStat: "",
			contains: []string{"## Summary", "Implementation for: Add feature X"},
		},
		{
			name: "bitbucket issue link",
			taskWork: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ExternalKey: "42",
				},
				Source: storage.SourceInfo{
					Type: ProviderName,
				},
			},
			specs:    nil,
			diffStat: "",
			contains: []string{"Closes #42"},
		},
		{
			name:     "with specs",
			taskWork: nil,
			specs: []*storage.Specification{
				{Title: "Spec 1", Content: "Details about spec 1"},
				{Title: "Spec 2", Content: "Details about spec 2"},
			},
			diffStat: "",
			contains: []string{"## Implementation Details", "### Spec 1", "### Spec 2"},
		},
		{
			name:     "with diff stat",
			taskWork: nil,
			specs:    nil,
			diffStat: " 3 files changed, 50 insertions(+), 10 deletions(-)",
			contains: []string{"## Changes", "3 files changed"},
		},
		{
			name:     "has footer",
			taskWork: nil,
			specs:    nil,
			diffStat: "",
			contains: []string{"Generated by [Mehrhof]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePRBody(tt.taskWork, tt.specs, tt.diffStat)
			for _, want := range tt.contains {
				if !containsString(got, want) {
					t.Errorf("GeneratePRBody() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestMapBitbucketState(t *testing.T) {
	tests := []struct {
		input string
		want  provider.Status
	}{
		{"new", provider.StatusOpen},
		{"open", provider.StatusOpen},
		{"resolved", provider.StatusClosed},
		{"on hold", provider.StatusOpen},
		{"invalid", provider.StatusClosed},
		{"duplicate", provider.StatusClosed},
		{"wontfix", provider.StatusClosed},
		{"closed", provider.StatusClosed},
		{"unknown", provider.StatusOpen},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapBitbucketState(tt.input)
			if got != tt.want {
				t.Errorf("mapBitbucketState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapBitbucketPriority(t *testing.T) {
	tests := []struct {
		input string
		want  provider.Priority
	}{
		{"trivial", provider.PriorityLow},
		{"minor", provider.PriorityNormal},
		{"major", provider.PriorityHigh},
		{"critical", provider.PriorityCritical},
		{"blocker", provider.PriorityCritical},
		{"unknown", provider.PriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapBitbucketPriority(tt.input)
			if got != tt.want {
				t.Errorf("mapBitbucketPriority(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Match tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderMatch(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		input string
		want  bool
	}{
		{"bitbucket:123", true},
		{"bb:456", true},
		{"bitbucket:workspace/repo#789", true},
		{"bb:myworkspace/myrepo#100", true},
		{"123", false},
		{"#456", false},
		{"workspace/repo#789", false},
		{"https://bitbucket.org/workspace/repo/issues/42", false},
		{"", false},
		{"github:123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := p.Match(tt.input)
			if got != tt.want {
				t.Errorf("Provider.Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Parse tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		workspace   string
		repoSlug    string
		want        string
		errContains string
		wantErr     bool
	}{
		{
			name:      "explicit workspace/repo",
			input:     "myworkspace/myrepo#42",
			workspace: "",
			repoSlug:  "",
			want:      "myworkspace/myrepo#42",
		},
		{
			name:      "uses configured workspace/repo",
			input:     "123",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			want:      "myworkspace/myrepo#123",
		},
		{
			name:        "error when workspace/repo not configured",
			input:       "123",
			workspace:   "",
			repoSlug:    "",
			errContains: "not configured",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				config: &Config{
					Workspace: tt.workspace,
					RepoSlug:  tt.repoSlug,
				},
			}

			got, err := p.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Provider.Parse(%q) expected error, got nil", tt.input)
				}
				if tt.errContains != "" && err != nil {
					if !containsString(err.Error(), tt.errContains) {
						t.Errorf("Provider.Parse(%q) error = %v, want to contain %q", tt.input, err, tt.errContains)
					}
				}

				return
			}

			if err != nil {
				t.Errorf("Provider.Parse(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got != tt.want {
				t.Errorf("Provider.Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapBitbucketKind tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapBitbucketKind(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"bug", "fix"},
		{"enhancement", "feature"},
		{"proposal", "feature"},
		{"task", "task"},
		{"unknown", "issue"},
		{"", "issue"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := mapBitbucketKind(tt.kind)
			if got != tt.want {
				t.Errorf("mapBitbucketKind(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractLinkedIssues tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractLinkedIssues(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []int
	}{
		{
			name: "single fix",
			text: "fixes #123",
			want: []int{123},
		},
		{
			name: "single close",
			text: "closes #456",
			want: []int{456},
		},
		{
			name: "single resolve",
			text: "resolves #789",
			want: []int{789},
		},
		{
			name: "multiple issues",
			text: "fixes #123 and closes #456",
			want: []int{123, 456},
		},
		{
			name: "case insensitive",
			text: "FIXES #123",
			want: []int{123},
		},
		{
			name: "no issues",
			text: "just regular text",
			want: nil,
		},
		{
			name: "deduplicates",
			text: "fixes #123 and also fixes #123",
			want: []int{123},
		},
		{
			name: "mixed keywords",
			text: "fixes #1, closes #2, resolves #3",
			want: []int{1, 2, 3},
		},
		{
			name: "empty string",
			text: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLinkedIssues(tt.text)
			if !slicesEqual(got, tt.want) {
				t.Errorf("ExtractLinkedIssues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractImageURLs tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractImageURLs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "png image",
			text: "![alt](https://example.com/image.png)",
			want: []string{"https://example.com/image.png"},
		},
		{
			name: "jpg image",
			text: "![alt](https://example.com/photo.jpg)",
			want: []string{"https://example.com/photo.jpg"},
		},
		{
			name: "jpeg image",
			text: "![alt](https://example.com/photo.jpeg)",
			want: []string{"https://example.com/photo.jpeg"},
		},
		{
			name: "gif image",
			text: "![alt](https://example.com/anim.gif)",
			want: []string{"https://example.com/anim.gif"},
		},
		{
			name: "webp image",
			text: "![alt](https://example.com/pic.webp)",
			want: []string{"https://example.com/pic.webp"},
		},
		{
			name: "bitbucket image",
			text: "![alt](https://bitbucket.org/account/repo/raw/HEAD/image.png)",
			want: []string{"https://bitbucket.org/account/repo/raw/HEAD/image.png"},
		},
		{
			name: "multiple images",
			text: "![a](img1.png) and ![b](img2.jpg)",
			want: []string{"img1.png", "img2.jpg"},
		},
		{
			name: "non-image markdown link ignored",
			text: "[link](https://example.com/file.pdf)",
			want: nil,
		},
		{
			name: "no images",
			text: "just regular text",
			want: nil,
		},
		{
			name: "uppercase extension",
			text: "![alt](https://example.com/image.PNG)",
			want: []string{"https://example.com/image.PNG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractImageURLs(tt.text)
			if !slicesEqualStr(got, tt.want) {
				t.Errorf("ExtractImageURLs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func slicesEqualStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ──────────────────────────────────────────────────────────────────────────────
// mapAssignee tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapAssignee(t *testing.T) {
	tests := []struct {
		name     string
		assignee *User
		want     []string // Names from result
		wantLen  int
	}{
		{
			name:     "nil assignee",
			assignee: nil,
			wantLen:  0,
		},
		{
			name: "assignee with display name",
			assignee: &User{
				UUID:        "uuid-123",
				DisplayName: "John Doe",
				Username:    "johndoe",
			},
			want:    []string{"John Doe"},
			wantLen: 1,
		},
		{
			name: "assignee with username only",
			assignee: &User{
				UUID:     "uuid-456",
				Username: "janedoe",
			},
			want:    []string{"janedoe"},
			wantLen: 1,
		},
		{
			name: "assignee with empty display name falls back to username",
			assignee: &User{
				UUID:        "uuid-789",
				DisplayName: "",
				Username:    "fallback",
			},
			want:    []string{"fallback"},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAssignee(tt.assignee)
			if got == nil {
				if tt.wantLen > 0 {
					t.Errorf("mapAssignee() = nil, want %d items", tt.wantLen)
				}

				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("mapAssignee() len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 && got[0].Name != tt.want[0] {
				t.Errorf("mapAssignee()[0].Name = %q, want %q", got[0].Name, tt.want[0])
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapComments tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapComments(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		comments     []Comment
		wantLen      int
		contains     string // Check if this string is in first comment body
		containsName string // Check if this string is the author name
	}{
		{
			name:     "nil comments",
			comments: nil,
			wantLen:  0,
		},
		{
			name:     "empty comments",
			comments: []Comment{},
			wantLen:  0,
		},
		{
			name: "single comment",
			comments: []Comment{
				{
					ID: 1,
					Content: &Content{
						Raw: "This is a comment",
					},
					User: &User{
						UUID:        "uuid-1",
						DisplayName: "Test User",
					},
					CreatedOn: baseTime,
					UpdatedOn: baseTime,
				},
			},
			wantLen:  1,
			contains: "This is a comment",
		},
		{
			name: "comment with username preferred",
			comments: []Comment{
				{
					ID: 1,
					Content: &Content{
						Raw: "Comment body",
					},
					User: &User{
						UUID:        "uuid-1",
						DisplayName: "Display Name",
						Username:    "username",
					},
					CreatedOn: baseTime,
					UpdatedOn: baseTime,
				},
			},
			wantLen:      1,
			containsName: "username",
		},
		{
			name: "comment with nil user",
			comments: []Comment{
				{
					ID:        1,
					Content:   &Content{Raw: "No user"},
					CreatedOn: baseTime,
					UpdatedOn: baseTime,
				},
			},
			wantLen: 1,
		},
		{
			name: "comment with nil content",
			comments: []Comment{
				{
					ID:        1,
					Content:   nil,
					CreatedOn: baseTime,
					UpdatedOn: baseTime,
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapComments(tt.comments)
			if len(got) != tt.wantLen {
				t.Errorf("mapComments() len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if tt.contains != "" && !containsString(got[0].Body, tt.contains) {
					t.Errorf("mapComments()[0].Body missing %q", tt.contains)
				}
				if tt.containsName != "" && got[0].Author.Name != tt.containsName {
					t.Errorf("mapComments()[0].Author.Name = %q, want %q", got[0].Author.Name, tt.containsName)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// formatIssueMarkdown tests
// ──────────────────────────────────────────────────────────────────────────────

func TestFormatIssueMarkdown(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		issue    *Issue
		contains []string
	}{
		{
			name: "basic issue",
			issue: &Issue{
				ID:       42,
				Title:    "Test Issue",
				State:    "open",
				Kind:     "bug",
				Priority: "major",
				Content: &Content{
					Raw: "Issue description",
				},
				Reporter: &User{
					DisplayName: "Reporter Name",
				},
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
				Links: Links{
					HTML: &Link{Href: "https://bitbucket.org/test/repo/issues/42"},
				},
			},
			contains: []string{"# #42: Test Issue", "**State:** open", "**Priority:** major", "**Kind:** bug", "Issue description"},
		},
		{
			name: "issue with assignee",
			issue: &Issue{
				ID:    42,
				Title: "Test",
				State: "open",
				Assignee: &User{
					DisplayName: "Assignee Name",
				},
				Content:   &Content{Raw: "Desc"},
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
			},
			contains: []string{"**Assignee:** Assignee Name"},
		},
		{
			name: "issue with component",
			issue: &Issue{
				ID:        42,
				Title:     "Test",
				State:     "open",
				Component: &Component{Name: "Backend"},
				Content:   &Content{Raw: "Desc"},
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
			},
			contains: []string{"**Component:** Backend"},
		},
		{
			name: "issue with milestone",
			issue: &Issue{
				ID:        42,
				Title:     "Test",
				State:     "open",
				Milestone: &Milestone{Name: "Sprint 1"},
				Content:   &Content{Raw: "Desc"},
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
			},
			contains: []string{"**Milestone:** Sprint 1"},
		},
		{
			name: "issue with nil content",
			issue: &Issue{
				ID:        42,
				Title:     "Test",
				State:     "open",
				Content:   nil,
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
			},
			contains: []string{"*No description*"},
		},
		{
			name: "issue with URL",
			issue: &Issue{
				ID:      42,
				Title:   "Test",
				State:   "open",
				Content: &Content{Raw: "Desc"},
				Links: Links{
					HTML: &Link{Href: "https://bitbucket.org/workspace/repo/issues/42"},
				},
				CreatedOn: baseTime,
				UpdatedOn: baseTime,
			},
			contains: []string{"**URL:** https://bitbucket.org/workspace/repo/issues/42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatIssueMarkdown(tt.issue)
			for _, want := range tt.contains {
				if !containsString(got, want) {
					t.Errorf("formatIssueMarkdown() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// formatCommentsMarkdown tests
// ──────────────────────────────────────────────────────────────────────────────

func TestFormatCommentsMarkdown(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		comments []Comment
		contains []string
	}{
		{
			name:     "no comments",
			comments: []Comment{},
			contains: []string{"# Comments"},
		},
		{
			name: "single comment",
			comments: []Comment{
				{
					ID: 1,
					Content: &Content{
						Raw: "This is a comment",
					},
					User: &User{
						DisplayName: "Test User",
					},
					CreatedOn: baseTime,
				},
			},
			contains: []string{"# Comments", "## Comment by Test User", "This is a comment"},
		},
		{
			name: "comment with display name preferred",
			comments: []Comment{
				{
					ID: 1,
					Content: &Content{
						Raw: "Body",
					},
					User: &User{
						DisplayName: "Display",
						Username:    "username",
					},
					CreatedOn: baseTime,
				},
			},
			contains: []string{"## Comment by Display"},
		},
		{
			name: "comment with no display name uses username",
			comments: []Comment{
				{
					ID: 1,
					Content: &Content{
						Raw: "Body",
					},
					User: &User{
						Username: "justusername",
					},
					CreatedOn: baseTime,
				},
			},
			contains: []string{"## Comment by justusername"},
		},
		{
			name: "comment with nil user",
			comments: []Comment{
				{
					ID:        1,
					Content:   &Content{Raw: "Body"},
					User:      nil,
					CreatedOn: baseTime,
				},
			},
			contains: []string{"## Comment by Unknown"},
		},
		{
			name: "multiple comments",
			comments: []Comment{
				{
					ID:        1,
					Content:   &Content{Raw: "First"},
					User:      &User{DisplayName: "User1"},
					CreatedOn: baseTime,
				},
				{
					ID:        2,
					Content:   &Content{Raw: "Second"},
					User:      &User{DisplayName: "User2"},
					CreatedOn: baseTime,
				},
			},
			contains: []string{"## Comment by User1", "First", "## Comment by User2", "Second", "---"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommentsMarkdown(tt.comments)
			for _, want := range tt.contains {
				if !containsString(got, want) {
					t.Errorf("formatCommentsMarkdown() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// wrapAPIError tests
// ──────────────────────────────────────────────────────────────────────────────

func TestWrapAPIError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantType    string // Error type substring to check
		wantContain string // Error message substring
	}{
		{
			name:        "nil error",
			err:         nil,
			wantContain: "",
		},
		{
			name:        "401 unauthorized",
			err:         httpclient.NewHTTPError(http.StatusUnauthorized, "bad credentials"),
			wantType:    "ErrUnauthorized",
			wantContain: "unauthorized",
		},
		{
			name:        "403 forbidden",
			err:         httpclient.NewHTTPError(http.StatusForbidden, "access denied"),
			wantType:    "ErrUnauthorized",
			wantContain: "unauthorized",
		},
		{
			name:        "429 rate limited",
			err:         httpclient.NewHTTPError(http.StatusTooManyRequests, "too many requests"),
			wantType:    "ErrRateLimited",
			wantContain: "rate limit",
		},
		{
			name:        "404 not found",
			err:         httpclient.NewHTTPError(http.StatusNotFound, "issue not found"),
			wantType:    "ErrIssueNotFound",
			wantContain: "not found",
		},
		{
			name:        "network error",
			err:         &net.DNSError{Err: "lookup failed"},
			wantType:    "ErrNetworkError",
			wantContain: "network error",
		},
		{
			name:        "unknown error",
			err:         errors.New("some other error"),
			wantContain: "some other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapAPIError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("wrapAPIError(nil) = %v, want nil", got)
				}

				return
			}
			if got == nil {
				t.Errorf("wrapAPIError(%v) = nil, want error", tt.err)

				return
			}
			if tt.wantType != "" && !containsString(got.Error(), tt.wantContain) {
				t.Errorf("wrapAPIError() error = %v, want to contain %q", got, tt.wantContain)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ResolveCredentials tests
// ──────────────────────────────────────────────────────────────────────────────

func TestResolveCredentials(t *testing.T) {
	tests := []struct {
		name            string
		username        string
		appPassword     string
		wantErr         bool
		errContains     string
		wantUsername    string
		wantAppPassword string
	}{
		{
			name:            "valid credentials",
			username:        "testuser",
			appPassword:     "testpass",
			wantErr:         false,
			wantUsername:    "testuser",
			wantAppPassword: "testpass",
		},
		{
			name:        "empty username",
			username:    "",
			appPassword: "testpass",
			wantErr:     true,
			errContains: "username not configured",
		},
		{
			name:        "empty password",
			username:    "testuser",
			appPassword: "",
			wantErr:     true,
			errContains: "token not found",
		},
		{
			name:        "both empty",
			username:    "",
			appPassword: "",
			wantErr:     true,
			errContains: "username not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsername, gotPassword, err := ResolveCredentials(tt.username, tt.appPassword)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveCredentials() expected error, got nil")
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("ResolveCredentials() error = %v, want to contain %q", err, tt.errContains)
				}

				return
			}
			if err != nil {
				t.Errorf("ResolveCredentials() unexpected error: %v", err)
			}
			if gotUsername != tt.wantUsername {
				t.Errorf("ResolveCredentials() username = %q, want %q", gotUsername, tt.wantUsername)
			}
			if gotPassword != tt.wantAppPassword {
				t.Errorf("ResolveCredentials() password = %q, want %q", gotPassword, tt.wantAppPassword)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapProviderPriorityToBitbucket tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapProviderPriorityToBitbucket(t *testing.T) {
	tests := []struct {
		name     string
		priority provider.Priority
		want     string
	}{
		{
			name:     "critical priority",
			priority: provider.PriorityCritical,
			want:     "critical",
		},
		{
			name:     "high priority",
			priority: provider.PriorityHigh,
			want:     "major",
		},
		{
			name:     "normal priority",
			priority: provider.PriorityNormal,
			want:     "minor",
		},
		{
			name:     "low priority",
			priority: provider.PriorityLow,
			want:     "trivial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProviderPriorityToBitbucket(tt.priority)
			if got != tt.want {
				t.Errorf("mapProviderPriorityToBitbucket(%v) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Reference.String tests
// ──────────────────────────────────────────────────────────────────────────────

func TestReferenceString(t *testing.T) {
	tests := []struct {
		name string
		ref  *Reference
		want string
	}{
		{
			name: "full reference",
			ref: &Reference{
				Workspace:  "myworkspace",
				RepoSlug:   "myrepo",
				IssueID:    42,
				IsExplicit: true,
			},
			want: "myworkspace/myrepo#42",
		},
		{
			name: "issue ID only",
			ref: &Reference{
				IssueID:    123,
				IsExplicit: false,
			},
			want: "123",
		},
		{
			name: "workspace only",
			ref: &Reference{
				Workspace: "workspace",
				IssueID:   1,
			},
			want: "1",
		},
		{
			name: "repo only",
			ref: &Reference{
				RepoSlug: "repo",
				IssueID:  1,
			},
			want: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("Reference.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		(len(haystack) > 0 && len(needle) > 0 && findString(haystack, needle)))
}

func findString(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
