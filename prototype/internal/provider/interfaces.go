package provider

import (
	"context"
	"io"
	"time"
)

// Reader fetches work units from a provider.
type Reader interface {
	Fetch(ctx context.Context, id string) (*WorkUnit, error)
}

// Identifier parses and validates references.
type Identifier interface {
	Parse(input string) (string, error)
	Match(input string) bool
}

// Lister enumerates work units.
type Lister interface {
	List(ctx context.Context, opts ListOptions) ([]*WorkUnit, error)
}

// AttachmentDownloader downloads attachments.
type AttachmentDownloader interface {
	DownloadAttachment(ctx context.Context, workUnitID, attachmentID string) (io.ReadCloser, error)
}

// CommentFetcher retrieves comments.
type CommentFetcher interface {
	FetchComments(ctx context.Context, workUnitID string) ([]Comment, error)
}

// Commenter adds comments to work units.
type Commenter interface {
	AddComment(ctx context.Context, workUnitID string, body string) (*Comment, error)
}

// StatusUpdater changes work unit status.
type StatusUpdater interface {
	UpdateStatus(ctx context.Context, workUnitID string, status Status) error
}

// LabelManager manages labels on work units.
type LabelManager interface {
	AddLabels(ctx context.Context, workUnitID string, labels []string) error
	RemoveLabels(ctx context.Context, workUnitID string, labels []string) error
}

// ReadOnlyProvider is the minimum interface for a provider.
type ReadOnlyProvider interface {
	Reader
	Identifier
}

// BidirectionalProvider supports read and write operations.
type BidirectionalProvider interface {
	Reader
	Identifier
	Commenter
	StatusUpdater
}

// ListOptions configures list operations.
type ListOptions struct {
	Status   Status
	Labels   []string
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string // asc, desc
}

// Snapshot contains captured source content (read-only copy).
type Snapshot struct {
	Type    string         // directory, file
	Ref     string         // original reference
	Files   []SnapshotFile // for directories
	Content string         // for single files
}

// SnapshotFile represents a single file in a snapshot.
type SnapshotFile struct {
	Path    string
	Content string
}

// Snapshotter captures source content for storage.
type Snapshotter interface {
	Snapshot(ctx context.Context, id string) (*Snapshot, error)
}

// ──────────────────────────────────────────────────────────────────────────────
// Extended interfaces for bidirectional providers (GitHub, Wrike, etc.)
// ──────────────────────────────────────────────────────────────────────────────

// PRCreator creates pull requests (for GitHub-like providers).
type PRCreator interface {
	CreatePullRequest(ctx context.Context, opts PullRequestOptions) (*PullRequest, error)
}

// PRFetcher retrieves pull request details and diffs.
type PRFetcher interface {
	FetchPullRequest(ctx context.Context, number int) (*PullRequest, error)
	FetchPullRequestDiff(ctx context.Context, number int) (*PullRequestDiff, error)
}

// PRCommenter posts comments to pull requests.
type PRCommenter interface {
	AddPullRequestComment(ctx context.Context, number int, body string) (*Comment, error)
}

// PRCommentFetcher retrieves existing comments from a PR/MR.
type PRCommentFetcher interface {
	FetchPullRequestComments(ctx context.Context, number int) ([]Comment, error)
}

// PRCommentUpdater updates existing comments on a PR/MR.
type PRCommentUpdater interface {
	UpdatePullRequestComment(ctx context.Context, number int, commentID string, body string) (*Comment, error)
}

// PullRequestOptions for creating a PR.
type PullRequestOptions struct {
	Title        string
	Body         string
	SourceBranch string
	TargetBranch string
	Labels       []string
	Reviewers    []string
	Draft        bool
}

// PullRequest represents a pull/merge request.
type PullRequest struct {
	ID         string
	URL        string
	Title      string
	State      string
	Number     int
	Body       string
	HeadSHA    string    // Commit SHA of the head branch
	HeadBranch string    // Name of the head branch
	BaseBranch string    // Name of the base branch
	Author     string    // Author username
	CreatedAt  time.Time // Creation time
	UpdatedAt  time.Time // Last update time
	Labels     []string  // PR labels
	Assignees  []string  // Assignee usernames
}

// PullRequestDiff contains PR diff information.
type PullRequestDiff struct {
	URL        string     // URL to view the diff
	BaseBranch string     // Base branch name
	HeadBranch string     // Head branch name
	Files      []FileDiff // Files changed
	Patch      string     // Full diff in unified format
	Additions  int        // Total lines added
	Deletions  int        // Total lines deleted
	Commits    int        // Number of commits
}

// FileDiff represents a single file's changes.
type FileDiff struct {
	Path      string // File path
	Mode      string // "added", "modified", "deleted", "renamed"
	Patch     string // Unified diff for this file
	Additions int    // Lines added
	Deletions int    // Lines deleted
}

// BranchLinker links work units to git branches.
type BranchLinker interface {
	LinkBranch(ctx context.Context, workUnitID, branch string) error
	UnlinkBranch(ctx context.Context, workUnitID, branch string) error
	GetLinkedBranch(ctx context.Context, workUnitID string) (string, error)
}

// WorkUnitCreator creates new work units (for Wrike, GitHub issues, etc.)
type WorkUnitCreator interface {
	CreateWorkUnit(ctx context.Context, opts CreateWorkUnitOptions) (*WorkUnit, error)
}

// SubtaskFetcher retrieves subtasks for a work unit.
type SubtaskFetcher interface {
	FetchSubtasks(ctx context.Context, workUnitID string) ([]*WorkUnit, error)
}

// CreateWorkUnitOptions for creating a work unit.
type CreateWorkUnitOptions struct {
	CustomFields  map[string]any
	Title         string
	Description   string
	ParentID      string
	Labels        []string
	Assignees     []string
	Priority      Priority
	DependencyIDs []string // Work unit IDs this unit depends on (predecessors)
}

// DependencyCreator creates dependencies between work units.
type DependencyCreator interface {
	// CreateDependency creates a dependency where predecessorID must complete before successorID.
	CreateDependency(ctx context.Context, predecessorID, successorID string) error
}

// DependencyFetcher retrieves dependencies for a work unit.
type DependencyFetcher interface {
	// GetDependencies returns the IDs of work units that the given work unit depends on.
	GetDependencies(ctx context.Context, workUnitID string) ([]string, error)
}

// FullProvider is a comprehensive interface combining all capabilities
// This is primarily for documentation; actual providers implement subsets.
//
//nolint:interfacebloat // documentation interface combining all provider capabilities
type FullProvider interface {
	ReadOnlyProvider
	Lister
	Commenter
	StatusUpdater
	LabelManager
	AttachmentDownloader
	CommentFetcher
	PRCreator
	BranchLinker
	WorkUnitCreator
	Snapshotter
	SubtaskFetcher
}
