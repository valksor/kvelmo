package bitbucket

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
			description: "This is an issue description.",
			expected:    nil,
		},
		{
			name:        "single dependency plain",
			description: "Depends on: #123",
			expected:    []string{"#123"},
		},
		{
			name:        "single dependency bold",
			description: "**Depends on:** #456",
			expected:    []string{"#456"},
		},
		{
			name:        "multiple dependencies comma separated",
			description: "**Depends on:** #100, #200, #300",
			expected:    []string{"#100", "#200", "#300"},
		},
		{
			name:        "multiple dependencies space separated",
			description: "Depends on: #10 #20 #30",
			expected:    []string{"#10", "#20", "#30"},
		},
		{
			name:        "dependencies in middle of description",
			description: "Some text\n**Depends on:** #42\nMore text",
			expected:    []string{"#42"},
		},
		{
			name:        "cross-repo reference",
			description: "Depends on: workspace/repo#123",
			expected:    []string{"workspace/repo#123"},
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
	p := &Provider{client: nil, config: &Config{}}

	// Test CreateDependency with nil client
	err := p.CreateDependency(context.Background(), "workspace/repo#123", "workspace/repo#456")
	if err == nil {
		t.Error("CreateDependency with nil client should return error")
	}

	// Test GetDependencies with nil client
	_, err = p.GetDependencies(context.Background(), "workspace/repo#123")
	if err == nil {
		t.Error("GetDependencies with nil client should return error")
	}
}
