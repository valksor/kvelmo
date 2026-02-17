package asana

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance.
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks.
var (
	_ workunit.AttachmentDownloader = (*Provider)(nil)
)

// ──────────────────────────────────────────────────────────────────────────────
// DownloadAttachment
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderDownloadAttachment(t *testing.T) {
	tests := []struct {
		name         string
		workUnitID   string
		attachmentID string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "success: downloads from URL",
			workUnitID:   "task123",
			attachmentID: "https://app.asana.com/attachment/123",
			wantErr:      false,
		},
		{
			name:         "error: invalid URL",
			workUnitID:   "task123",
			attachmentID: "://invalid-url",
			wantErr:      true,
			errContains:  "download",
		},
		{
			name:         "error: unauthorized",
			workUnitID:   "task123",
			attachmentID: "https://app.asana.com/attachment/401",
			wantErr:      true,
			errContains:  "download failed",
		},
		{
			name:         "error: not found",
			workUnitID:   "task123",
			attachmentID: "https://app.asana.com/attachment/404",
			wantErr:      true,
			errContains:  "download failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check authorization
				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Bearer ") {
					t.Errorf("expected Authorization header to start with 'Bearer ', got %q", auth)
				}

				// Simulate different responses based on test case
				if tt.name == "error: unauthorized" {
					w.WriteHeader(http.StatusUnauthorized)

					return
				}
				if tt.name == "error: not found" {
					w.WriteHeader(http.StatusNotFound)

					return
				}
				if tt.name == "error: invalid URL" {
					w.WriteHeader(http.StatusBadRequest)

					return
				}

				// Success case
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test attachment data"))
			}))
			defer server.Close()

			// Create provider
			client := NewClient("test-token", "")
			client.baseURL = server.URL

			provider := &Provider{
				client: client,
			}

			// For success test, use mock server URL
			attachmentID := tt.attachmentID
			if tt.name == "success: downloads from URL" {
				attachmentID = server.URL
			}

			// Execute - attachmentID in Asana is the URL
			rc, err := provider.DownloadAttachment(context.Background(), tt.workUnitID, attachmentID)

			// Verify
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
				// Only check error message if specified
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if rc == nil {
				t.Errorf("expected non-nil ReadCloser on success")

				return
			}
			defer func() { _ = rc.Close() }()

			// Verify we can read the content
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Errorf("failed to read attachment: %v", err)

				return
			}

			if string(data) != "test attachment data" {
				t.Errorf("got data %q, want %q", string(data), "test attachment data")
			}
		})
	}
}

func TestProviderDownloadAttachmentIntegration(t *testing.T) {
	// Test that the provider method correctly delegates to the client
	t.Run("delegates to client method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET request, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("attachment content"))
		}))
		defer server.Close()

		client := NewClient("test-token", "")
		client.baseURL = server.URL

		provider := &Provider{
			client: client,
		}

		rc, err := provider.DownloadAttachment(context.Background(), "task-id", server.URL)
		if err != nil {
			t.Errorf("unexpected error: %v", err)

			return
		}

		if rc == nil {
			t.Error("expected non-nil ReadCloser")

			return
		}
		defer func() { _ = rc.Close() }()
	})
}
