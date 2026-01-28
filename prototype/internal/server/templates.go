package server

import "embed"

// templateFS embeds all HTML templates.
// This is used by views.NewRenderer to load templates.
//
//go:embed templates/*.html templates/partials/*.html templates/partials/empty_states/*.html
var templateFS embed.FS
