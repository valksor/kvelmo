// Package apiclient provides a generic JSON API client for provider implementations.
//
// This package consolidates the common HTTP request/response handling patterns
// used across multiple provider clients, reducing duplication while allowing
// customization via authentication functions.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
)

// AuthFunc is a function that adds authentication headers to a request.
// Each provider implements this to add their specific auth (Bearer token, API key, etc).
type AuthFunc func(req *http.Request)

// Client is a generic JSON API client that handles the common request/response pattern.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authFn     AuthFunc
}

// New creates a new API client with the given base URL and authentication function.
func New(baseURL string, authFn AuthFunc) *Client {
	return &Client{
		httpClient: httpclient.NewHTTPClient(),
		baseURL:    baseURL,
		authFn:     authFn,
	}
}

// BearerAuth returns an AuthFunc that adds a Bearer token to requests.
func BearerAuth(token string) AuthFunc {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// BasicAuth returns an AuthFunc that adds Basic authentication to requests.
func BasicAuth(username, password string) AuthFunc {
	return func(req *http.Request) {
		req.SetBasicAuth(username, password)
	}
}

// HeaderAuth returns an AuthFunc that sets custom header(s) for authentication.
func HeaderAuth(headers map[string]string) AuthFunc {
	return func(req *http.Request) {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
}

// Do performs an HTTP request and decodes the JSON response into result.
// If result is nil, the response body is read but not decoded.
func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	respBody, err := c.DoRaw(ctx, method, path, body)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// DoRaw performs an HTTP request and returns the raw response body.
// This is useful when you need to handle the response manually.
func (c *Client) DoRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Apply authentication
	if c.authFn != nil {
		c.authFn(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, httpclient.NewHTTPError(resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DoWithHeaders performs an HTTP request with additional headers.
// This is useful for providers that require version headers or other custom headers.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, body, result any, headers map[string]string) error {
	respBody, err := c.DoRawWithHeaders(ctx, method, path, body, headers)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// DoRawWithHeaders performs an HTTP request with additional headers and returns raw body.
func (c *Client) DoRawWithHeaders(ctx context.Context, method, path string, body any, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Apply authentication
	if c.authFn != nil {
		c.authFn(req)
	}

	// Apply additional headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, httpclient.NewHTTPError(resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
