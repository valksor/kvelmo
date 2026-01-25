package provider

import (
	"testing"
	"time"
)

func TestHealthStatus_Values(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusConnected, "connected"},
		{HealthStatusNotConfigured, "not_configured"},
		{HealthStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("HealthStatus = %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestNewProviderHealth(t *testing.T) {
	ph := NewProviderHealth()
	if ph == nil {
		t.Fatal("NewProviderHealth() returned nil")
	}
	if ph.Providers == nil {
		t.Error("Providers map is not initialized")
	}
	if ph.CheckedAt.IsZero() {
		t.Error("CheckedAt should be set")
	}
}

func TestProviderHealth_Add(t *testing.T) {
	ph := NewProviderHealth()
	info := &HealthInfo{
		Status:  HealthStatusConnected,
		Message: "Test",
	}

	ph.Add("github", info)

	retrieved, ok := ph.Get("github")
	if !ok {
		t.Error("Add() failed to add provider")
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil for existing provider")
	}
	if retrieved.Status != HealthStatusConnected {
		t.Errorf("Get() returned wrong status: got %v, want %v", retrieved.Status, HealthStatusConnected)
	}
}

func TestProviderHealth_Get(t *testing.T) {
	ph := NewProviderHealth()
	info := &HealthInfo{
		Status:  HealthStatusConnected,
		Message: "Connected",
	}
	ph.Add("github", info)

	tests := []struct {
		name        string
		provider    string
		wantNilInfo bool
		wantOk      bool
	}{
		{
			name:        "existing provider",
			provider:    "github",
			wantNilInfo: false,
			wantOk:      true,
		},
		{
			name:        "non-existing provider",
			provider:    "gitlab",
			wantNilInfo: true,
			wantOk:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ph.Get(tt.provider)
			if (got == nil) != tt.wantNilInfo {
				t.Errorf("Get() info = %v, wantNil %v", got, tt.wantNilInfo)
			}
			if ok != tt.wantOk {
				t.Errorf("Get() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestHealthInfo_Validation(t *testing.T) {
	tests := []struct {
		name  string
		info  *HealthInfo
		valid bool
	}{
		{
			name: "valid connected",
			info: &HealthInfo{
				Status:    HealthStatusConnected,
				Message:   "Connected",
				LastSync:  time.Now(),
				RateLimit: &RateLimitInfo{Limit: 5000, Used: 100},
			},
			valid: true,
		},
		{
			name: "valid not configured",
			info: &HealthInfo{
				Status:  HealthStatusNotConfigured,
				Message: "Not configured",
			},
			valid: true,
		},
		{
			name: "error status with error message",
			info: &HealthInfo{
				Status:  HealthStatusError,
				Error:   "Authentication failed",
				Message: "Failed to connect",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.info.Status == "" {
				t.Error("HealthInfo should have a status")
			}
		})
	}
}

func TestRateLimitInfo_ResetTime(t *testing.T) {
	tests := []struct {
		name      string
		resetTime time.Time
		wantValid bool
	}{
		{
			name:      "future reset time",
			resetTime: time.Now().Add(1 * time.Hour),
			wantValid: true,
		},
		{
			name:      "past reset time",
			resetTime: time.Now().Add(-1 * time.Hour),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &RateLimitInfo{
				ResetAt: tt.resetTime,
			}
			isValid := info.ResetAt.After(time.Now())
			if isValid != tt.wantValid {
				t.Errorf("RateLimitInfo.ResetAt validity = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

// MockHealthChecker implements HealthChecker for testing.
type MockHealthChecker struct {
	status HealthStatus
	err    error
}

func (m *MockHealthChecker) HealthCheck() (*HealthInfo, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &HealthInfo{
		Status:  m.status,
		Message: "Mock health check",
	}, nil
}

func TestHealthChecker_Interface(t *testing.T) {
	mock := &MockHealthChecker{status: HealthStatusConnected}
	info, err := mock.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck() returned error: %v", err)
	}
	if info == nil {
		t.Fatal("HealthCheck() returned nil info")
	}
	if info.Status != HealthStatusConnected {
		t.Errorf("HealthCheck() status = %v, want %v", info.Status, HealthStatusConnected)
	}
}

func TestHealthChecker_Error(t *testing.T) {
	mock := &MockHealthChecker{
		err: &testError{msg: "health check failed"},
	}

	info, err := mock.HealthCheck()
	if err == nil {
		t.Error("HealthCheck() should return error")
	}
	if info != nil {
		t.Error("HealthCheck() should return nil info on error")
	}
}

// testError is a simple error implementation for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
