package azuredevops

import (
	"testing"

	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Compile-time interface compliance checks
// ──────────────────────────────────────────────────────────────────────────────

var (
	_ workunit.Reader          = (*Provider)(nil)
	_ workunit.Identifier      = (*Provider)(nil)
	_ workunit.Lister          = (*Provider)(nil)
	_ workunit.CommentFetcher  = (*Provider)(nil)
	_ workunit.Commenter       = (*Provider)(nil)
	_ workunit.StatusUpdater   = (*Provider)(nil)
	_ workunit.LabelManager    = (*Provider)(nil)
	_ snapshot.Snapshotter     = (*Provider)(nil)
	_ pullrequest.PRCreator    = (*Provider)(nil)
	_ workunit.WorkUnitCreator = (*Provider)(nil)
	_ workunit.SubtaskFetcher  = (*Provider)(nil)
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Reference
		wantErr bool
	}{
		{
			name:  "work item ID only",
			input: "123",
			want: &Reference{
				WorkItemID: 123,
				IsExplicit: false,
			},
		},
		{
			name:  "org/project#ID format",
			input: "myorg/myproject#456",
			want: &Reference{
				Organization: "myorg",
				Project:      "myproject",
				WorkItemID:   456,
				IsExplicit:   true,
			},
		},
		{
			name:  "dev.azure.com URL",
			input: "https://dev.azure.com/myorg/myproject/_workitems/edit/789",
			want: &Reference{
				Organization: "myorg",
				Project:      "myproject",
				WorkItemID:   789,
				IsExplicit:   true,
			},
		},
		{
			name:  "dev.azure.com URL without https",
			input: "dev.azure.com/myorg/myproject/_workitems/edit/101",
			want: &Reference{
				Organization: "myorg",
				Project:      "myproject",
				WorkItemID:   101,
				IsExplicit:   true,
			},
		},
		{
			name:  "visualstudio.com URL",
			input: "https://myorg.visualstudio.com/myproject/_workitems/edit/202",
			want: &Reference{
				Organization: "myorg",
				Project:      "myproject",
				WorkItemID:   202,
				IsExplicit:   true,
			},
		},
		{
			name:  "with azdo prefix",
			input: "azdo:303",
			want: &Reference{
				WorkItemID: 303,
				IsExplicit: false,
			},
		},
		{
			name:  "with azure prefix",
			input: "azure:404",
			want: &Reference{
				WorkItemID: 404,
				IsExplicit: false,
			},
		},
		{
			name:    "invalid input - not a number",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid input - negative number",
			input:   "-1",
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

			if got.WorkItemID != tt.want.WorkItemID {
				t.Errorf("ParseReference(%q).WorkItemID = %d, want %d", tt.input, got.WorkItemID, tt.want.WorkItemID)
			}
			if got.Organization != tt.want.Organization {
				t.Errorf("ParseReference(%q).Organization = %q, want %q", tt.input, got.Organization, tt.want.Organization)
			}
			if got.Project != tt.want.Project {
				t.Errorf("ParseReference(%q).Project = %q, want %q", tt.input, got.Project, tt.want.Project)
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

	expectedSchemes := []string{"azdo", "azure"}
	if len(info.Schemes) != len(expectedSchemes) {
		t.Errorf("Info().Schemes = %v, want %v", info.Schemes, expectedSchemes)
	}
	for i, scheme := range expectedSchemes {
		if info.Schemes[i] != scheme {
			t.Errorf("Info().Schemes[%d] = %q, want %q", i, info.Schemes[i], scheme)
		}
	}

	// Check capabilities
	if !info.Capabilities.Has(capability.CapRead) {
		t.Error("Info().Capabilities should have CapRead")
	}
	if !info.Capabilities.Has(capability.CapList) {
		t.Error("Info().Capabilities should have CapList")
	}
	if !info.Capabilities.Has(capability.CapFetchComments) {
		t.Error("Info().Capabilities should have CapFetchComments")
	}
	if !info.Capabilities.Has(capability.CapComment) {
		t.Error("Info().Capabilities should have CapComment")
	}
	if !info.Capabilities.Has(capability.CapUpdateStatus) {
		t.Error("Info().Capabilities should have CapUpdateStatus")
	}
	if !info.Capabilities.Has(capability.CapSnapshot) {
		t.Error("Info().Capabilities should have CapSnapshot")
	}
	if !info.Capabilities.Has(capability.CapCreatePR) {
		t.Error("Info().Capabilities should have CapCreatePR")
	}
	if !info.Capabilities.Has(capability.CapLinkBranch) {
		t.Error("Info().Capabilities should have CapLinkBranch")
	}
}

func TestMapAzureState(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  workunit.Status
	}{
		{
			name:  "New state",
			state: "New",
			want:  workunit.StatusOpen,
		},
		{
			name:  "To Do state",
			state: "To Do",
			want:  workunit.StatusOpen,
		},
		{
			name:  "Active state",
			state: "Active",
			want:  workunit.StatusInProgress,
		},
		{
			name:  "In Progress state",
			state: "In Progress",
			want:  workunit.StatusInProgress,
		},
		{
			name:  "Resolved state",
			state: "Resolved",
			want:  workunit.StatusClosed,
		},
		{
			name:  "Done state",
			state: "Done",
			want:  workunit.StatusClosed,
		},
		{
			name:  "Closed state",
			state: "Closed",
			want:  workunit.StatusClosed,
		},
		{
			name:  "In Review state",
			state: "In Review",
			want:  workunit.StatusReview,
		},
		{
			name:  "Unknown state",
			state: "Something Else",
			want:  workunit.StatusOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAzureState(tt.state)
			if got != tt.want {
				t.Errorf("mapAzureState(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestMapAzurePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     workunit.Priority
	}{
		{
			name:     "priority 1 (critical)",
			priority: 1,
			want:     workunit.PriorityCritical,
		},
		{
			name:     "priority 2 (high)",
			priority: 2,
			want:     workunit.PriorityHigh,
		},
		{
			name:     "priority 3 (normal)",
			priority: 3,
			want:     workunit.PriorityNormal,
		},
		{
			name:     "priority 4 (low)",
			priority: 4,
			want:     workunit.PriorityLow,
		},
		{
			name:     "priority 0 (default)",
			priority: 0,
			want:     workunit.PriorityNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAzurePriority(tt.priority)
			if got != tt.want {
				t.Errorf("mapAzurePriority(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestMapWorkItemType(t *testing.T) {
	tests := []struct {
		name   string
		wiType string
		want   string
	}{
		{
			name:   "Bug type",
			wiType: "Bug",
			want:   "fix",
		},
		{
			name:   "Feature type",
			wiType: "Feature",
			want:   "feature",
		},
		{
			name:   "User Story type",
			wiType: "User Story",
			want:   "feature",
		},
		{
			name:   "Task type",
			wiType: "Task",
			want:   "task",
		},
		{
			name:   "Epic type",
			wiType: "Epic",
			want:   "epic",
		},
		{
			name:   "Issue type",
			wiType: "Issue",
			want:   "issue",
		},
		{
			name:   "Unknown type",
			wiType: "Unknown",
			want:   "task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapWorkItemType(tt.wiType)
			if got != tt.want {
				t.Errorf("mapWorkItemType(%q) = %q, want %q", tt.wiType, got, tt.want)
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		name string
		tags string
		want []string
	}{
		{
			name: "empty tags",
			tags: "",
			want: nil,
		},
		{
			name: "single tag",
			tags: "bug",
			want: []string{"bug"},
		},
		{
			name: "multiple tags",
			tags: "bug; feature; urgent",
			want: []string{"bug", "feature", "urgent"},
		},
		{
			name: "tags with extra spaces",
			tags: "  bug  ;  feature  ",
			want: []string{"bug", "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTags(tt.tags)
			if len(got) != len(tt.want) {
				t.Errorf("parseTags(%q) returned %d tags, want %d", tt.tags, len(got), len(tt.want))

				return
			}
			for i, tag := range got {
				if tag != tt.want[i] {
					t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.tags, i, tag, tt.want[i])
				}
			}
		})
	}
}

func TestExtractWorkItemIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{
			name:  "AB# format",
			input: "Fixed AB#123 and AB#456",
			want:  []int{123, 456},
		},
		{
			name:  "# format",
			input: "Related to #789",
			want:  []int{789},
		},
		{
			name:  "mixed formats",
			input: "See AB#100 and #200 for details",
			want:  []int{100, 200},
		},
		{
			name:  "no work items",
			input: "Just regular text",
			want:  nil,
		},
		{
			name:  "duplicates",
			input: "AB#123 is related to AB#123",
			want:  []int{123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWorkItemIDs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractWorkItemIDs() returned %d IDs, want %d", len(got), len(tt.want))

				return
			}
			for i, id := range got {
				if id != tt.want[i] {
					t.Errorf("ExtractWorkItemIDs()[%d] = %d, want %d", i, id, tt.want[i])
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
			ref:  Reference{WorkItemID: 123},
			want: "123",
		},
		{
			ref:  Reference{Organization: "org", Project: "proj", WorkItemID: 456},
			want: "org/proj#456",
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

func TestParseAzureTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		empty bool
	}{
		{
			name:  "empty string",
			input: "",
			empty: true,
		},
		{
			name:  "RFC3339",
			input: "2024-01-01T00:00:00Z",
			empty: false,
		},
		{
			name:  "ISO 8601 without ms",
			input: "2024-01-15T10:30:00Z",
			empty: false,
		},
		{
			name:  "invalid timestamp",
			input: "not-a-timestamp",
			empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAzureTime(tt.input)
			if tt.empty && !got.IsZero() {
				t.Errorf("parseAzureTime(%q) = %v, want zero time", tt.input, got)
			}
			if !tt.empty && got.IsZero() {
				t.Errorf("parseAzureTime(%q) returned zero time, expected non-zero", tt.input)
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
		{"azdo:123", true},
		{"azure:456", true},
		// URL patterns
		{"https://dev.azure.com/org/project/_workitems/edit/789", true},
		{"dev.azure.com/org/project/_workitems/edit/101", true},
		{"https://myorg.visualstudio.com/project/_workitems/edit/202", true},
		// Numeric input is valid (ParseReference succeeds)
		{"123", true},
		{"org/project#456", true},
		// Empty and invalid
		{"", false},
		{"github:123", false},
		{"https://github.com/org/repo/issues/42", false},
		{"abc", false},
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
		org         string
		project     string
		want        string
		errContains string
		wantErr     bool
	}{
		{
			name:    "work item ID only",
			input:   "123",
			org:     "myorg",
			project: "myproject",
			want:    "123",
		},
		{
			name:    "org/project#ID format",
			input:   "myorg/myproject#456",
			org:     "",
			project: "",
			want:    "456",
		},
		{
			name:    "with azdo prefix",
			input:   "azdo:789",
			org:     "",
			project: "",
			want:    "789",
		},
		{
			name:    "with azure prefix",
			input:   "azure:101",
			org:     "",
			project: "",
			want:    "101",
		},
		{
			name:    "dev.azure.com URL",
			input:   "https://dev.azure.com/myorg/myproject/_workitems/edit/202",
			org:     "",
			project: "",
			want:    "202",
		},
		{
			name:        "invalid input",
			input:       "abc",
			org:         "",
			project:     "",
			errContains: "invalid azure devops reference",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				config: &Config{
					Organization: tt.org,
					Project:      tt.project,
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
// mapToAzureState tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapToAzureState(t *testing.T) {
	tests := []struct {
		status workunit.Status
		want   string
	}{
		{workunit.StatusOpen, "New"},
		{workunit.StatusInProgress, "Active"},
		{workunit.StatusReview, "Resolved"},
		{workunit.StatusDone, "Done"},
		{workunit.StatusClosed, "Done"},
		{workunit.Status("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := mapToAzureState(tt.status)
			if got != tt.want {
				t.Errorf("mapToAzureState(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		(len(haystack) > 0 && len(needle) > 0 && findInString(haystack, needle)))
}

func findInString(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}

// ──────────────────────────────────────────────────────────────────────────────
// buildWIQLQuery tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildWIQLQuery(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		opts   workunit.ListOptions
		want   string
	}{
		{
			name:   "no filters",
			config: &Config{},
			opts:   workunit.ListOptions{},
			want:   "SELECT [System.Id] FROM WorkItems ORDER BY [System.ChangedDate] DESC",
		},
		{
			name: "area path filter",
			config: &Config{
				AreaPath: "MyProject",
			},
			opts: workunit.ListOptions{},
			want: "SELECT [System.Id] FROM WorkItems WHERE [System.AreaPath] UNDER 'MyProject' ORDER BY [System.ChangedDate] DESC",
		},
		{
			name: "iteration path filter",
			config: &Config{
				IterationPath: "MyProject\\Sprint 1",
			},
			opts: workunit.ListOptions{},
			want: "SELECT [System.Id] FROM WorkItems WHERE [System.IterationPath] UNDER 'MyProject\\Sprint 1' ORDER BY [System.ChangedDate] DESC",
		},
		{
			name:   "status open filter",
			config: &Config{},
			opts:   workunit.ListOptions{Status: workunit.StatusOpen},
			want:   "SELECT [System.Id] FROM WorkItems WHERE [System.State] IN ('New', 'To Do', 'Proposed') ORDER BY [System.ChangedDate] DESC",
		},
		{
			name:   "status in progress filter",
			config: &Config{},
			opts:   workunit.ListOptions{Status: workunit.StatusInProgress},
			want:   "SELECT [System.Id] FROM WorkItems WHERE [System.State] IN ('Active', 'In Progress', 'Committed') ORDER BY [System.ChangedDate] DESC",
		},
		{
			name:   "status review filter",
			config: &Config{},
			opts:   workunit.ListOptions{Status: workunit.StatusReview},
			want:   "SELECT [System.Id] FROM WorkItems WHERE [System.State] IN ('Resolved', 'In Review') ORDER BY [System.ChangedDate] DESC",
		},
		{
			name:   "status closed filter",
			config: &Config{},
			opts:   workunit.ListOptions{Status: workunit.StatusClosed},
			want:   "SELECT [System.Id] FROM WorkItems WHERE [System.State] IN ('Done', 'Closed', 'Removed') ORDER BY [System.ChangedDate] DESC",
		},
		{
			name:   "labels filter",
			config: &Config{},
			opts: workunit.ListOptions{
				Labels: []string{"bug", "urgent"},
			},
			want: "SELECT [System.Id] FROM WorkItems WHERE [System.Tags] CONTAINS 'bug' AND [System.Tags] CONTAINS 'urgent' ORDER BY [System.ChangedDate] DESC",
		},
		{
			name: "combined filters",
			config: &Config{
				AreaPath:      "MyProject",
				IterationPath: "MyProject\\Sprint 1",
			},
			opts: workunit.ListOptions{
				Status: workunit.StatusOpen,
				Labels: []string{"bug"},
			},
			want: "SELECT [System.Id] FROM WorkItems WHERE [System.AreaPath] UNDER 'MyProject' AND [System.IterationPath] UNDER 'MyProject\\Sprint 1' AND [System.State] IN ('New', 'To Do', 'Proposed') AND [System.Tags] CONTAINS 'bug' ORDER BY [System.ChangedDate] DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWIQLQuery(tt.config, tt.opts)
			if got != tt.want {
				t.Errorf("buildWIQLQuery() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// buildSnapshotContent tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildSnapshotContent(t *testing.T) {
	tests := []struct {
		name string
		wi   *WorkItem
		want string
	}{
		{
			name: "basic work item",
			wi: &WorkItem{
				ID: 123,
				Fields: WorkItemFields{
					Title: "Test Bug",
				},
			},
			want: "# Test Bug\n\n**ID:** 123\n**Type:** \n**State:** \n\n",
		},
		{
			name: "work item with all fields",
			wi: &WorkItem{
				ID: 456,
				Fields: WorkItemFields{
					Title:         "Feature Request",
					WorkItemType:  "Task",
					State:         "Active",
					Priority:      2,
					Tags:          "feature; urgent",
					Description:   "Add new feature",
					AreaPath:      "MyProject",
					IterationPath: "Sprint 1",
				},
			},
			want: "# Feature Request\n\n**ID:** 456\n**Type:** Task\n**State:** Active\n**Area:** MyProject\n**Iteration:** Sprint 1\n**Priority:** 2\n**Tags:** feature; urgent\n\n## Description\n\nAdd new feature\n",
		},
		{
			name: "work item with assigned to",
			wi: &WorkItem{
				ID: 789,
				Fields: WorkItemFields{
					Title: "Assigned Task",
					AssignedTo: &Identity{
						DisplayName: "John Doe",
					},
				},
			},
			want: "# Assigned Task\n\n**ID:** 789\n**Type:** \n**State:** \n**Assigned To:** John Doe\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSnapshotContent(tt.wi)
			if got != tt.want {
				t.Errorf("buildSnapshotContent() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// extractAttachments tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractAttachments(t *testing.T) {
	tests := []struct {
		name string
		rels []WorkItemRelation
		want []workunit.Attachment
	}{
		{
			name: "no attachments",
			rels: []WorkItemRelation{},
			want: nil,
		},
		{
			name: "single attachment",
			rels: []WorkItemRelation{
				{
					Rel: "AttachedFile",
					URL: "https://example.com/file.txt",
					Attributes: map[string]interface{}{
						"name": "file.txt",
					},
				},
			},
			want: []workunit.Attachment{
				{
					ID:   "https://example.com/file.txt",
					URL:  "https://example.com/file.txt",
					Name: "file.txt",
				},
			},
		},
		{
			name: "multiple attachments",
			rels: []WorkItemRelation{
				{
					Rel: "AttachedFile",
					URL: "https://example.com/file1.txt",
					Attributes: map[string]interface{}{
						"name": "file1.txt",
					},
				},
				{
					Rel: "AttachedFile",
					URL: "https://example.com/file2.pdf",
					Attributes: map[string]interface{}{
						"name": "file2.pdf",
					},
				},
			},
			want: []workunit.Attachment{
				{
					ID:   "https://example.com/file1.txt",
					URL:  "https://example.com/file1.txt",
					Name: "file1.txt",
				},
				{
					ID:   "https://example.com/file2.pdf",
					URL:  "https://example.com/file2.pdf",
					Name: "file2.pdf",
				},
			},
		},
		{
			name: "mixed relations (non-attachments filtered out)",
			rels: []WorkItemRelation{
				{
					Rel: "AttachedFile",
					URL: "https://example.com/file.txt",
					Attributes: map[string]interface{}{
						"name": "file.txt",
					},
				},
				{
					Rel: "WorkItemLink",
					URL: "https://example.com/link/123",
				},
			},
			want: []workunit.Attachment{
				{
					ID:   "https://example.com/file.txt",
					URL:  "https://example.com/file.txt",
					Name: "file.txt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAttachments(tt.rels)
			if len(got) != len(tt.want) {
				t.Errorf("extractAttachments() returned %d items, want %d", len(got), len(tt.want))

				return
			}
			for i := range got {
				if got[i].URL != tt.want[i].URL || got[i].Name != tt.want[i].Name {
					t.Errorf("extractAttachments()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// workItemToWorkUnit tests
// ──────────────────────────────────────────────────────────────────────────────

func TestWorkItemToWorkUnit(t *testing.T) {
	tests := []struct {
		name string
		wi   *WorkItem
		want workunit.WorkUnit
	}{
		{
			name: "basic work item",
			wi: &WorkItem{
				ID: 123,
				Fields: WorkItemFields{
					Title: "Test Bug",
					State: "Active",
				},
				URL: "https://dev.azure.com/testorg/testproj/_workitems/edit/123",
			},
			want: workunit.WorkUnit{
				ID:          "123",
				Title:       "Test Bug",
				Status:      workunit.StatusInProgress,
				TaskType:    "task",
				Description: "",
			},
		},
		{
			name: "bug work item",
			wi: &WorkItem{
				ID: 456,
				Fields: WorkItemFields{
					Title:        "Fix bug",
					State:        "New",
					WorkItemType: "Bug",
					Priority:     1,
					Tags:         "urgent; bug",
					Description:  "Fix this bug",
				},
				URL: "https://dev.azure.com/org/proj/_workitems/edit/456",
			},
			want: workunit.WorkUnit{
				ID:          "456",
				Title:       "Fix bug",
				Status:      workunit.StatusOpen,
				TaskType:    "fix",
				Priority:    workunit.PriorityCritical,
				Labels:      []string{"urgent", "bug"},
				Description: "Fix this bug",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				config: &Config{
					Organization: "testorg",
					Project:      "testproj",
				},
			}
			got := p.workItemToWorkUnit(tt.wi)

			if got.ID != tt.want.ID {
				t.Errorf("workItemToWorkUnit().ID = %q, want %q", got.ID, tt.want.ID)
			}
			if got.Title != tt.want.Title {
				t.Errorf("workItemToWorkUnit().Title = %q, want %q", got.Title, tt.want.Title)
			}
			if got.Status != tt.want.Status {
				t.Errorf("workItemToWorkUnit().Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.TaskType != tt.want.TaskType {
				t.Errorf("workItemToWorkUnit().TaskType = %q, want %q", got.TaskType, tt.want.TaskType)
			}
			if got.Description != tt.want.Description {
				t.Errorf("workItemToWorkUnit().Description = %q, want %q", got.Description, tt.want.Description)
			}
		})
	}
}
