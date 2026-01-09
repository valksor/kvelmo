package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfigDir returns the default mehrhof config directory.
// Returns ~/.mehrhof/ (or platform equivalent).
func DefaultConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	return filepath.Join(homeDir, ".mehrhof"), nil
}

// CookiePath returns the path to the cookie file for a given profile.
func CookiePath(profile string) (string, error) {
	configDir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}

	if profile == "" {
		profile = "default"
	}

	return filepath.Join(configDir, "cookies-"+profile+".json"), nil
}

// NewCookieStorage creates a new cookie storage manager.
func NewCookieStorage(cookieDir string) *CookieStorage {
	return &CookieStorage{cookieDir: cookieDir}
}

// Load loads cookies from storage for the given profile.
func (cs *CookieStorage) Load(profile string) ([]Cookie, error) {
	cookiePath, err := cs.cookiePathForProfile(profile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Cookie{}, nil // No cookies yet
		}

		return nil, fmt.Errorf("read cookie file: %w", err)
	}

	var cookies []Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, fmt.Errorf("unmarshal cookies: %w", err)
	}

	return cookies, nil
}

// Save saves cookies to storage for the given profile.
func (cs *CookieStorage) Save(profile string, cookies []Cookie) error {
	cookiePath, err := cs.cookiePathForProfile(profile)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cookiePath), 0o755); err != nil {
		return fmt.Errorf("create cookie dir: %w", err)
	}

	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cookies: %w", err)
	}

	// Use restricted permissions (0600) - cookies may contain session tokens
	if err := os.WriteFile(cookiePath, data, 0o600); err != nil {
		return fmt.Errorf("write cookie file: %w", err)
	}

	return nil
}

// cookiePathForProfile returns the cookie file path for a profile.
// Uses cs.cookieDir if set, otherwise uses default location.
func (cs *CookieStorage) cookiePathForProfile(profile string) (string, error) {
	if cs.cookieDir != "" {
		if profile == "" {
			profile = "default"
		}

		return filepath.Join(cs.cookieDir, "cookies-"+profile+".json"), nil
	}

	return CookiePath(profile)
}
