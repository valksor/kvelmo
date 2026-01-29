package provider

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
//		_ provider.Reader = (*Provider)(nil)
//		_ provider.Identifier = (*Provider)(nil)
//		_ provider.Lister = (*Provider)(nil)
//		_ provider.Commenter = (*Provider)(nil)
//		_ provider.StatusUpdater = (*Provider)(nil)
//		_ provider.LabelManager = (*Provider)(nil)
//		_ provider.AttachmentDownloader = (*Provider)(nil)
//		_ provider.CommentFetcher = (*Provider)(nil)
//		_ provider.PRCreator = (*Provider)(nil)
//		_ provider.WorkUnitCreator = (*Provider)(nil)
//		_ provider.Snapshotter = (*Provider)(nil)
//		_ provider.SubtaskFetcher = (*Provider)(nil)
//	)

// CapabilityCheck defines a compile-time check for a single capability.
type CapabilityCheck struct {
	Capability Capability
	Interfaces []any // Example: (*Reader)(nil), (*Lister)(nil)
}

// CapabilityChecks maps provider names to their required capability checks.
var CapabilityChecks = map[string][]CapabilityCheck{
	"github": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapCreatePR, []any{(*PRCreator)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"gitlab": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreatePR, []any{(*PRCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"bitbucket": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreatePR, []any{(*PRCreator)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"azuredevops": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreatePR, []any{(*PRCreator)(nil)}},
		{CapLinkBranch, []any{(*BranchLinker)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"jira": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"asana": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"clickup": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"linear": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"notion": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
	},
	"wrike": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"youtrack": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"trello": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapList, []any{(*Lister)(nil)}},
		{CapFetchComments, []any{(*CommentFetcher)(nil)}},
		{CapComment, []any{(*Commenter)(nil)}},
		{CapUpdateStatus, []any{(*StatusUpdater)(nil)}},
		{CapManageLabels, []any{(*LabelManager)(nil)}},
		{CapDownloadAttachment, []any{(*AttachmentDownloader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
		{CapCreateWorkUnit, []any{(*WorkUnitCreator)(nil)}},
		{CapFetchSubtasks, []any{(*SubtaskFetcher)(nil)}},
		{CapFetchParent, []any{(*ParentFetcher)(nil)}},
	},
	"file": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
	},
	"directory": {
		{CapRead, []any{(*Reader)(nil)}},
		{CapSnapshot, []any{(*Snapshotter)(nil)}},
	},
}
