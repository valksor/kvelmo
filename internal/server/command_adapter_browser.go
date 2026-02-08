package server

import (
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

// parseBrowserSubcommand returns a ParseFn that sets the browser subcommand
// and merges any JSON body fields into options.
func parseBrowserSubcommand(sub string) func(r *http.Request) (commands.Invocation, error) {
	return func(r *http.Request) (commands.Invocation, error) {
		opts := map[string]any{"subcommand": sub}
		if r.Body != nil && r.ContentLength > 0 {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				for k, v := range body {
					opts[k] = v
				}
			}
		}

		return commands.Invocation{
			Source:  commands.SourceAPI,
			Options: opts,
		}, nil
	}
}

// parseBrowserGetSubcommand returns a ParseFn for GET-only browser routes
// that require no parameters beyond the subcommand name.
func parseBrowserGetSubcommand(sub string) func(r *http.Request) (commands.Invocation, error) {
	return func(_ *http.Request) (commands.Invocation, error) {
		return commands.Invocation{
			Source: commands.SourceAPI,
			Options: map[string]any{
				"subcommand": sub,
			},
		}, nil
	}
}

// parseBrowserCookiesGetInvocation parses GET /api/v1/browser/cookies,
// reading an optional tab_id from the query string.
func parseBrowserCookiesGetInvocation(r *http.Request) (commands.Invocation, error) {
	opts := map[string]any{"subcommand": "cookies-get"}
	if tabID := r.URL.Query().Get("tab_id"); tabID != "" {
		opts["tab_id"] = tabID
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: opts,
	}, nil
}
