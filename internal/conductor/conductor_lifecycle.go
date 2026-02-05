package conductor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/valksor/go-mehrhof/internal/coordination"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

// TaskConflictInfo provides details when starting a task conflicts with an active one.
// Used by CLI/Web handlers to present appropriate UI before calling Start().
type TaskConflictInfo struct {
	ActiveTaskID    string // ID of the currently active task
	ActiveTaskTitle string // Title of the active task (if available)
	ActiveBranch    string // Git branch of the active task
	UsingWorktree   bool   // Whether the active task uses a worktree
}

// Initialize sets up the conductor for a repository.
func (c *Conductor) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Early config load to detect code_dir before git init.
	// This allows git to be initialized from the code target directory
	// when the project hub is separate from the codebase.
	var earlyCfg *storage.WorkspaceConfig
	earlyConfigPath := filepath.Join(c.opts.WorkDir, ".mehrhof", "config.yaml")
	if data, err := os.ReadFile(earlyConfigPath); err == nil {
		earlyCfg = storage.NewDefaultWorkspaceConfig()
		if err := yaml.Unmarshal(data, earlyCfg); err != nil {
			return fmt.Errorf("parse config file: %w", err)
		}
	}

	// Determine git init directory: use code_dir if configured, otherwise opts.WorkDir
	gitInitDir := c.opts.WorkDir
	if earlyCfg != nil && earlyCfg.Project.CodeDir != "" {
		codeDir := os.ExpandEnv(earlyCfg.Project.CodeDir)
		if !filepath.IsAbs(codeDir) {
			codeDir = filepath.Join(c.opts.WorkDir, codeDir)
		}
		resolved, err := filepath.Abs(codeDir)
		if err != nil {
			slog.Warn("code_dir configured but not accessible, falling back to project root",
				"code_dir", earlyCfg.Project.CodeDir, "error", err)
		} else if info, statErr := os.Stat(resolved); statErr != nil {
			slog.Warn("code_dir configured but not accessible, falling back to project root",
				"code_dir", earlyCfg.Project.CodeDir, "resolved", resolved, "error", statErr)
		} else if !info.IsDir() {
			slog.Warn("code_dir configured but not a directory, falling back to project root",
				"code_dir", earlyCfg.Project.CodeDir, "resolved", resolved)
		} else {
			gitInitDir = resolved
		}
	}

	// Initialize git (optional - might not be in a git repo)
	git, err := vcs.New(ctx, gitInitDir)
	if err == nil {
		c.git = git
	}

	// Determine workspace root (project hub, where .mehrhof/ config lives)
	// When code_dir is set, the hub stays at opts.WorkDir regardless of git root.
	// When code_dir is NOT set, use git root as the hub (existing behavior).
	root := c.opts.WorkDir
	hasCodeDir := earlyCfg != nil && earlyCfg.Project.CodeDir != ""
	if !hasCodeDir && c.git != nil {
		if c.git.IsWorktree() {
			// Get main repo path for shared storage
			mainRepo, err := c.git.GetMainWorktreePath(ctx)
			if err != nil {
				return fmt.Errorf("get main repo from worktree: %w", err)
			}
			root = mainRepo
		} else {
			root = c.git.Root()
		}
	}

	// Load workspace config from the determined root
	var cfg *storage.WorkspaceConfig
	configPath := filepath.Join(root, ".mehrhof", "config.yaml")
	if data, err := os.ReadFile(configPath); err == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("parse config file: %w", err)
		}
	}

	// Apply HomeDir override from options (for testing)
	// If cfg is nil (no config file), create a default config to set HomeDir
	if c.opts.HomeDir != "" {
		if cfg == nil {
			cfg = storage.NewDefaultWorkspaceConfig()
		}
		cfg.Storage.HomeDir = c.opts.HomeDir
	}

	// Initialize workspace with config
	ws, err := storage.OpenWorkspace(ctx, root, cfg)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	c.workspace = ws

	// Auto-initialize if requested
	if c.opts.AutoInit {
		if err := ws.EnsureInitialized(); err != nil {
			return fmt.Errorf("auto-initialize workspace: %w", err)
		}
	}

	// Handle task detection differently based on context
	if c.git != nil && c.git.IsWorktree() {
		// Auto-detect task from worktree path
		active, err := ws.FindTaskByWorktreePath(c.git.Root())
		if err != nil {
			return fmt.Errorf("find task by worktree: %w", err)
		}
		if active != nil {
			c.activeTask = active
			// Load associated work
			work, err := ws.LoadWork(active.ID)
			if err == nil {
				c.taskWork = work
				// Restore state machine state
				c.machine.SetWorkUnit(c.buildWorkUnit())
			}
		}
	} else {
		// Standard behavior: check for existing active task from .active_task file
		if ws.HasActiveTask() {
			active, err := ws.LoadActiveTask()
			if err == nil {
				c.activeTask = active
				// Load associated work
				work, err := ws.LoadWork(active.ID)
				if err == nil {
					c.taskWork = work
					// Restore state machine state
					c.machine.SetWorkUnit(c.buildWorkUnit())
				}
			}
		}
	}

	// Register user-defined agent aliases from workspace config
	if c.workspace != nil {
		if cfg, err := c.workspace.LoadConfig(); err == nil {
			if err := c.registerAliasAgents(cfg); err != nil {
				return fmt.Errorf("register alias agents: %w", err)
			}

			// Load plugins
			if err := c.loadPlugins(ctx, cfg); err != nil {
				// Plugins are optional, but log the error for debugging
				// Don't fail initialization since plugins are optional
				c.logError(fmt.Errorf("load plugins (non-fatal): %w", err))
			}

			// Initialize memory system
			if err := c.InitializeMemory(ctx); err != nil {
				// Memory is optional, but log the error for debugging
				// Don't fail initialization since memory is optional
				c.logError(fmt.Errorf("initialize memory (non-fatal): %w", err))
			}

			// Initialize ML system
			if err := c.InitializeML(ctx); err != nil {
				// ML is optional, but log the error for debugging
				// Don't fail initialization since ML is optional
				c.logError(fmt.Errorf("initialize ML (non-fatal): %w", err))
			}

			// Initialize library system
			if err := c.InitializeLibrary(ctx); err != nil {
				// Library is optional, but store error for better UX when user tries to use it
				// Don't fail initialization since library is optional
				c.libraryInitErr = err
				c.logError(fmt.Errorf("initialize library (non-fatal): %w", err))
			}
		}
	}

	// Select agent with priority: CLI flag > stored task agent > auto-detect
	if c.opts.AgentName != "" {
		// Priority 1: CLI flag always wins
		agentInst, err := c.agents.Get(c.opts.AgentName)
		if err != nil {
			return fmt.Errorf("get agent %s: %w", c.opts.AgentName, err)
		}
		c.activeAgent = agentInst
	} else if c.taskWork != nil && c.taskWork.Agent.Name != "" {
		// Priority 2: Restore agent from stored task config (when resuming)
		agentInst, err := c.agents.Get(c.taskWork.Agent.Name)
		if err != nil {
			// Stored agent not available, fall back to auto-detect
			agentInst, err = c.agents.Detect()
			if err != nil {
				return fmt.Errorf("detect agent: %w", err)
			}
		} else {
			// Re-apply inline env vars if stored
			agentInst = coordination.ApplyEnvs(agentInst, c.taskWork.Agent.InlineEnv)
			// Re-apply args if stored
			if len(c.taskWork.Agent.Args) > 0 {
				agentInst = agentInst.WithArgs(c.taskWork.Agent.Args...)
			}
		}
		c.activeAgent = agentInst
	} else {
		// Priority 3: Auto-detect
		agentInst, err := c.agents.Detect()
		if err != nil {
			return fmt.Errorf("detect agent: %w", err)
		}
		c.activeAgent = agentInst
	}

	// Apply workspace env vars to agent (filtered by agent name prefix)
	if c.workspace != nil {
		if cfg, err := c.workspace.LoadConfig(); err == nil {
			agentEnv := cfg.GetEnvForAgent(c.activeAgent.Name())
			for k, v := range agentEnv {
				c.activeAgent = c.activeAgent.WithEnv(k, v)
			}
		}
	}

	return nil
}

