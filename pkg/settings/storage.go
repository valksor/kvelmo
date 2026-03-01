package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/valksor/kvelmo/pkg/meta"
	"gopkg.in/yaml.v3"
)

// GlobalPath returns the path to the global settings file.
func GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	return filepath.Join(home, meta.GlobalDir, meta.ConfigFile), nil
}

// GlobalDirPath returns the path to the global settings directory.
func GlobalDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	return filepath.Join(home, meta.GlobalDir), nil
}

// ProjectPath returns the path to the project settings file.
// projectRoot should be the root directory of the project.
func ProjectPath(projectRoot string) string {
	return filepath.Join(projectRoot, meta.OrgDir, meta.ConfigFile)
}

// ProjectDirPath returns the path to the project settings directory.
func ProjectDirPath(projectRoot string) string {
	return filepath.Join(projectRoot, meta.OrgDir)
}

// Load loads settings from the specified path.
// Returns nil if the file doesn't exist (not an error).
func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil // Documented behavior: nil means file not found
		}

		return nil, fmt.Errorf("read settings: %w", err)
	}

	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	return &s, nil
}

// Save saves settings to the specified path.
// Creates parent directories if they don't exist.
func Save(path string, s *Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	return nil
}

// LoadGlobal loads settings from the global path.
func LoadGlobal() (*Settings, error) {
	path, err := GlobalPath()
	if err != nil {
		return nil, err
	}

	return Load(path)
}

// LoadProject loads settings from the project path.
func LoadProject(projectRoot string) (*Settings, error) {
	return Load(ProjectPath(projectRoot))
}

// SaveGlobal saves settings to the global path.
func SaveGlobal(s *Settings) error {
	path, err := GlobalPath()
	if err != nil {
		return err
	}

	return Save(path, s)
}

// SaveProject saves settings to the project path.
func SaveProject(projectRoot string, s *Settings) error {
	return Save(ProjectPath(projectRoot), s)
}

// LoadEffective loads and merges global and project settings.
// Project settings override global settings.
// Also loads and injects environment variables from .env files.
func LoadEffective(projectRoot string) (*Settings, *Settings, *Settings, error) {
	// Load .env files into an in-memory map (project overrides global)
	envMap, err := LoadEnvMap(projectRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load env: %w", err)
	}

	// Load global settings
	global, err := LoadGlobal()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load global: %w", err)
	}

	// Load project settings
	project, err := LoadProject(projectRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load project: %w", err)
	}

	// Start with defaults
	effective := DefaultSettings()

	// Merge global (if exists)
	if global != nil {
		Merge(effective, global)
	}

	// Merge project (if exists, takes precedence)
	if project != nil {
		Merge(effective, project)
	}

	// Inject environment variables from .env files into sensitive fields
	InjectEnvVars(effective, envMap)

	return effective, global, project, nil
}

// Merge merges src into dst. Non-zero values in src override dst.
// This is a shallow merge for top-level fields, but preserves
// existing values in dst that are not set in src.
func Merge(dst, src *Settings) {
	if src == nil {
		return
	}

	// Agent settings
	if src.Agent.Default != "" {
		dst.Agent.Default = src.Agent.Default
	}
	if len(src.Agent.Allowed) > 0 {
		dst.Agent.Allowed = src.Agent.Allowed
	}

	// Provider settings
	if src.Providers.Default != "" {
		dst.Providers.Default = src.Providers.Default
	}
	mergeGitHubConfig(&dst.Providers.GitHub, &src.Providers.GitHub)
	mergeGitLabConfig(&dst.Providers.GitLab, &src.Providers.GitLab)
	mergeWrikeConfig(&dst.Providers.Wrike, &src.Providers.Wrike)
	mergeLinearConfig(&dst.Providers.Linear, &src.Providers.Linear)

	// Git settings
	if src.Git.BaseBranch != "" {
		dst.Git.BaseBranch = src.Git.BaseBranch
	}
	if src.Git.BranchPattern != "" {
		dst.Git.BranchPattern = src.Git.BranchPattern
	}
	if src.Git.CommitPrefix != "" {
		dst.Git.CommitPrefix = src.Git.CommitPrefix
	}
	// Pointer bools: non-nil means explicitly set (allows false to override true)
	if src.Git.CreateBranch != nil {
		dst.Git.CreateBranch = src.Git.CreateBranch
	}
	if src.Git.AutoCommit != nil {
		dst.Git.AutoCommit = src.Git.AutoCommit
	}
	if src.Git.SignCommits != nil {
		dst.Git.SignCommits = src.Git.SignCommits
	}
	if src.Git.AllowPRComment != nil {
		dst.Git.AllowPRComment = src.Git.AllowPRComment
	}

	// Workers settings
	if src.Workers.Max > 0 {
		dst.Workers.Max = src.Workers.Max
	}

	// Storage settings
	if src.Storage.SaveInProject != nil {
		dst.Storage.SaveInProject = src.Storage.SaveInProject
	}

	// Workflow settings
	if src.Workflow.UseWorktreeIsolation != nil {
		dst.Workflow.UseWorktreeIsolation = src.Workflow.UseWorktreeIsolation
	}

	// Custom agents - merge by key
	if len(src.CustomAgents) > 0 {
		if dst.CustomAgents == nil {
			dst.CustomAgents = make(map[string]CustomAgent)
		}
		for k, v := range src.CustomAgents {
			dst.CustomAgents[k] = v
		}
	}
}

