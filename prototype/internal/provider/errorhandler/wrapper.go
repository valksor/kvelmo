// Package errorhandler provides common HTTP error wrapping utilities for providers.
//
// Each provider has similar error wrapping logic that maps HTTP status codes
// to sentinel errors. This package consolidates that pattern while allowing
// providers to customize the mapping for their specific needs.
package errorhandler

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	providererrors "github.com/valksor/go-toolkit/errors"
)

// StatusMapping maps HTTP status codes to sentinel errors.
// Providers define their own mappings based on API behavior.
type StatusMapping map[int]error

// DefaultMapping provides standard HTTP status to error mappings.
// Providers can use this as a base and override specific codes.
func DefaultMapping() StatusMapping {
	return StatusMapping{
		http.StatusUnauthorized:       providererrors.ErrUnauthorized,
		http.StatusForbidden:          providererrors.ErrUnauthorized,
		http.StatusNotFound:           providererrors.ErrNotFound,
		http.StatusTooManyRequests:    providererrors.ErrRateLimited,
		http.StatusServiceUnavailable: providererrors.ErrNetworkError,
		http.StatusGatewayTimeout:     providererrors.ErrNetworkError,
		http.StatusBadGateway:         providererrors.ErrNetworkError,
	}
}

// WrapAPIError wraps an error with provider context using the given status mapping.
// It checks for HTTP errors and network errors, wrapping them with appropriate sentinels.
//
// Parameters:
//   - providerName: Name of the provider for error messages (e.g., "github", "notion")
//   - err: The error to wrap
//   - mapping: HTTP status code to sentinel error mapping
//
// Returns nil if err is nil.
func WrapAPIError(providerName string, err error, mapping StatusMapping) error {
	if err == nil {
		return nil
	}

	// Check for HTTP errors via interface
	var httpErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpErr) {
		code := httpErr.HTTPStatusCode()
		if sentinel, ok := mapping[code]; ok {
			return fmt.Errorf("%w: %w", sentinel, err)
		}
		// No mapping found, return generic wrapped error
		return fmt.Errorf("%s API error (HTTP %d): %w", providerName, code, err)
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %w", providererrors.ErrNetworkError, err)
	}

	// Return with provider context
	return fmt.Errorf("%s API error: %w", providerName, err)
}

// Wrap is a convenience function that uses the default mapping.
// Use WrapAPIError when you need custom status code mappings.
func Wrap(providerName string, err error) error {
	return WrapAPIError(providerName, err, DefaultMapping())
}

// MergeMapping creates a new mapping by merging base with overrides.
// Overrides take precedence for any duplicate keys.
func MergeMapping(base, overrides StatusMapping) StatusMapping {
	result := make(StatusMapping, len(base)+len(overrides))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overrides {
		result[k] = v
	}

	return result
}
