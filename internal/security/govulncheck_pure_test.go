package security

import (
	"testing"
)

func TestDeriveSeverityFromID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want Severity
	}{
		{
			name: "CVE ID returns high severity",
			id:   "CVE-2023-1234",
			want: SeverityHigh,
		},
		{
			name: "GHSA ID returns medium severity",
			id:   "GHSA-abc1-2345-6789",
			want: SeverityMedium,
		},
		{
			name: "unknown ID format defaults to medium",
			id:   "UNKNOWN-123",
			want: SeverityMedium,
		},
		{
			name: "empty ID defaults to medium",
			id:   "",
			want: SeverityMedium,
		},
		{
			name: "cve-lowercase not recognized as CVE",
			id:   "cve-2023-1234",
			want: SeverityMedium,
		},
		{
			name: "ghsa-lowercase not recognized as GHSA",
			id:   "ghsa-abc1-2345-6789",
			want: SeverityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveSeverityFromID(tt.id)
			if got != tt.want {
				t.Errorf("deriveSeverityFromID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCVE(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "CVE ID is extracted",
			id:   "CVE-2023-1234",
			want: "CVE-2023-1234",
		},
		{
			name: "GHSA ID returns empty",
			id:   "GHSA-abc1-2345-6789",
			want: "",
		},
		{
			name: "empty string returns empty",
			id:   "",
			want: "",
		},
		{
			name: "unknown format returns empty",
			id:   "OSV-2023-123",
			want: "",
		},
		{
			name: "CVE in middle of string is not extracted (must start with CVE)",
			id:   "prefix-CVE-2023-1234-suffix",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCVE(tt.id)
			if got != tt.want {
				t.Errorf("extractCVE() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractAliases(t *testing.T) {
	tests := []struct {
		name   string
		osvMap map[string]interface{}
		want   []string
	}{
		{
			name:   "nil map returns empty slice",
			osvMap: nil,
			want:   []string{},
		},
		{
			name:   "no aliases key returns empty slice",
			osvMap: map[string]interface{}{},
			want:   []string{},
		},
		{
			name: "aliases key with string array",
			osvMap: map[string]interface{}{
				"aliases": []interface{}{"CVE-2023-001", "CVE-2023-002"},
			},
			want: []string{"CVE-2023-001", "CVE-2023-002"},
		},
		{
			name: "aliases with non-string elements are filtered",
			osvMap: map[string]interface{}{
				"aliases": []interface{}{"CVE-2023-001", 123, true},
			},
			want: []string{"CVE-2023-001"},
		},
		{
			name: "empty aliases array returns empty slice",
			osvMap: map[string]interface{}{
				"aliases": []interface{}{},
			},
			want: []string{},
		},
		{
			name: "aliases key with non-array value returns empty",
			osvMap: map[string]interface{}{
				"aliases": "not-an-array",
			},
			want: []string{},
		},
		{
			name: "mixed valid types",
			osvMap: map[string]interface{}{
				"aliases": []interface{}{"CVE-2023-001", "GHSA-abc1-2345", "CVE-2023-002"},
			},
			want: []string{"CVE-2023-001", "GHSA-abc1-2345", "CVE-2023-002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAliases(tt.osvMap)
			if len(got) != len(tt.want) {
				t.Errorf("extractAliases() len = %d, want %d", len(got), len(tt.want))
			}
			for i, wantVal := range tt.want {
				if i < len(got) && got[i] != wantVal {
					t.Errorf("extractAliases()[%d] = %q, want %q", i, got[i], wantVal)
				}
			}
		})
	}
}
