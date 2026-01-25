package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v67/github"
	"github.com/valksor/go-mehrhof/internal/provider"
	providererrors "github.com/valksor/go-toolkit/errors"
)

// GitHub-specific error types.
var (
	ErrRepoNotDetected   = errors.New("could not detect repository from git remote")
	ErrRepoNotConfigured = errors.New("repository not configured")
	ErrIssueNotFound     = errors.New("issue not found")
	ErrInsufficientScope = errors.New("github token lacks required scope")
	ErrInvalidReference  = errors.New("invalid github reference")
)

// wrapAPIError converts GitHub API errors to typed errors with diagnostic hints.
func wrapAPIError(err error) error {
	if err == nil {
		return nil
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusUnauthorized:
			return &provider.DiagnosticError{
				Provider: "github",
				Message:  "authentication failed",
				Cause:    ghErr.Message,
				Suggestions: append(provider.CommonHints.Unauthorized,
					"Set GITHUB_TOKEN in .mehrhof/.env",
					"Or run: mehr provider validate github",
				),
				DocsURL: "https://mehrhof.valksor.com/docs/providers/github",
				Err:     err,
			}
		case http.StatusForbidden:
			if strings.Contains(ghErr.Message, "rate limit") {
				resetHeader := ghErr.Response.Header.Get("X-Ratelimit-Reset")

				return &provider.DiagnosticError{
					Provider: "github",
					Message:  "rate limit exceeded",
					Cause:    fmt.Sprintf("%s (retry after: %s)", ghErr.Message, resetHeader),
					Suggestions: append(provider.CommonHints.RateLimited,
						"Check current rate limit: mehr provider status github",
					),
					DocsURL: "https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting",
					Err:     err,
				}
			}

			return &provider.DiagnosticError{
				Provider: "github",
				Message:  "insufficient permissions",
				Cause:    ghErr.Message,
				Suggestions: append(provider.CommonHints.Unauthorized,
					"Token may lack required scopes (repo, issues, etc.)",
					"Check token permissions at: https://github.com/settings/tokens",
				),
				DocsURL: "https://mehrhof.valksor.com/docs/providers/github#authentication",
				Err:     err,
			}
		case http.StatusNotFound:
			return &provider.DiagnosticError{
				Provider: "github",
				Message:  "resource not found",
				Cause:    ghErr.Message,
				Suggestions: append(provider.CommonHints.NotFound,
					"Verify the issue/PR number is correct",
					"Check the repository owner and name",
				),
				DocsURL: "https://mehrhof.valksor.com/docs/providers/github#references",
				Err:     err,
			}
		}
	}

	// Use shared HTTP error wrapping for network and other common errors
	wrapped := providererrors.WrapHTTPError(err, "github", nil)
	if errors.Is(wrapped, providererrors.ErrNetworkError) {
		return &provider.DiagnosticError{
			Provider:    "github",
			Message:     "network error",
			Cause:       err.Error(),
			Suggestions: append(provider.CommonHints.Network, "Verify the provider API is accessible"),
			DocsURL:     "https://mehrhof.valksor.com/docs/providers/github#troubleshooting",
			Err:         wrapped,
		}
	}

	return wrapped
}
