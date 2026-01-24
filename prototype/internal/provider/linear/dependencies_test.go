package linear

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
			description: "Depends on: ENG-123",
			expected:    []string{"ENG-123"},
		},
		{
			name:        "single dependency bold",
			description: "**Depends on:** ENG-456",
			expected:    []string{"ENG-456"},
		},
		{
			name:        "multiple dependencies comma separated",
			description: "**Depends on:** ENG-100, ENG-200, ENG-300",
			expected:    []string{"ENG-100", "ENG-200", "ENG-300"},
		},
		{
			name:        "multiple dependencies space separated",
			description: "Depends on: ENG-10 ENG-20 ENG-30",
			expected:    []string{"ENG-10", "ENG-20", "ENG-30"},
		},
		{
			name:        "dependencies in middle of description",
			description: "Some text\n**Depends on:** ENG-42\nMore text",
			expected:    []string{"ENG-42"},
		},
		{
			name:        "dependencies at beginning",
			description: "**Depends on:** ENG-1, ENG-2\n\nRest of the description here.",
			expected:    []string{"ENG-1", "ENG-2"},
		},
		{
			name:        "with UUID style ID",
			description: "Depends on: abc123-def456",
			expected:    []string{"abc123-def456"},
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

func TestAddDependencyToDescription(t *testing.T) {
	tests := []struct {
		name         string
		originalDesc string
		depRef       string
		wantContains string
	}{
		{
			name:         "empty description",
			originalDesc: "",
			depRef:       "ENG-123",
			wantContains: "ENG-123",
		},
		{
			name:         "description without dependencies",
			originalDesc: "Some description",
			depRef:       "ENG-456",
			wantContains: "ENG-456",
		},
		{
			name:         "description with existing dependencies",
			originalDesc: "**Depends on:** ENG-100\n\nDescription",
			depRef:       "ENG-200",
			wantContains: "ENG-200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addDependencyToDescription(tt.originalDesc, tt.depRef)

			// Verify the dependency was added
			deps := parseDependenciesFromDescription(got)
			found := false
			for _, d := range deps {
				if d == tt.depRef {
					found = true

					break
				}
			}
			if !found {
				t.Errorf("Dependency %q not found in result: %q", tt.depRef, got)
			}
		})
	}
}

// addDependencyToDescription is a helper for testing.
func addDependencyToDescription(description, depRef string) string {
	// Check if already exists
	deps := parseDependenciesFromDescription(description)
	for _, d := range deps {
		if d == depRef {
			return description // No change needed
		}
	}

	// Simple implementation for testing
	if description != "" {
		return "**Depends on:** " + depRef + "\n\n" + description
	}

	return "**Depends on:** " + depRef
}

func TestDependencyDeduplication(t *testing.T) {
	// Test that adding the same dependency twice doesn't duplicate
	desc := "**Depends on:** ENG-100\n\nDescription"
	depRef := "ENG-100"

	result := addDependencyToDescription(desc, depRef)
	deps := parseDependenciesFromDescription(result)

	count := 0
	for _, d := range deps {
		if d == depRef {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 occurrence of %q, got %d", depRef, count)
	}
}
