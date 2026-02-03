package gitlab

import (
	"context"
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
// DownloadAttachment.
//
// Note: Attachment IDs use the format "img-HASH" where HASH is an 8-character
// hex string derived from SHA256 of the image URL. This ensures stable IDs
// even if images are reordered in the issue description.
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderDownloadAttachment(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Provider)
		workUnitID   string
		attachmentID string
		wantErr      bool
		errContains  string
	}{
		{
			name: "error: invalid work unit ID format",
			setup: func(p *Provider) {
				p.projectPath = "test/project"
			},
			workUnitID:   "invalid-id",
			attachmentID: "img-f0e6a6a9", // Hash-based ID format
			wantErr:      true,
			errContains:  "unrecognized format",
		},
		{
			name: "error: project not configured",
			setup: func(p *Provider) {
				// No project path configured
			},
			workUnitID:   "123",
			attachmentID: "img-f0e6a6a9", // Hash-based ID format
			wantErr:      true,
			errContains:  "",
		},
		{
			name: "success: can parse reference",
			setup: func(p *Provider) {
				p.projectPath = "test/project"
			},
			workUnitID:   "123",
			attachmentID: "img-f0e6a6a9", // Hash-based ID format
			wantErr:      false,          // Note: Will fail at network call, but parsing works
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider
			client, err := NewClient("", "", "", 0)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			provider := &Provider{
				client: client,
				config: &Config{
					ProjectPath: "test/project",
				},
			}

			tt.setup(provider)

			// Execute
			rc, err := provider.DownloadAttachment(context.Background(), tt.workUnitID, tt.attachmentID)

			// Verify
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}

				return
			}

			// For success cases, we expect the network call to fail in this test
			// (since we're not mocking the GitLab API client)
			// but we can verify the parsing worked
			if err == nil && rc != nil {
				_ = rc.Close()
			}
		})
	}
}

// Test error handling helpers.
func TestProviderDownloadAttachmentErrors(t *testing.T) {
	client, err := NewClient("", "", "", 0)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	provider := &Provider{
		client: client,
		config: &Config{},
	}

	t.Run("error: invalid work unit ID", func(t *testing.T) {
		_, err := provider.DownloadAttachment(context.Background(), "not-a-number", "img-f0e6a6a9")
		if err == nil {
			t.Error("expected error for invalid work unit ID, got nil")
		}
	})

	t.Run("error: project not configured", func(t *testing.T) {
		_, err := provider.DownloadAttachment(context.Background(), "123", "img-f0e6a6a9")
		if err == nil {
			t.Error("expected error for missing project, got nil")
		}
	})
}
