package display

import (
	"testing"

	"github.com/valksor/go-toolkit/version"
)

func TestDocsURL(t *testing.T) {
	// Save original version
	originalVersion := version.Version
	defer func() { version.Version = originalVersion }()

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "stable release version",
			version: "v1.0.0",
			want:    DocsBaseLatest,
		},
		{
			name:    "stable release with v prefix",
			version: "v2.5.3",
			want:    DocsBaseLatest,
		},
		{
			name:    "development version",
			version: "1.0.0-dev",
			want:    DocsBaseNightly,
		},
		{
			name:    "nightly build",
			version: "nightly",
			want:    DocsBaseNightly,
		},
		{
			name:    "empty version",
			version: "",
			want:    DocsBaseNightly,
		},
		{
			name:    "unstable version",
			version: "0.1.0-beta",
			want:    DocsBaseNightly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version.Version = tt.version
			got := DocsURL()
			if got != tt.want {
				t.Errorf("DocsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
