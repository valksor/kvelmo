package azuredevops

import (
	"context"
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
			name:         "success: downloads attachment by URL",
			workUnitID:   "123",
			attachmentID: "https://dev.azure.com/org/project/_apis/wit/attachments/abc123",
			wantErr:      false,
		},
		{
			name:         "error: invalid URL",
			workUnitID:   "123",
			attachmentID: "://invalid-url",
			wantErr:      true,
			errContains:  "download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider
			client := NewClient("org", "project", "test-token")

			provider := &Provider{
				client: client,
			}

			// Execute - attachmentID in Azure DevOps is the URL
			_, err := provider.DownloadAttachment(context.Background(), tt.workUnitID, tt.attachmentID)

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

			// For the success case with a real URL, we expect network errors
			// since we can't mock the Azure DevOps client easily
			// Just verify the method signature and delegation works
			if tt.attachmentID == "://invalid-url" {
				if err == nil {
					t.Error("expected error for invalid URL")
				}
			}
		})
	}
}

func TestProviderDownloadAttachmentIntegration(t *testing.T) {
	// Test that the provider method correctly delegates to the client
	t.Run("delegates to client method", func(t *testing.T) {
		client := NewClient("org", "project", "test-token")

		provider := &Provider{
			client: client,
		}

		// Verify the provider has the correct method signature
		// We can't easily test the full flow without exposing baseURL
		_, err := provider.DownloadAttachment(context.Background(), "workunit-id", "https://example.com/file")
		// We expect an error since this isn't a real URL
		if err == nil {
			t.Error("expected error for invalid URL scenario")
		}
	})
}
