import { useState, useEffect } from 'react'
import { Loader2, Save, AlertCircle, CheckCircle, Folder } from 'lucide-react'
import { useSettings, useSaveSettings, useAgents } from '@/api/settings'
import { useStatus } from '@/api/workflow'
import { useProjects } from '@/api/projects'
import {
  TextInput,
  NumberInput,
  Checkbox,
  Select,
  TextArea,
  CollapseSection,
} from '@/components/settings/FormField'
import type { WorkspaceConfig } from '@/types/api'

type TabId = 'core' | 'providers' | 'features' | 'automation'

export default function Settings() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const isGlobalMode = status?.mode === 'global'

  // In global mode, we need to select a project first
  const [selectedProjectId, setSelectedProjectId] = useState<string | undefined>(undefined)
  const { data: projectsData, isLoading: projectsLoading } = useProjects(isGlobalMode)

  // Fetch settings - in global mode, only when a project is selected
  const projectIdForSettings = isGlobalMode ? selectedProjectId : undefined
  const { data: settings, isLoading, error } = useSettings(projectIdForSettings)
  const { data: agents } = useAgents()
  const { mutate: saveSettings, isPending: isSaving, isSuccess, isError } = useSaveSettings(projectIdForSettings)

  const [activeTab, setActiveTab] = useState<TabId>('core')
  const [formData, setFormData] = useState<Partial<WorkspaceConfig>>({})
  const [hasChanges, setHasChanges] = useState(false)

  // Initialize form data when settings load - valid pattern for form sync
  useEffect(() => {
    if (settings) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- form initialization from fetched data
      setFormData(settings)
      setHasChanges(false)
    }
  }, [settings])

  // Helper to update nested fields
  const updateField = <T,>(path: string[], value: T) => {
    setFormData((prev) => {
      const newData = { ...prev }
      let current: Record<string, unknown> = newData
      for (let i = 0; i < path.length - 1; i++) {
        const key = path[i]
        if (!current[key] || typeof current[key] !== 'object') {
          current[key] = {}
        }
        current[key] = { ...(current[key] as Record<string, unknown>) }
        current = current[key] as Record<string, unknown>
      }
      current[path[path.length - 1]] = value
      return newData
    })
    setHasChanges(true)
  }

  const handleSave = () => {
    saveSettings(formData)
  }

  // Loading state
  if (statusLoading || (isGlobalMode && projectsLoading)) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project picker
  if (isGlobalMode && !selectedProjectId) {
    const projects = projectsData?.projects ?? []
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-base-content/60">Select a project to configure its settings.</p>

        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h3 className="card-title flex items-center gap-2">
              <Folder size={20} />
              Select Project
            </h3>

            {projects.length === 0 ? (
              <div className="text-center py-8">
                <Folder className="w-12 h-12 mx-auto text-base-content/40 mb-4" />
                <p className="text-base-content/60">No projects registered yet.</p>
                <p className="text-sm text-base-content/40 mt-2">
                  Register a project with <code className="kbd kbd-sm">mehr serve register</code>
                </p>
              </div>
            ) : (
              <select
                className="select select-bordered w-full"
                value=""
                onChange={(e) => setSelectedProjectId(e.target.value)}
              >
                <option value="" disabled>-- Select a project --</option>
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.name} ({project.id})
                  </option>
                ))}
              </select>
            )}
          </div>
        </div>
      </div>
    )
  }

  // Loading settings for selected project
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        <AlertCircle size={20} />
        <span>Failed to load settings: {error.message}</span>
      </div>
    )
  }

  const agentList = agents?.agents ?? []
  const agentOptions = agentList.length > 0
    ? agentList.map((a) => ({ value: a.name, label: a.name }))
    : [{ value: 'claude', label: 'claude' }, { value: 'gemini', label: 'gemini' }, { value: 'ollama', label: 'ollama' }]

  // Find selected project name for display
  const selectedProject = isGlobalMode
    ? projectsData?.projects?.find(p => p.id === selectedProjectId)
    : undefined

  return (
    <div className="space-y-4">
      {/* Project picker banner in global mode */}
      {isGlobalMode && selectedProject && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body py-3 flex-row items-center justify-between">
            <div className="flex items-center gap-3">
              <Folder size={18} className="text-primary" />
              <span className="font-medium">Editing: {selectedProject.name}</span>
              <span className="text-xs text-base-content/50 font-mono">{selectedProject.path}</span>
            </div>
            <button
              className="btn btn-ghost btn-sm"
              onClick={() => setSelectedProjectId(undefined)}
            >
              Change Project
            </button>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Settings</h1>
        <div className="flex items-center gap-2">
          {isSuccess && (
            <span className="text-success flex items-center gap-1 text-sm">
              <CheckCircle size={16} /> Saved
            </span>
          )}
          {isError && (
            <span className="text-error flex items-center gap-1 text-sm">
              <AlertCircle size={16} /> Failed to save
            </span>
          )}
          <button
            className="btn btn-primary"
            onClick={handleSave}
            disabled={isSaving || !hasChanges}
          >
            {isSaving ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
            Save Changes
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div role="tablist" className="tabs tabs-bordered">
        {(['core', 'providers', 'features', 'automation'] as TabId[]).map((tab) => (
          <button
            key={tab}
            role="tab"
            className={`tab ${activeTab === tab ? 'tab-active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="space-y-4">
        {activeTab === 'core' && (
          <CoreSettings
            data={formData}
            agentOptions={agentOptions}
            updateField={updateField}
          />
        )}
        {activeTab === 'providers' && (
          <ProviderSettings data={formData} updateField={updateField} />
        )}
        {activeTab === 'features' && (
          <FeatureSettings data={formData} updateField={updateField} />
        )}
        {activeTab === 'automation' && (
          <AutomationSettings data={formData} updateField={updateField} />
        )}
      </div>
    </div>
  )
}

// =============================================================================
// Core Settings Tab
// =============================================================================

interface SectionProps {
  data: Partial<WorkspaceConfig>
  updateField: <T>(path: string[], value: T) => void
}

interface CoreSettingsProps extends SectionProps {
  agentOptions: { value: string; label: string }[]
}

function CoreSettings({ data, agentOptions, updateField }: CoreSettingsProps) {
  return (
    <div className="space-y-4">
      {/* Git Settings */}
      <CollapseSection title="Git" defaultOpen>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Commit Prefix"
            hint="Pattern for commit messages. Use {key}, {type}, {slug}"
            value={data.git?.commit_prefix}
            onChange={(v) => updateField(['git', 'commit_prefix'], v)}
          />
          <TextInput
            label="Branch Pattern"
            hint="Pattern for branch names. Use {key}, {type}, {slug}"
            value={data.git?.branch_pattern}
            onChange={(v) => updateField(['git', 'branch_pattern'], v)}
          />
          <TextInput
            label="Default Branch"
            hint="Override branch detection (e.g., main, develop)"
            value={data.git?.default_branch}
            onChange={(v) => updateField(['git', 'default_branch'], v)}
          />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4">
          <Checkbox
            label="Auto Commit"
            hint="Commit after implementation"
            checked={data.git?.auto_commit}
            onChange={(v) => updateField(['git', 'auto_commit'], v)}
          />
          <Checkbox
            label="Sign Commits"
            hint="GPG sign commits"
            checked={data.git?.sign_commits}
            onChange={(v) => updateField(['git', 'sign_commits'], v)}
          />
          <Checkbox
            label="Stash on Start"
            hint="Auto-stash changes"
            checked={data.git?.stash_on_start}
            onChange={(v) => updateField(['git', 'stash_on_start'], v)}
          />
          <Checkbox
            label="Auto Pop Stash"
            hint="Pop stash after branch creation"
            checked={data.git?.auto_pop_stash}
            onChange={(v) => updateField(['git', 'auto_pop_stash'], v)}
          />
        </div>
      </CollapseSection>

      {/* Agent Settings */}
      <CollapseSection title="Agent" defaultOpen>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Select
            label="Default Agent"
            hint="Agent to use when not specified"
            value={data.agent?.default}
            onChange={(v) => updateField(['agent', 'default'], v)}
            options={agentOptions}
          />
          <NumberInput
            label="Timeout (seconds)"
            hint="Maximum time for agent execution"
            value={data.agent?.timeout}
            onChange={(v) => updateField(['agent', 'timeout'], v)}
            min={30}
            max={3600}
          />
          <NumberInput
            label="Max Retries"
            hint="Retry count on transient failures"
            value={data.agent?.max_retries}
            onChange={(v) => updateField(['agent', 'max_retries'], v)}
            min={0}
            max={10}
          />
        </div>
        <TextArea
          label="Instructions"
          hint="Global instructions included in all agent prompts"
          value={data.agent?.instructions}
          onChange={(v) => updateField(['agent', 'instructions'], v)}
          rows={4}
        />
        <Checkbox
          label="Optimize Prompts"
          hint="Optimize prompts for token efficiency"
          checked={data.agent?.optimize_prompts}
          onChange={(v) => updateField(['agent', 'optimize_prompts'], v)}
        />
      </CollapseSection>

      {/* Workflow Settings */}
      <CollapseSection title="Workflow">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <NumberInput
            label="Session Retention (days)"
            hint="How long to keep session logs"
            value={data.workflow?.session_retention_days}
            onChange={(v) => updateField(['workflow', 'session_retention_days'], v)}
            min={1}
            max={365}
          />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mt-4">
          <Checkbox
            label="Auto Init"
            hint="Auto-initialize workspace"
            checked={data.workflow?.auto_init}
            onChange={(v) => updateField(['workflow', 'auto_init'], v)}
          />
          <Checkbox
            label="Delete Work on Finish"
            hint="Clean up work directory after finish"
            checked={data.workflow?.delete_work_on_finish}
            onChange={(v) => updateField(['workflow', 'delete_work_on_finish'], v)}
          />
          <Checkbox
            label="Delete Work on Abandon"
            hint="Clean up work directory on abandon"
            checked={data.workflow?.delete_work_on_abandon}
            onChange={(v) => updateField(['workflow', 'delete_work_on_abandon'], v)}
          />
        </div>
      </CollapseSection>

      {/* Budget Settings */}
      <CollapseSection title="Budget">
        <h4 className="font-medium text-sm text-base-content/70 mb-2">Per Task</h4>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <NumberInput
            label="Max Cost ($)"
            value={data.budget?.per_task?.max_cost}
            onChange={(v) => updateField(['budget', 'per_task', 'max_cost'], v)}
            min={0}
            step={0.01}
          />
          <NumberInput
            label="Max Tokens"
            value={data.budget?.per_task?.max_tokens}
            onChange={(v) => updateField(['budget', 'per_task', 'max_tokens'], v)}
            min={0}
            step={1000}
          />
          <Select
            label="On Limit"
            value={data.budget?.per_task?.on_limit}
            onChange={(v) => updateField(['budget', 'per_task', 'on_limit'], v)}
            options={[
              { value: 'warn', label: 'Warn' },
              { value: 'pause', label: 'Pause' },
              { value: 'stop', label: 'Stop' },
            ]}
          />
          <NumberInput
            label="Warning At (%)"
            hint="0-100"
            value={(data.budget?.per_task?.warning_at ?? 0.8) * 100}
            onChange={(v) => updateField(['budget', 'per_task', 'warning_at'], v / 100)}
            min={0}
            max={100}
          />
        </div>
        <h4 className="font-medium text-sm text-base-content/70 mb-2 mt-4">Monthly</h4>
        <Checkbox
          label="Enable Monthly Budget"
          hint="Track spending across the workspace"
          checked={data.budget?.monthly?.enabled ?? false}
          onChange={(v) => updateField(['budget', 'monthly', 'enabled'], v)}
        />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-2">
          <NumberInput
            label="Max Cost ($)"
            value={data.budget?.monthly?.max_cost}
            onChange={(v) => updateField(['budget', 'monthly', 'max_cost'], v)}
            min={0}
            step={1}
          />
          <TextInput
            label="Currency"
            value={data.budget?.monthly?.currency}
            onChange={(v) => updateField(['budget', 'monthly', 'currency'], v)}
          />
          <NumberInput
            label="Warning At (%)"
            value={(data.budget?.monthly?.warning_at ?? 0.8) * 100}
            onChange={(v) => updateField(['budget', 'monthly', 'warning_at'], v / 100)}
            min={0}
            max={100}
          />
        </div>
      </CollapseSection>

      {/* Project & Storage */}
      <CollapseSection title="Project & Storage">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TextInput
            label="Code Directory"
            hint="Separate code directory (relative or absolute)"
            value={data.project?.code_dir}
            onChange={(v) => updateField(['project', 'code_dir'], v)}
          />
          <TextInput
            label="Project Directory"
            hint="Where to store work files in project"
            value={data.storage?.project_dir}
            onChange={(v) => updateField(['storage', 'project_dir'], v)}
          />
        </div>
        <Checkbox
          label="Save in Project"
          hint="Store work in project directory instead of global"
          checked={data.storage?.save_in_project}
          onChange={(v) => updateField(['storage', 'save_in_project'], v)}
        />
      </CollapseSection>

      {/* Stack Settings */}
      <CollapseSection title="Stack (Feature Branches)">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Select
            label="Auto Rebase"
            hint="When to auto-rebase child branches"
            value={data.stack?.auto_rebase}
            onChange={(v) => updateField(['stack', 'auto_rebase'], v)}
            options={[
              { value: 'disabled', label: 'Disabled' },
              { value: 'on_finish', label: 'On Finish' },
            ]}
          />
          <Checkbox
            label="Block on Conflicts"
            hint="Block auto-rebase if conflicts detected"
            checked={data.stack?.block_on_conflicts}
            onChange={(v) => updateField(['stack', 'block_on_conflicts'], v)}
          />
        </div>
      </CollapseSection>

      {/* Update & Patterns */}
      <CollapseSection title="Updates & Patterns">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Checkbox
              label="Enable Update Checks"
              checked={data.update?.enabled}
              onChange={(v) => updateField(['update', 'enabled'], v)}
            />
            <NumberInput
              label="Check Interval (hours)"
              value={data.update?.check_interval}
              onChange={(v) => updateField(['update', 'check_interval'], v)}
              min={1}
              max={168}
            />
          </div>
          <div>
            <TextInput
              label="Specification Pattern"
              hint="Pattern for spec filenames"
              value={data.specification?.filename_pattern}
              onChange={(v) => updateField(['specification', 'filename_pattern'], v)}
            />
            <TextInput
              label="Review Pattern"
              hint="Pattern for review filenames"
              value={data.review?.filename_pattern}
              onChange={(v) => updateField(['review', 'filename_pattern'], v)}
            />
          </div>
        </div>
      </CollapseSection>
    </div>
  )
}

// =============================================================================
// Provider Settings Tab
// =============================================================================

function ProviderSettings({ data, updateField }: SectionProps) {
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

function FeatureSettings({ data, updateField }: SectionProps) {
  return (
    <div className="space-y-4">
      {/* Browser */}
      <CollapseSection title="Browser Automation">
        <Checkbox
          label="Enable Browser"
          hint="Allow AI agents to control a browser"
          checked={data.browser?.enabled}
          onChange={(v) => updateField(['browser', 'enabled'], v)}
        />
        {data.browser?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
            <NumberInput
              label="Port"
              hint="0 = random, 9222 = existing Chrome"
              value={data.browser?.port}
              onChange={(v) => updateField(['browser', 'port'], v)}
              min={0}
            />
            <NumberInput
              label="Timeout (seconds)"
              value={data.browser?.timeout}
              onChange={(v) => updateField(['browser', 'timeout'], v)}
              min={5}
              max={300}
            />
            <TextInput
              label="Screenshot Directory"
              value={data.browser?.screenshot_dir}
              onChange={(v) => updateField(['browser', 'screenshot_dir'], v)}
            />
            <Checkbox
              label="Headless"
              hint="Run browser without UI"
              checked={data.browser?.headless}
              onChange={(v) => updateField(['browser', 'headless'], v)}
            />
            <Checkbox
              label="Auto-load Cookies"
              checked={data.browser?.cookie_auto_load}
              onChange={(v) => updateField(['browser', 'cookie_auto_load'], v)}
            />
            <Checkbox
              label="Auto-save Cookies"
              checked={data.browser?.cookie_auto_save}
              onChange={(v) => updateField(['browser', 'cookie_auto_save'], v)}
            />
          </div>
        )}
      </CollapseSection>

      {/* MCP */}
      <CollapseSection title="MCP (Model Context Protocol)">
        <Checkbox
          label="Enable MCP Server"
          hint="Allow AI agents to call Mehrhof commands via MCP"
          checked={data.mcp?.enabled}
          onChange={(v) => updateField(['mcp', 'enabled'], v)}
        />
        {data.mcp?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            <NumberInput
              label="Rate Limit (req/sec)"
              value={data.mcp?.rate_limit?.rate}
              onChange={(v) => updateField(['mcp', 'rate_limit', 'rate'], v)}
              min={1}
              max={100}
            />
            <NumberInput
              label="Burst Size"
              value={data.mcp?.rate_limit?.burst}
              onChange={(v) => updateField(['mcp', 'rate_limit', 'burst'], v)}
              min={1}
              max={200}
            />
          </div>
        )}
      </CollapseSection>

      {/* Security */}
      <CollapseSection title="Security Scanning">
        <Checkbox
          label="Enable Security Scanning"
          hint="Scan code for vulnerabilities and secrets"
          checked={data.security?.enabled}
          onChange={(v) => updateField(['security', 'enabled'], v)}
        />
        {data.security?.enabled && (
          <>
            <h4 className="font-medium text-sm mt-4 mb-2">Run On</h4>
            <div className="grid grid-cols-3 gap-4">
              <Checkbox
                label="Planning"
                checked={data.security?.run_on?.planning}
                onChange={(v) => updateField(['security', 'run_on', 'planning'], v)}
              />
              <Checkbox
                label="Implementing"
                checked={data.security?.run_on?.implementing}
                onChange={(v) => updateField(['security', 'run_on', 'implementing'], v)}
              />
              <Checkbox
                label="Reviewing"
                checked={data.security?.run_on?.reviewing}
                onChange={(v) => updateField(['security', 'run_on', 'reviewing'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Fail On</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Severity Level"
                value={data.security?.fail_on?.level}
                onChange={(v) => updateField(['security', 'fail_on', 'level'], v)}
                options={[
                  { value: 'critical', label: 'Critical' },
                  { value: 'high', label: 'High' },
                  { value: 'medium', label: 'Medium' },
                  { value: 'low', label: 'Low' },
                  { value: 'any', label: 'Any' },
                ]}
              />
              <Checkbox
                label="Block Finish"
                hint="Block task completion on failures"
                checked={data.security?.fail_on?.block_finish}
                onChange={(v) => updateField(['security', 'fail_on', 'block_finish'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Scanners</h4>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Checkbox
                label="SAST"
                checked={data.security?.scanners?.sast?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'sast', 'enabled'], v)}
              />
              <Checkbox
                label="Secrets"
                checked={data.security?.scanners?.secrets?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'secrets', 'enabled'], v)}
              />
              <Checkbox
                label="Dependencies"
                checked={data.security?.scanners?.dependencies?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'dependencies', 'enabled'], v)}
              />
              <Checkbox
                label="License"
                checked={data.security?.scanners?.license?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'license', 'enabled'], v)}
              />
            </div>
          </>
        )}
      </CollapseSection>

      {/* Memory */}
      <CollapseSection title="Memory System">
        <Checkbox
          label="Enable Memory"
          hint="Semantic search and learning from past tasks"
          checked={data.memory?.enabled}
          onChange={(v) => updateField(['memory', 'enabled'], v)}
        />
        {data.memory?.enabled && (
          <>
            <h4 className="font-medium text-sm mt-4 mb-2">Vector Database</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Backend"
                value={data.memory?.vector_db?.backend}
                onChange={(v) => updateField(['memory', 'vector_db', 'backend'], v)}
                options={[
                  { value: 'chromadb', label: 'ChromaDB' },
                  { value: 'pinecone', label: 'Pinecone' },
                  { value: 'weaviate', label: 'Weaviate' },
                  { value: 'qdrant', label: 'Qdrant' },
                ]}
              />
              <TextInput
                label="Connection String"
                value={data.memory?.vector_db?.connection_string}
                onChange={(v) => updateField(['memory', 'vector_db', 'connection_string'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Embedding Model</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Model Type"
                hint="ONNX provides true semantic similarity"
                value={data.memory?.vector_db?.embedding_model || 'default'}
                onChange={(v) => updateField(['memory', 'vector_db', 'embedding_model'], v)}
                options={[
                  { value: 'default', label: 'Hash-based (default)' },
                  { value: 'onnx', label: 'ONNX Neural (semantic)' },
                ]}
              />
              {data.memory?.vector_db?.embedding_model === 'onnx' && (
                <Select
                  label="ONNX Model"
                  hint="Downloaded on first use"
                  value={data.memory?.vector_db?.onnx?.model || 'all-MiniLM-L6-v2'}
                  onChange={(v) => updateField(['memory', 'vector_db', 'onnx', 'model'], v)}
                  options={[
                    { value: 'all-MiniLM-L6-v2', label: 'all-MiniLM-L6-v2 (22MB, fast)' },
                    { value: 'all-MiniLM-L12-v2', label: 'all-MiniLM-L12-v2 (33MB, better)' },
                  ]}
                />
              )}
            </div>
            {data.memory?.vector_db?.embedding_model === 'onnx' && (
              <p className="text-xs text-muted-foreground mt-2">
                Switching embedding models invalidates existing vectors. Run <code className="bg-muted px-1 rounded">mehr memory clear</code> after changing.
              </p>
            )}
            <h4 className="font-medium text-sm mt-4 mb-2">Search</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <NumberInput
                label="Max Results"
                value={data.memory?.search?.max_results}
                onChange={(v) => updateField(['memory', 'search', 'max_results'], v)}
                min={1}
                max={50}
              />
              <NumberInput
                label="Similarity Threshold"
                hint="0-1"
                value={data.memory?.search?.similarity_threshold}
                onChange={(v) => updateField(['memory', 'search', 'similarity_threshold'], v)}
                min={0}
                max={1}
                step={0.1}
              />
            </div>
          </>
        )}
      </CollapseSection>

      {/* Sandbox */}
      <CollapseSection title="Sandbox">
        <Checkbox
          label="Enable Sandbox"
          hint="Isolate agent execution for security"
          checked={data.sandbox?.enabled}
          onChange={(v) => updateField(['sandbox', 'enabled'], v)}
        />
        {data.sandbox?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            <Checkbox
              label="Allow Network"
              hint="Required for LLM APIs"
              checked={data.sandbox?.network}
              onChange={(v) => updateField(['sandbox', 'network'], v)}
            />
            <TextInput
              label="Tmp Directory"
              value={data.sandbox?.tmp_dir}
              onChange={(v) => updateField(['sandbox', 'tmp_dir'], v)}
            />
          </div>
        )}
      </CollapseSection>

      {/* Quality */}
      <CollapseSection title="Quality & Linters">
        <Checkbox
          label="Enable Quality Checks"
          checked={data.quality?.enabled}
          onChange={(v) => updateField(['quality', 'enabled'], v)}
        />
        <Checkbox
          label="Use Defaults"
          hint="Auto-enable default linters for detected languages"
          checked={data.quality?.use_defaults}
          onChange={(v) => updateField(['quality', 'use_defaults'], v)}
        />
      </CollapseSection>

      {/* Links */}
      <CollapseSection title="Links (Bidirectional Linking)">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Checkbox
            label="Enabled"
            checked={data.links?.enabled}
            onChange={(v) => updateField(['links', 'enabled'], v)}
          />
          <Checkbox
            label="Auto Index"
            checked={data.links?.auto_index}
            onChange={(v) => updateField(['links', 'auto_index'], v)}
          />
          <Checkbox
            label="Case Sensitive"
            checked={data.links?.case_sensitive}
            onChange={(v) => updateField(['links', 'case_sensitive'], v)}
          />
          <NumberInput
            label="Max Context Length"
            value={data.links?.max_context_length}
            onChange={(v) => updateField(['links', 'max_context_length'], v)}
            min={50}
            max={500}
          />
        </div>
      </CollapseSection>

      {/* Context */}
      <CollapseSection title="Hierarchical Context">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Checkbox
            label="Include Parent"
            checked={data.context?.include_parent}
            onChange={(v) => updateField(['context', 'include_parent'], v)}
          />
          <Checkbox
            label="Include Siblings"
            checked={data.context?.include_siblings}
            onChange={(v) => updateField(['context', 'include_siblings'], v)}
          />
          <NumberInput
            label="Max Siblings"
            value={data.context?.max_siblings}
            onChange={(v) => updateField(['context', 'max_siblings'], v)}
            min={1}
            max={20}
          />
          <NumberInput
            label="Description Limit"
            value={data.context?.description_limit}
            onChange={(v) => updateField(['context', 'description_limit'], v)}
            min={100}
            max={2000}
          />
        </div>
      </CollapseSection>

      {/* Labels */}
      <CollapseSection title="Labels">
        <Checkbox
          label="Enable Labels"
          checked={data.labels?.enabled}
          onChange={(v) => updateField(['labels', 'enabled'], v)}
        />
      </CollapseSection>

      {/* Library */}
      <CollapseSection title="Library (Documentation)">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <NumberInput
            label="Auto Include Max"
            hint="Max collections to auto-include"
            value={data.library?.auto_include_max}
            onChange={(v) => updateField(['library', 'auto_include_max'], v)}
            min={0}
            max={10}
          />
          <NumberInput
            label="Max Pages Per Prompt"
            value={data.library?.max_pages_per_prompt}
            onChange={(v) => updateField(['library', 'max_pages_per_prompt'], v)}
            min={1}
            max={100}
          />
          <NumberInput
            label="Max Token Budget"
            value={data.library?.max_token_budget}
            onChange={(v) => updateField(['library', 'max_token_budget'], v)}
            min={1000}
            max={50000}
          />
        </div>
      </CollapseSection>
    </div>
  )
}

// =============================================================================
// Automation Settings Tab
// =============================================================================

function AutomationSettings({ data, updateField }: SectionProps) {
  return (
    <div className="space-y-4">
      {/* Master Enable */}
      <CollapseSection title="Webhook Automation" defaultOpen>
        <Checkbox
          label="Enable Automation"
          hint="Process GitHub/GitLab webhooks automatically"
          checked={data.automation?.enabled}
          onChange={(v) => updateField(['automation', 'enabled'], v)}
        />
      </CollapseSection>

      {data.automation?.enabled && (
        <>
          {/* GitHub Provider */}
          <CollapseSection title="GitHub Triggers">
            <Checkbox
              label="Enable GitHub"
              checked={data.automation?.providers?.github?.enabled}
              onChange={(v) => updateField(['automation', 'providers', 'github', 'enabled'], v)}
            />
            {data.automation?.providers?.github?.enabled && (
              <>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                  <TextInput
                    label="Webhook Secret"
                    type="password"
                    value={data.automation?.providers?.github?.webhook_secret}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'github', 'webhook_secret'], v)
                    }
                  />
                  <TextInput
                    label="Command Prefix"
                    hint="Comment trigger (default: @mehrhof)"
                    value={data.automation?.providers?.github?.command_prefix}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'github', 'command_prefix'], v)
                    }
                  />
                </div>
                <h4 className="font-medium text-sm mt-4 mb-2">Trigger On</h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Checkbox
                    label="Issue Opened"
                    checked={data.automation?.providers?.github?.trigger_on?.issue_opened}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'github', 'trigger_on', 'issue_opened'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="PR Opened"
                    checked={data.automation?.providers?.github?.trigger_on?.pr_opened}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'github', 'trigger_on', 'pr_opened'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="PR Updated"
                    checked={data.automation?.providers?.github?.trigger_on?.pr_updated}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'github', 'trigger_on', 'pr_updated'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="Comment Commands"
                    checked={data.automation?.providers?.github?.trigger_on?.comment_commands}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'github', 'trigger_on', 'comment_commands'],
                        v
                      )
                    }
                  />
                </div>
                <div className="grid grid-cols-2 gap-4 mt-4">
                  <Checkbox
                    label="Use Worktrees"
                    hint="Isolate work with git worktrees"
                    checked={data.automation?.providers?.github?.use_worktrees}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'github', 'use_worktrees'], v)
                    }
                  />
                  <Checkbox
                    label="Dry Run"
                    hint="Log actions without executing"
                    checked={data.automation?.providers?.github?.dry_run}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'github', 'dry_run'], v)
                    }
                  />
                </div>
              </>
            )}
          </CollapseSection>

          {/* GitLab Provider */}
          <CollapseSection title="GitLab Triggers">
            <Checkbox
              label="Enable GitLab"
              checked={data.automation?.providers?.gitlab?.enabled}
              onChange={(v) => updateField(['automation', 'providers', 'gitlab', 'enabled'], v)}
            />
            {data.automation?.providers?.gitlab?.enabled && (
              <>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                  <TextInput
                    label="Webhook Secret"
                    type="password"
                    value={data.automation?.providers?.gitlab?.webhook_secret}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'gitlab', 'webhook_secret'], v)
                    }
                  />
                  <TextInput
                    label="Command Prefix"
                    value={data.automation?.providers?.gitlab?.command_prefix}
                    onChange={(v) =>
                      updateField(['automation', 'providers', 'gitlab', 'command_prefix'], v)
                    }
                  />
                </div>
                <h4 className="font-medium text-sm mt-4 mb-2">Trigger On</h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Checkbox
                    label="Issue Opened"
                    checked={data.automation?.providers?.gitlab?.trigger_on?.issue_opened}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'gitlab', 'trigger_on', 'issue_opened'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="MR Opened"
                    checked={data.automation?.providers?.gitlab?.trigger_on?.mr_opened}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'gitlab', 'trigger_on', 'mr_opened'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="MR Updated"
                    checked={data.automation?.providers?.gitlab?.trigger_on?.mr_updated}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'gitlab', 'trigger_on', 'mr_updated'],
                        v
                      )
                    }
                  />
                  <Checkbox
                    label="Comment Commands"
                    checked={data.automation?.providers?.gitlab?.trigger_on?.comment_commands}
                    onChange={(v) =>
                      updateField(
                        ['automation', 'providers', 'gitlab', 'trigger_on', 'comment_commands'],
                        v
                      )
                    }
                  />
                </div>
              </>
            )}
          </CollapseSection>

          {/* Access Control */}
          <CollapseSection title="Access Control">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Mode"
                value={data.automation?.access_control?.mode}
                onChange={(v) => updateField(['automation', 'access_control', 'mode'], v)}
                options={[
                  { value: 'all', label: 'All Users' },
                  { value: 'allowlist', label: 'Allowlist Only' },
                  { value: 'blocklist', label: 'Blocklist' },
                ]}
              />
              <div className="flex gap-4">
                <Checkbox
                  label="Allow Bots"
                  checked={data.automation?.access_control?.allow_bots}
                  onChange={(v) => updateField(['automation', 'access_control', 'allow_bots'], v)}
                />
                <Checkbox
                  label="Require Org"
                  checked={data.automation?.access_control?.require_org}
                  onChange={(v) => updateField(['automation', 'access_control', 'require_org'], v)}
                />
              </div>
            </div>
          </CollapseSection>

          {/* Queue Settings */}
          <CollapseSection title="Queue Settings">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <NumberInput
                label="Max Concurrent"
                value={data.automation?.queue?.max_concurrent}
                onChange={(v) => updateField(['automation', 'queue', 'max_concurrent'], v)}
                min={1}
                max={10}
              />
              <TextInput
                label="Job Timeout"
                hint="e.g., 30m, 1h"
                value={data.automation?.queue?.job_timeout}
                onChange={(v) => updateField(['automation', 'queue', 'job_timeout'], v)}
              />
              <NumberInput
                label="Retry Attempts"
                value={data.automation?.queue?.retry_attempts}
                onChange={(v) => updateField(['automation', 'queue', 'retry_attempts'], v)}
                min={0}
                max={5}
              />
              <TextInput
                label="Retry Delay"
                hint="e.g., 30s, 5m"
                value={data.automation?.queue?.retry_delay}
                onChange={(v) => updateField(['automation', 'queue', 'retry_delay'], v)}
              />
            </div>
          </CollapseSection>

          {/* Labels */}
          <CollapseSection title="Automation Labels">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <TextInput
                label="Generated Label"
                hint="Label for mehrhof PRs"
                value={data.automation?.labels?.mehr_generated}
                onChange={(v) => updateField(['automation', 'labels', 'mehr_generated'], v)}
              />
              <TextInput
                label="In Progress Label"
                value={data.automation?.labels?.in_progress}
                onChange={(v) => updateField(['automation', 'labels', 'in_progress'], v)}
              />
              <TextInput
                label="Failed Label"
                value={data.automation?.labels?.failed}
                onChange={(v) => updateField(['automation', 'labels', 'failed'], v)}
              />
              <TextInput
                label="Skip Review Label"
                value={data.automation?.labels?.skip_review}
                onChange={(v) => updateField(['automation', 'labels', 'skip_review'], v)}
              />
            </div>
          </CollapseSection>
        </>
      )}
    </div>
  )
}
