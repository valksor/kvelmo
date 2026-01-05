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
			cfg.Set("folder_id", workspaceCfg.Wrike.Folder)
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

		// Note: asana, clickup, azuredevops, bitbucket, trello are not currently
		// represented in WorkspaceConfig but could be added in the future
	}

	return cfg
}