// Start registers a new task from a reference (does not run planning).
func (c *Conductor) Start(ctx context.Context, reference string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reject starting new tasks from within a worktree
	if c.git != nil && c.git.IsWorktree() {
		mainRepo, _ := c.git.GetMainWorktreePath(ctx)

		return fmt.Errorf("this command must be run from the main repository; you are currently in a worktree, return to the main repository first: cd %s", mainRepo)
	}

	// Check for existing active task (only applies in main repo)
	if c.activeTask != nil {
		return fmt.Errorf("task already active: %s (use 'task status' to check)", c.activeTask.ID)
	}

	// If git is available and branch creation requested, check for clean workspace FIRST
	if err := c.prepareWorkspace(ctx); err != nil {
		return err
	}

	// Detect provider and fetch work unit
	p, workUnit, err := c.fetchWorkUnit(ctx, reference)
	if err != nil {
		return err
	}

	// Check if there's a local queue task with matching external ID.
	// This merges local metadata (code examples, file references, custom frontmatter)
	// into the provider work unit so the agent sees both provider data and local enrichments.
	var localQueueTask *storage.QueuedTask
	if workUnit.ExternalID != "" && c.workspace != nil {
		var queueErr error
		localQueueTask, queueErr = c.workspace.FindQueueTaskByExternalID(workUnit.ExternalID)
		if queueErr != nil {
			slog.Warn("search local queues for metadata enrichment", "external_id", workUnit.ExternalID, "error", queueErr)
		}
		if localQueueTask != nil {
			c.mergeLocalMetadata(workUnit, localQueueTask)
		}
	}

	// Capture task agent config from workUnit (if specified in task frontmatter)
	c.taskAgentConfig = workUnit.AgentConfig

	// Determine task ID: use external key with timestamp suffix to prevent collisions on restart.
	// The timestamp ensures uniqueness if the same external task is started multiple times
	// (e.g., after abandon/finish) while preserving human-readable provider ID prefix.
	taskID := workUnit.ExternalKey
	if taskID != "" {
		taskID = fmt.Sprintf("%s-%d", workUnit.ExternalKey, time.Now().UnixNano())
	} else {
		taskID = storage.GenerateTaskID()
	}

	// Resolve naming (external key, branch pattern, commit prefix)
	namingInfo := c.resolveNaming(workUnit, taskID)

	// Create and switch to branch (or worktree) BEFORE creating work directory
	gitInfo, err := c.createBranchOrWorktree(ctx, taskID, namingInfo)
	if err != nil {
		return err
	}

	// Snapshot the source (read-only copy)
	snapshot := c.snapshotSource(ctx, p, reference, workUnit)

	// Merge local source files into snapshot if a matching queue task was found.
	// This ensures the agent sees both the provider content and local file content.
	if localQueueTask != nil && localQueueTask.SourcePath != "" {
		c.mergeLocalSourceIntoSnapshot(snapshot, localQueueTask.SourcePath)
	}

	// Register the task with workspace (writes source files)
	if err := c.registerTask(taskID, reference, workUnit, snapshot, gitInfo, namingInfo); err != nil {
		return err
	}

	c.publishProgress("Task registered", 100)

	return nil
}

