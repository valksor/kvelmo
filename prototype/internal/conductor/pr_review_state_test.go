package conductor

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// TestSignVerifyState tests HMAC signature generation and verification.
func TestSignVerifyState(t *testing.T) {
	// Set test secret - t.Setenv automatically cleans up when test ends
	t.Setenv("MEHRHOF_STATE_SECRET", "test-secret-key-that-is-at-least-32-chars-long")

	stateJSON := []byte(`{"test":"data"}`)

	// Test signing
	sig, err := SignState(stateJSON)
	if err != nil {
		t.Fatalf("SignState: %v", err)
	}
	if sig == "" {
		t.Fatal("signature is empty")
	}

	// Test verification
	verified, err := VerifyState(stateJSON, sig)
	if err != nil {
		t.Fatalf("VerifyState: %v", err)
	}
	if !verified {
		t.Error("signature verification failed")
	}

	// Test invalid signature
	verified, _ = VerifyState(stateJSON, "invalid")
	if verified {
		t.Error("invalid signature was verified")
	}
}

// TestSignVerifyStateNoSecret tests error when no secret is configured.
func TestSignVerifyStateNoSecret(t *testing.T) {
	t.Setenv("MEHRHOF_STATE_SECRET", "")

	stateJSON := []byte(`{"test":"data"}`)

	_, err := SignState(stateJSON)
	if err == nil {
		t.Error("expected error when no secret configured")
	}

	_, err = VerifyState(stateJSON, "sig")
	if err == nil {
		t.Error("expected error when no secret configured")
	}
}

// TestSignVerifyStateShortSecret tests error with short secret.
func TestSignVerifyStateShortSecret(t *testing.T) {
	t.Setenv("MEHRHOF_STATE_SECRET", "short")

	_, err := SignState([]byte("data"))
	if err == nil {
		t.Error("expected error with short secret")
	}

	_, err = VerifyState([]byte("data"), "sig")
	if err == nil {
		t.Error("expected error with short secret")
	}
}

// TestEmbedExtractState tests round-trip state embedding and extraction.
func TestEmbedExtractState(t *testing.T) {
	t.Setenv("MEHRHOF_STATE_SECRET", "test-secret-key-that-is-at-least-32-chars-long")

	state := &PRReviewState{
		Provider:     "github",
		PRNumber:     123,
		CommitSHA:    "abc123",
		HeadBranch:   "feature",
		LastReviewAt: time.Now(),
		Issues: []ReviewIssue{
			{
				ID:       "test-id",
				File:     "test.go",
				Line:     42,
				Category: "correctness",
				Severity: "high",
				Message:  "Test issue",
				Status:   "open",
			},
		},
	}

	comment := "## Review\n\nSome content"
	embedded := EmbedStateInComment(comment, state)

	// Should have marker
	if !contains(embedded, StateMarker) {
		t.Error("state marker not found")
	}

	// Should extract back
	extracted, err := ExtractStateFromComment(embedded)
	if err != nil {
		t.Fatalf("ExtractStateFromComment: %v", err)
	}
	if extracted == nil {
		t.Fatal("extracted state is nil")
	}

	// Verify fields
	if extracted.Provider != state.Provider {
		t.Errorf("provider: got %s, want %s", extracted.Provider, state.Provider)
	}
	if extracted.PRNumber != state.PRNumber {
		t.Errorf("PR number: got %d, want %d", extracted.PRNumber, state.PRNumber)
	}
	if len(extracted.Issues) != len(state.Issues) {
		t.Errorf("issues: got %d, want %d", len(extracted.Issues), len(state.Issues))
	}
}

// TestExtractStateTampered tests tampered state detection.
func TestExtractStateTampered(t *testing.T) {
	t.Setenv("MEHRHOF_STATE_SECRET", "test-secret-key-that-is-at-least-32-chars-long")

	state := &PRReviewState{
		Provider: "github",
		PRNumber: 123,
	}

	comment := EmbedStateInComment("test", state)

	// Tamper with the state by modifying the base64 portion
	// Find the start of base64 data
	markerIdx := strings.Index(comment, StateMarker)
	if markerIdx == -1 {
		t.Fatal("state marker not found in embedded comment")
	}

	// Get the base64 part and modify it
	base64Start := markerIdx + len(StateMarker) + 1
	endMarkerIdx := strings.Index(comment[base64Start:], "-->")
	if endMarkerIdx == -1 {
		t.Fatal("end marker not found")
	}
	base64Part := comment[base64Start : base64Start+endMarkerIdx]

	// Modify a character in the base64 (this will corrupt it)
	tamperedBase64 := "A" + base64Part[1:]

	// Reconstruct the comment with tampered data
	tampered := comment[:base64Start] + tamperedBase64 + comment[base64Start+endMarkerIdx:]

	_, err := ExtractStateFromComment(tampered)
	if err == nil {
		t.Error("expected error for tampered state")
	}
}

