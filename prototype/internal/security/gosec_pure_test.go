package security

import (
	"errors"
	"testing"
)

func TestMapGosecSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     Severity
	}{
		{
			name:     "high severity",
			severity: "HIGH",
			want:     SeverityHigh,
		},
		{
			name:     "medium severity",
			severity: "MEDIUM",
			want:     SeverityMedium,
		},
		{
			name:     "low severity",
			severity: "LOW",
			want:     SeverityLow,
		},
		{
			name:     "unknown severity defaults to info",
			severity: "UNKNOWN",
			want:     SeverityInfo,
		},
		{
			name:     "empty severity defaults to info",
			severity: "",
			want:     SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapGosecSeverity(tt.severity)
			if got != tt.want {
				t.Errorf("mapGosecSeverity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCommandNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "executable file not found",
			err:  errors.New("exec: \"gcc\": executable file not found in $PATH"),
			want: true,
		},
		{
			name: "command not found",
			err:  errors.New("exec: \"gcc\": command not found"),
			want: true,
		},
		{
			name: "no such file or directory",
			err:  errors.New("exec: \"gcc\": no such file or directory"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "mixed case executable file not found",
			err:  errors.New("exec: \"test\": Executable File Not Found in $PATH"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommandNotFound(tt.err)
			if got != tt.want {
				t.Errorf("isCommandNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
