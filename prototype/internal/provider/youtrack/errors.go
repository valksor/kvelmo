package youtrack

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
	providererrors "github.com/valksor/go-toolkit/errors"
)

// newHTTPError creates a new HTTP error using the shared httpclient.HTTPError type.
func newHTTPError(code int, message string) *httpclient.HTTPError {
	return httpclient.NewHTTPError(code, message)
}

// wrapAPIError wraps an error with appropriate typed errors using shared error package.
func wrapAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already a provider error
	if errors.Is(err, providererrors.ErrNoToken) ||
		errors.Is(err, providererrors.ErrNotFound) ||
		errors.Is(err, providererrors.ErrRateLimited) ||
		errors.Is(err, providererrors.ErrNetworkError) ||
		errors.Is(err, providererrors.ErrUnauthorized) ||
		errors.Is(err, providererrors.ErrInvalidReference) {
		return err
	}

	// Check for HTTP errors via interface
	var httpErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpErr) {
		switch httpErr.HTTPStatusCode() {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %w", providererrors.ErrUnauthorized, err)
		case http.StatusForbidden:
			return fmt.Errorf("%w: %w", providererrors.ErrRateLimited, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %w", providererrors.ErrNotFound, err)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %w", providererrors.ErrRateLimited, err)
		case http.StatusServiceUnavailable:
			return fmt.Errorf("%w: %w", providererrors.ErrRateLimited, err)
		default:
			return err
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %w", providererrors.ErrNetworkError, err)
	}

	return err
}
