package azuredevops

import (
	"errors"
	"fmt"
	"net/http"

	providererrors "github.com/valksor/go-mehrhof/internal/provider/errors"
)

// Sentinel errors for the Azure DevOps provider.
var (
	ErrNoToken          = errors.New("azure devops token not found")
	ErrOrgRequired      = errors.New("organization is required")
	ErrProjectRequired  = errors.New("project is required")
	ErrInvalidReference = errors.New("invalid azure devops reference")
	ErrWorkItemNotFound = errors.New("work item not found")
	ErrRateLimited      = errors.New("rate limited")
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
			return fmt.Errorf("%w: %w", ErrWorkItemNotFound, err)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %w", ErrRateLimited, err)
		}
	}

	return fmt.Errorf("azure devops API error: %w", err)
}
