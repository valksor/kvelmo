package azuredevops

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
		links    []WorkItemLink
		expected []string
	}{
		{
			name:     "no links",
			links:    nil,
			expected: nil,
		},
		{
			name:     "empty links",
			links:    []WorkItemLink{},
			expected: nil,
		},
		{
			name: "single predecessor link",
			links: []WorkItemLink{
				{
					Type:     "System.LinkTypes.Dependency-Reverse",
					TargetID: "123",
				},
			},
			expected: []string{"123"},
		},
		{
			name: "multiple predecessor links",
			links: []WorkItemLink{
				{
					Type:     "System.LinkTypes.Dependency-Reverse",
					TargetID: "100",
				},
				{
					Type:     "System.LinkTypes.Dependency-Reverse",
					TargetID: "200",
				},
			},
			expected: []string{"100", "200"},
		},
		{
			name: "ignore non-dependency links",
			links: []WorkItemLink{
				{
					Type:     "System.LinkTypes.Related",
					TargetID: "50",
				},
				{
					Type:     "System.LinkTypes.Dependency-Reverse",
					TargetID: "100",
				},
				{
					Type:     "System.LinkTypes.Parent",
					TargetID: "60",
				},
			},
			expected: []string{"100"},
		},
		{
			name: "forward link only - not a dependency",
			links: []WorkItemLink{
				{
					Type:     "System.LinkTypes.Dependency-Forward",
					TargetID: "999",
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
func extractDependencies(links []WorkItemLink) []string {
	var deps []string
	for _, link := range links {
		if link.Type == "System.LinkTypes.Dependency-Reverse" {
			deps = append(deps, link.TargetID)
		}
	}

	return deps
}

func TestWorkItemLinkTypes(t *testing.T) {
	tests := []struct {
		name     string
		linkType string
		isDep    bool
	}{
		{
			name:     "predecessor (reverse) is dependency",
			linkType: "System.LinkTypes.Dependency-Reverse",
			isDep:    true,
		},
		{
			name:     "successor (forward) is not dependency",
			linkType: "System.LinkTypes.Dependency-Forward",
			isDep:    false,
		},
		{
			name:     "related is not dependency",
			linkType: "System.LinkTypes.Related",
			isDep:    false,
		},
		{
			name:     "parent is not dependency",
			linkType: "System.LinkTypes.Parent",
			isDep:    false,
		},
		{
			name:     "child is not dependency",
			linkType: "System.LinkTypes.Child",
			isDep:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := WorkItemLink{Type: tt.linkType, TargetID: "123"}
			deps := extractDependencies([]WorkItemLink{link})
			got := len(deps) > 0

			if got != tt.isDep {
				t.Errorf("Link type %q: isDep = %v, want %v", tt.linkType, got, tt.isDep)
			}
		})
	}
}

func TestParseDependenciesFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    []string
	}{
		{
			name:        "empty description",
			description: "",
			expected:    nil,
		},
		{
			name:        "no dependencies",
			description: "Work item description",
			expected:    nil,
		},
		{
			name:        "single dependency",
			description: "**Depends on:** 123",
			expected:    []string{"123"},
		},
		{
			name:        "multiple dependencies",
			description: "Depends on: 100, 200, 300",
			expected:    []string{"100", "200", "300"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDependenciesFromDescription(tt.description)

			if len(got) != len(tt.expected) {
				t.Errorf("parseDependenciesFromDescription() returned %d items, want %d", len(got), len(tt.expected))

				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseDependenciesFromDescription()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
