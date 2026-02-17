package conductor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/pullrequest"
	"github.com/valksor/go-toolkit/workunit"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ErrNoStateFound is returned when no state marker is found in a comment.
var ErrNoStateFound = errors.New("no state found")

const (
	// StateMarker is the HTML comment marker used to embed state in PR comments.
	StateMarker = "<!-- MEHRHOF_REVIEW_STATE"

	// MaxStateSize is the maximum size of embedded state (50KB).
	MaxStateSize = 50000

	// maxDiffSize is the maximum size of diff to hash directly (10MB).
	maxDiffSize = 10 * 1024 * 1024

	// currentStateVersion is the current state format version.
	currentStateVersion = 1
)

// hashDiffPatch generates a SHA256 hash of a diff patch with memory protection.
// For large diffs (>10MB), it hashes a prefix plus the length to avoid memory exhaustion.
func hashDiffPatch(patch string) string {
	if len(patch) > maxDiffSize {
		// Hash first N bytes + length for approximation
		h := sha256.New()
		h.Write([]byte(patch[:maxDiffSize]))
		_, _ = fmt.Fprintf(h, "...truncated...len=%d", len(patch))

		return hex.EncodeToString(h.Sum(nil))
	}
	h := sha256.Sum256([]byte(patch))

	return hex.EncodeToString(h[:])
}

// getStateSecretKey returns the secret key for HMAC signatures.
// Requires the MEHRHOF_STATE_SECRET environment variable to be set with at least 32 characters.
func getStateSecretKey() ([]byte, error) {
	key := os.Getenv("MEHRHOF_STATE_SECRET")
	if key == "" {
		return nil, errors.New("MEHRHOF_STATE_SECRET environment variable must be set for PR review state verification")
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("MEHRHOF_STATE_SECRET must be at least 32 characters (got %d)", len(key))
	}

	return []byte(key), nil
}

