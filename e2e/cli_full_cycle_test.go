//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/valksor/kvelmo/pkg/agent/claude"
	"github.com/valksor/kvelmo/pkg/socket"
	"golang.org/x/oauth2"
)

// TestCLIFullCycle tests the complete kvelmo workflow via CLI.
// Uses real Claude binary and real GitHub API.
//
// Run with:
//
//	E2E_GITHUB_REPO=ozo2003/e2e-test \
//	GITHUB_TOKEN=github_pat_... \
//	go test -tags=e2e -v -timeout=30m ./test/e2e/... -run TestCLIFullCycle
func TestCLIFullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full cycle test in short mode")
	}

	// 1. Check prerequisites
	checkClaudeAvailable(t)
	token, repo := getE2EConfig(t)
	parts := strings.SplitN(repo, "/", 2)
	owner, repoName := parts[0], parts[1]

	// 2. Build kvelmo binary
	kvelmoPath := buildKvelmo(t)

	// 3. Create test issue
	client := newGitHubClient(token)
	issueNum := createTestIssue(t, client, owner, repoName)
	t.Logf("Created test issue #%d", issueNum)

	// Setup cleanup
	var createdBranch string
	var createdPRNum int
	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") != "" {
			t.Log("Skipping cleanup (E2E_SKIP_CLEANUP set)")
			return
		}

		ctx := context.Background()

		// Close PR if created
		if createdPRNum > 0 {
			state := "closed"
			_, _, _ = client.PullRequests.Edit(ctx, owner, repoName, createdPRNum, &github.PullRequest{State: &state})
			t.Logf("Closed PR #%d", createdPRNum)
		}

		// Delete branch if created
		if createdBranch != "" {
			_, _ = client.Git.DeleteRef(ctx, owner, repoName, "refs/heads/"+createdBranch)
			t.Logf("Deleted branch: %s", createdBranch)
		}

		// Close issue
		state := "closed"
		_, _, _ = client.Issues.Edit(ctx, owner, repoName, issueNum, &github.IssueRequest{State: &state})
		t.Logf("Closed issue #%d", issueNum)
	})

	// 4. Setup work directory (clone repo)
	workDir := setupWorkDir(t, token, owner, repoName)

	// 5. Start kvelmo socket in background
	t.Log("Step 1: Starting kvelmo socket in background...")
	stopKvelmo := startKvelmoBackground(t, kvelmoPath, workDir, token, repo, issueNum)
	defer stopKvelmo()

	// Wait for socket to be ready
	waitForSocket(t, workDir)

	// Test infrastructure commands early
	t.Log("Testing infrastructure commands...")
	runKvelmo(t, kvelmoPath, workDir, "config", "show")
	runKvelmo(t, kvelmoPath, workDir, "diagnose")

	// 6. Wait for task to be loaded (start command loads it asynchronously while holding lock)
	// Use waitForState with timeout since the conductor holds a lock during GitHub fetch
	waitForState(t, kvelmoPath, workDir, "loaded", 60*time.Second)
	t.Log("Task loaded successfully")

	// Test status (human-readable) and list commands
	runKvelmo(t, kvelmoPath, workDir, "status")
	runKvelmo(t, kvelmoPath, workDir, "list")

	// Get branch name for cleanup
	statusOutput := runKvelmoCapture(t, kvelmoPath, workDir, "status", "--json")
	var statusResult map[string]any
	if err := json.Unmarshal([]byte(statusOutput), &statusResult); err == nil {
		if branch, ok := statusResult["branch"].(string); ok {
			createdBranch = branch
			t.Logf("Created branch: %s", createdBranch)
		}
	}

	// 7. Run planning phase
	t.Log("Step 2: Running planning phase with Claude...")
	runKvelmo(t, kvelmoPath, workDir, "plan")
	waitForState(t, kvelmoPath, workDir, "planned", 5*time.Minute)
	t.Log("Planning completed")

	// Verify checkpoints work
	runKvelmo(t, kvelmoPath, workDir, "checkpoints")

	// 8. Run implementation phase
	t.Log("Step 3: Running implementation phase with Claude...")
	runKvelmo(t, kvelmoPath, workDir, "implement")
	waitForState(t, kvelmoPath, workDir, "implemented", 8*time.Minute)
	t.Log("Implementation completed")

	// Test git commands after implementation
	t.Log("Testing git and file commands...")
	runKvelmo(t, kvelmoPath, workDir, "git", "status")
	runKvelmo(t, kvelmoPath, workDir, "git", "log")
	runKvelmo(t, kvelmoPath, workDir, "files", "list", ".")
	runKvelmo(t, kvelmoPath, workDir, "jobs", "list")

	// 9. Run simplify pass (optional but test it)
	t.Log("Step 4: Running simplify pass...")
	runKvelmo(t, kvelmoPath, workDir, "simplify")
	waitForState(t, kvelmoPath, workDir, "implemented", 5*time.Minute)
	t.Log("Simplify completed")

	// 10. Run optimize pass (optional but test it)
	t.Log("Step 5: Running optimize pass...")
	runKvelmo(t, kvelmoPath, workDir, "optimize")
	waitForState(t, kvelmoPath, workDir, "implemented", 5*time.Minute)
	t.Log("Optimize completed")

	// 11. Test undo/redo functionality (skip if less than 2 checkpoints - need previous state to undo to)
	t.Log("Step 6: Testing undo/redo and checkpoint navigation...")
	checkpointsOutput := runKvelmoCapture(t, kvelmoPath, workDir, "checkpoints")
	// Count checkpoint lines (they start with "  " or "* " followed by a number)
	checkpointCount := strings.Count(checkpointsOutput, ". ")
	if strings.Contains(checkpointsOutput, "No checkpoints") || checkpointCount < 2 {
		t.Logf("Skipping undo/redo - need at least 2 checkpoints, got %d", checkpointCount)
	} else {
		runKvelmo(t, kvelmoPath, workDir, "undo")
		runKvelmo(t, kvelmoPath, workDir, "redo")
		t.Log("Undo/redo completed")
	}

	// 12. Run quality check explicitly before review
	t.Log("Step 7: Running quality check...")
	runKvelmo(t, kvelmoPath, workDir, "quality")

	// 13. Review and submit
	t.Log("Step 8: Reviewing and submitting...")
	runKvelmo(t, kvelmoPath, workDir, "review", "--approve")
	runKvelmo(t, kvelmoPath, workDir, "submit")

	submitState := getKvelmoState(t, kvelmoPath, workDir)
	if submitState != "submitted" {
		t.Fatalf("Expected state 'submitted', got '%s'", submitState)
	}
	t.Log("PR submitted")

	// Get PR number for cleanup
	prs, _, _ := client.PullRequests.List(context.Background(), owner, repoName, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, createdBranch),
	})
	if len(prs) > 0 {
		createdPRNum = prs[0].GetNumber()
		t.Logf("Created PR #%d: %s", createdPRNum, prs[0].GetHTMLURL())
	}

	// 14. Approve and merge via remote commands
	// Note: Self-approval is not allowed on GitHub, so we skip approval and merge directly.
	// In a real workflow, approval would come from a different user/token.
	t.Log("Step 9: Merging PR (skipping approval - GitHub doesn't allow self-approval)...")
	runKvelmo(t, kvelmoPath, workDir, "remote", "merge", "--method", "squash")
	t.Log("PR merged")

	// 15. Refresh and finish
	t.Log("Step 10: Refreshing and finishing...")
	runKvelmo(t, kvelmoPath, workDir, "refresh")
	runKvelmo(t, kvelmoPath, workDir, "finish")

	// 16. Verify final state
	finalState := getKvelmoState(t, kvelmoPath, workDir)
	if finalState != "none" {
		t.Fatalf("Expected final state 'none', got '%s'", finalState)
	}
	t.Log("Workflow completed successfully!")

	// Branch was deleted by finish, clear for cleanup
	createdBranch = ""
	createdPRNum = 0
}

