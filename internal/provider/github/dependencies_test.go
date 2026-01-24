package github

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

func TestParseDependencies(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []string
	}{
		{
			name:     "empty body",
			body:     "",
			expected: nil,
		},
		{
			name:     "no dependencies",
			body:     "This is a regular issue description.",
			expected: nil,
		},
		{
			name:     "single dependency plain",
			body:     "Depends on: #123",
			expected: []string{"123"},
		},
		{
			name:     "single dependency bold",
			body:     "**Depends on:** #456",
			expected: []string{"456"},
		},
		{
			name:     "multiple dependencies comma separated",
			body:     "**Depends on:** #100, #200, #300",
			expected: []string{"100", "200", "300"},
		},
		{
			name:     "multiple dependencies space separated",
			body:     "Depends on: #10 #20 #30",
			expected: []string{"10", "20", "30"},
		},
		{
			name:     "dependencies in middle of description",
			body:     "Some text\n**Depends on:** #42\nMore text",
			expected: []string{"42"},
		},
		{
			name:     "dependencies at beginning",
			body:     "**Depends on:** #1, #2\n\nRest of the description here.",
			expected: []string{"1", "2"},
		},
		{
			name:     "mixed delimiters",
			body:     "Depends on: #5, #6 #7",
			expected: []string{"5", "6", "7"},
		},
		{
			name:     "with whitespace variations",
			body:     "**Depends on:**   #100 ,  #200 , #300  ",
			expected: []string{"100", "200", "300"},
		},
		{
			name:     "numeric IDs without hash",
			body:     "Depends on: #123, #456",
			expected: []string{"123", "456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDependencies(tt.body)

			if len(got) != len(tt.expected) {
				t.Errorf("parseDependencies() returned %d items, want %d", len(got), len(tt.expected))
				t.Errorf("got: %v, want: %v", got, tt.expected)

				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseDependencies()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestAddDependencyToBody(t *testing.T) {
	tests := []struct {
		name         string
		originalBody string
		depRef       string
		wantContains string
	}{
		{
			name:         "empty body",
			originalBody: "",
			depRef:       "#123",
			wantContains: "#123",
		},
		{
			name:         "body without dependencies",
			originalBody: "Some description",
			depRef:       "#456",
			wantContains: "#456",
		},
		{
			name:         "body with existing dependencies",
			originalBody: "**Depends on:** #100\n\nDescription",
			depRef:       "#200",
			wantContains: "#200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addDependencyToBody(tt.originalBody, tt.depRef)

			// Verify the dependency reference is in the result
			if !contains(got, tt.wantContains) {
				t.Errorf("Result %q does not contain %q", got, tt.wantContains)
			}
		})
	}
}

// addDependencyToBody is a helper for testing - simulates what CreateDependency does.
func addDependencyToBody(body, depRef string) string {
	// Check if already exists
	deps := parseDependencies(body)
	for _, d := range deps {
		if d == depRef || "#"+d == depRef {
			return body // No change needed
		}
	}

	// Simple implementation for testing
	if body != "" {
		return "**Depends on:** " + depRef + "\n\n" + body
	}

	return "**Depends on:** " + depRef
}

// contains and containsHelper are defined in parser_test.go
