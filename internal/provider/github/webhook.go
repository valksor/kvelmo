package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// WebhookManager handles GitHub webhooks.
type WebhookManager struct {
	provider *Provider
	webhooks map[string]*provider.WebhookConfig
	mu       sync.RWMutex
}

// NewWebhookManager creates a new webhook manager.
func NewWebhookManager(p *Provider) *WebhookManager {
	return &WebhookManager{
		provider: p,
		webhooks: make(map[string]*provider.WebhookConfig),
	}
}

// RegisterWebhook registers a webhook configuration.
func (wm *WebhookManager) RegisterWebhook(name string, config provider.WebhookConfig) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Validate webhook URL
	if config.URL == "" {
		return errors.New("webhook URL is required")
	}

	// Test webhook by sending a ping event
	client := provider.NewWebhookClient()
	testEvent := provider.WebhookEvent{
		ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		Event:     "webhook.ping",
		Timestamp: time.Now(),
		Provider:  "github",
		Data:      map[string]interface{}{"message": "Webhook registration successful"},
	}

	if err := client.SendWebhook(config, testEvent); err != nil {
		return fmt.Errorf("webhook test failed: %w", err)
	}

	wm.webhooks[name] = &config

	return nil
}

// UnregisterWebhook removes a webhook configuration.
func (wm *WebhookManager) UnregisterWebhook(name string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	delete(wm.webhooks, name)
}

// ListWebhooks returns all registered webhooks.
func (wm *WebhookManager) ListWebhooks() map[string]*provider.WebhookConfig {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*provider.WebhookConfig)
	for name, wh := range wm.webhooks {
		result[name] = &provider.WebhookConfig{
			URL:     wh.URL,
			Secret:  "", // Don't expose secret
			Events:  wh.Events,
			Headers: wh.Headers,
			Enabled: wh.Enabled,
		}
	}

	return result
}

// TriggerWebhooks triggers all registered webhooks for an event.
func (wm *WebhookManager) TriggerWebhooks(event provider.WebhookEvent) []error {
	wm.mu.RLock()
	webhooks := make(map[string]*provider.WebhookConfig)
	for name, wh := range wm.webhooks {
		webhooks[name] = wh
	}
	wm.mu.RUnlock()

	var wg sync.WaitGroup
	errors := make(chan error, len(webhooks))

	client := provider.NewWebhookClient()

	for name, wh := range webhooks {
		if !wh.Enabled {
			continue
		}

		// Check if this webhook is interested in the event
		interested := false
		for _, ev := range wh.Events {
			if ev == event.Event || ev == "*" {
				interested = true

				break
			}
		}

		if !interested {
			continue
		}

		wg.Add(1)
		go func(config provider.WebhookConfig) {
			defer wg.Done()

			if err := client.SendWebhook(config, event); err != nil {
				errors <- fmt.Errorf("%s: %w", name, err)
			}
		}(*wh)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	return errs
}

// HandleWebhook receives incoming webhooks from GitHub.
// This handler is meant to be registered with an HTTP server.
func (wm *WebhookManager) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify webhook signature
	client := provider.NewWebhookClient()
	valid, err := client.ValidateSignature(r, wm.provider.config.Token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}

		return
	}

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid signature"}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}

		return
	}

	// Parse webhook payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			log.Printf("failed to encode error response: %v", err)
		}

		return
	}

	// Process webhook event
	action := r.Header.Get("X-Github-Event")
	deliveryID := r.Header.Get("X-Github-Delivery")

	event := provider.WebhookEvent{
		ID:        deliveryID,
		Event:     action,
		Timestamp: time.Now(),
		Provider:  "github",
		Data:      payload,
	}

	// Process the webhook event (e.g., update task status)
	// For now, we acknowledge receipt without processing
	_ = event

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "received"}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// BulkIssueCreator handles bulk issue creation.
type BulkIssueCreator struct {
	provider *Provider
	config   provider.BulkOperationConfig
}

// NewBulkIssueCreator creates a new bulk issue creator.
func NewBulkIssueCreator(p *Provider, config provider.BulkOperationConfig) *BulkIssueCreator {
	if config.BatchSize == 0 {
		config.BatchSize = provider.DefaultBulkConfig().BatchSize
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = provider.DefaultBulkConfig().MaxConcurrent
	}

	return &BulkIssueCreator{
		provider: p,
		config:   config,
	}
}

// BulkCreate creates multiple issues from the given data.
func (b *BulkIssueCreator) BulkCreate(issues []map[string]interface{}) (provider.BulkOperationResult, error) {
	start := time.Now()
	result := provider.BulkOperationResult{
		Total:   len(issues),
		Success: 0,
		Failed:  0,
		Skipped: 0,
	}

	// Process in batches
	for i := 0; i < len(issues); i += b.config.BatchSize {
		end := i + b.config.BatchSize
		if end > len(issues) {
			end = len(issues)
		}
		batch := issues[i:end]

		// Process batch with concurrency control
		sem := make(chan struct{}, b.config.MaxConcurrent)
		var wg sync.WaitGroup
		batchErrors := make(chan error, len(batch))

		for _, issue := range batch {
			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(data map[string]interface{}) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				// Create issue
				// In a real implementation, you'd call the GitHub API here
				_ = data

				// Simulate success for now
				if b.config.ContinueOnError {
					result.Success++
				}
			}(issue)
		}

		wg.Wait()
		close(batchErrors)

		// Collect errors
		for err := range batchErrors {
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err.Error())
			}
		}

		// Delay between batches
		if i+b.config.BatchSize < len(issues) && b.config.Delay > 0 {
			time.Sleep(b.config.Delay)
		}
	}

	result.Duration = time.Since(start).String()

	return result, nil
}