// CheckActiveTaskConflict checks if starting a new task would conflict with an existing active task.
// Returns nil if no conflict exists (no active task, or using worktree mode).
// Returns TaskConflictInfo if a conflict exists and the caller should prompt the user.
// Called by CLI/Web handlers before Start() to handle the conflict UI appropriately.
func (c *Conductor) CheckActiveTaskConflict(_ context.Context) *TaskConflictInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// No conflict if no active task
	if c.activeTask == nil {
		return nil
	}

	// No conflict if using worktree mode (parallel tasks supported)
	if c.opts.UseWorktree {
		return nil
	}

	// Conflict exists - gather information for the caller
	info := &TaskConflictInfo{
		ActiveTaskID:  c.activeTask.ID,
		ActiveBranch:  c.activeTask.Branch,
		UsingWorktree: c.activeTask.WorktreePath != "",
	}

	// Get title from taskWork if available
	if c.taskWork != nil {
		info.ActiveTaskTitle = c.taskWork.Metadata.Title
	}

	return info
}

// ContinueWithExisting reuses an existing work directory for an updated task.
// This is used when a user wants to continue work on a previously finished task
// with new/updated content from the provider.
func (c *Conductor) ContinueWithExisting(ctx context.Context, reference string, existingTaskID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for existing active task
	if c.activeTask != nil {
		return fmt.Errorf("task already active: %s (use 'task status' to check)", c.activeTask.ID)
	}

	// Fetch updated work unit from provider
	p, workUnit, err := c.fetchWorkUnit(ctx, reference)
	if err != nil {
		return fmt.Errorf("fetch updated work unit: %w", err)
	}

	// Merge local metadata if a matching queue task exists
	var localQueueTask *storage.QueuedTask
	if workUnit.ExternalID != "" && c.workspace != nil {
		var queueErr error
		localQueueTask, queueErr = c.workspace.FindQueueTaskByExternalID(workUnit.ExternalID)
		if queueErr != nil {
			slog.Warn("search local queues for metadata enrichment", "external_id", workUnit.ExternalID, "error", queueErr)
		}
		if localQueueTask != nil {
			c.mergeLocalMetadata(workUnit, localQueueTask)
		}
	}

	// Capture task agent config from workUnit
	c.taskAgentConfig = workUnit.AgentConfig

	// Load existing work
	work, err := c.workspace.LoadWork(existingTaskID)
	if err != nil {
		return fmt.Errorf("load existing work: %w", err)
	}

	// Snapshot the updated source
	snapshot := c.snapshotSource(ctx, p, reference, workUnit)

	// Merge local source files into snapshot if available
	if localQueueTask != nil && localQueueTask.SourcePath != "" {
		c.mergeLocalSourceIntoSnapshot(snapshot, localQueueTask.SourcePath)
	}

	// Write updated source files to existing directory
	if err := c.writeSourceFiles(existingTaskID, snapshot); err != nil {
		return fmt.Errorf("write updated source files: %w", err)
	}

	// Update source info in work
	sourceInfo := c.buildSourceInfo(snapshot)
	work.Source = sourceInfo

	// Update metadata from work unit
	work.Metadata.Title = workUnit.Title
	if workUnit.ExternalKey != "" {
		work.Metadata.ExternalKey = workUnit.ExternalKey
	}

	// Resolve and update agent for this task
	agentInst, agentSource, err := c.resolveAgentForTask()
	if err != nil {
		return fmt.Errorf("resolve agent: %w", err)
	}
	c.activeAgent = agentInst
	work.Agent = storage.AgentInfo{
		Name:   agentInst.Name(),
		Source: agentSource,
	}
	if c.taskAgentConfig != nil && len(c.taskAgentConfig.Env) > 0 {
		work.Agent.InlineEnv = c.taskAgentConfig.Env
	}

	// Save updated work
	if err := c.workspace.SaveWork(work); err != nil {
		return fmt.Errorf("save updated work: %w", err)
	}

	// Create active task reference with state="idle"
	cfg, _ := c.workspace.LoadConfig()
	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}
	active := storage.NewActiveTask(existingTaskID, reference, c.workspace.EffectiveWorkDir(existingTaskID, cfg))

	// Preserve git info if it exists
	if c.git != nil {
		active.UseGit = true
		if work.Git.Branch != "" {
			active.Branch = work.Git.Branch
		}
		if work.Git.WorktreePath != "" {
			active.WorktreePath = work.Git.WorktreePath
		}
	}

	// Save active task
	if err := c.workspace.SaveActiveTask(active); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	c.activeTask = active
	c.taskWork = work

	// Set up state machine with idle state
	c.machine.SetWorkUnit(c.buildWorkUnit())

	c.publishProgress("Resumed with existing work directory", 100)

	return nil
}