// checkClaudeAvailable verifies Claude CLI is installed.
func checkClaudeAvailable(t *testing.T) {
	t.Helper()
	claudeAgent := claude.New()
	if err := claudeAgent.Available(); err != nil {
		t.Skipf("Claude CLI not available: %v", err)
	}
}

// getE2EConfig gets configuration from environment.
func getE2EConfig(t *testing.T) (token, repo string) {
	t.Helper()

	repo = os.Getenv("E2E_GITHUB_REPO")
	if repo == "" {
		t.Skip("E2E_GITHUB_REPO not set")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("E2E_GITHUB_REPO must be in owner/repo format, got: %s", repo)
	}

	token = os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("E2E_GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("GITHUB_TOKEN or E2E_GITHUB_TOKEN not set")
	}

	return token, repo
}

// newGitHubClient creates a GitHub client.
func newGitHubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(httpClient)
}

// buildKvelmo compiles the kvelmo binary and returns the path.
func buildKvelmo(t *testing.T) string {
	t.Helper()

	// Build to temp directory
	tmpDir, err := os.MkdirTemp("", "kvelmo-e2e-bin-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}

	binaryPath := filepath.Join(tmpDir, "kvelmo")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/kvelmo")
	cmd.Dir = findProjectRoot(t)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Build kvelmo: %v\n%s", err, output)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return binaryPath
}

// findProjectRoot finds the kvelmo project root.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory and walk up to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

// createTestIssue creates a simple test issue.
// Uses a unique timestamp-based filename to avoid conflicts with previous test runs.
func createTestIssue(t *testing.T, client *github.Client, owner, repo string) int {
	t.Helper()

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("test-output-%d.txt", timestamp)
	title := fmt.Sprintf("E2E CLI Test %d", timestamp)
	body := fmt.Sprintf(`## Description

Create a simple test file with timestamp %d.

## Acceptance Criteria

- [ ] Create a file called %s
- [ ] The file should contain "Test completed at timestamp %d"

## Implementation Notes

This is a minimal task for E2E testing. Create the file with the exact specified filename and content.
The unique filename ensures this test doesn't conflict with previous test runs.`, timestamp, filename, timestamp)

	issue, _, err := client.Issues.Create(context.Background(), owner, repo, &github.IssueRequest{
		Title: &title,
		Body:  &body,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	return issue.GetNumber()
}

// setupWorkDir clones the repo to a temp directory.
func setupWorkDir(t *testing.T, token, owner, repo string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kvelmo-e2e-work-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}

	t.Cleanup(func() {
		if os.Getenv("E2E_SKIP_CLEANUP") == "" {
			os.RemoveAll(tmpDir)
		} else {
			t.Logf("Keeping temp dir: %s", tmpDir)
		}
	})

	// Clone the repo
	repoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)
	cmd := exec.Command("git", "clone", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v (check repo access and token permissions)", err)
	} else {
		t.Log("Clone completed successfully")
		_ = output
	}

	// Configure git user for commits
	runGitCmd(t, tmpDir, "config", "user.email", "test@e2e.local")
	runGitCmd(t, tmpDir, "config", "user.name", "E2E Test")

	// Create .valksor directory with config
	valksorDir := filepath.Join(tmpDir, ".valksor")
	if err := os.MkdirAll(valksorDir, 0o755); err != nil {
		t.Fatalf("Create .valksor dir: %v", err)
	}

	// Write .env with the GitHub token (kvelmo loads tokens from .env files)
	envContent := fmt.Sprintf("GITHUB_TOKEN=%s\n", token)
	if err := os.WriteFile(filepath.Join(valksorDir, ".env"), []byte(envContent), 0o600); err != nil {
		t.Fatalf("Write .env file: %v", err)
	}

	// Write kvelmo.yaml with save_in_project=true to isolate state per test
	// Also set coderabbit mode to "never" to avoid interactive prompts in quality gate
	settingsContent := `storage:
  save_in_project: true
workflow:
  coderabbit:
    mode: never
`
	if err := os.WriteFile(filepath.Join(valksorDir, "kvelmo.yaml"), []byte(settingsContent), 0o644); err != nil {
		t.Fatalf("Write kvelmo.yaml: %v", err)
	}

	return tmpDir
}

// startKvelmoBackground starts kvelmo in foreground mode as a subprocess and returns a stop function.
func startKvelmoBackground(t *testing.T, kvelmoPath, workDir, token, repo string, issueNum int) func() {
	t.Helper()

	// Start kvelmo with the --from flag to load the task
	cmd := exec.Command(kvelmoPath, "start", "--from", fmt.Sprintf("github:%s#%d", repo, issueNum))
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+token)

	// Create a pipe to capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start kvelmo: %v", err)
	}
	t.Logf("Started kvelmo (PID: %d)", cmd.Process.Pid)

	// Channel to signal goroutines to stop logging
	stopLogging := make(chan struct{})
	var logWg sync.WaitGroup

	// Read output in background for debugging
	logWg.Add(2)
	go func() {
		defer logWg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				select {
				case <-stopLogging:
					return
				default:
					t.Logf("kvelmo stdout: %s", buf[:n])
				}
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		defer logWg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				select {
				case <-stopLogging:
					return
				default:
					t.Logf("kvelmo stderr: %s", buf[:n])
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Return stop function
	return func() {
		t.Log("Stopping kvelmo...")

		// Signal logging goroutines to stop
		close(stopLogging)

		if cmd.Process != nil {
			// Send SIGTERM for graceful shutdown
			_ = cmd.Process.Signal(syscall.SIGTERM)

			// Wait with timeout
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case <-done:
				t.Log("kvelmo stopped gracefully")
			case <-time.After(5 * time.Second):
				t.Log("kvelmo didn't stop, killing...")
				_ = cmd.Process.Kill()
			}
		}

		// Wait for logging goroutines to finish
		logWg.Wait()
	}
}

