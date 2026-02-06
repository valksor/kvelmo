package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-rod/rod/lib/proto"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// DefaultConfigDir returns the default mehrhof config directory.
// Returns ~/.valksor/mehrhof/ (or platform equivalent).
func DefaultConfigDir() (string, error) {
	return storage.GetMehrhofHomeDir()
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

// GetCookies retrieves all cookies from the browser.
func (c *controller) GetCookies(ctx context.Context) ([]Cookie, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.browser == nil {
		return nil, errNotFound("not connected")
	}

	pages, err := c.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("get pages: %w", err)
	}

	if len(pages) == 0 {
		return []Cookie{}, nil
	}

	result, err := proto.StorageGetCookies{}.Call(pages[0])
	if err != nil {
		return nil, fmt.Errorf("get cookies via CDP: %w", err)
	}

	cookies := make([]Cookie, len(result.Cookies))
	for i, rc := range result.Cookies {
		cookies[i] = cookieFromRod(*rc)
	}

	return cookies, nil
}

// SetCookies sets cookies in the browser.
func (c *controller) SetCookies(ctx context.Context, cookies []Cookie) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.browser == nil {
		return errNotFound("not connected")
	}

	pages, err := c.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	if len(pages) == 0 {
		return errors.New("no pages available to set cookies")
	}

	// Convert to Rod format
	rodCookies := make([]*proto.NetworkCookieParam, len(cookies))
	for i, cookie := range cookies {
		c := cookieToRod(cookie)
		rodCookies[i] = &c
	}

	// Set cookies via CDP
	setCookiesAction := proto.StorageSetCookies{Cookies: rodCookies}
	if err := setCookiesAction.Call(pages[0]); err != nil {
		return fmt.Errorf("set cookies via CDP: %w", err)
	}

	return nil
}

// loadCookies loads cookies from storage and sets them in the browser.
func (c *controller) loadCookies() error {
	if c.cookieStorage == nil {
		return errors.New("cookie storage not initialized")
	}

	cookies, err := c.cookieStorage.Load(c.cookieProfile)
	if err != nil {
		return fmt.Errorf("load cookies from storage: %w", err)
	}

	if len(cookies) == 0 {
		slog.Debug("no cookies to load", "profile", c.cookieProfile)

		return nil
	}

	// Get a page to set cookies
	pages, err := c.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	if len(pages) == 0 {
		slog.Debug("no pages available, skipping cookie load")

		return nil
	}

	// Convert to Rod format and set cookies
	rodCookies := make([]*proto.NetworkCookieParam, len(cookies))
	for i, cookie := range cookies {
		c := cookieToRod(cookie)
		rodCookies[i] = &c
	}

	// Set cookies via CDP
	setCookiesAction := proto.StorageSetCookies{Cookies: rodCookies}
	if err := setCookiesAction.Call(pages[0]); err != nil {
		return fmt.Errorf("set cookies via CDP: %w", err)
	}

	slog.Info("loaded cookies", "profile", c.cookieProfile, "count", len(cookies))

	return nil
}

// saveCookies saves current browser cookies to storage.
func (c *controller) saveCookies() error {
	if c.cookieStorage == nil {
		return errors.New("cookie storage not initialized")
	}

	// Get all cookies from the browser
	pages, err := c.browser.Pages()
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}

	if len(pages) == 0 {
		slog.Debug("no pages available, skipping cookie save")

		return nil
	}

	result, err := proto.StorageGetCookies{}.Call(pages[0])
	if err != nil {
		return fmt.Errorf("get cookies via CDP: %w", err)
	}

	// Convert to our format
	cookies := make([]Cookie, len(result.Cookies))
	for i, rc := range result.Cookies {
		cookies[i] = cookieFromRod(*rc)
	}

	// Save to storage
	if err := c.cookieStorage.Save(c.cookieProfile, cookies); err != nil {
		return err
	}

	slog.Info("saved cookies", "profile", c.cookieProfile, "count", len(cookies))

	return nil
}

// cookieToRod converts our Cookie type to Rod's proto.NetworkCookieParam.
func cookieToRod(c Cookie) proto.NetworkCookieParam {
	return proto.NetworkCookieParam{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HTTPOnly: c.HTTPOnly,
		SameSite: proto.NetworkCookieSameSite(c.SameSite),
		Expires:  proto.TimeSinceEpoch(c.Expires),
	}
}

// cookieFromRod converts Rod's proto.NetworkCookie to our Cookie type.
func cookieFromRod(c proto.NetworkCookie) Cookie {
	return Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HTTPOnly: c.HTTPOnly,
		SameSite: string(c.SameSite),
		Expires:  int64(c.Expires),
	}
}
