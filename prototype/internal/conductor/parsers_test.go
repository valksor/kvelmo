package conductor

import (
	"testing"
)

func TestParseSimplifiedSpecifications(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantCount    int
		wantFirstNum int
	}{
		{
			name: "single specification",
			content: `--- specification-1.md ---
This is the specification content.
--- end ---`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name: "multiple specifications",
			content: `--- specification-1.md ---
First spec content.
--- end ---
--- specification-2.md ---
Second spec content.
--- end ---`,
			wantCount:    2,
			wantFirstNum: 1,
		},
		{
			name: "specification with newlines",
			content: `--- specification-1.md ---
Line 1
Line 2
Line 3
--- end ---`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name:         "no specification markers - fallback",
			content:      `This is just plain text without markers.`,
			wantCount:    1,
			wantFirstNum: 1,
		},
		{
			name: "specification number 10",
			content: `--- specification-10.md ---
Content for spec 10.
--- end ---`,
			wantCount:    1,
			wantFirstNum: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs := parseSimplifiedSpecifications(tt.content)

			if len(specs) != tt.wantCount {
				t.Errorf("parseSimplifiedSpecifications() got %d specs, want %d", len(specs), tt.wantCount)

				return
			}

			if len(specs) > 0 && specs[0].Number != tt.wantFirstNum {
				t.Errorf("parseSimplifiedSpecifications() first spec number = %d, want %d", specs[0].Number, tt.wantFirstNum)
			}
		})
	}
}

func TestParseSimplifiedCode(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFiles []string
		wantError bool
	}{
		{
			name: "single file",
			content: `--- main.go ---
package main

func main() {
	println("hello")
}
--- end ---`,
			wantCount: 1,
			wantFiles: []string{"main.go"},
			wantError: false,
		},
		{
			name: "multiple files",
			content: `--- main.go ---
package main
--- end ---
--- utils/helper.go ---
package utils
--- end ---`,
			wantCount: 2,
			wantFiles: []string{"main.go", "utils/helper.go"},
			wantError: false,
		},
		{
			name:      "no file markers",
			content:   `Just plain text without any markers.`,
			wantCount: 0,
			wantFiles: []string{},
			wantError: true,
		},
		{
			name: "file with spaces in path",
			content: `--- path/to/my file.go ---
content
--- end ---`,
			wantCount: 1,
			wantFiles: []string{"path/to/my file.go"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := parseSimplifiedCode(tt.content)

			if (err != nil) != tt.wantError {
				t.Errorf("parseSimplifiedCode() error = %v, wantError %v", err, tt.wantError)

				return
			}

			if len(files) != tt.wantCount {
				t.Errorf("parseSimplifiedCode() got %d files, want %d", len(files), tt.wantCount)

				return
			}

			for _, wantFile := range tt.wantFiles {
				if _, ok := files[wantFile]; !ok {
					t.Errorf("parseSimplifiedCode() missing file %s", wantFile)
				}
			}
		})
	}
}