// TestExtractStateNoSecret tests that extraction fails gracefully without secret.
func TestExtractStateNoSecret(t *testing.T) {
	t.Setenv("MEHRHOF_STATE_SECRET", "")

	// Create a comment with state marker but no valid state
	comment := "## Review\n\n" + StateMarker + " invalid-base64-data -->"

	_, err := ExtractStateFromComment(comment)
	if err == nil {
		t.Error("expected error for invalid state without secret")
	}
}

// TestGenerateIssueID tests ID stability.
func TestGenerateIssueID(t *testing.T) {
	id1 := GenerateIssueID("test.go", "issue message", 42)
	id2 := GenerateIssueID("test.go", "issue message", 42)

	if id1 != id2 {
		t.Errorf("IDs not stable: %s != %s", id1, id2)
	}

	id3 := GenerateIssueID("test.go", "different message", 42)
	if id1 == id3 {
		t.Error("different inputs produced same ID")
	}

	id4 := GenerateIssueID("different.go", "issue message", 42)
	if id1 == id4 {
		t.Error("different file produced same ID")
	}

	id5 := GenerateIssueID("test.go", "issue message", 100)
	if id1 == id5 {
		t.Error("different line number produced same ID")
	}
}

// TestComputeReviewDelta tests delta computation.
func TestComputeReviewDelta(t *testing.T) {
	prevState := &PRReviewState{
		Issues: []ReviewIssue{
			{ID: "1", Status: "open", Message: "Old issue"},
			{ID: "2", Status: "open", Message: "Fixed issue"},
		},
	}

	current := &ParsedReview{
		Issues: []ReviewIssue{
			{ID: "1", Status: "open", Message: "Old issue"},
			{ID: "3", Status: "open", Message: "New issue"},
		},
	}

	delta := ComputeReviewDelta(prevState, current)

	if len(delta.NewIssues) != 1 {
		t.Errorf("new issues: got %d, want 1", len(delta.NewIssues))
	}
	if len(delta.FixedIssues) != 1 {
		t.Errorf("fixed issues: got %d, want 1", len(delta.FixedIssues))
	}
	if len(delta.Unchanged) != 1 {
		t.Errorf("unchanged: got %d, want 1", len(delta.Unchanged))
	}

	// Verify new issue
	if delta.NewIssues[0].ID != "3" {
		t.Errorf("new issue ID: got %s, want 3", delta.NewIssues[0].ID)
	}

	// Verify fixed issue
	if delta.FixedIssues[0].ID != "2" {
		t.Errorf("fixed issue ID: got %s, want 2", delta.FixedIssues[0].ID)
	}

	// Verify unchanged issue
	if delta.Unchanged[0].ID != "1" {
		t.Errorf("unchanged issue ID: got %s, want 1", delta.Unchanged[0].ID)
	}
}

// TestComputeReviewDeltaNilPrevious tests delta with nil previous state.
func TestComputeReviewDeltaNilPrevious(t *testing.T) {
	current := &ParsedReview{
		Issues: []ReviewIssue{
			{ID: "1", Status: "open", Message: "Issue 1"},
			{ID: "2", Status: "open", Message: "Issue 2"},
		},
	}

	delta := ComputeReviewDelta(nil, current)

	if len(delta.NewIssues) != 2 {
		t.Errorf("new issues: got %d, want 2", len(delta.NewIssues))
	}
	if len(delta.FixedIssues) != 0 {
		t.Errorf("fixed issues: got %d, want 0", len(delta.FixedIssues))
	}
	if len(delta.Unchanged) != 0 {
		t.Errorf("unchanged: got %d, want 0", len(delta.Unchanged))
	}
}

// TestHashDiffPatch tests diff hash generation with memory protection.
func TestHashDiffPatch(t *testing.T) {
	// Small diff - should hash directly
	smallDiff := "line 1\nline 2\nline 3"
	hash1 := hashDiffPatch(smallDiff)
	hash2 := hashDiffPatch(smallDiff)
	if hash1 != hash2 {
		t.Error("hash of same diff is not stable")
	}

	// Different content should produce different hash
	differentDiff := "line 1\nline 2\nmodified"
	hash3 := hashDiffPatch(differentDiff)
	if hash1 == hash3 {
		t.Error("different content produced same hash")
	}

	// Empty diff should produce valid hash
	emptyHash := hashDiffPatch("")
	if emptyHash == "" {
		t.Error("empty diff produced empty hash")
	}
}

