import { Plus, Trash2 } from 'lucide-react'
import {
  TextInput,
  NumberInput,
  Checkbox,
  Select,
  TextArea,
  CollapseSection,
} from '@/components/settings/FormField'
import type { CoreSettingsProps } from './types'

export function CoreSettings({ data, agentOptions, updateField, mode }: CoreSettingsProps) {
  const isWorkMode = mode === 'work'

  return (
    <div className="space-y-4">
      {isWorkMode && (
        <>
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
        </>
      )}

      {!isWorkMode && (
        <>
          {/* Agent Aliases */}
          <CollapseSection title="Agent Aliases" defaultOpen>
            <p className="text-sm text-base-content/60 mb-4">
              Create aliases for agents with custom binary paths, environment variables, or CLI arguments.
            </p>
            {Object.entries(data.agents ?? {}).map(([name, alias]) => (
              <div key={name} className="card bg-base-200 mb-3">
                <div className="card-body p-4">
                  <div className="flex justify-between items-start">
                    <h4 className="font-medium">{name}</h4>
                    <button
                      type="button"
                      className="btn btn-ghost btn-xs text-error"
                      onClick={() => {
                        const newAgents = { ...data.agents }
                        delete newAgents[name]
                        updateField(['agents'], Object.keys(newAgents).length > 0 ? newAgents : undefined)
                      }}
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mt-2">
                    <Select
                      label="Extends"
                      hint="Base agent to wrap"
                      value={alias.extends}
                      onChange={(v) => updateField(['agents', name, 'extends'], v)}
                      options={agentOptions}
                    />
                    <TextInput
                      label="Binary Path"
                      hint="Custom binary path (optional)"
                      value={alias.binary_path}
                      onChange={(v) => updateField(['agents', name, 'binary_path'], v || undefined)}
                      placeholder="/path/to/binary"
                    />
                    <TextInput
                      label="Description"
                      hint="Human-readable description"
                      value={alias.description}
                      onChange={(v) => updateField(['agents', name, 'description'], v || undefined)}
                    />
                    <TextInput
                      label="Args"
                      hint="CLI arguments (space-separated)"
                      value={alias.args?.join(' ')}
                      onChange={(v) =>
                        updateField(
                          ['agents', name, 'args'],
                          v ? v.split(/\s+/).filter(Boolean) : undefined
                        )
                      }
                      placeholder="--model opus"
                    />
                  </div>
                </div>
              </div>
            ))}
            <button
              type="button"
              className="btn btn-outline btn-sm gap-2"
              onClick={() => {
                const name = prompt('Alias name:')
                if (name && name.trim()) {
                  updateField(['agents', name.trim()], { extends: 'claude' })
                }
              }}
            >
              <Plus size={16} />
              Add Alias
            </button>
          </CollapseSection>

          {/* Budget Settings */}
          <CollapseSection title="Budget">
            <Checkbox
              label="Enable Budget Tracking"
              hint="Track costs per task and monthly"
              checked={data.budget?.enabled ?? false}
              onChange={(v) => updateField(['budget', 'enabled'], v)}
            />
            {(data.budget?.enabled ?? false) && (
              <>
                <h4 className="font-medium text-sm text-base-content/70 mb-2 mt-4">Per Task</h4>
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
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
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
              </>
            )}
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
        </>
      )}
    </div>
  )
}
