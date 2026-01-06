package browser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// AuthRequirement represents detected authentication requirements.
type AuthRequirement struct {
	Type     string // "login_form", "http_auth", "session_expired", "auth_wall"
	URL      string
	Selector string // For login forms
	Hint     string // User-friendly hint
}

// DetectAuth checks if the current page requires authentication.
func (c *controller) DetectAuth(ctx context.Context, tabID string) (*AuthRequirement, error) {
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	// Get page info
	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	url := info.URL

	// Check for HTTP 401/403 via network monitor (if available)
	// This is more accurate than matching "401" in URL strings
	requests, _ := c.GetNetworkRequests(ctx, tabID, 0) // Get cached requests, no wait
	for _, req := range requests {
		if req.Status == 401 {
			return &AuthRequirement{
				Type: "http_auth",
				URL:  req.URL,
				Hint: "HTTP 401 - Authentication required",
			}, nil
		}
		if req.Status == 403 {
			return &AuthRequirement{
				Type: "forbidden",
				URL:  req.URL,
				Hint: "HTTP 403 - Access forbidden",
			}, nil
		}
	}

	// Check for common login form patterns
	loginSelectors := []string{
		"input[type='password']",
		"input[name*='password' i]",
		"input[id*='password' i]",
		"#password",
		".password",
	}

	for _, selector := range loginSelectors {
		elem, err := page.Element(selector)
		if err == nil && elem != nil {
			// Found a password field - check if it's a login form
			visible, err := elem.Visible()
			if err == nil && visible {
				// Look for username/email field nearby
				hasUsername := false
				usernameSelectors := []string{
					"input[type='email']",
					"input[type='text']",
					"input[name*='user' i]",
					"input[name*='email' i]",
					"#username",
					"#email",
				}

				for _, userSel := range usernameSelectors {
					if _, err := page.Element(userSel); err == nil {
						hasUsername = true

						break
					}
				}

				if hasUsername {
					return &AuthRequirement{
						Type:     "login_form",
						URL:      url,
						Selector: selector,
						Hint:     "Login form detected. Please enter your credentials.",
					}, nil
				}
			}
		}
	}

	// Check for common auth wall patterns by page title and URL
	authPatterns := []struct {
		pattern string
		hint    string
	}{
		{`(?i)sign in`, "Please sign in to continue"},
		{`(?i)log in`, "Please log in to continue"},
		{`(?i)authentication required`, "Authentication required"},
		{`(?i)unauthorized`, "You are not authorized to view this page"},
		{`(?i)401`, "Unauthorized (HTTP 401)"},
		{`(?i)403`, "Forbidden (HTTP 403)"},
		{`(?i)login required`, "Login required"},
	}

	title := info.Title
	for _, p := range authPatterns {
		matched, _ := regexp.MatchString(p.pattern, url)
		if matched {
			return &AuthRequirement{
				Type: "auth_wall",
				URL:  url,
				Hint: p.hint,
			}, nil
		}

		matched, _ = regexp.MatchString(p.pattern, title)
		if matched {
			return &AuthRequirement{
				Type: "auth_wall",
				URL:  url,
				Hint: p.hint,
			}, nil
		}
	}

	// Check for session expired indicators
	sessionExpiredPatterns := []string{
		"session has expired",
		"you have been logged out",
		"please login again",
		"session timeout",
	}

	for _, pattern := range sessionExpiredPatterns {
		// Search in page body
		results, err := page.Elements(fmt.Sprintf("*:contains-text('%s')", pattern))
		if err == nil && len(results) > 0 {
			return &AuthRequirement{
				Type: "session_expired",
				URL:  url,
				Hint: "Your session has expired. Please log in again.",
			}, nil
		}
	}

	// No authentication requirement detected
	return nil, nil //nolint:nilnil
}

// WaitForLogin waits for the user to manually login in the browser.
// It prompts the user and blocks until the user indicates they are done.
// Returns error if stdin is not a TTY (e.g., running in CI or headless mode).
func (c *controller) WaitForLogin(ctx context.Context, tabID string, auth *AuthRequirement) error {
	// Check if stdin is a TTY - can't prompt for input in non-interactive mode
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("cannot wait for login: stdin is not a terminal (running in non-interactive mode?)")
	}

	// Publish event for UI visibility
	// (Event publishing would happen here through event bus if available)

	fmt.Printf("\n🔐 %s\n", displayBold("Authentication Required"))
	fmt.Printf("   %s\n", displayMuted(auth.URL))
	fmt.Printf("\n%s\n", auth.Hint)
	fmt.Printf("\n%s\n", displayInfo("Please login in the browser window"))
	fmt.Printf("%s\n", displayInfo("Press Enter when done to continue the workflow"))

	// Wait for user input with context cancellation support
	inputCh := make(chan error, 1)
	go func() {
		fmt.Print("\n> ")
		reader := bufio.NewReader(os.Stdin)
		_, err := reader.ReadString('\n')
		inputCh <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-inputCh:
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
	}

	// Verify login succeeded by checking if we're still on the same page
	c.mu.RLock()
	page, err := c.getPage(tabID)
	c.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("get page: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("get page info: %w", err)
	}

	// Check if URL changed (login successful usually redirects)
	if info.URL != auth.URL && !isAuthURL(info.URL) {
		fmt.Printf("\n%s\n", displaySuccess("✓ Login successful! Continuing workflow..."))

		return nil
	}

	fmt.Printf("\n%s\n", displayWarning("⚠ Warning: Still on login page. If login failed, the workflow may encounter errors."))

	return nil
}

// isAuthURL checks if a URL is an authentication-related URL.
func isAuthURL(url string) bool {
	authURLPatterns := []string{
		"/login", "/signin", "/auth", "/session", "/oauth",
	}

	for _, pattern := range authURLPatterns {
		if strings.Contains(strings.ToLower(url), pattern) {
			return true
		}
	}

	return false
}

// Helper functions for display formatting.
// ANSI codes are only used when stdout is a TTY.

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func displayBold(s string) string {
	if !isTTY() {
		return s
	}

	return fmt.Sprintf("\033[1m%s\033[0m", s)
}

func displayMuted(s string) string {
	if !isTTY() {
		return s
	}

	return fmt.Sprintf("\033[90m%s\033[0m", s)
}

func displayInfo(s string) string {
	if !isTTY() {
		return "→ " + s
	}

	return fmt.Sprintf("\033[34m→ %s\033[0m", s)
}

func displaySuccess(s string) string {
	if !isTTY() {
		return s
	}

	return fmt.Sprintf("\033[32m%s\033[0m", s)
}

func displayWarning(s string) string {
	if !isTTY() {
		return s
	}

	return fmt.Sprintf("\033[33m%s\033[0m", s)
}
