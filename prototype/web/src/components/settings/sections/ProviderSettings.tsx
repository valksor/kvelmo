import {
  TextInput,
  Checkbox,
  Select,
  CollapseSection,
} from '@/components/settings/FormField'
import type { SettingsSectionProps } from './types'

export function ProviderSettings({ data, updateField }: SettingsSectionProps) {
  return (
    <div className="space-y-4">
      {/* Default Provider */}
      <CollapseSection title="Default Provider" defaultOpen>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Select
            label="Default Provider"
            hint="For bare task references without scheme"
            value={data.providers?.default}
            onChange={(v) => updateField(['providers', 'default'], v)}
            options={[
              { value: 'file', label: 'File' },
              { value: 'directory', label: 'Directory' },
              { value: 'github', label: 'GitHub' },
              { value: 'gitlab', label: 'GitLab' },
              { value: 'jira', label: 'Jira' },
              { value: 'linear', label: 'Linear' },
            ]}
          />
          <TextInput
            label="Default Mention"
            hint="Mention text when submitting tasks"
            value={data.providers?.default_mention}
            onChange={(v) => updateField(['providers', 'default_mention'], v)}
          />
        </div>
      </CollapseSection>

      {/* GitHub */}
      <CollapseSection title="GitHub">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="GitHub API token (or set GITHUB_TOKEN env var)"
            value={data.github?.token}
            onChange={(v) => updateField(['github', 'token'], v)}
          />
          <TextInput
            label="Owner"
            hint="Repository owner (auto-detected from remote)"
            value={data.github?.owner}
            onChange={(v) => updateField(['github', 'owner'], v)}
          />
          <TextInput
            label="Repository"
            value={data.github?.repo}
            onChange={(v) => updateField(['github', 'repo'], v)}
          />
          <TextInput
            label="Target Branch"
            hint="Default branch for PRs"
            value={data.github?.target_branch}
            onChange={(v) => updateField(['github', 'target_branch'], v)}
          />
        </div>
        <Checkbox
          label="Draft PRs"
          hint="Create PRs as draft by default"
          checked={data.github?.draft_pr}
          onChange={(v) => updateField(['github', 'draft_pr'], v)}
        />
      </CollapseSection>

      {/* GitLab */}
      <CollapseSection title="GitLab">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="GitLab API token (or set GITLAB_TOKEN env var)"
            value={data.gitlab?.token}
            onChange={(v) => updateField(['gitlab', 'token'], v)}
          />
          <TextInput
            label="Host"
            hint="GitLab host (default: gitlab.com)"
            value={data.gitlab?.host}
            onChange={(v) => updateField(['gitlab', 'host'], v)}
          />
          <TextInput
            label="Project Path"
            hint="e.g., group/project"
            value={data.gitlab?.project_path}
            onChange={(v) => updateField(['gitlab', 'project_path'], v)}
          />
        </div>
      </CollapseSection>

      {/* Jira */}
      <CollapseSection title="Jira">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="Jira API token"
            value={data.jira?.token}
            onChange={(v) => updateField(['jira', 'token'], v)}
          />
          <TextInput
            label="Email"
            hint="Email for Cloud auth"
            value={data.jira?.email}
            onChange={(v) => updateField(['jira', 'email'], v)}
          />
          <TextInput
            label="Base URL"
            hint="Jira instance URL"
            value={data.jira?.base_url}
            onChange={(v) => updateField(['jira', 'base_url'], v)}
          />
          <TextInput
            label="Project"
            hint="Default project key"
            value={data.jira?.project}
            onChange={(v) => updateField(['jira', 'project'], v)}
          />
        </div>
      </CollapseSection>

      {/* Linear */}
      <CollapseSection title="Linear">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="Linear API key"
            value={data.linear?.token}
            onChange={(v) => updateField(['linear', 'token'], v)}
          />
          <TextInput
            label="Team"
            hint="Default team key"
            value={data.linear?.team}
            onChange={(v) => updateField(['linear', 'team'], v)}
          />
        </div>
      </CollapseSection>

      {/* Notion */}
      <CollapseSection title="Notion">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="Notion integration token"
            value={data.notion?.token}
            onChange={(v) => updateField(['notion', 'token'], v)}
          />
          <TextInput
            label="Database ID"
            value={data.notion?.database_id}
            onChange={(v) => updateField(['notion', 'database_id'], v)}
          />
          <TextInput
            label="Status Property"
            hint="Property name for status (default: Status)"
            value={data.notion?.status_property}
            onChange={(v) => updateField(['notion', 'status_property'], v)}
          />
        </div>
      </CollapseSection>

      {/* Bitbucket */}
      <CollapseSection title="Bitbucket">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Username"
            value={data.bitbucket?.username}
            onChange={(v) => updateField(['bitbucket', 'username'], v)}
          />
          <TextInput
            label="App Password"
            type="password"
            value={data.bitbucket?.app_password}
            onChange={(v) => updateField(['bitbucket', 'app_password'], v)}
          />
          <TextInput
            label="Workspace"
            value={data.bitbucket?.workspace}
            onChange={(v) => updateField(['bitbucket', 'workspace'], v)}
          />
          <TextInput
            label="Repository"
            value={data.bitbucket?.repo}
            onChange={(v) => updateField(['bitbucket', 'repo'], v)}
          />
        </div>
        <Checkbox
          label="Close Source Branch"
          hint="Delete source branch when PR is merged"
          checked={data.bitbucket?.close_source_branch}
          onChange={(v) => updateField(['bitbucket', 'close_source_branch'], v)}
        />
      </CollapseSection>

      {/* Azure DevOps */}
      <CollapseSection title="Azure DevOps">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Token"
            type="password"
            hint="Personal Access Token"
            value={data.azure_devops?.token}
            onChange={(v) => updateField(['azure_devops', 'token'], v)}
          />
          <TextInput
            label="Organization"
            value={data.azure_devops?.organization}
            onChange={(v) => updateField(['azure_devops', 'organization'], v)}
          />
          <TextInput
            label="Project"
            value={data.azure_devops?.project}
            onChange={(v) => updateField(['azure_devops', 'project'], v)}
          />
          <TextInput
            label="Repository Name"
            value={data.azure_devops?.repo_name}
            onChange={(v) => updateField(['azure_devops', 'repo_name'], v)}
          />
        </div>
      </CollapseSection>

      {/* Other providers collapsed by default */}
      <CollapseSection title="Other Providers (Asana, ClickUp, Trello, Wrike, YouTrack)">
        <div className="space-y-4">
          <h4 className="font-medium text-sm">Asana</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <TextInput
              label="Token"
              type="password"
              value={data.asana?.token}
              onChange={(v) => updateField(['asana', 'token'], v)}
            />
            <TextInput
              label="Workspace GID"
              value={data.asana?.workspace_gid}
              onChange={(v) => updateField(['asana', 'workspace_gid'], v)}
            />
            <TextInput
              label="Default Project"
              value={data.asana?.default_project}
              onChange={(v) => updateField(['asana', 'default_project'], v)}
            />
          </div>

          <h4 className="font-medium text-sm mt-4">ClickUp</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <TextInput
              label="Token"
              type="password"
              value={data.clickup?.token}
              onChange={(v) => updateField(['clickup', 'token'], v)}
            />
            <TextInput
              label="Team ID"
              value={data.clickup?.team_id}
              onChange={(v) => updateField(['clickup', 'team_id'], v)}
            />
            <TextInput
              label="Default List"
              value={data.clickup?.default_list}
              onChange={(v) => updateField(['clickup', 'default_list'], v)}
            />
          </div>

          <h4 className="font-medium text-sm mt-4">Trello</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <TextInput
              label="API Key"
              type="password"
              value={data.trello?.api_key}
              onChange={(v) => updateField(['trello', 'api_key'], v)}
            />
            <TextInput
              label="Token"
              type="password"
              value={data.trello?.token}
              onChange={(v) => updateField(['trello', 'token'], v)}
            />
            <TextInput
              label="Board"
              value={data.trello?.board}
              onChange={(v) => updateField(['trello', 'board'], v)}
            />
          </div>

          <h4 className="font-medium text-sm mt-4">Wrike</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <TextInput
              label="Token"
              type="password"
              value={data.wrike?.token}
              onChange={(v) => updateField(['wrike', 'token'], v)}
            />
            <TextInput
              label="Space"
              value={data.wrike?.space}
              onChange={(v) => updateField(['wrike', 'space'], v)}
            />
            <TextInput
              label="Project"
              value={data.wrike?.project}
              onChange={(v) => updateField(['wrike', 'project'], v)}
            />
          </div>

          <h4 className="font-medium text-sm mt-4">YouTrack</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <TextInput
              label="Token"
              type="password"
              value={data.youtrack?.token}
              onChange={(v) => updateField(['youtrack', 'token'], v)}
            />
            <TextInput
              label="Host"
              value={data.youtrack?.host}
              onChange={(v) => updateField(['youtrack', 'host'], v)}
            />
          </div>
        </div>
      </CollapseSection>
    </div>
  )
}

// =============================================================================
// Feature Settings Tab
// =============================================================================
