package hierarchy

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-toolkit/workunit"
)

func TestSubtaskPattern_IsSubtaskID(t *testing.T) {
	tests := []struct {
		name    string
		pattern SubtaskPattern
		id      string
		want    bool
	}{
		// Contains pattern
		{
			name:    "github subtask",
			pattern: GitHubSubtaskPattern,
			id:      "owner/repo#123-task-1",
			want:    true,
		},
		{
			name:    "github regular issue",
			pattern: GitHubSubtaskPattern,
			id:      "owner/repo#123",
			want:    false,
		},
		{
			name:    "gitlab subtask",
			pattern: GitLabSubtaskPattern,
			id:      "group/project#456-task-2",
			want:    true,
		},
		{
			name:    "bitbucket subtask",
			pattern: BitbucketSubtaskPattern,
			id:      "project:123:task-1",
			want:    true,
		},
		{
			name:    "bitbucket regular issue",
			pattern: BitbucketSubtaskPattern,
			id:      "project:123",
			want:    false,
		},
		{
			name:    "trello checklist item",
			pattern: TrelloSubtaskPattern,
			id:      "board/card/checkitem/item123",
			want:    true,
		},
		{
			name:    "trello regular card",
			pattern: TrelloSubtaskPattern,
			id:      "board/card/abc123",
			want:    false,
		},
		// Prefix pattern
		{
			name:    "prefix match",
			pattern: SubtaskPattern{Prefix: "subtask:"},
			id:      "subtask:123",
			want:    true,
		},
		{
			name:    "prefix no match",
			pattern: SubtaskPattern{Prefix: "subtask:"},
			id:      "task:123",
			want:    false,
		},
		// Empty pattern
		{
			name:    "empty pattern",
			pattern: SubtaskPattern{},
			id:      "anything",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pattern.IsSubtaskID(tt.id)
			if got != tt.want {
				t.Errorf("IsSubtaskID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestExtractParentID(t *testing.T) {
	tests := []struct {
		name       string
		subtaskID  string
		pattern    SubtaskPattern
		wantParent string
		wantOK     bool
	}{
		{
			name:       "github subtask",
			subtaskID:  "owner/repo#123-task-1",
			pattern:    GitHubSubtaskPattern,
			wantParent: "owner/repo#123",
			wantOK:     true,
		},
		{
			name:       "github subtask with multiple tasks",
			subtaskID:  "owner/repo#456-task-5",
			pattern:    GitHubSubtaskPattern,
			wantParent: "owner/repo#456",
			wantOK:     true,
		},
		{
			name:       "gitlab subtask",
			subtaskID:  "group/project#789-task-3",
			pattern:    GitLabSubtaskPattern,
			wantParent: "group/project#789",
			wantOK:     true,
		},
		{
			name:       "bitbucket subtask",
			subtaskID:  "project:100:task-2",
			pattern:    BitbucketSubtaskPattern,
			wantParent: "project:100",
			wantOK:     true,
		},
		{
			name:       "not a subtask",
			subtaskID:  "owner/repo#123",
			pattern:    GitHubSubtaskPattern,
			wantParent: "owner/repo#123",
			wantOK:     false,
		},
		{
			name:       "prefix pattern - cannot extract",
			subtaskID:  "checkitem/xyz",
			pattern:    SubtaskPattern{Prefix: "checkitem/"},
			wantParent: "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotParent, gotOK := ExtractParentID(tt.subtaskID, tt.pattern)
			if gotParent != tt.wantParent {
				t.Errorf("ExtractParentID() parentID = %q, want %q", gotParent, tt.wantParent)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ExtractParentID() isSubtask = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestFetchParentByID(t *testing.T) {
	parentWorkUnit := &workunit.WorkUnit{
		ID:    "owner/repo#123",
		Title: "Parent Issue",
	}

	tests := []struct {
		name      string
		subtaskID string
		pattern   SubtaskPattern
		fetcher   FetcherFunc
		wantErr   error
		wantID    string
	}{
		{
			name:      "successful fetch",
			subtaskID: "owner/repo#123-task-1",
			pattern:   GitHubSubtaskPattern,
			fetcher: func(_ context.Context, id string) (*workunit.WorkUnit, error) {
				if id == "owner/repo#123" {
					return parentWorkUnit, nil
				}

				return nil, errors.New("not found")
			},
			wantErr: nil,
			wantID:  "owner/repo#123",
		},
		{
			name:      "not a subtask",
			subtaskID: "owner/repo#123",
			pattern:   GitHubSubtaskPattern,
			fetcher: func(_ context.Context, _ string) (*workunit.WorkUnit, error) {
				t.Error("fetcher should not be called for non-subtask")

				return nil, errors.New("should not be called")
			},
			wantErr: ErrNotASubtask,
			wantID:  "",
		},
		{
			name:      "fetcher error",
			subtaskID: "owner/repo#456-task-1",
			pattern:   GitHubSubtaskPattern,
			fetcher: func(_ context.Context, _ string) (*workunit.WorkUnit, error) {
				return nil, errors.New("API error")
			},
			wantErr: nil, // Generic error, not our sentinel
			wantID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FetchParentByID(context.Background(), tt.subtaskID, tt.pattern, tt.fetcher)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FetchParentByID() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if tt.wantID == "" && err != nil && !errors.Is(err, ErrNotASubtask) {
				// Expected a fetcher error
				return
			}

			if err != nil {
				t.Fatalf("FetchParentByID() unexpected error = %v", err)
			}

			if result.ID != tt.wantID {
				t.Errorf("FetchParentByID() result.ID = %q, want %q", result.ID, tt.wantID)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	// Verify sentinel errors are distinct
	if errors.Is(ErrNotASubtask, ErrNoParent) {
		t.Error("ErrNotASubtask and ErrNoParent should be distinct")
	}

	// Verify error messages
	if ErrNotASubtask.Error() != "not a subtask" {
		t.Errorf("ErrNotASubtask.Error() = %q, want %q", ErrNotASubtask.Error(), "not a subtask")
	}
	if ErrNoParent.Error() != "no parent found" {
		t.Errorf("ErrNoParent.Error() = %q, want %q", ErrNoParent.Error(), "no parent found")
	}
}

func TestPredefinedPatterns(t *testing.T) {
	// Verify predefined patterns are properly configured
	patterns := []struct {
		name    string
		pattern SubtaskPattern
		example string
	}{
		{"GitHubSubtaskPattern", GitHubSubtaskPattern, "owner/repo#123-task-1"},
		{"GitLabSubtaskPattern", GitLabSubtaskPattern, "project#456-task-2"},
		{"BitbucketSubtaskPattern", BitbucketSubtaskPattern, "project:789:task-3"},
		{"TrelloSubtaskPattern", TrelloSubtaskPattern, "board/card/checkitem/item"},
	}

	for _, p := range patterns {
		t.Run(p.name, func(t *testing.T) {
			if !p.pattern.IsSubtaskID(p.example) {
				t.Errorf("%s.IsSubtaskID(%q) = false, want true", p.name, p.example)
			}
		})
	}
}
