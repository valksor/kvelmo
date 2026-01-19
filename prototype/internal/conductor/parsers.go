package conductor

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// SimplifiedSpec represents a parsed simplified specification.
type SimplifiedSpec struct {
	Number  int
	Content string
}

// parseSimplifiedSpecifications extracts simplified specs from agent response.
// Expected format: --- specification-N.md ---\n[content]\n--- end ---.
func parseSimplifiedSpecifications(content string) []SimplifiedSpec {
	pattern := regexp.MustCompile(`(?s)---\s+specification-(\d+)\.md\s*---\s*\n(.*?)\n---\s+end\s*---`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		// Fallback: treat entire response as single spec
		return []SimplifiedSpec{{Number: 1, Content: strings.TrimSpace(content)}}
	}

	var specs []SimplifiedSpec
	for _, match := range matches {
		num, _ := strconv.Atoi(match[1])
		specs = append(specs, SimplifiedSpec{
			Number:  num,
			Content: strings.TrimSpace(match[2]),
		})
	}

	return specs
}

// parseSimplifiedCode extracts simplified code files from agent response.
// Expected format: --- path/to/file.ext ---\n[code]\n--- end ---.
func parseSimplifiedCode(content string) (map[string]string, error) {
	pattern := regexp.MustCompile(`(?s)---\s+(.+?)\s+---\s*\n(.*?)\n---\s+end\s*---`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return nil, errors.New("no simplified files found in response")
	}

	files := make(map[string]string)
	for _, match := range matches {
		filePath := strings.TrimSpace(match[1])
		files[filePath] = match[2]
	}

	return files, nil
}
