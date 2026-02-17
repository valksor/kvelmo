package trello

import (
	"context"
	"testing"

	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/workunit"
)

func TestInfo_DependencyCapabilities(t *testing.T) {
	info := Info()

	expectedCaps := []capability.Capability{
		capability.CapCreateDependency,
		capability.CapFetchDependencies,
	}

	for _, cap := range expectedCaps {
		if !info.Capabilities.Has(cap) {
			t.Errorf("Capabilities missing %q", cap)
		}
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
			description: "This is a card description.",
			expected:    nil,
		},
		{
			name:        "single dependency plain",
			description: "Depends on: 5a1b2c3d4e5f",
			expected:    []string{"5a1b2c3d4e5f"},
		},
		{
			name:        "single dependency bold",
			description: "**Depends on:** abc123def456",
			expected:    []string{"abc123def456"},
		},
		{
			name:        "multiple dependencies comma separated",
			description: "**Depends on:** card1, card2, card3",
			expected:    []string{"card1", "card2", "card3"},
		},
		{
			name:        "multiple dependencies space separated",
			description: "Depends on: id1 id2 id3",
			expected:    []string{"id1", "id2", "id3"},
		},
		{
			name:        "dependencies in middle of description",
			description: "Some text\n**Depends on:** cardXYZ\nMore text",
			expected:    []string{"cardXYZ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDependenciesFromDescription(tt.description)

			if len(got) != len(tt.expected) {
				t.Errorf("parseDependenciesFromDescription() returned %d items, want %d", len(got), len(tt.expected))
				t.Errorf("got: %v, want: %v", got, tt.expected)

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

func TestDependencyInterfaceImplementation(t *testing.T) {
	// Verify Provider implements the dependency interfaces
	var _ workunit.DependencyCreator = (*Provider)(nil)
	var _ workunit.DependencyFetcher = (*Provider)(nil)
}

func TestProviderNotInitialized(t *testing.T) {
	p := &Provider{client: nil}

	// Test CreateDependency with nil client
	err := p.CreateDependency(context.Background(), "card123", "card456")
	if err == nil {
		t.Error("CreateDependency with nil client should return error")
	}

	// Test GetDependencies with nil client
	_, err = p.GetDependencies(context.Background(), "card123")
	if err == nil {
		t.Error("GetDependencies with nil client should return error")
	}
}
