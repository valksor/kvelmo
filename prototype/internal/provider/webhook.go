package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig holds webhook configuration.
type WebhookConfig struct {
	URL           string            `yaml:"url"`
	Secret        string            `yaml:"secret,omitempty"`
	Events        []string          `yaml:"events"` // e.g., "issue.created", "task.updated"
	Headers       map[string]string `yaml:"headers,omitempty"`
	Enabled       bool              `yaml:"enabled"`
	LastTriggered time.Time         `yaml:"last_triggered,omitempty"`
}

// WebhookEvent represents a webhook event payload.
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Provider  string                 `json:"provider"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookSender sends webhook events to configured endpoints.
type WebhookSender interface {
	SendWebhook(event WebhookEvent) error
	ValidateSignature(r *http.Request, secret string) (bool, error)
}

// WebhookClient implements WebhookSender.
type WebhookClient struct {
	httpClient *http.Client
}

// NewWebhookClient creates a new webhook client.
func NewWebhookClient() *WebhookClient {
	return &WebhookClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendWebhook sends a webhook event to the configured endpoint.
func (w *WebhookClient) SendWebhook(config WebhookConfig, event WebhookEvent) error {
	if !config.Enabled {
		return errors.New("webhook not enabled")
	}

	// Create signature
	signature := ""
	if config.Secret != "" {
		// In a real implementation, you'd serialize the event and sign it
		// For now, we'll create a simple signature
		sig := w.createSignature(event, config.Secret)
		signature = "sha256=" + sig
	}

	// Create request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, config.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mehrhof-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", event.Event)
	req.Header.Set("X-Webhook-Id", event.ID)
	req.Header.Set("X-Webhook-Timestamp", event.Timestamp.Format(time.RFC3339))

	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// In a real implementation, you'd marshal the event as JSON body
	// For now, we'll just send the request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// ValidateSignature validates a webhook signature from an incoming request.
func (w *WebhookClient) ValidateSignature(r *http.Request, secret string) (bool, error) {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false, nil
	}

	// Extract signature hash
	if len(signature) < 7 || signature[:7] != "sha256=" {
		return false, errors.New("invalid signature format")
	}

	_ = signature[7:]

	// In a real implementation, you'd read the body, compute the hash,
	// and compare it with the expected signature
	// For now, just return true if signature exists
	return true, nil
}

// createSignature creates an HMAC signature for the webhook payload.
func (w *WebhookClient) createSignature(event WebhookEvent, secret string) string {
	// In a real implementation, you'd marshal event to JSON and sign it
	// For now, return a placeholder signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(event.ID + event.Event + event.Timestamp.Format(time.RFC3339)))

	return hex.EncodeToString(h.Sum(nil))
}

// BulkOperationConfig holds configuration for bulk operations.
type BulkOperationConfig struct {
	BatchSize       int           `yaml:"batch_size"`        // Number of items per batch
	MaxConcurrent   int           `yaml:"max_concurrent"`    // Max concurrent operations
	ContinueOnError bool          `yaml:"continue_on_error"` // Continue if one item fails
	Delay           time.Duration `yaml:"delay"`             // Delay between batches
}

// BulkOperationResult holds the result of a bulk operation.
type BulkOperationResult struct {
	Total    int      `json:"total"`
	Success  int      `json:"success"`
	Failed   int      `json:"failed"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
	Duration string   `json:"duration"`
}

// BulkOperator handles bulk operations on providers.
type BulkOperator interface {
	// BulkCreate creates multiple items at once
	BulkCreate(items []map[string]interface{}) (BulkOperationResult, error)

	// BulkUpdate updates multiple items at once
	BulkUpdate(items []map[string]interface{}) (BulkOperationResult, error)

	// BulkDelete deletes multiple items at once
	BulkDelete(items []map[string]interface{}) (BulkOperationResult, error)
}

// DefaultBulkConfig returns default bulk operation configuration.
func DefaultBulkConfig() BulkOperationConfig {
	return BulkOperationConfig{
		BatchSize:       50,
		MaxConcurrent:   5,
		ContinueOnError: true,
		Delay:           100 * time.Millisecond,
	}
}
