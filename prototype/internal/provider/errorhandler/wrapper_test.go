package errorhandler

import (
	"errors"
	"net"
	"net/http"
	"testing"

	providererrors "github.com/valksor/go-toolkit/errors"
)

// mockHTTPError implements the HTTPStatusCode() interface.
type mockHTTPError struct {
	code    int
	message string
}

func (e *mockHTTPError) Error() string {
	return e.message
}

func (e *mockHTTPError) HTTPStatusCode() int {
	return e.code
}

// mockNetError implements net.Error.
type mockNetError struct {
	message   string
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return e.message }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestDefaultMapping(t *testing.T) {
	mapping := DefaultMapping()

	tests := []struct {
		code    int
		wantErr error
		wantOK  bool
	}{
		{http.StatusUnauthorized, providererrors.ErrUnauthorized, true},
		{http.StatusForbidden, providererrors.ErrUnauthorized, true},
		{http.StatusNotFound, providererrors.ErrNotFound, true},
		{http.StatusTooManyRequests, providererrors.ErrRateLimited, true},
		{http.StatusServiceUnavailable, providererrors.ErrNetworkError, true},
		{http.StatusGatewayTimeout, providererrors.ErrNetworkError, true},
		{http.StatusBadGateway, providererrors.ErrNetworkError, true},
		{http.StatusOK, nil, false},
		{http.StatusInternalServerError, nil, false},
	}

	for _, tt := range tests {
		got, ok := mapping[tt.code]
		if ok != tt.wantOK {
			t.Errorf("DefaultMapping()[%d] ok = %v, want %v", tt.code, ok, tt.wantOK)
		}
		if ok && !errors.Is(got, tt.wantErr) {
			t.Errorf("DefaultMapping()[%d] = %v, want %v", tt.code, got, tt.wantErr)
		}
	}
}

func TestWrapAPIError_Nil(t *testing.T) {
	err := WrapAPIError("test", nil, DefaultMapping())
	if err != nil {
		t.Errorf("WrapAPIError(nil) = %v, want nil", err)
	}
}

func TestWrapAPIError_HTTPError(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		wantSentinel error
	}{
		{"unauthorized", http.StatusUnauthorized, providererrors.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, providererrors.ErrUnauthorized},
		{"not found", http.StatusNotFound, providererrors.ErrNotFound},
		{"rate limited", http.StatusTooManyRequests, providererrors.ErrRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpErr := &mockHTTPError{code: tt.code, message: "test error"}
			err := WrapAPIError("github", httpErr, DefaultMapping())

			if err == nil {
				t.Fatal("WrapAPIError() returned nil")
			}
			if !errors.Is(err, tt.wantSentinel) {
				t.Errorf("WrapAPIError() does not wrap %v", tt.wantSentinel)
			}
		})
	}
}

func TestWrapAPIError_UnmappedHTTPError(t *testing.T) {
	httpErr := &mockHTTPError{code: http.StatusInternalServerError, message: "server error"}
	err := WrapAPIError("github", httpErr, DefaultMapping())

	if err == nil {
		t.Fatal("WrapAPIError() returned nil")
	}

	// Should contain provider name and status code
	errStr := err.Error()
	if !contains(errStr, "github") {
		t.Errorf("error %q does not contain provider name", errStr)
	}
	if !contains(errStr, "500") {
		t.Errorf("error %q does not contain status code", errStr)
	}
}

func TestWrapAPIError_NetError(t *testing.T) {
	netErr := &mockNetError{message: "connection refused", timeout: false, temporary: false}
	err := WrapAPIError("github", netErr, DefaultMapping())

	if err == nil {
		t.Fatal("WrapAPIError() returned nil")
	}
	if !errors.Is(err, providererrors.ErrNetworkError) {
		t.Errorf("WrapAPIError() does not wrap ErrNetworkError")
	}
}

func TestWrapAPIError_GenericError(t *testing.T) {
	genericErr := errors.New("some random error")
	err := WrapAPIError("notion", genericErr, DefaultMapping())

	if err == nil {
		t.Fatal("WrapAPIError() returned nil")
	}

	// Should contain provider name
	errStr := err.Error()
	if !contains(errStr, "notion") {
		t.Errorf("error %q does not contain provider name", errStr)
	}
	// Should wrap the original error
	if !errors.Is(err, genericErr) {
		t.Errorf("WrapAPIError() does not wrap original error")
	}
}

func TestWrap(t *testing.T) {
	httpErr := &mockHTTPError{code: http.StatusUnauthorized, message: "invalid token"}
	err := Wrap("linear", httpErr)

	if err == nil {
		t.Fatal("Wrap() returned nil")
	}
	if !errors.Is(err, providererrors.ErrUnauthorized) {
		t.Errorf("Wrap() does not wrap ErrUnauthorized")
	}
}

func TestMergeMapping(t *testing.T) {
	base := StatusMapping{
		http.StatusUnauthorized: providererrors.ErrUnauthorized,
		http.StatusNotFound:     providererrors.ErrNotFound,
	}

	customNotFound := errors.New("custom not found")
	customGone := errors.New("resource gone")

	overrides := StatusMapping{
		http.StatusNotFound: customNotFound, // Override existing
		http.StatusGone:     customGone,     // Add new
	}

	merged := MergeMapping(base, overrides)

	// Base mapping should be preserved
	if !errors.Is(merged[http.StatusUnauthorized], providererrors.ErrUnauthorized) {
		t.Error("MergeMapping() lost base mapping for 401")
	}

	// Override should take effect
	if !errors.Is(merged[http.StatusNotFound], customNotFound) {
		t.Error("MergeMapping() did not apply override for 404")
	}

	// New mapping should be added
	if !errors.Is(merged[http.StatusGone], customGone) {
		t.Error("MergeMapping() did not add new mapping for 410")
	}

	// Original maps should not be modified
	if base[http.StatusGone] != nil {
		t.Error("MergeMapping() modified base map")
	}
}

func TestWrapAPIError_CustomMapping(t *testing.T) {
	customNotFound := errors.New("task not found")
	customMapping := StatusMapping{
		http.StatusNotFound: customNotFound,
	}

	httpErr := &mockHTTPError{code: http.StatusNotFound, message: "not found"}
	err := WrapAPIError("clickup", httpErr, customMapping)

	if err == nil {
		t.Fatal("WrapAPIError() returned nil")
	}
	if !errors.Is(err, customNotFound) {
		t.Errorf("WrapAPIError() does not wrap custom sentinel error")
	}
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// Verify mockNetError implements net.Error.
var _ net.Error = (*mockNetError)(nil)
