package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Workflow action request/response types.

type finishRequest struct {
	SquashMerge  bool   `json:"squash_merge"`
	DeleteBranch bool   `json:"delete_branch"`
	TargetBranch string `json:"target_branch"`
	PushAfter    bool   `json:"push_after"`
	ForceMerge   bool   `json:"force_merge"`
	DraftPR      bool   `json:"draft_pr"`
	PRTitle      string `json:"pr_title"`
	PRBody       string `json:"pr_body"`
	FinishAction string `json:"finish_action"`
}

// handleFileUpload processes file upload and returns a file: ref.
func (s *Server) handleFileUpload(r *http.Request) (string, error) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return "", err
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".md" && ext != ".txt" && ext != ".markdown" {
		return "", &invalidFileError{ext: ext}
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return s.saveContentToFile(string(content))
}

// saveContentToFile saves content to a temp file and returns a file: ref.
// Storage location respects storage.save_in_project setting:
//   - save_in_project=true: .mehrhof/tasks/
//   - save_in_project=false: ~/.valksor/mehrhof/workspaces/<id>/tasks/
func (s *Server) saveContentToFile(content string) (string, error) {
	// Get tasks directory from conductor (respects storage config)
	var tasksDir string
	if s.config.Conductor != nil {
		tasksDir = s.config.Conductor.TasksDir()
	}
	// Fallback for global mode or empty result
	if tasksDir == "" {
		homeDir, err := storage.GetMehrhofHomeDir()
		if err != nil {
			return "", fmt.Errorf("get mehrhof home dir: %w", err)
		}
		tasksDir = filepath.Join(homeDir, "tasks")
	}

	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return "", err
	}

	// Create temp file
	f, err := os.CreateTemp(tasksDir, "task-*.md")
	if err != nil {
		return "", err
	}

	name := f.Name()

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()

		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return "file:" + name, nil
}

// fetchAndSaveURL fetches content from a URL and saves it to a temp file.
func (s *Server) fetchAndSaveURL(ctx context.Context, url string) (string, error) {
	client := httpclient.NewHTTPClient()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed: status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	return s.saveContentToFile(string(content))
}

// invalidFileError represents an invalid file extension error.
type invalidFileError struct {
	ext string
}

func (e *invalidFileError) Error() string {
	return "invalid file type: " + e.ext + " (expected .md, .txt, or .markdown)"
}
