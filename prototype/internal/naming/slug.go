package naming

import (
	"github.com/valksor/go-toolkit/slug"
)

// Slugify converts a title to a URL-safe slug suitable for branch names.
// It lowercases, removes diacritics, replaces spaces/special chars with hyphens,
// and truncates to maxLen characters (at word boundary if possible).
//
// This delegates to go-toolkit/slug.Slugify for the implementation.
func Slugify(title string, maxLen int) string {
	return slug.Slugify(title, maxLen)
}
