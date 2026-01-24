package jira

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestInfo_DependencyCapabilities(t *testing.T) {
	info := Info()

	expectedCaps := []provider.Capability{
		provider.CapCreateDependency,
		provider.CapFetchDependencies,
	}

	for _, cap := range expectedCaps {
		if !info.Capabilities.Has(cap) {
			t.Errorf("Capabilities missing %q", cap)
		}
	}
}

func TestExtractDependenciesFromLinks(t *testing.T) {
	tests := []struct {
		name     string
		links    []IssueLink
		expected []string
	}{
		{
			name:     "no links",
			links:    nil,
			expected: nil,
		},
		{
			name:     "empty links",
			links:    []IssueLink{},
			expected: nil,
		},
		{
			name: "single blocks link",
			links: []IssueLink{
				{
					Type:        IssueLinkType{Name: "Blocks"},
					InwardIssue: &LinkedIssue{Key: "PROJ-123"},
				},
			},
			expected: []string{"PROJ-123"},
		},
		{
			name: "multiple blocks links",
			links: []IssueLink{
				{
					Type:        IssueLinkType{Name: "Blocks"},
					InwardIssue: &LinkedIssue{Key: "PROJ-100"},
				},
				{
					Type:        IssueLinkType{Name: "Blocks"},
					InwardIssue: &LinkedIssue{Key: "PROJ-200"},
				},
			},
			expected: []string{"PROJ-100", "PROJ-200"},
		},
		{
			name: "ignore non-blocks links",
			links: []IssueLink{
				{
					Type:        IssueLinkType{Name: "Relates"},
					InwardIssue: &LinkedIssue{Key: "PROJ-50"},
				},
				{
					Type:        IssueLinkType{Name: "Blocks"},
					InwardIssue: &LinkedIssue{Key: "PROJ-100"},
				},
				{
					Type:        IssueLinkType{Name: "Duplicate"},
					InwardIssue: &LinkedIssue{Key: "PROJ-60"},
				},
			},
			expected: []string{"PROJ-100"},
		},
		{
			name: "outward link only - no dependency",
			links: []IssueLink{
				{
					Type:         IssueLinkType{Name: "Blocks"},
					OutwardIssue: &LinkedIssue{Key: "PROJ-999"},
				},
			},
			expected: nil,
		},
		{
			name: "nil inward issue",
			links: []IssueLink{
				{
					Type:        IssueLinkType{Name: "Blocks"},
					InwardIssue: nil,
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDependencies(tt.links)

			if len(got) != len(tt.expected) {
				t.Errorf("extractDependencies() returned %d items, want %d", len(got), len(tt.expected))
				t.Errorf("got: %v, want: %v", got, tt.expected)

				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractDependencies()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// extractDependencies is a helper for testing - simulates what GetDependencies does.
func extractDependencies(links []IssueLink) []string {
	var deps []string
	for _, link := range links {
		if link.Type.Name == "Blocks" && link.InwardIssue != nil {
			deps = append(deps, link.InwardIssue.Key)
		}
	}

	return deps
}

func TestIssueLinkTypeFields(t *testing.T) {
	linkType := IssueLinkType{
		ID:      "10000",
		Name:    "Blocks",
		Inward:  "is blocked by",
		Outward: "blocks",
	}

	if linkType.Name != "Blocks" {
		t.Errorf("Name = %q, want %q", linkType.Name, "Blocks")
	}
	if linkType.Inward != "is blocked by" {
		t.Errorf("Inward = %q, want %q", linkType.Inward, "is blocked by")
	}
	if linkType.Outward != "blocks" {
		t.Errorf("Outward = %q, want %q", linkType.Outward, "blocks")
	}
}

func TestLinkedIssueFields(t *testing.T) {
	linked := LinkedIssue{
		ID:   "12345",
		Key:  "PROJ-100",
		Self: "https://jira.example.com/rest/api/3/issue/12345",
	}
	linked.Fields.Summary = "Test issue"
	linked.Fields.Status = &Status{Name: "Open"}

	if linked.Key != "PROJ-100" {
		t.Errorf("Key = %q, want %q", linked.Key, "PROJ-100")
	}
	if linked.Fields.Summary != "Test issue" {
		t.Errorf("Fields.Summary = %q, want %q", linked.Fields.Summary, "Test issue")
	}
	if linked.Fields.Status.Name != "Open" {
		t.Errorf("Fields.Status.Name = %q, want %q", linked.Fields.Status.Name, "Open")
	}
}