func mergeGitHubConfig(dst, src *GitHubConfig) {
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.Owner != "" {
		dst.Owner = src.Owner
	}
	if src.AllowTicketComment {
		dst.AllowTicketComment = true
	}
}

func mergeGitLabConfig(dst, src *GitLabConfig) {
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.AllowTicketComment {
		dst.AllowTicketComment = true
	}
}

func mergeWrikeConfig(dst, src *WrikeConfig) {
	if src.Token != "" {
		dst.Token = src.Token
	}
	// Boolean fields: only override dst when src explicitly sets them to false
	// (the zero value). We use a yaml-aware approach — if src was loaded from
	// YAML and a bool field is present, it overrides. Since we can't distinguish
	// "not set" from false for plain booleans, we follow the pattern used by the
	// rest of the Merge function: only set when src is true (opt-in fields that
	// are on by default stay on unless explicitly turned off at the project level).
	//
	// For fields that default to true, we propagate a false override only if the
	// src Settings actually came from a file (non-nil). The caller is responsible
	// for passing a non-nil src only when the file was loaded.
	if src.IncludeParentContext {
		dst.IncludeParentContext = true
	}
	if src.IncludeSiblingContext {
		dst.IncludeSiblingContext = true
	}
	if src.AllowTicketComment {
		dst.AllowTicketComment = true
	}
}

func mergeLinearConfig(dst, src *LinearConfig) {
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.Team != "" {
		dst.Team = src.Team
	}
	if src.IncludeParentContext {
		dst.IncludeParentContext = true
	}
	if src.IncludeSiblingContext {
		dst.IncludeSiblingContext = true
	}
	if src.AllowTicketComment {
		dst.AllowTicketComment = true
	}
}

