package youtrack

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
			description: "This is a regular issue description.",
			expected:    nil,
		},
		{
			name:        "single dependency plain",
			description: "Depends on: ABC-123",
			expected:    []string{"ABC-123"},
		},
		{
			name:        "single dependency bold",
			description: "**Depends on:** ABC-456",
			expected:    []string{"ABC-456"},
		},
		{
			name:        "multiple dependencies comma separated",
			description: "**Depends on:** ABC-100, ABC-200, ABC-300",
			expected:    []string{"ABC-100", "ABC-200", "ABC-300"},
		},
		{
			name:        "multiple dependencies space separated",
			description: "Depends on: ABC-10 ABC-20 ABC-30",
			expected:    []string{"ABC-10", "ABC-20", "ABC-30"},
		},
		{
			name:        "dependencies in middle of description",
			description: "Some text\n**Depends on:** ABC-42\nMore text",
			expected:    []string{"ABC-42"},
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
	err := p.CreateDependency(context.Background(), "ABC-123", "ABC-456")
	if err == nil {
		t.Error("CreateDependency with nil client should return error")
	}

	// Test GetDependencies with nil client
	_, err = p.GetDependencies(context.Background(), "ABC-123")
	if err == nil {
		t.Error("GetDependencies with nil client should return error")
	}
}
