package commands

import "strings"

// capitalizeFirst capitalizes the first letter of a string.
// Used by renderResult in interactive_events.go for category headers.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
