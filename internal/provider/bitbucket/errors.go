package bitbucket

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Error types for the Bitbucket provider.
var (
	ErrNoToken           = errors.New("bitbucket token not found")
	ErrNoUsername        = errors.New("bitbucket username not configured")
	ErrRepoNotConfigured = errors.New("repository not configured")
	ErrIssueNotFound     = errors.New("issue not found")
	ErrRateLimited       = errors.New("bitbucket api rate limit exceeded")
	ErrNetworkError      = errors.New("network error communicating with bitbucket")
	ErrUnauthorized      = errors.New("bitbucket credentials unauthorized or expired")
	ErrInvalidReference  = errors.New("invalid bitbucket reference")
	ErrIssuesNotEnabled  = errors.New("issues not enabled for this repository")
)

// wrapAPIError converts Bitbucket API errors to typed errors.
func wrapAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Check for HTTP errors via interface
	var httpErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpErr) {
		switch httpErr.HTTPStatusCode() {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %w", ErrUnauthorized, err)
		case http.StatusForbidden:
			return fmt.Errorf("%w: %w", ErrUnauthorized, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %w", ErrIssueNotFound, err)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %w", ErrRateLimited, err)
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %w", ErrNetworkError, err)
	}

	return err
}