// prepareWorkspace ensures workspace is ready for branch creation.
// If StashChanges is enabled, uncommitted changes are stashed.
// Otherwise, returns an error if workspace has uncommitted changes.
func (c *Conductor) prepareWorkspace(ctx context.Context) error {
	if c.git == nil || !c.opts.CreateBranch {
		return nil
	}

	hasChanges, err := c.git.HasChanges(ctx)
	if err != nil {
		return fmt.Errorf("check git status: %w", err)
	}

	if !hasChanges {
		return nil
	}

	if c.opts.StashChanges {
		message := "mehrhof: stash before task " + time.Now().Format("2006-01-02T15:04:05")
		if err := c.git.Stash(ctx, message); err != nil {
			return fmt.Errorf("stash changes: %w", err)
		}

		// Verify stash was created and display reference for manual recovery
		stashes, err := c.git.StashList(ctx)
		if err == nil && len(stashes) > 0 {
			// Display stash reference (e.g., "stash@{0}")
			stashRef := strings.Split(stashes[0], ":")[0]
			c.publishProgress(fmt.Sprintf("Stashed uncommitted changes (%s)", stashRef), 5)
		} else {
			c.publishProgress("Stashed uncommitted changes", 5)
		}

		return nil
	}

	return errors.New("workspace has uncommitted changes\nPlease commit or stash your changes, or use --no-branch to work on the current branch")
}

