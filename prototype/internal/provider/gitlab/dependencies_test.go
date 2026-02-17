package gitlab

import (
	"testing"

	"github.com/valksor/go-toolkit/capability"
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
			name:     "cross-project reference extracts local number",
			body:     "**Depends on:** group/project#42",
			expected: []string{"42"},
		},
		{
			name:     "mixed local and cross-project extracts numbers",
			body:     "Depends on: #1, other/repo#2",
			expected: []string{"1", "2"},
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
