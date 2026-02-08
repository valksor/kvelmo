import { useEffect, useState } from 'react'
import {
  AlertCircle,
  CheckCircle,
  Folder,
  Layers,
  Loader2,
  Save,
  Wrench,
  type LucideIcon,
} from 'lucide-react'
import { useProjects } from '@/api/projects'
import { useSaveSettings, useSettings } from '@/api/settings'
import { useStatus } from '@/api/workflow'
import { useSettingsMode } from '@/hooks/useSettingsMode'
import { SettingsModeToggle } from '@/components/settings/SettingsModeToggle'
import { DynamicSettings } from '@/components/settings/DynamicSettings'
import type { WorkspaceConfig } from '@/types/api'

type SectionID = 'work' | 'advanced'

interface SectionMeta {
  id: SectionID
  label: string
  description: string
  icon: LucideIcon
}

const sectionNavigation: SectionMeta[] = [
  {
    id: 'work',
    label: 'Work',
    description: 'Project workflow, git, agents, and provider defaults',
    icon: Layers,
  },
  {
    id: 'advanced',
    label: 'System',
    description: 'Memory, security, browser, sandbox, and power features',
    icon: Wrench,
  },
]

// Section IDs grouped by tab - core/providers for Work, features for System
const workSections = [
  'git', 'agent', 'workflow', 'budget', 'project', 'storage',
  'specification', 'review', 'display', 'providers', 'github', 'gitlab',
  'jira', 'linear', 'notion', 'bitbucket', 'asana', 'clickup',
  'azure_devops', 'trello', 'wrike', 'youtrack',
]

const advancedSections = [
  'browser', 'mcp', 'security', 'memory', 'library', 'orchestration',
  'ml', 'sandbox', 'labels', 'quality', 'links', 'context', 'stack',
  'update', 'plugins',
]

