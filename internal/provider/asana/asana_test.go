package asana

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// ──────────────────────────────────────────────────────────────────────────────
// Compile-time interface compliance checks
// ──────────────────────────────────────────────────────────────────────────────

var (
	_ provider.Reader         = (*Provider)(nil)
	_ provider.Identifier     = (*Provider)(nil)
	_ provider.Lister         = (*Provider)(nil)
	_ provider.CommentFetcher = (*Provider)(nil)
	_ provider.Commenter      = (*Provider)(nil)
	_ provider.StatusUpdater  = (*Provider)(nil)
	_ provider.Snapshotter    = (*Provider)(nil)
	_ provider.SubtaskFetcher = (*Provider)(nil)
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Reference
		wantErr bool
	}{
		{
			name:  "task GID only",
			input: "1234567890123456",
			want: &Reference{
				TaskGID:    "1234567890123456",
				IsExplicit: false,
			},
		},
		{
			name:  "project/task format",
			input: "1111111111111111/2222222222222222",
			want: &Reference{
				ProjectGID: "1111111111111111",
				TaskGID:    "2222222222222222",
				IsExplicit: true,
			},
		},
		{
			name:  "full Asana URL",
			input: "https://app.asana.com/0/1234567890123456/9876543210987654",
			want: &Reference{
				ProjectGID: "1234567890123456",
				TaskGID:    "9876543210987654",
				IsExplicit: true,
			},
		},
		{
			name:  "Asana URL without project (0 placeholder)",
			input: "https://app.asana.com/0/0/9876543210987654",
			want: &Reference{
				ProjectGID: "",
				TaskGID:    "9876543210987654",
				IsExplicit: false,
			},
		},
		{
			name:  "URL with /f suffix",
			input: "https://app.asana.com/0/1234567890123456/9876543210987654/f",
			want: &Reference{
				ProjectGID: "1234567890123456",
				TaskGID:    "9876543210987654",
				IsExplicit: true,
			},
		},
		{
			name:  "URL without https",
			input: "app.asana.com/0/1234567890123456/9876543210987654",
			want: &Reference{
				ProjectGID: "1234567890123456",
				TaskGID:    "9876543210987654",
				IsExplicit: true,
			},
		},
		{
			name:    "invalid input - too short",
			input:   "123456",
			wantErr: true,
		},
		{
			name:    "invalid input - non-numeric",
			input:   "abc-task",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
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

			if got.TaskGID != tt.want.TaskGID {
				t.Errorf("ParseReference(%q).TaskGID = %q, want %q", tt.input, got.TaskGID, tt.want.TaskGID)
			}
			if got.ProjectGID != tt.want.ProjectGID {
				t.Errorf("ParseReference(%q).ProjectGID = %q, want %q", tt.input, got.ProjectGID, tt.want.ProjectGID)
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

	expectedSchemes := []string{"asana", "as"}
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
}

func TestMapAsanaStatus(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want provider.Status
	}{
		{
			name: "completed task",
			task: &Task{Completed: true},
			want: provider.StatusClosed,
		},
		{
			name: "open task",
			task: &Task{Completed: false},
			want: provider.StatusOpen,
		},
		{
			name: "task in Done section",
			task: &Task{
				Completed: false,
				Memberships: []Membership{
					{Section: &Section{Name: "Done"}},
				},
			},
			want: provider.StatusClosed,
		},
		{
			name: "task in In Progress section",
			task: &Task{
				Completed: false,
				Memberships: []Membership{
					{Section: &Section{Name: "In Progress"}},
				},
			},
			want: provider.StatusInProgress,
		},
		{
			name: "task in Review section",
			task: &Task{
				Completed: false,
				Memberships: []Membership{
					{Section: &Section{Name: "Code Review"}},
				},
			},
			want: provider.StatusReview,
		},
		{
			name: "approval task pending",
			task: &Task{
				ResourceSubtype: "approval",
				ApprovalStatus:  "pending",
			},
			want: provider.StatusInProgress,
		},
		{
			name: "approval task approved",
			task: &Task{
				Completed:       true,
				ResourceSubtype: "approval",
				ApprovalStatus:  "approved",
			},
			want: provider.StatusClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAsanaStatus(tt.task)
			if got != tt.want {
				t.Errorf("mapAsanaStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapTaskType(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want string
	}{
		{
			name: "regular task",
			task: &Task{},
			want: "task",
		},
		{
			name: "bug tag",
			task: &Task{
				Tags: []Tag{{Name: "Bug"}},
			},
			want: "fix",
		},
		{
			name: "feature tag",
			task: &Task{
				Tags: []Tag{{Name: "Feature"}},
			},
			want: "feature",
		},
		{
			name: "milestone subtype",
			task: &Task{
				ResourceSubtype: "milestone",
			},
			want: "milestone",
		},
		{
			name: "approval subtype",
			task: &Task{
				ResourceSubtype: "approval",
			},
			want: "approval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTaskType(tt.task)
			if got != tt.want {
				t.Errorf("mapTaskType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTaskGIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single task URL",
			input: "Check out https://app.asana.com/0/123456/789012",
			want:  []string{"789012"},
		},
		{
			name:  "multiple task URLs",
			input: "Related: app.asana.com/0/123/456 and app.asana.com/0/789/012",
			want:  []string{"456", "012"},
		},
		{
			name:  "no task URLs",
			input: "This is just regular text",
			want:  nil,
		},
		{
			name:  "duplicate task URLs",
			input: "See app.asana.com/0/123/456 and app.asana.com/0/789/456",
			want:  []string{"456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTaskGIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractTaskGIDs() returned %d GIDs, want %d", len(got), len(tt.want))

				return
			}
			for i, gid := range got {
				if gid != tt.want[i] {
					t.Errorf("ExtractTaskGIDs()[%d] = %q, want %q", i, gid, tt.want[i])
				}
			}
		})
	}
}

func TestReferenceString(t *testing.T) {
	tests := []struct {
		ref  Reference
		want string
	}{
		{
			ref:  Reference{TaskGID: "123456"},
			want: "123456",
		},
		{
			ref:  Reference{TaskGID: "789012", ProjectGID: "123456"},
			want: "123456/789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("Reference.String() = %q, want %q", got, tt.want)
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
		// Scheme prefixes
		{"asana:1234567890123456", true},
		{"as:1234567890123456", true},
		// URL patterns
		{"https://app.asana.com/0/1234567890123456/9876543210987654", true},
		{"app.asana.com/0/1234567890123456/9876543210987654", true},
		// Bare GID (valid - ParseReference succeeds)
		{"1234567890123456", true},
		{"1111111111111111/2222222222222222", true},
		// Invalid patterns (ParseReference fails)
		{"", false},
		{"123", false},
		{"abc", false},
		// Other providers
		{"github:123", false},
		{"https://github.com/org/repo/issues/42", false},
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
	p := &Provider{}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "asana scheme",
			input: "asana:1234567890123456",
			want:  "1234567890123456",
		},
		{
			name:  "as scheme",
			input: "as:1234567890123456",
			want:  "1234567890123456",
		},
		{
			name:  "bare task GID",
			input: "1234567890123456",
			want:  "1234567890123456",
		},
		{
			name:  "project/task format",
			input: "1111111111111111/2222222222222222",
			want:  "2222222222222222",
		},
		{
			name:  "asana URL",
			input: "https://app.asana.com/0/1234567890123456/9876543210987654",
			want:  "9876543210987654",
		},
		{
			name:  "URL without project",
			input: "https://app.asana.com/0/0/9876543210987654",
			want:  "9876543210987654",
		},
		{
			name:    "invalid input - too short",
			input:   "123",
			wantErr: true,
		},
		{
			name:    "invalid input - non-numeric",
			input:   "abc-task",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Provider.Parse(%q) expected error, got nil", tt.input)
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
