import { useEffect, useState } from 'react'
import {
  AlertCircle,
  CheckCircle,
  Folder,
  Layers,
  Loader2,
  Save,
  Shield,
  Wrench,
  type LucideIcon,
} from 'lucide-react'
import { useProjects } from '@/api/projects'
import { useAgents, useSaveSettings, useSettings } from '@/api/settings'
import { useStatus } from '@/api/workflow'
import { AutomationSettings } from '@/components/settings/sections/AutomationSettings'
import { CoreSettings } from '@/components/settings/sections/CoreSettings'
import { FeatureSettings } from '@/components/settings/sections/FeatureSettings'
import { ProviderSettings } from '@/components/settings/sections/ProviderSettings'
import type { WorkspaceConfig } from '@/types/api'

type SectionID = 'work' | 'advanced' | 'admin'

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
    label: 'Advanced',
    description: 'Memory, security, browser, sandbox, and power features',
    icon: Wrench,
  },
  {
    id: 'admin',
    label: 'Admin',
    description: 'Automation controls and webhook behavior',
    icon: Shield,
  },
]

export default function Settings() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const isGlobalMode = status?.mode === 'global'

  const [selectedProjectID, setSelectedProjectID] = useState<string | undefined>(undefined)
  const { data: projectsData, isLoading: projectsLoading } = useProjects(isGlobalMode)

  const projectIDForSettings = isGlobalMode ? selectedProjectID : undefined
  const { data: settings, isLoading, error } = useSettings(projectIDForSettings)
  const { data: agents } = useAgents()
  const {
    mutate: saveSettings,
    isPending: isSaving,
    isSuccess,
    isError,
  } = useSaveSettings(projectIDForSettings)

  const [activeSection, setActiveSection] = useState<SectionID>('work')
  const [formData, setFormData] = useState<Partial<WorkspaceConfig>>({})
  const [hasChanges, setHasChanges] = useState(false)

  useEffect(() => {
    if (settings) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- form initialization from fetched data
      setFormData(settings)
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
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
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
  const agentOptions =
    agentList.length > 0
      ? agentList.map((agent) => ({ value: agent.name, label: agent.name }))
      : [
          { value: 'claude', label: 'claude' },
          { value: 'gemini', label: 'gemini' },
          { value: 'ollama', label: 'ollama' },
        ]

  const selectedProject = isGlobalMode
    ? projectsData?.projects?.find((project) => project.id === selectedProjectID)
    : undefined

  return (
    <div className="space-y-4">
      {isGlobalMode && selectedProject && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body py-3 flex-row items-center justify-between">
            <div className="flex items-center gap-3">
              <Folder size={18} className="text-primary" />
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
            Organized to match navigation groups: Work, Advanced, and Admin.
          </p>
        </div>
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
            onClick={() => saveSettings(formData)}
            disabled={isSaving || !hasChanges}
          >
            {isSaving ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
            Save Changes
          </button>
        </div>
      </div>

      <div className="card bg-base-100 shadow-sm border border-base-300/70">
        <div className="card-body p-2 sm:p-3">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
            {sectionNavigation.map(({ id, label, description, icon: Icon }) => (
              <button
                key={id}
                type="button"
                className={`rounded-xl border px-4 py-3 text-left transition-colors ${
                  activeSection === id
                    ? 'border-primary bg-primary/10 shadow-sm'
                    : 'border-base-300 bg-base-100 hover:bg-base-200/60'
                }`}
                onClick={() => setActiveSection(id)}
                aria-pressed={activeSection === id}
                aria-label={`${label} settings section`}
              >
                <div className="flex items-start gap-3">
                  <Icon size={18} className={activeSection === id ? 'text-primary' : 'text-base-content/60'} />
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
          <>
            <CoreSettings
              data={formData}
              agentOptions={agentOptions}
              updateField={updateField}
              mode="work"
            />
            <ProviderSettings data={formData} updateField={updateField} />
          </>
        )}

        {activeSection === 'advanced' && (
          <>
            <CoreSettings
              data={formData}
              agentOptions={agentOptions}
              updateField={updateField}
              mode="advanced"
            />
            <FeatureSettings data={formData} updateField={updateField} />
          </>
        )}

        {activeSection === 'admin' && (
          <AutomationSettings data={formData} updateField={updateField} />
        )}
      </div>
    </div>
  )
}
