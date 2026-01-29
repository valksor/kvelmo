package conductor

import (
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// parseScheme extracts the provider scheme from a reference (e.g., "github:123" -> "github").
func parseScheme(reference string) string {
	parts := strings.SplitN(reference, ":", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	// No scheme - will use default provider
	return ""
}

// buildProviderConfig creates a provider.Config from workspace configuration.
// It extracts provider-specific settings and makes them available via provider.Config methods.
func buildProviderConfig(workspaceCfg *storage.WorkspaceConfig, providerName string) provider.Config {
	if workspaceCfg == nil || providerName == "" {
		return provider.NewConfig()
	}

	cfg := provider.NewConfig()

	switch strings.ToLower(providerName) {
	case "github", "gh":
		if workspaceCfg.GitHub != nil {
			cfg.Set("token", workspaceCfg.GitHub.Token)
			cfg.Set("owner", workspaceCfg.GitHub.Owner)
			cfg.Set("repo", workspaceCfg.GitHub.Repo)
			cfg.Set("branch_pattern", workspaceCfg.GitHub.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.GitHub.CommitPrefix)
			cfg.Set("target_branch", workspaceCfg.GitHub.TargetBranch)
			cfg.Set("draft_pr", workspaceCfg.GitHub.DraftPR)
			if workspaceCfg.GitHub.Comments != nil {
				cfg.Set("comments.enabled", workspaceCfg.GitHub.Comments.Enabled)
				cfg.Set("comments.on_branch_created", workspaceCfg.GitHub.Comments.OnBranchCreated)
				cfg.Set("comments.on_plan_done", workspaceCfg.GitHub.Comments.OnPlanDone)
				cfg.Set("comments.on_implement_done", workspaceCfg.GitHub.Comments.OnImplementDone)
				cfg.Set("comments.on_pr_created", workspaceCfg.GitHub.Comments.OnPRCreated)
			}
		}

	case "gitlab", "gl":
		if workspaceCfg.GitLab != nil {
			cfg.Set("token", workspaceCfg.GitLab.Token)
			cfg.Set("host", workspaceCfg.GitLab.Host)
			cfg.Set("project_path", workspaceCfg.GitLab.ProjectPath)
			cfg.Set("branch_pattern", workspaceCfg.GitLab.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.GitLab.CommitPrefix)
		}

	case "wrike":
		if workspaceCfg.Wrike != nil {
			cfg.Set("token", workspaceCfg.Wrike.Token)
			cfg.Set("host", workspaceCfg.Wrike.Host)
			cfg.Set("space_id", workspaceCfg.Wrike.Space)
			cfg.Set("folder_id", workspaceCfg.Wrike.Folder)
			cfg.Set("project_id", workspaceCfg.Wrike.Project)
		}

	case "notion", "nt":
		if workspaceCfg.Notion != nil {
			cfg.Set("token", workspaceCfg.Notion.Token)
			cfg.Set("database_id", workspaceCfg.Notion.DatabaseID)
			cfg.Set("status_property", workspaceCfg.Notion.StatusProperty)
			cfg.Set("description_property", workspaceCfg.Notion.DescriptionProperty)
			cfg.Set("labels_property", workspaceCfg.Notion.LabelsProperty)
		}

	case "jira", "j":
		if workspaceCfg.Jira != nil {
			cfg.Set("token", workspaceCfg.Jira.Token)
			cfg.Set("email", workspaceCfg.Jira.Email)
			cfg.Set("base_url", workspaceCfg.Jira.BaseURL)
			cfg.Set("project", workspaceCfg.Jira.Project)
		}

	case "linear", "ln":
		if workspaceCfg.Linear != nil {
			cfg.Set("token", workspaceCfg.Linear.Token)
			cfg.Set("team", workspaceCfg.Linear.Team)
		}

	case "youtrack", "yt":
		if workspaceCfg.YouTrack != nil {
			cfg.Set("token", workspaceCfg.YouTrack.Token)
			cfg.Set("host", workspaceCfg.YouTrack.Host)
		}

	case "bitbucket", "bb":
		if workspaceCfg.Bitbucket != nil {
			cfg.Set("username", workspaceCfg.Bitbucket.Username)
			cfg.Set("app_password", workspaceCfg.Bitbucket.AppPassword)
			cfg.Set("workspace", workspaceCfg.Bitbucket.Workspace)
			cfg.Set("repo", workspaceCfg.Bitbucket.RepoSlug)
			cfg.Set("branch_pattern", workspaceCfg.Bitbucket.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.Bitbucket.CommitPrefix)
			cfg.Set("target_branch", workspaceCfg.Bitbucket.TargetBranch)
			cfg.Set("close_source_branch", workspaceCfg.Bitbucket.CloseSourceBranch)
		}

	case "asana", "as":
		if workspaceCfg.Asana != nil {
			cfg.Set("token", workspaceCfg.Asana.Token)
			cfg.Set("workspace_gid", workspaceCfg.Asana.WorkspaceGID)
			cfg.Set("default_project", workspaceCfg.Asana.DefaultProject)
			cfg.Set("branch_pattern", workspaceCfg.Asana.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.Asana.CommitPrefix)
		}

	case "clickup", "cu":
		if workspaceCfg.ClickUp != nil {
			cfg.Set("token", workspaceCfg.ClickUp.Token)
			cfg.Set("team_id", workspaceCfg.ClickUp.TeamID)
			cfg.Set("default_list", workspaceCfg.ClickUp.DefaultList)
			cfg.Set("branch_pattern", workspaceCfg.ClickUp.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.ClickUp.CommitPrefix)
		}

	case "azuredevops", "azdo", "azure":
		if workspaceCfg.AzureDevOps != nil {
			cfg.Set("token", workspaceCfg.AzureDevOps.Token)
			cfg.Set("organization", workspaceCfg.AzureDevOps.Organization)
			cfg.Set("project", workspaceCfg.AzureDevOps.Project)
			cfg.Set("area_path", workspaceCfg.AzureDevOps.AreaPath)
			cfg.Set("iteration_path", workspaceCfg.AzureDevOps.IterationPath)
			cfg.Set("repo_name", workspaceCfg.AzureDevOps.RepoName)
			cfg.Set("target_branch", workspaceCfg.AzureDevOps.TargetBranch)
			cfg.Set("branch_pattern", workspaceCfg.AzureDevOps.BranchPattern)
			cfg.Set("commit_prefix", workspaceCfg.AzureDevOps.CommitPrefix)
		}

	case "trello", "tr":
		if workspaceCfg.Trello != nil {
			cfg.Set("api_key", workspaceCfg.Trello.APIKey)
			cfg.Set("token", workspaceCfg.Trello.Token)
			cfg.Set("board", workspaceCfg.Trello.Board)
		}
	}

	return cfg
}
