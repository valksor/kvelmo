import type { WorkspaceConfig } from '@/types/api'

export interface SettingsSectionProps {
  data: Partial<WorkspaceConfig>
  updateField: <T>(path: string[], value: T) => void
}

export interface CoreSettingsProps extends SettingsSectionProps {
  agentOptions: { value: string; label: string }[]
  mode: 'work' | 'advanced'
}