// mergeLocalMetadata merges metadata from a local queue task into a provider work unit.
// Local data fills gaps in provider data — it does not overwrite existing provider fields.
// This enables enriching external provider tasks (e.g., Wrike, GitHub) with local context
// such as code examples, file references, and custom frontmatter from local task files.
func (c *Conductor) mergeLocalMetadata(workUnit *provider.WorkUnit, local *storage.QueuedTask) {
	// Merge local description if provider description is shorter or empty
	if local.Description != "" && len(local.Description) > len(workUnit.Description) {
		slog.Debug("local description overrides provider",
			"local_len", len(local.Description), "provider_len", len(workUnit.Description))
		workUnit.Description = local.Description
	}

	// Merge local metadata (arbitrary frontmatter fields)
	if len(local.Metadata) > 0 {
		if workUnit.Metadata == nil {
			workUnit.Metadata = make(map[string]any)
		}
		for k, v := range local.Metadata {
			// Local fills gaps — doesn't overwrite provider data
			if _, exists := workUnit.Metadata[k]; !exists {
				slog.Debug("local metadata fills gap", "key", k)
				workUnit.Metadata[k] = v
			}
		}
	}

	// Store source path in metadata for the agent prompt
	if local.SourcePath != "" {
		if workUnit.Metadata == nil {
			workUnit.Metadata = make(map[string]any)
		}
		workUnit.Metadata["source_path"] = local.SourcePath
	}
}

// mergeLocalSourceIntoSnapshot reads local source files and appends them to the snapshot.
// This ensures the agent sees both provider content and local file content (code examples, etc.).
func (c *Conductor) mergeLocalSourceIntoSnapshot(snapshot *provider.Snapshot, sourcePath string) {
	if snapshot == nil || sourcePath == "" {
		return
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		// File doesn't exist — merge metadata only, skip snapshot
		return
	}

	if !info.IsDir() {
		// Single file: read and append
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			c.logError(fmt.Errorf("read local source %s (non-fatal): %w", sourcePath, err))

			return
		}
		snapshot.Files = append(snapshot.Files, provider.SnapshotFile{
			Path:    "local/" + filepath.Base(sourcePath),
			Content: string(content),
		})

		return
	}

	// Directory: walk and append all files (skip hidden dirs, limit per-file and total size)
	const maxMergeBytes int64 = 10 << 20 // 10MB total cap
	var accumulated int64
	_ = filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // WalkDir: skip inaccessible entries, continue walking
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}

			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return nil //nolint:nilerr // WalkDir: skip entries with stat errors
		}

		// Skip large files (>100KB)
		if fi.Size() > 100*1024 {
			slog.Debug("skipping large local source file", "path", path, "size", fi.Size())

			return nil
		}

		// Stop walking if total size limit exceeded
		if accumulated+fi.Size() > maxMergeBytes {
			slog.Warn("local source merge size limit reached, skipping remaining files",
				"accumulated", accumulated, "limit", maxMergeBytes)

			return filepath.SkipAll
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // WalkDir: skip unreadable files, continue walking
		}

		accumulated += int64(len(content))
		relPath, _ := filepath.Rel(sourcePath, path)
		snapshot.Files = append(snapshot.Files, provider.SnapshotFile{
			Path:    "local/" + relPath,
			Content: string(content),
		})

		return nil
	})
}

