package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	taskDirName     = ".mehrhof"
	workSubDirName  = "work"
	plannedDirName  = "planned"
	activeTaskFile  = ".active_task"
	workFileName    = "work.yaml"
	notesFileName   = "notes.md"
	specsDirName    = "specifications"
	sessionsDirName = "sessions"
	configFileName  = "config.yaml"
	envFileName     = ".env"

	// Usage buffer configuration.
	defaultUsageFlushInterval  = 5 * time.Second // Auto-flush interval
	defaultUsageFlushThreshold = 10              // Flush after N updates
)

// usageBuffer tracks accumulated usage for a task/step combination.
type usageBuffer struct {
	inputTokens  int
	outputTokens int
	cachedTokens int
	costUSD      float64
	callCount    int
}

// Workspace manages task storage within a repository.
type Workspace struct {
	root          string // Repository root
	taskRoot      string // .mehrhof directory in project (config, .env)
	workspaceRoot string // ~/.valksor/mehrhof/workspaces/<project-id>/ (work/, .active_task)
	workRoot      string // ~/.valksor/mehrhof/workspaces/<project-id>/work/

	// Usage buffering for reducing I/O
	usageMu   sync.RWMutex
	usageBuf  map[string]map[string]*usageBuffer // taskID -> step -> buffer
	lastFlush time.Time
}

// OpenWorkspace opens or creates a workspace in the given directory.
// Split structure:
//   - .mehrhof/ in project: config.yaml, .env
//   - ~/.mehrhof/workspaces/<project-id>/: work/, .active_task
//
// If cfg is nil, defaults are used for work directory path.
func OpenWorkspace(ctx context.Context, repoRoot string, cfg *WorkspaceConfig) (*Workspace, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// taskRoot is in project directory (for config and .env)
	taskRoot := filepath.Join(absRoot, taskDirName)

	// Extract home directory override from config (for tests and custom setups)
	homeDirOverride := ""
	if cfg != nil && cfg.Storage.HomeDir != "" {
		homeDirOverride = cfg.Storage.HomeDir
	}

	// workspaceRoot is in home directory (for work/ and .active_task)
	workspaceRoot, err := GetWorkspaceDataDir(ctx, absRoot, homeDirOverride)
	if err != nil {
		return nil, fmt.Errorf("get workspace data directory: %w", err)
	}

	// Determine work directory path from config or default
	workDir := "work" // default - relative to workspaceRoot
	if cfg != nil && cfg.Storage.WorkDir != "" {
		workDir = cfg.Storage.WorkDir
		// If workDir starts with ".mehrhof/", make it relative to new location
		workDir = strings.TrimPrefix(workDir, ".mehrhof/")
	}
	// Work directory is relative to workspaceRoot (in home directory)
	workRoot := filepath.Join(workspaceRoot, workDir)

	return &Workspace{
		root:          absRoot,       // Repository root (for git operations)
		taskRoot:      taskRoot,      // .mehrhof in project (config, .env)
		workspaceRoot: workspaceRoot, // ~/.mehrhof/workspaces/<project-id>/ (work/, .active_task)
		workRoot:      workRoot,      // ~/.mehrhof/workspaces/<project-id>/work/
		usageBuf:      make(map[string]map[string]*usageBuffer),
		lastFlush:     time.Now(),
	}, nil
}

// Root returns the repository root path.
func (w *Workspace) Root() string {
	return w.root
}

// TaskRoot returns the .mehrhof directory path (in project, for config/.env).
func (w *Workspace) TaskRoot() string {
	return w.taskRoot
}

// WorkRoot returns the .mehrhof/work directory path.
func (w *Workspace) WorkRoot() string {
	return w.workRoot
}

// ConfigPath returns the path to the config file.
func (w *Workspace) ConfigPath() string {
	return filepath.Join(w.taskRoot, configFileName)
}

// HasConfig returns true if the config file exists.
func (w *Workspace) HasConfig() bool {
	_, err := os.Stat(w.ConfigPath())

	return err == nil
}

// EnvPath returns the path to the .env file.
func (w *Workspace) EnvPath() string {
	return filepath.Join(w.taskRoot, envFileName)
}

// LoadEnv reads the .env file and returns key-value pairs.
// Returns empty map if file doesn't exist.
func (w *Workspace) LoadEnv() (map[string]string, error) {
	data, err := os.ReadFile(w.EnvPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}

		return nil, fmt.Errorf("read .env: %w", err)
	}

	result := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return result, nil
}

// EnsureInitialized ensures that the workspace directories exist.
// Creates both:
// - .mehrhof/ in project (for config.yaml and .env)
// - work/ in home directory (for task data)
// Note: Does NOT update .gitignore - use UpdateGitignore() for that (typically only in init command).
func (w *Workspace) EnsureInitialized() error {
	// Create .mehrhof/ in project directory (for config.yaml and .env)
	if err := os.MkdirAll(w.taskRoot, 0o755); err != nil {
		return fmt.Errorf("create project .mehrhof directory: %w", err)
	}

	// Create work/ in home directory (for task data)
	if err := os.MkdirAll(w.workRoot, 0o755); err != nil {
		return fmt.Errorf("create workspace data directory: %w", err)
	}

	return nil
}

// UpdateGitignore updates .gitignore with mehrhof entries.
// Adds .mehrhof/.env (gitignored) but NOT .active_task or .mehrhof/work/
// since those are now stored in the home directory.
// This should only be called from the init command.
func (w *Workspace) UpdateGitignore() error {
	gitignorePath := filepath.Join(w.root, ".gitignore")

	// Read existing .gitignore
	var content string
	if _, err := os.Stat(gitignorePath); err == nil {
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("read .gitignore: %w", err)
		}
		content = string(data)
	}

	// Entries to ADD
	entriesToAdd := []string{
		taskDirName + "/.env", // .mehrhof/.env,
		taskDirName + "browser.json",
	}

	// Entries to REMOVE (legacy - no longer in project)
	entriesToRemove := []string{
		taskDirName + "/work/", // .mehrhof/work/ (moved to home)
		activeTaskFile,         // .active_task (moved to home)
	}

	modified := false
	lines := strings.Split(content, "\n")
	var newLines []string

	// First pass: remove legacy entries
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		shouldRemove := false
		for _, entry := range entriesToRemove {
			if trimmed == entry {
				shouldRemove = true

				break
			}
		}
		if !shouldRemove {
			newLines = append(newLines, line)
		} else {
			modified = true
		}
	}

	// Rebuild content
	content = strings.Join(newLines, "\n")

	// Second pass: add new entries if not present
	for _, entry := range entriesToAdd {
		found := false
		for _, line := range newLines {
			if strings.TrimSpace(line) == entry {
				found = true

				break
			}
		}

		if !found {
			if len(content) > 0 && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += entry + "\n"
			modified = true
		}
	}

	if modified {
		// Ensure trailing newline
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write .gitignore: %w", err)
		}
	}

	return nil
}

// NeedsMigration checks if the old .mehrhof directory exists in repo root.
func (w *Workspace) NeedsMigration() bool {
	oldTaskRoot := filepath.Join(w.root, taskDirName)
	info, err := os.Stat(oldTaskRoot)

	return err == nil && info.IsDir()
}

// GetLegacyTaskRoot returns the old .mehrhof path for migration.
func (w *Workspace) GetLegacyTaskRoot() string {
	return filepath.Join(w.root, taskDirName)
}