// SignState generates HMAC signature for state verification.
func SignState(stateJSON []byte) (string, error) {
	key, err := getStateSecretKey()
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, key)
	h.Write(stateJSON)

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyState validates state signature.
func VerifyState(stateJSON []byte, signature string) (bool, error) {
	key, err := getStateSecretKey()
	if err != nil {
		return false, err
	}
	h := hmac.New(sha256.New, key)
	h.Write(stateJSON)
	expected := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// PRReviewState stores the review history for incremental reviews.
// This state is embedded in PR comments (no local files written).
type PRReviewState struct {
	Version          int           `json:"version"`            // State format version
	Provider         string        `json:"provider"`           // "github", "gitlab", "bitbucket", "azuredevops"
	PRNumber         int           `json:"pr_number"`          // PR/MR number
	CommitSHA        string        `json:"commit_sha"`         // Last reviewed commit SHA
	HeadBranch       string        `json:"head_branch"`        // Head branch name
	LastReviewAt     time.Time     `json:"last_review_at"`     // When last review ran
	Issues           []ReviewIssue `json:"issues"`             // Current issues found
	Signature        string        `json:"sig,omitempty"`      // HMAC signature for tamper detection
	ReviewedDiffHash string        `json:"reviewed_diff_hash"` // Hash of the reviewed diff for validation
	AgentVersion     string        `json:"agent_version"`      // Agent or config version
}

// ReviewIssue represents a single finding from the review.
type ReviewIssue struct {
	ID       string `json:"id"`       // Unique ID: hash of (file + line + normalized message)
	File     string `json:"file"`     // File path
	Line     int    `json:"line"`     // Line number (0 if not line-specific)
	Category string `json:"category"` // correctness, security, performance, style
	Severity string `json:"severity"` // critical, high, medium, low
	Message  string `json:"message"`  // The issue description
	Status   string `json:"status"`   // "open", "fixed", "acknowledged"
}

// EmbedStateInComment embeds state as a hidden HTML comment in markdown.
// This allows state to travel with the PR without writing local files.
// Adds HMAC signature for tamper protection.
// Uses base64 encoding to prevent XSS from AI-generated content.
func EmbedStateInComment(commentBody string, state *PRReviewState) string {
	if state == nil {
		return commentBody
	}

	// Marshal state to JSON (without signature first)
	stateCopy := *state
	stateCopy.Signature = ""

	stateJSON, err := json.Marshal(stateCopy)
	if err != nil {
		// If we can't marshal state, just return the comment as-is
		return commentBody
	}

	// Check size limit
	if len(stateJSON) > MaxStateSize {
		// Truncate issues list linearly if needed
		truncatedState := stateCopy
		// Leave 25% headroom to avoid edge cases
		const targetSize = MaxStateSize * 3 / 4
		for len(stateJSON) > MaxStateSize && len(truncatedState.Issues) > 0 {
			// Remove issues from the end (oldest first)
			truncatedState.Issues = truncatedState.Issues[:len(truncatedState.Issues)-1]
			var err error
			stateJSON, err = json.Marshal(truncatedState)
			if err != nil {
				// If marshal fails, return comment without state
				return commentBody
			}
			// Check if we've hit the target size
			if len(stateJSON) <= targetSize {
				break
			}
		}
	}

	// Generate HMAC signature
	signature, err := SignState(stateJSON)
	if err != nil {
		// If we can't sign, return comment without state (secure by default)
		return commentBody
	}

	// Add signature to state
	var finalState map[string]interface{}
	if err := json.Unmarshal(stateJSON, &finalState); err != nil {
		// Can't unmarshal - return comment without state
		return commentBody
	}
	finalState["sig"] = signature

	finalJSON, err := json.Marshal(finalState)
	if err != nil {
		// Can't marshal with signature - return comment without state
		return commentBody
	}

	// Base64 encode the JSON to prevent XSS
	encodedState := base64.StdEncoding.EncodeToString(finalJSON)

	// Add hidden HTML comment with embedded state (base64 encoded)
	stateComment := fmt.Sprintf("%s %s -->", StateMarker, encodedState)

	// Append to comment body
	return commentBody + "\n\n" + stateComment
}

// ExtractStateFromComment parses state from our bot's previous comment.
// Searches for the state marker and extracts the JSON.
// Verifies HMAC signature if present.
// Handles base64-encoded state to prevent XSS.
func ExtractStateFromComment(commentBody string) (*PRReviewState, error) {
	// Find the state marker in the comment
	startIdx := strings.Index(commentBody, StateMarker)
	if startIdx == -1 {
		return nil, ErrNoStateFound
	}

	// Find the end of the JSON (closing brace followed by -->)
	jsonStart := startIdx + len(StateMarker) + 1 // Skip the marker and space

	endMarker := "-->"
	endIdx := strings.Index(commentBody[jsonStart:], endMarker)
	if endIdx == -1 {
		return nil, errors.New("state marker end not found")
	}

	encodedStr := strings.TrimSpace(commentBody[jsonStart : jsonStart+endIdx])

	// Base64 decode the JSON
	jsonBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}

	// Parse JSON
	var state PRReviewState
	if err := json.Unmarshal(jsonBytes, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	// Verify signature if present
	if state.Signature != "" {
		// Remarshal without signature to verify
		stateCopy := state
		stateCopy.Signature = ""
		stateJSON, err := json.Marshal(stateCopy)
		if err != nil {
			return nil, fmt.Errorf("marshal state for verification: %w", err)
		}

		verified, err := VerifyState(stateJSON, state.Signature)
		if err != nil {
			return nil, fmt.Errorf("verify signature: %w", err)
		}
		if !verified {
			return nil, errors.New("state signature verification failed - state may have been tampered with")
		}
	}

	return &state, nil
}

// FindOurPreviousComment looks for our bot's previous comment with embedded state.
// Returns the comment and its extracted state.
// Optionally validates comment author if botUsername is provided.
func FindOurPreviousComment(comments []workunit.Comment, botUsername string) (*workunit.Comment, *PRReviewState, error) {
	// Look for comments that contain our state marker
	for i := range comments {
		comment := &comments[i]

		// Check if comment is from our bot (if username provided)
		if botUsername != "" {
			if comment.Author.ID != botUsername && comment.Author.Name != botUsername {
				continue
			}
		}

		// Check if comment contains our state marker
		if !strings.Contains(comment.Body, StateMarker) {
			continue
		}

		// Try to extract state from this comment
		state, err := ExtractStateFromComment(comment.Body)
		if err != nil {
			// Comment has marker but state is invalid/corrupted - skip it
			continue
		}

		if state != nil {
			return comment, state, nil
		}
	}

	return nil, nil, nil // No previous comment with valid state found
}

// GenerateIssueID creates a stable unique ID for a review issue.
// Uses SHA256 for collision resistance (128 bits = 16 hex chars).
func GenerateIssueID(file, message string, line int) string {
	// Normalize the message for consistent hashing
	normalized := strings.ToLower(strings.TrimSpace(message))
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Normalize file path
	filepath := strings.TrimPrefix(file, "./")
	filepath = strings.TrimPrefix(filepath, "/")

	// Create hash input: file:line:message
	hashInput := fmt.Sprintf("%s:%d:%s", filepath, line, normalized)

	// Generate SHA256 hash and take first 8 bytes (16 hex chars = 128 bits, collision resistant)
	h := sha256.Sum256([]byte(hashInput))

	return hex.EncodeToString(h[:8]) // 16 hex characters
}

// ReviewDelta represents the difference between reviews.
type ReviewDelta struct {
	NewIssues   []ReviewIssue // Issues not seen before → POST
	FixedIssues []ReviewIssue // Issues that existed before but now gone → ACKNOWLEDGE
	Unchanged   []ReviewIssue // Issues still present → SKIP
}

// ComputeReviewDelta compares current review with previous state.
// Returns what's new, what's fixed, and what hasn't changed.
func ComputeReviewDelta(prevState *PRReviewState, current *ParsedReview) ReviewDelta {
	prevIssues := make(map[string]ReviewIssue)
	if prevState != nil {
		for _, issue := range prevState.Issues {
			if issue.Status == "open" || issue.Status == "" {
				prevIssues[issue.ID] = issue
			}
		}
	}

	currentIssues := make(map[string]ReviewIssue)
	for _, issue := range current.Issues {
		currentIssues[issue.ID] = ReviewIssue{
			ID:       issue.ID,
			File:     issue.File,
			Line:     issue.Line,
			Category: issue.Category,
			Severity: issue.Severity,
			Message:  issue.Message,
			Status:   "open",
		}
	}

	delta := ReviewDelta{}

	// Find new issues
	for id, issue := range currentIssues {
		if _, exists := prevIssues[id]; !exists {
			delta.NewIssues = append(delta.NewIssues, issue)
		} else {
			// Issue still exists - unchanged
			delta.Unchanged = append(delta.Unchanged, issue)
		}
	}

	// Find fixed issues (existed before but not in current)
	for id, issue := range prevIssues {
		if _, exists := currentIssues[id]; !exists {
			// Mark as fixed
			issue.Status = "fixed"
			delta.FixedIssues = append(delta.FixedIssues, issue)
		}
	}

	return delta
}

// BuildPRReviewState creates a new PRReviewState from PR info and review.
func BuildPRReviewState(pr *pullrequest.PullRequest, diff *pullrequest.PullRequestDiff, review *ParsedReview) *PRReviewState {
	state := &PRReviewState{
		Version:      currentStateVersion,
		Provider:     detectProviderFromPR(pr),
		PRNumber:     pr.Number,
		CommitSHA:    pr.HeadSHA,
		HeadBranch:   pr.HeadBranch,
		LastReviewAt: time.Now(),
		Issues:       make([]ReviewIssue, len(review.Issues)),
	}

	// Compute diff hash for validation - handle nil diff
	if diff != nil && diff.Patch != "" {
		state.ReviewedDiffHash = hashDiffPatch(diff.Patch)
	}

	for i, issue := range review.Issues {
		// Generate ID if not provided
		id := issue.ID
		if id == "" {
			id = GenerateIssueID(issue.File, issue.Message, issue.Line)
		}

		state.Issues[i] = ReviewIssue{
			ID:       id,
			File:     issue.File,
			Line:     issue.Line,
			Category: issue.Category,
			Severity: issue.Severity,
			Message:  issue.Message,
			Status:   "open",
		}
	}

	return state
}

// detectProviderFromPR detects the provider from a PR struct.
// This is used when the provider might not be explicitly passed.
func detectProviderFromPR(pr *pullrequest.PullRequest) string {
	if pr == nil {
		return ""
	}

	return provider.DetectProviderFromURL(pr.URL)
}

// FormatReviewComment formats a review result as a markdown comment.
func FormatReviewComment(review *ParsedReview, delta ReviewDelta, opts PRReviewOptions) string {
	var sb strings.Builder

	sb.WriteString("## 🤖 AI PR Review\n\n")

	// Summary section
	if review.Summary != "" {
		sb.WriteString("### Summary\n\n")
		sb.WriteString(review.Summary)
		sb.WriteString("\n\n")
	}

	// Overall assessment
	if review.Overall != "" {
		sb.WriteString("**Assessment:** ")
		sb.WriteString(review.Overall)
		sb.WriteString("\n\n")
	}

	// Issues section
	if len(review.Issues) > 0 {
		sb.WriteString("### Issues Found\n\n")

		// Group by category
		byCategory := groupIssuesByCategory(review.Issues)

		for category, issues := range byCategory {
			if len(issues) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("#### %s\n\n", cases.Title(language.English, cases.NoLower).String(category)))

			for _, issue := range issues {
				lineInfo := ""
				if issue.Line > 0 {
					lineInfo = fmt.Sprintf("%s:%d", issue.File, issue.Line)
				} else if issue.File != "" {
					lineInfo = issue.File
				}

				if lineInfo != "" {
					lineInfo = "`" + lineInfo + "` "
				}

				sb.WriteString(fmt.Sprintf("- **%s** [%s] %s\n",
					issue.Severity, lineInfo, issue.Message))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("### ✅ No Issues Found\n\n")
		sb.WriteString("The code looks great! No issues were detected.\n\n")
	}

	// Fixed issues (if any and enabled)
	if opts.AcknowledgeFixes && len(delta.FixedIssues) > 0 {
		sb.WriteString("### ✅ Fixed Issues\n\n")
		sb.WriteString("The following issues from the previous review have been resolved:\n\n")

		for _, issue := range delta.FixedIssues {
			lineInfo := ""
			if issue.Line > 0 {
				lineInfo = fmt.Sprintf("%s:%d", issue.File, issue.Line)
			} else if issue.File != "" {
				lineInfo = issue.File
			}

			if lineInfo != "" {
				lineInfo = "`" + lineInfo + "` "
			}

			sb.WriteString(fmt.Sprintf("- ✓ %s [%s] %s\n",
				lineInfo, issue.Severity, issue.Message))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Review by Mehrhof* • *%s*\n",
		time.Now().Format("2006-01-02 15:04 MST")))

	return sb.String()
}

// groupIssuesByCategory groups issues by category.
func groupIssuesByCategory(issues []ReviewIssue) map[string][]ReviewIssue {
	grouped := make(map[string][]ReviewIssue)

	for _, issue := range issues {
		grouped[issue.Category] = append(grouped[issue.Category], issue)
	}

	return grouped
}

// PRReviewOptions holds options for PR review.
type PRReviewOptions struct {
	Provider         string
	PRNumber         int
	Format           string
	Scope            string
	AgentName        string
	AcknowledgeFixes bool
	UpdateExisting   bool
	MaxComments      int
	ExcludePatterns  []string
	Token            string // Override token for CI
}

// PRReviewResult holds the result of a PR review.
type PRReviewResult struct {
	CommentsPosted int    `json:"comments_posted"`
	URL            string `json:"url"`
	Skipped        bool   `json:"skipped"`
	Reason         string `json:"reason,omitempty"`
}

// ParsedReview holds the parsed review response from the AI.
type ParsedReview struct {
	Summary string
	Overall string
	Issues  []ReviewIssue
}