// fetchWorkUnit resolves the provider and fetches the work unit.
func (c *Conductor) fetchWorkUnit(ctx context.Context, reference string) (any, *provider.WorkUnit, error) {
	resolveOpts := provider.ResolveOptions{
		DefaultProvider: c.opts.DefaultProvider,
	}

	// Load workspace config and build provider config
	workspaceCfg, _ := c.workspace.LoadConfig() // ignore error, use defaults
	scheme := parseScheme(reference)
	providerCfg := buildProviderConfig(workspaceCfg, scheme)

	p, id, err := c.providers.Resolve(ctx, reference, providerCfg, resolveOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve provider: %w", err)
	}

	reader, ok := p.(provider.Reader)
	if !ok {
		return nil, nil, errors.New("provider does not support reading")
	}

	workUnit, err := reader.Fetch(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch work unit: %w", err)
	}

	return p, workUnit, nil
}

// snapshotSource creates a snapshot of the source content.
func (c *Conductor) snapshotSource(ctx context.Context, p any, reference string, workUnit *provider.WorkUnit) *provider.Snapshot {
	if snapshotter, ok := p.(provider.Snapshotter); ok {
		// Use ExternalID which contains the full reference (e.g., file path, issue URL).
		// workUnit.ID may be a generated short identifier that providers can't resolve.
		snapshot, err := snapshotter.Snapshot(ctx, workUnit.ExternalID)
		if err != nil {
			c.logError(fmt.Errorf("snapshot source (non-fatal): %w", err))
		} else if snapshot != nil {
			return snapshot
		}
	}

	// Fallback: return minimal snapshot with reference only
	return &provider.Snapshot{
		Type: workUnit.Provider,
		Ref:  reference,
	}
}

// buildSourceInfo creates storage.SourceInfo from provider snapshot (metadata only).
func (c *Conductor) buildSourceInfo(snapshot *provider.Snapshot) storage.SourceInfo {
	info := storage.SourceInfo{
		Type:   snapshot.Type,
		Ref:    snapshot.Ref,
		ReadAt: time.Now(),
	}

	// For single file content, store path reference
	if snapshot.Content != "" {
		// Generate filename from reference or use default
		filename := "source.md"
		if snapshot.Ref != "" {
			// Extract filename from reference if possible
			if idx := strings.LastIndex(snapshot.Ref, "/"); idx != -1 {
				filename = snapshot.Ref[idx+1:]
			} else if idx := strings.LastIndex(snapshot.Ref, ":"); idx != -1 {
				filename = snapshot.Ref[idx+1:] + ".md"
			}
		}
		info.Files = []string{"source/" + filename}
	}

	// For directory/multiple files, store file paths
	for _, f := range snapshot.Files {
		info.Files = append(info.Files, "source/"+f.Path)
	}

	return info
}

// writeSourceFiles writes snapshot content to the work directory's source/ subdirectory.
func (c *Conductor) writeSourceFiles(taskID string, snapshot *provider.Snapshot) error {
	if snapshot == nil {
		return nil
	}

	cfg, _ := c.workspace.LoadConfig()
	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}
	workPath := c.workspace.EffectiveWorkDir(taskID, cfg)
	sourceDir := filepath.Join(workPath, "source")

	// Create source directory if it doesn't exist
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return fmt.Errorf("create source directory: %w", err)
	}

	// Write single file content
	if snapshot.Content != "" {
		filename := "source.md"
		if snapshot.Ref != "" {
			// Extract filename from reference
			if idx := strings.LastIndex(snapshot.Ref, "/"); idx != -1 {
				filename = snapshot.Ref[idx+1:]
			} else if idx := strings.LastIndex(snapshot.Ref, ":"); idx != -1 {
				filename = snapshot.Ref[idx+1:] + ".md"
			}
		}
		destPath := filepath.Join(sourceDir, filename)
		if err := os.WriteFile(destPath, []byte(snapshot.Content), 0o644); err != nil {
			return fmt.Errorf("write source file: %w", err)
		}
	}

	// Write multiple files (directory provider)
	for _, f := range snapshot.Files {
		destPath := filepath.Join(sourceDir, f.Path)
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
		if err := os.WriteFile(destPath, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", f.Path, err)
		}
	}

	return nil
}

