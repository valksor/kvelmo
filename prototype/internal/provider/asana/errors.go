package asana

import (
	"errors"
	"fmt"
	"net/http"

	providererrors "github.com/valksor/go-mehrhof/internal/provider/errors"
)

// Sentinel errors for the Asana provider.
var (
	ErrNoToken           = errors.New("asana token not found")
	ErrWorkspaceRequired = errors.New("workspace GID is required")
	ErrProjectRequired   = errors.New("project GID is required for list operations")
	ErrInvalidReference  = errors.New("invalid asana reference")
	ErrTaskNotFound      = errors.New("task not found")
	ErrRateLimited       = errors.New("rate limited")
)

// wrapAPIError wraps API errors with appropriate provider error types.
func wrapAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Check for HTTP errors via interface
	var httpErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpErr) {
		switch httpErr.HTTPStatusCode() {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %w", providererrors.ErrUnauthorized, err)
		case http.StatusForbidden:
			return fmt.Errorf("%w: %w", providererrors.ErrUnauthorized, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %w", ErrTaskNotFound, err)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %w", ErrRateLimited, err)
		}
	}

	return fmt.Errorf("asana API error: %w", err)
}
