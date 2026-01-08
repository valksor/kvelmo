//go:build no_browser
// +build no_browser

package browser

import (
	"context"
	"errors"
)

// DetectAuth stub returns an error - browser is disabled.
func (c *stubController) DetectAuth(ctx context.Context, tabID string) (*AuthRequirement, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

// WaitForLogin stub returns an error - browser is disabled.
func (c *stubController) WaitForLogin(ctx context.Context, tabID string, auth *AuthRequirement) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

// GetCookies stub returns an error - browser is disabled.
func (c *stubController) GetCookies(ctx context.Context) ([]Cookie, error) {
	return nil, errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

// SetCookies stub returns an error - browser is disabled.
func (c *stubController) SetCookies(ctx context.Context, cookies []Cookie) error {
	return errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}