// SetValue sets a value at a dot-notation path in the settings.
// Returns an error if the path is invalid.
func SetValue(s *Settings, path string, value any) error {
	switch path {
	// Agent
	case "agent.default":
		if v, ok := value.(string); ok {
			s.Agent.Default = v

			return nil
		}

		return errors.New("agent.default must be a string")
	case "agent.allowed":
		if v, ok := value.([]string); ok {
			s.Agent.Allowed = v

			return nil
		}
		// Handle []any from JSON unmarshaling
		if arr, ok := value.([]any); ok {
			strs := make([]string, len(arr))
			for i, item := range arr {
				if str, ok := item.(string); ok {
					strs[i] = str
				} else {
					return fmt.Errorf("agent.allowed[%d] must be a string", i)
				}
			}
			s.Agent.Allowed = strs

			return nil
		}

		return errors.New("agent.allowed must be a string array")

	// Providers
	case "providers.default":
		if v, ok := value.(string); ok {
			s.Providers.Default = v

			return nil
		}

		return errors.New("providers.default must be a string")
	case "providers.github.token":
		if v, ok := value.(string); ok {
			s.Providers.GitHub.Token = v

			return nil
		}

		return errors.New("providers.github.token must be a string")
	case "providers.github.owner":
		if v, ok := value.(string); ok {
			s.Providers.GitHub.Owner = v

			return nil
		}

		return errors.New("providers.github.owner must be a string")
	case "providers.github.allow_ticket_comment":
		if v, ok := value.(bool); ok {
			s.Providers.GitHub.AllowTicketComment = v

			return nil
		}

		return errors.New("providers.github.allow_ticket_comment must be a boolean")
	case "providers.gitlab.token":
		if v, ok := value.(string); ok {
			s.Providers.GitLab.Token = v

			return nil
		}

		return errors.New("providers.gitlab.token must be a string")
	case "providers.gitlab.base_url":
		if v, ok := value.(string); ok {
			s.Providers.GitLab.BaseURL = v

			return nil
		}

		return errors.New("providers.gitlab.base_url must be a string")
	case "providers.wrike.token":
		if v, ok := value.(string); ok {
			s.Providers.Wrike.Token = v

			return nil
		}

		return errors.New("providers.wrike.token must be a string")
	case "providers.wrike.include_parent_context":
		if v, ok := value.(bool); ok {
			s.Providers.Wrike.IncludeParentContext = v

			return nil
		}

		return errors.New("providers.wrike.include_parent_context must be a boolean")
	case "providers.wrike.include_sibling_context":
		if v, ok := value.(bool); ok {
			s.Providers.Wrike.IncludeSiblingContext = v

			return nil
		}

		return errors.New("providers.wrike.include_sibling_context must be a boolean")
	case "providers.gitlab.allow_ticket_comment":
		if v, ok := value.(bool); ok {
			s.Providers.GitLab.AllowTicketComment = v

			return nil
		}

		return errors.New("providers.gitlab.allow_ticket_comment must be a boolean")
	case "providers.wrike.allow_ticket_comment":
		if v, ok := value.(bool); ok {
			s.Providers.Wrike.AllowTicketComment = v

			return nil
		}

		return errors.New("providers.wrike.allow_ticket_comment must be a boolean")
	case "providers.linear.token":
		if v, ok := value.(string); ok {
			s.Providers.Linear.Token = v

			return nil
		}

		return errors.New("providers.linear.token must be a string")
	case "providers.linear.team":
		if v, ok := value.(string); ok {
			s.Providers.Linear.Team = v

			return nil
		}

		return errors.New("providers.linear.team must be a string")
	case "providers.linear.include_parent_context":
		if v, ok := value.(bool); ok {
			s.Providers.Linear.IncludeParentContext = v

			return nil
		}

		return errors.New("providers.linear.include_parent_context must be a boolean")
	case "providers.linear.include_sibling_context":
		if v, ok := value.(bool); ok {
			s.Providers.Linear.IncludeSiblingContext = v

			return nil
		}

		return errors.New("providers.linear.include_sibling_context must be a boolean")
	case "providers.linear.allow_ticket_comment":
		if v, ok := value.(bool); ok {
			s.Providers.Linear.AllowTicketComment = v

			return nil
		}

		return errors.New("providers.linear.allow_ticket_comment must be a boolean")

	// Git
	case "git.base_branch":
		if v, ok := value.(string); ok {
			s.Git.BaseBranch = v

			return nil
		}

		return errors.New("git.base_branch must be a string")
	case "git.branch_pattern":
		if v, ok := value.(string); ok {
			s.Git.BranchPattern = v

			return nil
		}

		return errors.New("git.branch_pattern must be a string")
	case "git.commit_prefix":
		if v, ok := value.(string); ok {
			s.Git.CommitPrefix = v

			return nil
		}

		return errors.New("git.commit_prefix must be a string")
	case "git.create_branch":
		if v, ok := value.(bool); ok {
			s.Git.CreateBranch = &v

			return nil
		}

		return errors.New("git.create_branch must be a boolean")
	case "git.auto_commit":
		if v, ok := value.(bool); ok {
			s.Git.AutoCommit = &v

			return nil
		}

		return errors.New("git.auto_commit must be a boolean")
	case "git.sign_commits":
		if v, ok := value.(bool); ok {
			s.Git.SignCommits = &v

			return nil
		}

		return errors.New("git.sign_commits must be a boolean")
	case "git.allow_pr_comment":
		if v, ok := value.(bool); ok {
			s.Git.AllowPRComment = &v

			return nil
		}

		return errors.New("git.allow_pr_comment must be a boolean")

	// Workers
	case "workers.max":
		switch v := value.(type) {
		case int:
			s.Workers.Max = v

			return nil
		case float64:
			s.Workers.Max = int(v)

			return nil
		}

		return errors.New("workers.max must be a number")

	// Storage
	case "storage.save_in_project":
		if v, ok := value.(bool); ok {
			s.Storage.SaveInProject = &v

			return nil
		}

		return errors.New("storage.save_in_project must be a boolean")

	// Workflow
	case "workflow.use_worktree_isolation":
		if v, ok := value.(bool); ok {
			s.Workflow.UseWorktreeIsolation = &v

			return nil
		}

		return errors.New("workflow.use_worktree_isolation must be a boolean")

	// Custom Agents
	case "custom_agents":
		// Expect a map of agent name -> agent config
		if v, ok := value.(map[string]any); ok {
			agents := make(map[string]CustomAgent)
			for name, agentData := range v {
				agentMap, ok := agentData.(map[string]any)
				if !ok {
					return fmt.Errorf("custom_agents.%s must be an object", name)
				}
				agent := CustomAgent{}
				if extends, ok := agentMap["extends"].(string); ok {
					agent.Extends = extends
				}
				if desc, ok := agentMap["description"].(string); ok {
					agent.Description = desc
				}
				if argsAny, ok := agentMap["args"].([]any); ok {
					args := make([]string, 0, len(argsAny))
					for _, a := range argsAny {
						if s, ok := a.(string); ok {
							args = append(args, s)
						}
					}
					agent.Args = args
				}
				if envMap, ok := agentMap["env"].(map[string]any); ok {
					env := make(map[string]string)
					for k, v := range envMap {
						if s, ok := v.(string); ok {
							env[k] = s
						}
					}
					agent.Env = env
				}
				agents[name] = agent
			}
			s.CustomAgents = agents

			return nil
		}

		return errors.New("custom_agents must be an object")

	default:
		return fmt.Errorf("unknown path: %s", path)
	}
}

