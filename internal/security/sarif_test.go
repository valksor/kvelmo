package security

import (
	"testing"
)

func TestSeverityToLevel(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{
			name:     "critical severity",
			severity: SeverityCritical,
			want:     "error",
		},
		{
			name:     "high severity",
			severity: SeverityHigh,
			want:     "error",
		},
		{
			name:     "medium severity",
			severity: SeverityMedium,
			want:     "warning",
		},
		{
			name:     "low severity",
			severity: SeverityLow,
			want:     "note",
		},
		{
			name:     "info severity",
			severity: SeverityInfo,
			want:     "note",
		},
		{
			name:     "unknown severity defaults to note",
			severity: Severity("unknown"),
			want:     "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityToLevel(tt.severity)
			if got != tt.want {
				t.Errorf("severityToLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarshalJSONResults(t *testing.T) {
	results := []*ScanResult{
		{
			Scanner: "test-scanner",
			Findings: []Finding{
				{
					ID:          "TEST-001",
					Severity:    SeverityHigh,
					Title:       "Test Finding",
					Description: "This is a test finding",
					Location: Location{
						File:   "test.go",
						Line:   42,
						Column: 10,
					},
				},
			},
			Summary: Summary{
				Total: 1,
				BySeverity: map[Severity]int{
					SeverityHigh: 1,
				},
			},
		},
	}

	data, err := MarshalJSONResults(results)
	if err != nil {
		t.Fatalf("MarshalJSONResults() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalJSONResults() returned empty data")
	}

	// Check that it contains expected JSON fields
	jsonStr := string(data)
	expectedStrings := []string{
		"\"scanner\"",
		"\"test-scanner\"",
		"\"TEST-001\"",
		"\"high\"", // Severity values are lowercase
		"\"test.go\"",
		"42",
	}

	for _, s := range expectedStrings {
		if !containsString(jsonStr, s) {
			t.Errorf("MarshalJSONResults() should contain %q", s)
		}
	}
}

func TestMarshalJSONResults_Empty(t *testing.T) {
	data, err := MarshalJSONResults([]*ScanResult{})
	if err != nil {
		t.Fatalf("MarshalJSONResults() error = %v", err)
	}

	// Empty array should be represented
	jsonStr := string(data)
	if jsonStr != "[]\n" && jsonStr != "[]" {
		t.Errorf("MarshalJSONResults() empty should be '[]', got %q", jsonStr)
	}
}

func TestMarshalJSONResults_Nil(t *testing.T) {
	data, err := MarshalJSONResults(nil)
	if err != nil {
		t.Fatalf("MarshalJSONResults() error = %v", err)
	}

	// Nil should be represented as null or empty array
	jsonStr := string(data)
	if jsonStr != "null\n" && jsonStr != "null" && jsonStr != "[]\n" && jsonStr != "[]" {
		t.Errorf("MarshalJSONResults() nil should be 'null' or '[]', got %q", jsonStr)
	}
}

// Helper function.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