// registerTask creates the work directory and active task reference.
func (c *Conductor) registerTask(taskID, reference string, workUnit *provider.WorkUnit, snapshot *provider.Snapshot, gi *gitInfo, ni *namingInfo) error {
	// Resolve agent for this task (uses priority: CLI > task > workspace > auto)
	agentInst, agentSource, err := c.resolveAgentForTask()
	if err != nil {
		return fmt.Errorf("resolve agent: %w", err)
	}
	c.activeAgent = agentInst

	// Build SourceInfo with metadata (files will be written separately)
	sourceInfo := c.buildSourceInfo(snapshot)

	// Create work directory (creates source/ subdirectory)
	work, err := c.workspace.CreateWork(taskID, sourceInfo)
	if err != nil {
		return fmt.Errorf("create work: %w", err)
	}

	// Write source files to work directory
	if err := c.writeSourceFiles(taskID, snapshot); err != nil {
		return fmt.Errorf("write source files: %w", err)
	}

	// Set metadata from work unit and naming info
	work.Metadata.Title = workUnit.Title
	work.Metadata.ExternalKey = ni.externalKey
	work.Metadata.TaskType = ni.taskType
	work.Metadata.Slug = ni.slug

	// Store git info if branch was created
	if gi.branchName != "" {
		work.Git.Branch = gi.branchName
		work.Git.BaseBranch = gi.baseBranch
		work.Git.WorktreePath = gi.worktreePath
		work.Git.CreatedAt = time.Now()
		work.Git.CommitPrefix = gi.commitPrefix
		work.Git.BranchPattern = gi.branchPattern
	}

	// Store agent info for persistence (so subsequent commands use the same agent)
	work.Agent = storage.AgentInfo{
		Name:   agentInst.Name(),
		Source: agentSource,
	}
	// Store inline env vars if specified in task (for reference, not resolved values)
	if c.taskAgentConfig != nil && len(c.taskAgentConfig.Env) > 0 {
		work.Agent.InlineEnv = c.taskAgentConfig.Env
	}
	// Store task budget if provided by the source
	if workUnit.Budget != nil {
		work.Budget = &storage.BudgetConfig{
			MaxTokens: workUnit.Budget.MaxTokens,
			MaxCost:   workUnit.Budget.MaxCost,
			Currency:  workUnit.Budget.Currency,
			OnLimit:   workUnit.Budget.OnLimit,
			WarningAt: workUnit.Budget.WarningAt,
		}
	}

	if err := c.workspace.SaveWork(work); err != nil {
		return fmt.Errorf("save work: %w", err)
	}

	// Create active task reference
	cfg, _ := c.workspace.LoadConfig()
	if cfg == nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}
	active := storage.NewActiveTask(taskID, reference, c.workspace.EffectiveWorkDir(taskID, cfg))

	// Set git info on active task
	if c.git != nil {
		active.UseGit = true
		if gi.branchName != "" {
			active.Branch = gi.branchName
		}
		if gi.worktreePath != "" {
			active.WorktreePath = gi.worktreePath
		}
	}

	// Save active task
	if err := c.workspace.SaveActiveTask(active); err != nil {
		return fmt.Errorf("save active task: %w", err)
	}

	c.activeTask = active
	c.taskWork = work

	// Set up state machine
	c.machine.SetWorkUnit(c.buildWorkUnit())

	return nil
}

// Resume loads an existing active task.
func (c *Conductor) Resume(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.workspace.HasActiveTask() {
		return errors.New("no active task")
	}

	active, err := c.workspace.LoadActiveTask()
	if err != nil {
		return fmt.Errorf("load active task: %w", err)
	}

	work, err := c.workspace.LoadWork(active.ID)
	if err != nil {
		return fmt.Errorf("load work: %w", err)
	}

	c.activeTask = active
	c.taskWork = work
	c.machine.SetWorkUnit(c.buildWorkUnit())

	return nil
}

