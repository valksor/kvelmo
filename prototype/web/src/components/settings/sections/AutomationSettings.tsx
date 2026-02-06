import {
  TextInput,
  NumberInput,
  Checkbox,
  Select,
  CollapseSection,
} from '@/components/settings/FormField'
import type { SettingsSectionProps } from './types'

export function AutomationSettings({ data, updateField }: SettingsSectionProps) {
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