// GetValue gets a value at a dot-notation path from the settings.
func GetValue(s *Settings, path string) (any, error) {
	switch path {
	// Agent
	case "agent.default":
		return s.Agent.Default, nil
	case "agent.allowed":
		return s.Agent.Allowed, nil

	// Providers
	case "providers.default":
		return s.Providers.Default, nil
	case "providers.github.token":
		return s.Providers.GitHub.Token, nil
	case "providers.github.owner":
		return s.Providers.GitHub.Owner, nil
	case "providers.github.allow_ticket_comment":
		return s.Providers.GitHub.AllowTicketComment, nil
	case "providers.gitlab.token":
		return s.Providers.GitLab.Token, nil
	case "providers.gitlab.base_url":
		return s.Providers.GitLab.BaseURL, nil
	case "providers.wrike.token":
		return s.Providers.Wrike.Token, nil
	case "providers.wrike.include_parent_context":
		return s.Providers.Wrike.IncludeParentContext, nil
	case "providers.wrike.include_sibling_context":
		return s.Providers.Wrike.IncludeSiblingContext, nil
	case "providers.gitlab.allow_ticket_comment":
		return s.Providers.GitLab.AllowTicketComment, nil
	case "providers.wrike.allow_ticket_comment":
		return s.Providers.Wrike.AllowTicketComment, nil
	case "providers.linear.token":
		return s.Providers.Linear.Token, nil
	case "providers.linear.team":
		return s.Providers.Linear.Team, nil
	case "providers.linear.include_parent_context":
		return s.Providers.Linear.IncludeParentContext, nil
	case "providers.linear.include_sibling_context":
		return s.Providers.Linear.IncludeSiblingContext, nil
	case "providers.linear.allow_ticket_comment":
		return s.Providers.Linear.AllowTicketComment, nil

	// Git
	case "git.base_branch":
		return s.Git.BaseBranch, nil
	case "git.branch_pattern":
		return s.Git.BranchPattern, nil
	case "git.commit_prefix":
		return s.Git.CommitPrefix, nil
	case "git.create_branch":
		return BoolValue(s.Git.CreateBranch, true), nil
	case "git.auto_commit":
		return BoolValue(s.Git.AutoCommit, true), nil
	case "git.sign_commits":
		return BoolValue(s.Git.SignCommits, false), nil
	case "git.allow_pr_comment":
		return BoolValue(s.Git.AllowPRComment, false), nil

	// Workers
	case "workers.max":
		return s.Workers.Max, nil

	// Storage
	case "storage.save_in_project":
		return BoolValue(s.Storage.SaveInProject, false), nil

	// Workflow
	case "workflow.use_worktree_isolation":
		return BoolValue(s.Workflow.UseWorktreeIsolation, true), nil

	// Custom Agents
	case "custom_agents":
		return s.CustomAgents, nil

	default:
		return nil, fmt.Errorf("unknown path: %s", path)
	}
}

// SensitivePaths maps setting paths to their corresponding environment variable names.
// These paths should be stored in .env files rather than settings.json.
//

var SensitivePaths = map[string]string{
	"providers.github.token": "GITHUB_TOKEN",
	"providers.gitlab.token": "GITLAB_TOKEN",
	"providers.wrike.token":  "WRIKE_TOKEN",
	"providers.linear.token": "LINEAR_TOKEN",
}

// IsSensitivePath returns true if the path should be stored in .env.
func IsSensitivePath(path string) bool {
	_, ok := SensitivePaths[path]

	return ok
}

// GetEnvVarForPath returns the environment variable name for a sensitive path.
func GetEnvVarForPath(path string) string {
	return SensitivePaths[path]
}