// Delete abandons the current task without merging.
func (c *Conductor) Delete(ctx context.Context, opts DeleteOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeTask == nil {
		return errors.New("no active task")
	}

	taskID := c.activeTask.ID

	// Handle git operations if applicable
	if c.git != nil && c.activeTask.UseGit && c.activeTask.Branch != "" && !opts.KeepBranch {
		currentBranch, _ := c.git.CurrentBranch(ctx)
		taskBranch := c.activeTask.Branch
		worktreePath := c.activeTask.WorktreePath

		// NOTE: Cleanup errors below are logged but not returned intentionally.
		// Delete operation should succeed even if cleanup partially fails.
		// This is best-effort cleanup that should not block task deletion.

		// If using worktree, remove it first
		if worktreePath != "" {
			if err := c.git.RemoveWorktree(ctx, worktreePath, true); err != nil {
				c.logError(fmt.Errorf("remove worktree: %w", err))
			}
		} else if currentBranch == taskBranch {
			// If we're on the task branch (not worktree), switch to base branch first
			var baseBranch string
			if c.taskWork != nil && c.taskWork.Git.BaseBranch != "" {
				baseBranch = c.taskWork.Git.BaseBranch
			} else {
				var err error
				baseBranch, err = c.git.GetBaseBranch(ctx)
				if err != nil {
					return fmt.Errorf("get base branch: %w", err)
				}
			}

			if err := c.git.Checkout(ctx, baseBranch); err != nil {
				return fmt.Errorf("checkout base branch: %w", err)
			}
		}

		// Checkpoint deletion is best-effort; ignore errors
		_ = c.git.DeleteAllCheckpoints(ctx, taskID)

		// Delete the branch
		if err := c.git.DeleteBranch(ctx, taskBranch, true); err != nil {
			c.logError(fmt.Errorf("delete branch: %w", err))
		}
	}

	// Delete work directory based on: CLI flag > config > default (delete)
	var shouldDelete bool
	if opts.DeleteWork != nil {
		shouldDelete = *opts.DeleteWork // CLI explicitly set
	} else {
		cfg, _ := c.workspace.LoadConfig()              // ignore error, use defaults
		shouldDelete = cfg.Workflow.DeleteWorkOnAbandon // default: true
	}
	if shouldDelete {
		if err := c.workspace.DeleteWork(taskID); err != nil {
			c.logError(fmt.Errorf("delete work directory: %w", err))
		}
	}

	// Clear active task
	if err := c.workspace.ClearActiveTask(); err != nil {
		c.logError(fmt.Errorf("clear active task: %w", err))
	}

	c.activeTask = nil
	c.taskWork = nil

	c.publishProgress("Task deleted", 100)

	return nil
}

// Status returns the current task status.
func (c *Conductor) Status(ctx context.Context) (*TaskStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.activeTask == nil {
		return nil, errors.New("no active task")
	}

	// Count specifications - errors ignored; empty list is acceptable for status display
	specifications, _ := c.workspace.ListSpecifications(c.activeTask.ID)

	status := &TaskStatus{
		TaskID:         c.activeTask.ID,
		State:          c.activeTask.State,
		Ref:            c.activeTask.Ref,
		Branch:         c.activeTask.Branch,
		WorktreePath:   c.activeTask.WorktreePath,
		Specifications: len(specifications),
		Checkpoints:    c.countCheckpoints(ctx),
		Started:        c.activeTask.Started,
	}

	// Add work metadata if available (may be nil if work directory is missing)
	if c.taskWork != nil {
		status.Title = c.taskWork.Metadata.Title
		status.ExternalKey = c.taskWork.Metadata.ExternalKey
	}

	// Add agent info
	if c.taskWork != nil && c.taskWork.Agent.Name != "" {
		status.Agent = c.taskWork.Agent.Name
		status.AgentSource = c.taskWork.Agent.Source
	} else if c.activeAgent != nil {
		status.Agent = c.activeAgent.Name()
		status.AgentSource = "auto"
	}

	return status, nil
}

// ListExistingWorkDirs returns all task IDs with existing work directories.
func (c *Conductor) ListExistingWorkDirs() ([]string, error) {
	return c.workspace.ListWorks()
}

// ArchiveWorkDir archives a specific work directory.
func (c *Conductor) ArchiveWorkDir(taskID string) error {
	return c.workspace.ArchiveWorkDir(taskID)
}

// TaskStatus represents the current task state.
type TaskStatus struct {
	TaskID         string
	Title          string
	ExternalKey    string // User-facing key (e.g., "FEATURE-123")
	State          string
	Ref            string
	Branch         string
	WorktreePath   string
	Specifications int
	Checkpoints    int
	Started        time.Time
	Agent          string // Agent name being used
	AgentSource    string // Where agent was configured from: "cli", "task", "workspace", "auto"
}
