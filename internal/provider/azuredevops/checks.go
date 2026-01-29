package azuredevops

import (
	"github.com/valksor/go-mehrhof/internal/provider"
)

// Compile-time interface checks to ensure Provider implements declared capabilities.
var (
	_ provider.Reader               = (*Provider)(nil)
	_ provider.Identifier           = (*Provider)(nil)
	_ provider.Lister               = (*Provider)(nil)
	_ provider.Commenter            = (*Provider)(nil)
	_ provider.StatusUpdater        = (*Provider)(nil)
	_ provider.LabelManager         = (*Provider)(nil)
	_ provider.Snapshotter          = (*Provider)(nil)
	_ provider.PRCreator            = (*Provider)(nil)
	_ provider.WorkUnitCreator      = (*Provider)(nil)
	_ provider.SubtaskFetcher       = (*Provider)(nil)
	_ provider.ParentFetcher        = (*Provider)(nil)
	_ provider.CommentFetcher       = (*Provider)(nil)
	_ provider.AttachmentDownloader = (*Provider)(nil)
)
