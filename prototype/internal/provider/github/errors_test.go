package github

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v67/github"
)

func TestWrapAPIError_NetworkError(t *testing.T) {
	// Test network error wrapping with diagnostic hints
	netErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	got := wrapAPIError(netErr)

	if got == nil {
		t.Fatal("wrapAPIError() = nil, want error")
	}

	// The underlying error should wrap network error
	var netErrCheck *net.OpError
	if !errors.As(got, &netErrCheck) {
		t.Errorf("wrapAPIError() should wrap network error, got: %T", got)
	}

	// Should contain diagnostic hints
	errMsg := got.Error()
	if !strings.Contains(errMsg, "network error") {
		t.Errorf("wrapAPIError() should contain 'network error', got: %v", got)
	}
	if !strings.Contains(errMsg, "Recovery steps:") {
		t.Errorf("wrapAPIError() should contain recovery hints, got: %v", got)
	}
}

func TestWrapAPIError_Unauthorized(t *testing.T) {
	// Test 401 error with diagnostic hints
	authErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusUnauthorized},
		Message:  "Bad credentials",
	}
	got := wrapAPIError(authErr)

	if got == nil {
		t.Fatal("wrapAPIError() = nil, want error")
	}

	errMsg := got.Error()
	if !strings.Contains(errMsg, "authentication failed") {
		t.Errorf("wrapAPIError() should contain 'authentication failed', got: %v", got)
	}
	if !strings.Contains(errMsg, "GITHUB_TOKEN") {
		t.Errorf("wrapAPIError() should mention GITHUB_TOKEN, got: %v", got)
	}
	if !strings.Contains(errMsg, "Recovery steps:") {
		t.Errorf("wrapAPIError() should contain recovery hints, got: %v", got)
	}
}

func TestWrapAPIError_RateLimited(t *testing.T) {
	// Test rate limit error with diagnostic hints
	rateErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Ratelimit-Reset": []string{"1234567890"}},
		},
		Message: "API rate limit exceeded",
	}
	got := wrapAPIError(rateErr)

	if got == nil {
		t.Fatal("wrapAPIError() = nil, want error")
	}

	errMsg := got.Error()
	if !strings.Contains(errMsg, "rate limit exceeded") {
		t.Errorf("wrapAPIError() should contain 'rate limit exceeded', got: %v", got)
	}
	if !strings.Contains(errMsg, "retry after") {
		t.Errorf("wrapAPIError() should mention retry time, got: %v", got)
	}
}

func TestWrapAPIError_NotFound(t *testing.T) {
	// Test 404 error with diagnostic hints
	notFoundErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound},
		Message:  "Not Found",
	}
	got := wrapAPIError(notFoundErr)

	if got == nil {
		t.Fatal("wrapAPIError() = nil, want error")
	}

	errMsg := got.Error()
	if !strings.Contains(errMsg, "resource not found") {
		t.Errorf("wrapAPIError() should contain 'resource not found', got: %v", got)
	}
}

func TestWrapAPIError_500ServerError(t *testing.T) {
	// Test that 500 errors pass through unchanged
	serverErr := &github.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusInternalServerError},
		Message:  "Internal Server Error",
	}
	got := wrapAPIError(serverErr)

	if got == nil {
		t.Fatal("wrapAPIError() = nil, want error")
	}

	// Should pass through as-is, not wrapped
	var ghErr *github.ErrorResponse
	if !errors.As(got, &ghErr) {
		t.Errorf("wrapAPIError() should return original github.ErrorResponse for 500")
	}
}

func TestErrorVariables(t *testing.T) {
	// Test that GitHub-specific error variables are properly defined
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrRepoNotDetected", ErrRepoNotDetected, "could not detect repository from git remote"},
		{"ErrRepoNotConfigured", ErrRepoNotConfigured, "repository not configured"},
		{"ErrIssueNotFound", ErrIssueNotFound, "issue not found"},
		{"ErrInsufficientScope", ErrInsufficientScope, "github token lacks required scope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}
