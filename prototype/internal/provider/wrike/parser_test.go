package wrike

import (
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// ParseReference tests
// ──────────────────────────────────────────────────────────────────────────────

func TestParseReference(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantTaskID    string
		wantPermalink string
		errContains   string
		wantErr       bool
	}{
		{
			name:       "wrike scheme with numeric ID",
			input:      "wrike:1234567890",
			wantTaskID: "1234567890",
		},
		{
			name:       "wrike scheme with API ID",
			input:      "wrike:IEAAJXXXXXXXX",
			wantTaskID: "IEAAJXXXXXXXX",
		},
		{
			name:       "wk short scheme with numeric ID",
			input:      "wk:1234567890",
			wantTaskID: "1234567890",
		},
		{
			name:       "wk short scheme with API ID",
			input:      "wk:IEAAJXXXXXXXX",
			wantTaskID: "IEAAJXXXXXXXX",
		},
		{
			name:          "permalink URL",
			input:         "https://www.wrike.com/open.htm?id=1234567890",
			wantTaskID:    "1234567890",
			wantPermalink: "https://www.wrike.com/open.htm?id=1234567890",
		},
		{
			name:          "wrike scheme with permalink URL (FIXED BUG)",
			input:         "wrike:https://www.wrike.com/open.htm?id=4341623772",
			wantTaskID:    "4341623772",
			wantPermalink: "https://www.wrike.com/open.htm?id=4341623772",
		},
		{
			name:          "wk short scheme with permalink URL (FIXED BUG)",
			input:         "wk:https://www.wrike.com/open.htm?id=4341623772",
			wantTaskID:    "4341623772",
			wantPermalink: "https://www.wrike.com/open.htm?id=4341623772",
		},
		{
			name:          "permalink with more digits",
			input:         "https://www.wrike.com/open.htm?id=12345678901234",
			wantTaskID:    "12345678901234",
			wantPermalink: "https://www.wrike.com/open.htm?id=12345678901234",
		},
		{
			name:       "bare API ID format",
			input:      "IEAAJXXXXXXXX",
			wantTaskID: "IEAAJXXXXXXXX",
		},
		{
			name:       "bare API ID with numbers",
			input:      "IEAAJ123456789",
			wantTaskID: "IEAAJ123456789",
		},
		{
			name:       "new Wrike API ID format (v4)",
			input:      "MAAAAAECx-vc",
			wantTaskID: "MAAAAAECx-vc",
		},
		{
			name:       "wrike scheme with new API ID format (v4)",
			input:      "wrike:MAAAAAECx-vc",
			wantTaskID: "MAAAAAECx-vc",
		},
		{
			name:       "bare numeric ID (10 digits)",
			input:      "1234567890",
			wantTaskID: "1234567890",
		},
		{
			name:       "bare numeric ID (more than 10 digits)",
			input:      "123456789012345",
			wantTaskID: "123456789012345",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid format - too short",
			input:       "wrike:123",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "invalid format - letters only",
			input:       "wrike:abcdef",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "invalid format - mix that doesn't match",
			input:       "wrike:123abc",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "not a wrike reference - file scheme",
			input:       "file:task.md",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "not a wrike reference - github scheme",
			input:       "github:123",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "numeric ID with 9 digits (too short)",
			input:       "123456789",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "API ID with lowercase (must start with uppercase)",
			input:       "ieaajxxxxxxxx",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "API ID starting with number (must start with uppercase letter)",
			input:       "1AAAJXXXXXXXX",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "API ID too short (needs at least 2 chars)",
			input:       "A",
			wantErr:     true,
			errContains: "unrecognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseReference(%q) expected error, got nil", tt.input)

					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseReference(%q) error = %q, want to contain %q", tt.input, err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Errorf("ParseReference(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got.TaskID != tt.wantTaskID {
				t.Errorf("ParseReference(%q).TaskID = %q, want %q", tt.input, got.TaskID, tt.wantTaskID)
			}

			if got.Permalink != tt.wantPermalink {
				t.Errorf("ParseReference(%q).Permalink = %q, want %q", tt.input, got.Permalink, tt.wantPermalink)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ExtractNumericID tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractNumericID(t *testing.T) {
	tests := []struct {
		name      string
		permalink string
		want      string
	}{
		{
			name:      "valid permalink with 10 digits",
			permalink: "https://www.wrike.com/open.htm?id=1234567890",
			want:      "1234567890",
		},
		{
			name:      "valid permalink with more digits",
			permalink: "https://www.wrike.com/open.htm?id=12345678901234",
			want:      "12345678901234",
		},
		{
			name:      "permalink with additional query params",
			permalink: "https://www.wrike.com/open.htm?id=1234567890&other=value",
			want:      "1234567890",
		},
		{
			name:      "not a permalink - regular URL",
			permalink: "https://example.com/page",
			want:      "",
		},
		{
			name:      "not a permalink - missing query param",
			permalink: "https://www.wrike.com/open.htm",
			want:      "",
		},
		{
			name:      "not a permalink - wrong query param name",
			permalink: "https://www.wrike.com/open.htm?task=1234567890",
			want:      "",
		},
		{
			name:      "empty string",
			permalink: "",
			want:      "",
		},
		{
			name:      "just text",
			permalink: "not a url",
			want:      "",
		},
		{
			name:      "ID in middle of URL",
			permalink: "https://www.wrike.com/open.htm?id=1234567890&extra=true",
			want:      "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractNumericID(tt.permalink)
			if got != tt.want {
				t.Errorf("ExtractNumericID(%q) = %q, want %q", tt.permalink, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// BuildPermalinkURL tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildPermalinkURL(t *testing.T) {
	tests := []struct {
		name      string
		numericID string
		want      string
	}{
		{
			name:      "valid 10-digit ID",
			numericID: "1234567890",
			want:      "https://www.wrike.com/open.htm?id=1234567890",
		},
		{
			name:      "valid longer numeric ID",
			numericID: "4341623772",
			want:      "https://www.wrike.com/open.htm?id=4341623772",
		},
		{
			name:      "very long numeric ID",
			numericID: "123456789012345",
			want:      "https://www.wrike.com/open.htm?id=123456789012345",
		},
		{
			name:      "API ID format - returns empty",
			numericID: "IEAAJXXXXXXXX",
			want:      "",
		},
		{
			name:      "too short - returns empty",
			numericID: "123456789",
			want:      "",
		},
		{
			name:      "empty string - returns empty",
			numericID: "",
			want:      "",
		},
		{
			name:      "mixed alphanumeric - returns empty",
			numericID: "123abc5678",
			want:      "",
		},
		{
			name:      "just letters - returns empty",
			numericID: "abcdefghij",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildPermalinkURL(tt.numericID); got != tt.want {
				t.Errorf("BuildPermalinkURL(%q) = %q, want %q", tt.numericID, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Ref.String tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRefString(t *testing.T) {
	tests := []struct {
		name string
		ref  Ref
		want string
	}{
		{
			name: "with permalink",
			ref: Ref{
				TaskID:    "1234567890",
				Permalink: "https://www.wrike.com/open.htm?id=1234567890",
			},
			want: "https://www.wrike.com/open.htm?id=1234567890",
		},
		{
			name: "without permalink",
			ref: Ref{
				TaskID:    "IEAAJXXXXXXXX",
				Permalink: "",
			},
			want: "IEAAJXXXXXXXX",
		},
		{
			name: "empty ref",
			ref: Ref{
				TaskID:    "",
				Permalink: "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("Ref.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
