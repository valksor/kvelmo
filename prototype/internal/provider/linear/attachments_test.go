package linear

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider Interface Compliance.
// ──────────────────────────────────────────────────────────────────────────────

// Compile-time interface checks.
var (
	_ provider.AttachmentDownloader = (*Provider)(nil)
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
			workUnitID:   "LIN-123",
			attachmentID: "https://api.linear.app/attachment/uuid",
			wantErr:      false,
		},
		{
			name:         "error: invalid URL",
			workUnitID:   "LIN-123",
			attachmentID: "://invalid-url",
			wantErr:      true,
			errContains:  "download",
		},
		{
			name:         "error: unauthorized",
			workUnitID:   "LIN-123",
			attachmentID: "https://api.linear.app/unauthorized",
			wantErr:      true,
			errContains:  "download failed",
		},
		{
			name:         "error: not found",
			workUnitID:   "LIN-123",
			attachmentID: "https://api.linear.app/notfound",
			wantErr:      true,
			errContains:  "download failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check authorization (Bearer token)
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

				// Success case.
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test attachment data"))
			}))
			defer server.Close()

			// Create provider
			client := NewClient("test-token")
			client.baseURL = server.URL

			provider := &Provider{
				client: client,
			}

			// For success test, use mock server URL
			attachmentID := tt.attachmentID
			if tt.name == "success: downloads from URL" {
				attachmentID = server.URL
			}

			// Execute - attachmentID in Linear is the URL
			rc, err := provider.DownloadAttachment(context.Background(), tt.workUnitID, attachmentID)

			// Verify
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)

					return
				}
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

		client := NewClient("test-token")

		provider := &Provider{
			client: client,
		}

		rc, err := provider.DownloadAttachment(context.Background(), "issue-id", server.URL)
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
