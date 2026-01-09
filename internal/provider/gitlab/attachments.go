package gitlab

import (
	"context"
	"fmt"
	"io"
)

// DownloadAttachment downloads an attachment by ID.
// For GitLab, attachments are typically image URLs embedded in the issue body.
// The attachment ID format is "img-HASH" where HASH is derived from the URL for stability.
func (p *Provider) DownloadAttachment(ctx context.Context, workUnitID, attachmentID string) (io.ReadCloser, error) {
	// Parse the work unit ID
	ref, err := ParseReference(workUnitID)
	if err != nil {
		return nil, err
	}

	// Determine project
	projectPath := ref.ProjectPath
	if projectPath == "" {
		projectPath = p.projectPath
		if projectPath == "" {
			projectPath = p.config.ProjectPath
		}
	}

	if ref.ProjectID > 0 {
		p.client.SetProjectID(ref.ProjectID)
	} else if projectPath != "" {
		p.client.SetProjectPath(projectPath)
	} else {
		return nil, ErrProjectNotConfigured
	}

	// Fetch the issue to find the attachment URL
	issue, err := p.client.GetIssue(ctx, ref.IssueIID)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	// Extract image URLs from issue body
	urls := ExtractImageURLs(issue.Description)

	// Find the attachment by ID (format: img-HASH)
	var targetURL string
	for _, url := range urls {
		expectedID := "img-" + hashURL(url)
		if attachmentID == expectedID {
			targetURL = url

			break
		}
	}

	if targetURL == "" {
		return nil, fmt.Errorf("attachment not found: %s", attachmentID)
	}

	// Download using authenticated client with URL validation
	return p.client.DownloadAttachment(ctx, targetURL)
}
