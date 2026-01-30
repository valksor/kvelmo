package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const reviewsDirName = "reviews"

// resolveReviewFilenamePattern converts a pattern like "CODERABBIT-{n}.txt" with a number into a filename.
func resolveReviewFilenamePattern(pattern string, number int) string {
	if pattern == "" {
		pattern = "review-{n}.txt" // fallback default
	}

	return strings.Replace(pattern, "{n}", strconv.Itoa(number), 1)
}

// ReviewsDir returns the reviews directory path (internal storage).
func (w *Workspace) ReviewsDir(taskID string) string {
	return filepath.Join(w.WorkPath(taskID), reviewsDirName)
}

// ReviewPath returns the path for a review file (internal storage).
func (w *Workspace) ReviewPath(taskID string, number int, cfg *WorkspaceConfig) string {
	pattern := "review-{n}.txt"
	if cfg != nil && cfg.Review.FilenamePattern != "" {
		pattern = cfg.Review.FilenamePattern
	}

	return filepath.Join(w.ReviewsDir(taskID), resolveReviewFilenamePattern(pattern, number))
}

// ProjectReviewPath returns the project-local path for a review file.
// Reviews are stored alongside specs in the same project directory.
func (w *Workspace) ProjectReviewPath(taskID string, number int, cfg *WorkspaceConfig) string {
	pattern := "review-{n}.txt"
	if cfg != nil && cfg.Review.FilenamePattern != "" {
		pattern = cfg.Review.FilenamePattern
	}
	// Reviews go in the same directory as specs (uses spec's ProjectDir setting)
	return filepath.Join(w.ProjectSpecificationsDir(taskID, cfg), resolveReviewFilenamePattern(pattern, number))
}

// buildReviewPatternRegex builds a regex to match filenames based on the config pattern.
func buildReviewPatternRegex(pattern string) *regexp.Regexp {
	if pattern == "" {
		pattern = "review-{n}.txt"
	}
	escaped := regexp.QuoteMeta(pattern)
	regexStr := strings.Replace(escaped, `\{n\}`, `(\d+)`, 1)

	return regexp.MustCompile(`^` + regexStr + `$`)
}

// SaveReview saves a review file with atomic write.
// Saves to internal storage, optionally to project-local.
func (w *Workspace) SaveReview(taskID string, number int, content string) error {
	if !isValidTaskID(taskID) {
		return fmt.Errorf("invalid task ID %q: must contain only alphanumeric characters, hyphens, and underscores", taskID)
	}

	cfg, err := w.LoadConfig()
	if err != nil {
		slog.Warn("failed to load workspace config, using defaults",
			"task_id", taskID,
			"review_number", number,
			"error", err,
		)
		cfg = NewDefaultWorkspaceConfig()
	}

	// Step 1: Always save to internal storage
	reviewPath := w.ReviewPath(taskID, number, cfg)
	reviewDir := filepath.Dir(reviewPath)
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		return fmt.Errorf("create reviews directory: %w", err)
	}
	if err := os.WriteFile(reviewPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("save internal review: %w", err)
	}

	// Step 2: Save to project-local storage if enabled
	if cfg.Review.SaveInProject {
		projectReviewPath := w.ProjectReviewPath(taskID, number, cfg)
		projectDir := filepath.Dir(projectReviewPath)

		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			slog.Error("failed to create project reviews directory",
				"task_id", taskID,
				"review_number", number,
				"path", projectDir,
				"error", err,
			)

			return nil
		}

		// Atomic write
		tmpPath := projectReviewPath + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 36)
		if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
			slog.Error("failed to write project-local review (temp file)",
				"task_id", taskID,
				"review_number", number,
				"path", tmpPath,
				"error", err,
			)

			return nil
		}

		if err := syncFile(tmpPath); err != nil {
			slog.Error("failed to sync project-local review before rename",
				"task_id", taskID,
				"review_number", number,
				"path", tmpPath,
				"error", err,
			)
			_ = os.Remove(tmpPath)

			return nil
		}

		if err := os.Rename(tmpPath, projectReviewPath); err != nil {
			_ = os.Remove(tmpPath)
			slog.Error("failed to finalize project-local review (rename)",
				"task_id", taskID,
				"review_number", number,
				"error", err,
			)

			return nil
		}
	}

	return nil
}

// LoadReview loads a review file content.
func (w *Workspace) LoadReview(taskID string, number int) (string, error) {
	cfg, _ := w.LoadConfig()
	reviewPath := w.ReviewPath(taskID, number, cfg)
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ListReviews returns all review numbers for a task.
func (w *Workspace) ListReviews(taskID string) ([]int, error) {
	cfg, _ := w.LoadConfig()
	reviewsDir := w.ReviewsDir(taskID)

	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}

		return nil, fmt.Errorf("read reviews directory: %w", err)
	}

	pattern := buildReviewPatternRegex(cfg.Review.FilenamePattern)
	var numbers []int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := pattern.FindStringSubmatch(entry.Name())
		if matches != nil {
			num, _ := strconv.Atoi(matches[1])
			numbers = append(numbers, num)
		}
	}

	slices.Sort(numbers)

	return numbers, nil
}

// NextReviewNumber returns the next available review number.
func (w *Workspace) NextReviewNumber(taskID string) (int, error) {
	reviews, err := w.ListReviews(taskID)
	if err != nil {
		return 0, err
	}

	if len(reviews) == 0 {
		return 1, nil
	}

	return reviews[len(reviews)-1] + 1, nil
}