export default function Settings() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const isGlobalMode = status?.mode === 'global'

  const [selectedProjectID, setSelectedProjectID] = useState<string | undefined>(undefined)
  const { data: projectsData, isLoading: projectsLoading } = useProjects(isGlobalMode)

  const projectIDForSettings = isGlobalMode ? selectedProjectID : undefined
  const { data: settings, isLoading, error } = useSettings(projectIDForSettings)

  const {
    mutate: saveSettings,
    isPending: isSaving,
    isSuccess,
    isError,
  } = useSaveSettings(projectIDForSettings)

  const [activeSection, setActiveSection] = useState<SectionID>('work')
  const [formData, setFormData] = useState<Partial<WorkspaceConfig>>({})
  const [hasChanges, setHasChanges] = useState(false)
  const { isSimple, toggleMode } = useSettingsMode()

  useEffect(() => {
    if (settings?.values) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- form initialization from fetched data
      setFormData(settings.values)
      setHasChanges(false)
    }
  }, [settings])

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

  if (statusLoading || (isGlobalMode && projectsLoading)) {
    return (
      <div className="flex items-center justify-center min-h-[400px]" role="status" aria-label="Loading">
        <Loader2 className="w-8 h-8 animate-spin text-primary" aria-hidden="true" />
      </div>
    )
  }

  if (isGlobalMode && !selectedProjectID) {
    const projects = projectsData?.projects ?? []
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-base-content/60">Select a project to configure its settings.</p>

        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h2 className="card-title flex items-center gap-2">
              <Folder size={20} aria-hidden="true" />
              Select Project
            </h2>

            {projects.length === 0 ? (
              <div className="text-center py-8">
                <Folder className="w-12 h-12 mx-auto text-base-content/40 mb-4" aria-hidden="true" />
                <p className="text-base-content/60">No projects registered yet.</p>
                <p className="text-sm text-base-content/40 mt-2">
                  Register a project with <code className="kbd kbd-sm">mehr serve register</code>
                </p>
              </div>
            ) : (
              <select
                className="select select-bordered w-full"
                value=""
                onChange={(e) => setSelectedProjectID(e.target.value)}
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]" role="status" aria-label="Loading">
        <Loader2 className="w-8 h-8 animate-spin text-primary" aria-hidden="true" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error" role="alert">
        <AlertCircle size={20} aria-hidden="true" />
        <span>Failed to load settings: {error.message}</span>
      </div>
    )
  }

  const selectedProject = isGlobalMode
    ? projectsData?.projects?.find((project) => project.id === selectedProjectID)
    : undefined

  return (
    <div className="space-y-4">
      {isGlobalMode && selectedProject && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body py-3 flex-row items-center justify-between">
            <div className="flex items-center gap-3">
              <Folder size={18} className="text-primary" aria-hidden="true" />
              <span className="font-medium">Editing: {selectedProject.name}</span>
              <span className="text-xs text-base-content/50 font-mono">{selectedProject.path}</span>
            </div>
            <button className="btn btn-ghost btn-sm" onClick={() => setSelectedProjectID(undefined)}>
              Change Project
            </button>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Settings</h1>
          <p className="text-base-content/60 text-sm">
            {isSimple ? 'Showing essential settings only.' : 'Showing all settings.'}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <SettingsModeToggle isSimple={isSimple} onToggle={toggleMode} />
          <div className="flex items-center gap-2">
          {isSuccess && (
            <span className="text-success flex items-center gap-1 text-sm" role="status">
              <CheckCircle size={16} aria-hidden="true" /> Saved
            </span>
          )}
          {isError && (
            <span className="text-error flex items-center gap-1 text-sm" role="alert">
              <AlertCircle size={16} aria-hidden="true" /> Failed to save
            </span>
          )}
            <button
              className="btn btn-primary"
              onClick={() => saveSettings(formData)}
              disabled={isSaving || !hasChanges}
            >
              {isSaving ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : <Save size={16} aria-hidden="true" />}
              Save Changes
            </button>
          </div>
        </div>
      </div>

      {isSimple && (
        <div className="alert alert-info" role="status">
          <Wrench size={18} aria-hidden="true" />
          <div>
            <p className="font-medium">Looking for more settings?</p>
            <p className="text-sm opacity-80">
              Switch to Advanced mode to see all configuration options.
            </p>
          </div>
          <button className="btn btn-sm btn-ghost" onClick={toggleMode}>
            Show All Settings
          </button>
        </div>
      )}

      <div className="card bg-base-100 shadow-sm border border-base-300/70">
        <div className="card-body p-2 sm:p-3">
          <div role="tablist" aria-label="Settings sections" className="grid grid-cols-1 md:grid-cols-2 gap-2">
            {sectionNavigation.map(({ id, label, description, icon: Icon }) => (
              <button
                key={id}
                type="button"
                role="tab"
                id={`tab-settings-${id}`}
                aria-selected={activeSection === id}
                aria-controls={`tabpanel-settings-${id}`}
                className={`rounded-xl border px-4 py-3 text-left transition-colors ${
                  activeSection === id
                    ? 'border-primary bg-primary/10 shadow-sm'
                    : 'border-base-300 bg-base-100 hover:bg-base-200/60'
                }`}
                onClick={() => setActiveSection(id)}
              >
                <div className="flex items-start gap-3">
                  <Icon size={18} className={activeSection === id ? 'text-primary' : 'text-base-content/60'} aria-hidden="true" />
                  <div className="space-y-1">
                    <p className="font-semibold">{label}</p>
                    <p className="text-xs text-base-content/65">{description}</p>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {activeSection === 'work' && (
          <div role="tabpanel" id="tabpanel-settings-work" aria-labelledby="tab-settings-work" className="space-y-4">
            <DynamicSettings
              projectId={projectIDForSettings}
              sectionIds={workSections}
              values={formData as Record<string, unknown>}
              onChange={updateField}
              simpleMode={isSimple}
            />
          </div>
        )}

        {activeSection === 'advanced' && (
          <div role="tabpanel" id="tabpanel-settings-advanced" aria-labelledby="tab-settings-advanced" className="space-y-4">
            <DynamicSettings
              projectId={projectIDForSettings}
              sectionIds={advancedSections}
              values={formData as Record<string, unknown>}
              onChange={updateField}
              simpleMode={isSimple}
            />
          </div>
        )}

      </div>
    </div>
  )
}