// waitForSocket waits for the kvelmo socket to be ready.
func waitForSocket(t *testing.T, workDir string) {
	t.Helper()

	socketPath := socket.WorktreeSocketPath(workDir)
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		if socket.SocketExists(socketPath) {
			t.Logf("Socket ready: %s", socketPath)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for socket: %s", socketPath)
}

// runGitCmd runs a git command.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// runKvelmo runs kvelmo and checks for errors.
func runKvelmo(t *testing.T, kvelmoPath, workDir string, args ...string) {
	t.Helper()

	cmd := exec.Command(kvelmoPath, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.Logf("Running: kvelmo %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		t.Fatalf("kvelmo %v: %v\nstdout: %s\nstderr: %s",
			args, err, stdout.String(), stderr.String())
	}

	if stdout.Len() > 0 {
		t.Logf("stdout: %s", stdout.String())
	}
}

// runKvelmoCapture runs kvelmo and returns stdout.
func runKvelmoCapture(t *testing.T, kvelmoPath, workDir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(kvelmoPath, args...)
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("kvelmo %v: %v\nstderr: %s", args, err, exitErr.Stderr)
		}
		t.Fatalf("kvelmo %v: %v", args, err)
	}

	return string(output)
}

// getKvelmoState returns the current kvelmo state.
func getKvelmoState(t *testing.T, kvelmoPath, workDir string) string {
	t.Helper()

	output := runKvelmoCapture(t, kvelmoPath, workDir, "status", "--json")

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Parse status JSON: %v\nOutput: %s", err, output)
	}

	state, ok := result["state"].(string)
	if !ok {
		t.Fatalf("No state in status output: %v", result)
	}

	return state
}

// tryGetKvelmoState tries to get the state, returning empty string on error.
func tryGetKvelmoState(kvelmoPath, workDir string) string {
	cmd := exec.Command(kvelmoPath, "status", "--json", "--timeout", "10s")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "" // Return empty on error, caller will retry
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return ""
	}

	state, _ := result["state"].(string)
	return state
}

// waitForState polls until the expected state is reached.
func waitForState(t *testing.T, kvelmoPath, workDir string, expected string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		// Use tryGetKvelmoState to handle temporary timeouts during task loading
		state := tryGetKvelmoState(kvelmoPath, workDir)
		if state == expected {
			return
		}

		// Check for failure states
		if state == "failed" {
			t.Fatalf("Task entered failed state while waiting for %s", expected)
		}

		// Empty state means error/timeout, just retry
		time.Sleep(pollInterval)
	}

	t.Fatalf("Timeout waiting for state %s (current: %s)", expected, getKvelmoState(t, kvelmoPath, workDir))
}
