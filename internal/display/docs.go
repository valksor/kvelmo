package display

import (
	"strings"

	"github.com/valksor/go-toolkit/version"
)

const (
	DocsBaseNightly = "https://valksor.com/docs/mehrhof/nightly"
	DocsBaseLatest  = "https://valksor.com/docs/mehrhof/latest"
)

// DocsURL returns the documentation base URL for the current build.
// Stable releases (v*) link to /docs/latest, all others to /docs/nightly.
func DocsURL() string {
	if strings.HasPrefix(version.Version, "v") {
		return DocsBaseLatest
	}

	return DocsBaseNightly
}
