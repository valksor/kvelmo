package commands

import (
	"errors"
	"testing"
)

// TestFormatError verifies error formatting behavior.
func TestFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "simple error",
			err:      errors.New("failed"),
			expected: "Error: failed\n",
		},
		{
			name:     "multi-line error",
			err:      errors.New("line1\nline2"),
			expected: "line1\nline2\n",
		},
		{
			name:     "error with suggestions",
			err:      errors.New("cmd not found\n\nDid you mean: help"),
			expected: "cmd not found\n\nDid you mean: help\n",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "empty error",
			err:      errors.New(""),
			expected: "Error: \n",
		},
		{
			name:     "multi-line with trailing newline",
			err:      errors.New("error message\n"),
			expected: "error message\n\n",
		},
		{
			name:     "complex multi-line error",
			err:      errors.New("step 1 failed\nstep 2 failed\nstep 3 failed"),
			expected: "step 1 failed\nstep 2 failed\nstep 3 failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			if result != tt.expected {
				t.Errorf("FormatError() = %q, want %q", result, tt.expected)
			}
		})
	}
}
