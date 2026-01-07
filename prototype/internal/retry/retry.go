// Package retry provides utilities for retrying operations with exponential backoff.
// Ported from Python version: .backup/aerones-super-code/aerones_super_code/retry.py
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Default configuration values.
const (
	DefaultMaxAttempts     = 3
	DefaultBaseDelay       = 1.0 * time.Second
	DefaultMaxDelay        = 60.0 * time.Second
	DefaultExponentialBase = 2.0
)

// Config holds retry configuration parameters.
type Config struct {
	MaxAttempts     int           // Maximum number of retry attempts
	BaseDelay       time.Duration // Base delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	ExponentialBase float64       // Multiplier for exponential backoff
	Jitter          bool          // Add randomness to delay to prevent thundering herd
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:     DefaultMaxAttempts,
		BaseDelay:       DefaultBaseDelay,
		MaxDelay:        DefaultMaxDelay,
		ExponentialBase: DefaultExponentialBase,
		Jitter:          true,
	}
}

// IsRetryable returns true if the error should trigger a retry.
// By default, network-related errors are retryable.
func (c Config) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for common retryable error types
	// In Go, we typically check for temporary errors or specific error types
	type temporary interface {
		Temporary() bool
	}

	if temp, ok := err.(temporary); ok && temp.Temporary() {
		return true
	}

	// Could be extended with specific error type checks
	// For now, default to not retrying unknown errors
	return false
}

// CalculateDelay computes the delay for a given attempt number.
// Attempt is 0-indexed (0 = first retry).
func (c Config) CalculateDelay(attempt int) time.Duration {
	// Exponential backoff: base_delay * (exponential_base ^ attempt)
	floatDelay := float64(c.BaseDelay) * math.Pow(c.ExponentialBase, float64(attempt))
	delay := time.Duration(floatDelay)

	// Cap at max delay
	if delay > c.MaxDelay {
		delay = c.MaxDelay
	}

	// Add jitter: ±25% random variation
	if c.Jitter {
		//nolint:gosec // G404 - Math/rand is sufficient for jitter (non-cryptographic)
		jitterFactor := 1.0 + (rand.Float64()*0.5 - 0.25) // [-0.25, +0.25]
		delay = time.Duration(float64(delay) * jitterFactor)
	}

	return delay
}

// Do executes the given function, retrying on retryable errors.
// Returns the function's result or the last error encountered.
func (c Config) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := range c.MaxAttempts {
		if attempt > 0 {
			// Wait before retry
			delay := c.CalculateDelay(attempt - 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.IsRetryable(err) {
			return err
		}

		// If this was the last attempt, we'll return the error below
		if attempt == c.MaxAttempts-1 {
			break
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", c.MaxAttempts, lastErr)
}

// DoWithContext is like Do but allows the function to receive the context.
func (c Config) DoWithContext(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for attempt := range c.MaxAttempts {
		if attempt > 0 {
			// Wait before retry
			delay := c.CalculateDelay(attempt - 1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.IsRetryable(err) {
			return err
		}

		// If this was the last attempt, we'll return the error below
		if attempt == c.MaxAttempts-1 {
			break
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", c.MaxAttempts, lastErr)
}

// RetryContext manages retry state for manual retry control.
type RetryContext struct {
	config  Config
	attempt int
	lastErr error
}

// NewRetryContext creates a new retry context.
func NewRetryContext(config Config) *RetryContext {
	return &RetryContext{
		config: config,
	}
}

// ShouldContinue returns true if more attempts are allowed.
func (r *RetryContext) ShouldContinue() bool {
	return r.attempt < r.config.MaxAttempts
}

// HandleError records an error and determines if retry should continue.
// Returns true if the operation should be retried, false if it should stop.
func (r *RetryContext) HandleError(err error) bool {
	r.lastErr = err
	r.attempt++

	if !r.config.IsRetryable(err) {
		return false
	}

	if r.attempt >= r.config.MaxAttempts {
		return false
	}

	return true
}

// Delay waits for the appropriate delay before the next retry.
func (r *RetryContext) Delay(ctx context.Context) error {
	delay := r.config.CalculateDelay(r.attempt - 1)
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// LastError returns the most recent error encountered.
func (r *RetryContext) LastError() error {
	return r.lastErr
}

// AttemptCount returns the number of attempts made so far.
func (r *RetryContext) AttemptCount() int {
	return r.attempt
}
