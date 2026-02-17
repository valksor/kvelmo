package youtrack

import (
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// Compile-time interface checks to ensure Provider implements declared capabilities.
var (
	_ workunit.Reader               = (*Provider)(nil)
	_ workunit.Identifier           = (*Provider)(nil)
	_ workunit.Lister               = (*Provider)(nil)
	_ workunit.Commenter            = (*Provider)(nil)
	_ workunit.StatusUpdater        = (*Provider)(nil)
	_ workunit.LabelManager         = (*Provider)(nil)
	_ workunit.AttachmentDownloader = (*Provider)(nil)
	_ snapshot.Snapshotter          = (*Provider)(nil)
	_ workunit.WorkUnitCreator      = (*Provider)(nil)
	_ workunit.SubtaskFetcher       = (*Provider)(nil)
	_ workunit.ParentFetcher        = (*Provider)(nil)
	_ workunit.CommentFetcher       = (*Provider)(nil)
)
