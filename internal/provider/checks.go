package provider

import (
	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/snapshot"
	"github.com/valksor/go-toolkit/workunit"
)

// Compile-time interface checks for provider implementations.
//
// Each provider package should import this package and include a check
// in a file (e.g., provider_checks.go) to verify it implements the
// interfaces corresponding to its declared capabilities.
//
// Example:
//
//	package github
//
//	import (
//		"github.com/valksor/go-mehrhof/internal/provider"
//	)
//
//	// Compile-time interface checks
//	var (
//		_ workunit.Reader = (*Provider)(nil)
//		_ workunit.Identifier = (*Provider)(nil)
//		_ workunit.Lister = (*Provider)(nil)
//		_ workunit.Commenter = (*Provider)(nil)
//		_ workunit.StatusUpdater = (*Provider)(nil)
//		_ workunit.LabelManager = (*Provider)(nil)
//		_ workunit.AttachmentDownloader = (*Provider)(nil)
//		_ workunit.CommentFetcher = (*Provider)(nil)
//		_ pullrequest.PRCreator = (*Provider)(nil)
//		_ workunit.WorkUnitCreator = (*Provider)(nil)
//		_ snapshot.Snapshotter = (*Provider)(nil)
//		_ workunit.SubtaskFetcher = (*Provider)(nil)
//	)

// CapabilityCheck defines a compile-time check for a single capability.
type CapabilityCheck struct {
	Capability capability.Capability
	Interfaces []any // Example: (*workunit.Reader)(nil), (*workunit.Lister)(nil)
}

// CapabilityChecks maps provider names to their required capability checks.
var CapabilityChecks = map[string][]CapabilityCheck{
	"github": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapCreatePR, []any{(*pullrequest.PRCreator)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"gitlab": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreatePR, []any{(*pullrequest.PRCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"bitbucket": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreatePR, []any{(*pullrequest.PRCreator)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"azuredevops": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreatePR, []any{(*pullrequest.PRCreator)(nil)}},
		{capability.CapLinkBranch, []any{(*pullrequest.BranchLinker)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"jira": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"asana": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"clickup": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"linear": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"notion": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
	},
	"wrike": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"youtrack": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"trello": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapList, []any{(*workunit.Lister)(nil)}},
		{capability.CapFetchComments, []any{(*workunit.CommentFetcher)(nil)}},
		{capability.CapComment, []any{(*workunit.Commenter)(nil)}},
		{capability.CapUpdateStatus, []any{(*workunit.StatusUpdater)(nil)}},
		{capability.CapManageLabels, []any{(*workunit.LabelManager)(nil)}},
		{capability.CapDownloadAttachment, []any{(*workunit.AttachmentDownloader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
		{capability.CapCreateWorkUnit, []any{(*workunit.WorkUnitCreator)(nil)}},
		{capability.CapFetchSubtasks, []any{(*workunit.SubtaskFetcher)(nil)}},
		{capability.CapFetchParent, []any{(*workunit.ParentFetcher)(nil)}},
	},
	"file": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
	},
	"directory": {
		{capability.CapRead, []any{(*workunit.Reader)(nil)}},
		{capability.CapSnapshot, []any{(*snapshot.Snapshotter)(nil)}},
	},
}