// TestHashDiffPatchLarge tests diff hash with large content (>10MB).
func TestHashDiffPatchLarge(t *testing.T) {
	// Create a large diff (>10MB)
	largeDiff := strings.Repeat("a", maxDiffSize+1)
	hash1 := hashDiffPatch(largeDiff)
	hash2 := hashDiffPatch(largeDiff)

	if hash1 != hash2 {
		t.Error("hash of large diff is not stable")
	}

	// Verify it's different from hash of just the prefix
	prefixHash := hashDiffPatch(largeDiff[:maxDiffSize])
	if hash1 == prefixHash {
		t.Error("large diff hash should differ from prefix hash")
	}
}

// TestEmbedStateNil tests embedding nil state.
func TestEmbedStateNil(t *testing.T) {
	comment := "## Review\n\nSome content"
	result := EmbedStateInComment(comment, nil)

	if result != comment {
		t.Error("embedding nil state should return comment unchanged")
	}
}

// TestExtractStateNoMarker tests extraction without state marker.
func TestExtractStateNoMarker(t *testing.T) {
	comment := "## Review\n\nSome content with no state"

	state, err := ExtractStateFromComment(comment)
	if err != nil && !errors.Is(err, ErrNoStateFound) {
		t.Errorf("unexpected error: %v", err)
	}
	if state != nil {
		t.Error("expected nil state when no marker present")
	}
}

// TestDetectProviderFromPR tests provider detection.
func TestDetectProviderFromPR(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		provider string
	}{
		{"GitHub", "https://github.com/owner/repo/pull/123", "github"},
		{"GitLab", "https://gitlab.com/owner/repo/merge_requests/123", "gitlab"},
		{"Bitbucket", "https://bitbucket.org/owner/repo/pull-requests/123", "bitbucket"},
		{"Azure DevOps", "https://dev.azure.com/org/project/_git/repo/pullrequest/123", "azuredevops"},
		{"Visual Studio", "https://visualstudio.com/org/project/_git/repo/pullrequest/123", "azuredevops"},
		{"Unknown", "https://unknown.com/repo/pull/123", ""},
		{"Nil PR", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pr *provider.PullRequest
			if tt.url != "" {
				pr = &provider.PullRequest{URL: tt.url}
			}
			result := detectProviderFromPR(pr)
			if result != tt.provider {
				t.Errorf("detectProviderFromPR(%q) = %q, want %q", tt.url, result, tt.provider)
			}
		})
	}
}

// TestFormatReviewComment tests review comment formatting.
func TestFormatReviewComment(t *testing.T) {
	review := &ParsedReview{
		Summary: "This is a summary",
		Overall: "approved",
		Issues: []ReviewIssue{
			{
				ID:       "1",
				File:     "test.go",
				Line:     42,
				Category: "correctness",
				Severity: "high",
				Message:  "Test issue",
			},
		},
	}

	delta := ReviewDelta{
		NewIssues:   review.Issues,
		FixedIssues: []ReviewIssue{},
		Unchanged:   []ReviewIssue{},
	}

	opts := PRReviewOptions{
		AcknowledgeFixes: true,
	}

	comment := FormatReviewComment(review, delta, opts)

	if !contains(comment, "AI PR Review") {
		t.Error("comment should contain 'AI PR Review'")
	}
	if !contains(comment, "Summary") {
		t.Error("comment should contain 'Summary'")
	}
	if !contains(comment, "approved") {
		t.Error("comment should contain 'approved'")
	}
	if !contains(comment, "test.go:42") {
		t.Error("comment should contain file location")
	}
	if !contains(comment, "Test issue") {
		t.Error("comment should contain issue message")
	}
}

// TestGroupIssuesByCategory tests issue grouping.
func TestGroupIssuesByCategory(t *testing.T) {
	issues := []ReviewIssue{
		{ID: "1", Category: "security", Message: "Sec issue"},
		{ID: "2", Category: "performance", Message: "Perf issue"},
		{ID: "3", Category: "security", Message: "Another sec issue"},
		{ID: "4", Category: "style", Message: "Style issue"},
	}

	grouped := groupIssuesByCategory(issues)

	if len(grouped["security"]) != 2 {
		t.Errorf("security issues: got %d, want 2", len(grouped["security"]))
	}
	if len(grouped["performance"]) != 1 {
		t.Errorf("performance issues: got %d, want 1", len(grouped["performance"]))
	}
	if len(grouped["style"]) != 1 {
		t.Errorf("style issues: got %d, want 1", len(grouped["style"]))
	}
}
