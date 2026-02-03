package storage

import (
	"os"
	"strings"
	"testing"
)

func TestReviewsDir(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	path := ws.ReviewsDir("test123")
	if !strings.HasSuffix(path, "/test123/reviews") {
		t.Errorf("ReviewsDir() = %q, want suffix /test123/reviews", path)
	}
}

func TestReviewPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	cfg := NewDefaultWorkspaceConfig()

	// Default pattern: review-{n}.txt
	path := ws.ReviewPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "/test123/reviews/review-1.txt") {
		t.Errorf("ReviewPath() = %q, want suffix /test123/reviews/review-1.txt", path)
	}

	// Custom pattern: CODERABBIT-{n}.txt
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"
	path = ws.ReviewPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "/test123/reviews/CODERABBIT-1.txt") {
		t.Errorf("ReviewPath(CODERABBIT) = %q, want suffix /test123/reviews/CODERABBIT-1.txt", path)
	}
}

func TestProjectReviewPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	cfg := NewDefaultWorkspaceConfig()

	// Default: reviews go alongside specs in .mehrhof/work/<task-id>/
	path := ws.ProjectReviewPath("test123", 1, cfg)
	if !strings.HasSuffix(path, ".mehrhof/work/test123/review-1.txt") {
		t.Errorf("ProjectReviewPath() = %q, want suffix .mehrhof/work/test123/review-1.txt", path)
	}

	// Custom ProjectDir: tickets/
	cfg.Storage.ProjectDir = "tickets"
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"
	path = ws.ProjectReviewPath("test123", 1, cfg)
	if !strings.HasSuffix(path, "tickets/test123/CODERABBIT-1.txt") {
		t.Errorf("ProjectReviewPath(tickets) = %q, want suffix tickets/test123/CODERABBIT-1.txt", path)
	}
}

func TestSaveAndLoadReview(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	content := "# Code Review\n\nThis is a test review."
	if err := ws.SaveReview("test123", 1, content); err != nil {
		t.Fatalf("SaveReview failed: %v", err)
	}

	loaded, err := ws.LoadReview("test123", 1)
	if err != nil {
		t.Fatalf("LoadReview failed: %v", err)
	}

	if loaded != content {
		t.Errorf("LoadReview() = %q, want %q", loaded, content)
	}
}

func TestListReviews(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// Save multiple reviews
	for i := 1; i <= 3; i++ {
		if err := ws.SaveReview("test123", i, "Review content"); err != nil {
			t.Fatalf("SaveReview(%d) failed: %v", i, err)
		}
	}

	reviews, err := ws.ListReviews("test123")
	if err != nil {
		t.Fatalf("ListReviews failed: %v", err)
	}

	if len(reviews) != 3 {
		t.Errorf("ListReviews() returned %d reviews, want 3", len(reviews))
	}

	// Should be sorted
	for i, num := range reviews {
		if num != i+1 {
			t.Errorf("reviews[%d] = %d, want %d", i, num, i+1)
		}
	}
}

func TestNextReviewNumber(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	// No reviews yet
	num, err := ws.NextReviewNumber("test123")
	if err != nil {
		t.Fatalf("NextReviewNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("NextReviewNumber() = %d, want 1", num)
	}

	// Save review 1
	if err := ws.SaveReview("test123", 1, "Review 1"); err != nil {
		t.Fatalf("SaveReview(1) failed: %v", err)
	}

	num, err = ws.NextReviewNumber("test123")
	if err != nil {
		t.Fatalf("NextReviewNumber failed: %v", err)
	}
	if num != 2 {
		t.Errorf("NextReviewNumber() = %d, want 2", num)
	}
}

func TestSaveReviewInProject(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Enable project-local saving (mutually exclusive - project ONLY)
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Storage.SaveInProject = true
	cfg.Storage.ProjectDir = "tickets"
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork("test123", source); err != nil {
		t.Fatalf("CreateWork(test123): %v", err)
	}

	content := "# CodeRabbit Review\n\nIssues found."
	if err := ws.SaveReview("test123", 1, content); err != nil {
		t.Fatalf("SaveReview failed: %v", err)
	}

	// Reload config
	cfg, _ = ws.LoadConfig()

	// Verify project-local storage exists (mutually exclusive: project ONLY)
	projectPath := ws.ProjectReviewPath("test123", 1, cfg)
	if _, err := os.Stat(projectPath); err != nil {
		t.Errorf("Project-local review not found: %s, error: %v", projectPath, err)
	}

	// Verify internal storage does NOT exist (mutually exclusive)
	internalPath := ws.ReviewPath("test123", 1, cfg)
	if _, err := os.Stat(internalPath); err == nil {
		t.Errorf("Internal review should NOT exist when save_in_project=true: %s", internalPath)
	}

	// Verify it's in tickets/<task-id>/CODERABBIT-1.txt
	if !strings.HasSuffix(projectPath, "tickets/test123/CODERABBIT-1.txt") {
		t.Errorf("Project review path = %q, want suffix tickets/test123/CODERABBIT-1.txt", projectPath)
	}

	// Verify content matches
	loaded, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile(projectPath) failed: %v", err)
	}
	if string(loaded) != content {
		t.Errorf("Project review content mismatch, got %q, want %q", string(loaded), content)
	}

	// Verify LoadReview returns correct content
	loadedViaAPI, err := ws.LoadReview("test123", 1)
	if err != nil {
		t.Fatalf("LoadReview failed: %v", err)
	}
	if loadedViaAPI != content {
		t.Errorf("LoadReview content mismatch, got %q, want %q", loadedViaAPI, content)
	}
}

func TestResolveReviewFilenamePattern(t *testing.T) {
	tests := []struct {
		pattern  string
		number   int
		expected string
	}{
		{"review-{n}.txt", 1, "review-1.txt"},
		{"review-{n}.txt", 42, "review-42.txt"},
		{"CODERABBIT-{n}.txt", 1, "CODERABBIT-1.txt"},
		{"CODERABBIT-{n}.txt", 5, "CODERABBIT-5.txt"},
		{"", 1, "review-1.txt"}, // empty pattern uses default
	}

	for _, tt := range tests {
		result := resolveReviewFilenamePattern(tt.pattern, tt.number)
		if result != tt.expected {
			t.Errorf("resolveReviewFilenamePattern(%q, %d) = %q, want %q",
				tt.pattern, tt.number, result, tt.expected)
		}
	}
}
