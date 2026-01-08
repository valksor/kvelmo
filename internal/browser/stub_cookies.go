//go:build no_browser
// +build no_browser

package browser

import (
	"errors"
)

// DefaultConfigDir returns an error when browser is disabled.
func DefaultConfigDir() (string, error) {
	return "", errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

// CookiePath returns an error when browser is disabled.
func CookiePath(profile string) (string, error) {
	return "", errors.New("browser support is disabled in this build (built with 'no_browser' tag)")
}

// NewCookieStorage returns a stub storage when browser is disabled.
func NewCookieStorage(cookieDir string) *CookieStorage {
	return &CookieStorage{}
}

// Load returns empty cookies when browser is disabled.
func (cs *CookieStorage) Load(profile string) ([]Cookie, error) {
	return []Cookie{}, nil
}

// Save is a no-op when browser is disabled.
func (cs *CookieStorage) Save(profile string, cookies []Cookie) error {
	return nil
}
