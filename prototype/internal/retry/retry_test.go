package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// mockTemporaryError is an error that implements Temporary() interface.
type mockTemporaryError struct {
	error

	temporary bool
}

func (e mockTemporaryError) Temporary() bool {
	return e.temporary
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxAttempts != DefaultMaxAttempts {
		t.Errorf("expected MaxAttempts %d, got %d", DefaultMaxAttempts, config.MaxAttempts)
	}
	if config.BaseDelay != DefaultBaseDelay {
		t.Errorf("expected BaseDelay %v, got %v", DefaultBaseDelay, config.BaseDelay)
	}
	if config.MaxDelay != DefaultMaxDelay {
		t.Errorf("expected MaxDelay %v, got %v", DefaultMaxDelay, config.MaxDelay)
	}
	if config.ExponentialBase != DefaultExponentialBase {
		t.Errorf("expected ExponentialBase %f, got %f", DefaultExponentialBase, config.ExponentialBase)
	}
	if !config.Jitter {
		t.Errorf("expected Jitter true, got false")
	}
}

func TestCalculateDelay(t *testing.T) {
	config := Config{
		BaseDelay:       1 * time.Second,
		MaxDelay:        60 * time.Second,
		ExponentialBase: 2.0,
		Jitter:          false, // Disable jitter for predictable tests
	}

	tests := []struct {
		name        string
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "first retry",
			attempt:     0,
			expectedMin: 1 * time.Second,
			expectedMax: 1 * time.Second,
		},
		{
			name:        "second retry",
			attempt:     1,
			expectedMin: 2 * time.Second,
			expectedMax: 2 * time.Second,
		},
		{
			name:        "third retry",
			attempt:     2,
			expectedMin: 4 * time.Second,
			expectedMax: 4 * time.Second,
		},
		{
			name:        "fourth retry",
			attempt:     3,
			expectedMin: 8 * time.Second,
			expectedMax: 8 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := config.CalculateDelay(tt.attempt)
			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("expected delay between %v and %v, got %v", tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestCalculateDelayWithMax(t *testing.T) {
	config := Config{
		BaseDelay:       1 * time.Second,
		MaxDelay:        5 * time.Second,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	// With max delay of 5s, even high attempts should be capped
	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"attempt 0", 0, 1 * time.Second},
		{"attempt 1", 1, 2 * time.Second},
		{"attempt 2", 2, 4 * time.Second},
		{"attempt 3", 3, 5 * time.Second}, // Capped at MaxDelay
		{"attempt 4", 4, 5 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := config.CalculateDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("expected delay %v, got %v", tt.expected, delay)
			}
		})
	}
}

func TestCalculateDelayWithJitter(t *testing.T) {
	config := Config{
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		ExponentialBase: 2.0,
		Jitter:          true,
	}

	// With jitter enabled, delays should vary but be in reasonable range
	attempts := make([]time.Duration, 100)
	for i := range 100 {
		attempts[i] = config.CalculateDelay(1) // Second attempt = 2x base
	}

	// Find minDelay and maxDelay
	minDelay := attempts[0]
	maxDelay := attempts[0]
	for _, d := range attempts[1:] {
		if d < minDelay {
			minDelay = d
		}
		if d > maxDelay {
			maxDelay = d
		}
	}

	// Without jitter: would be exactly 200ms
	// With jitter (±25%): should be between 150ms and 250ms
	expected := 200 * time.Millisecond
	minExpected := time.Duration(float64(expected) * 0.75)
	maxExpected := time.Duration(float64(expected) * 1.25)

	if minDelay < minExpected {
		t.Errorf("minimum delay %v is less than expected minimum %v", minDelay, minExpected)
	}
	if maxDelay > maxExpected {
		t.Errorf("maximum delay %v exceeds expected maximum %v", maxDelay, maxExpected)
	}
}

func TestIsRetryable(t *testing.T) {
	config := DefaultConfig()

	// Non-temporary error
	nonTempErr := errors.New("permanent error")
	if config.IsRetryable(nonTempErr) {
		t.Error("non-temporary error should not be retryable")
	}

	// Temporary error
	tempErr := mockTemporaryError{error: errors.New("temporary error"), temporary: true}
	if !config.IsRetryable(tempErr) {
		t.Error("temporary error should be retryable")
	}

	// Nil error
	if config.IsRetryable(nil) {
		t.Error("nil error should not be retryable")
	}
}

func TestDo_Success(t *testing.T) {
	config := Config{
		MaxAttempts:     3,
		BaseDelay:       10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	ctx := context.Background()
	callCount := 0

	err := config.Do(ctx, func() error {
		callCount++
		if callCount < 2 {
			return mockTemporaryError{error: errors.New("temp fail"), temporary: true}
		}

		return nil
	})
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	config := Config{
		MaxAttempts:     3,
		BaseDelay:       10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	ctx := context.Background()
	callCount := 0

	err := config.Do(ctx, func() error {
		callCount++

		return mockTemporaryError{error: errors.New("always fails"), temporary: true}
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	config := Config{
		MaxAttempts:     3,
		BaseDelay:       10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	ctx := context.Background()
	callCount := 0

	err := config.Do(ctx, func() error {
		callCount++

		return errors.New("permanent failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries for non-retryable error), got %d", callCount)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	config := Config{
		MaxAttempts:     10,
		BaseDelay:       1 * time.Second,
		MaxDelay:        10 * time.Second,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Cancel immediately
	cancel()

	err := config.Do(ctx, func() error {
		callCount++

		return mockTemporaryError{error: errors.New("temp fail"), temporary: true}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
	// Should only have tried once before context cancellation
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryContext(t *testing.T) {
	config := Config{
		MaxAttempts:     3,
		BaseDelay:       10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		ExponentialBase: 2.0,
		Jitter:          false,
	}

	ctx := context.Background()
	rc := NewRetryContext(config)

	// Simulate 3 retry attempts
	for i := range 3 {
		err := mockTemporaryError{error: fmt.Errorf("attempt %d failed", i), temporary: true}
		if !rc.HandleError(err) {
			break
		}
		_ = rc.Delay(ctx)
	}

	if rc.AttemptCount() != 3 {
		t.Errorf("expected AttemptCount 3, got %d", rc.AttemptCount())
	}
	if rc.LastError() == nil {
		t.Error("expected last error to be set")
	}
	// After 3 attempts, should not be able to continue.
	if rc.ShouldContinue() {
		t.Error("expected ShouldContinue to return false after MaxAttempts")
	}
}

// Benchmark for delay calculation.
func BenchmarkCalculateDelay(b *testing.B) {
	config := DefaultConfig()
	b.ResetTimer()
	for range b.N {
		_ = config.CalculateDelay(5)
	}
}
