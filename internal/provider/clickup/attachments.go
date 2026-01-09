package clickup

import (
	"context"
	"fmt"
	"io"
)

// DownloadAttachment downloads an attachment by ID.
// For ClickUp, the attachmentID should be a direct URL to the attachment.
func (p *Provider) DownloadAttachment(ctx context.Context, workUnitID, attachmentID string) (io.ReadCloser, error) {
	// Download the attachment using the URL (attachmentID)
	rc, err := p.client.DownloadAttachment(ctx, attachmentID)
	if err != nil {
		return nil, fmt.Errorf("download attachment: %w", err)
	}

	return rc, nil
}
