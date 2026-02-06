import { useState } from 'react'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import { QueuesPanel } from '@/components/project/QueuesPanel'
import { TasksPanel } from '@/components/project/TasksPanel'
import { EditTaskModal } from '@/components/project/EditTaskModal'
import { FolderKanban, ListTree, FilePlus2, Loader2 } from 'lucide-react'
import type { PlanTask } from '@/api/project-planning'

type Tab = 'create' | 'queues' | 'tasks'

export default function Project() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const [activeTab, setActiveTab] = useState<Tab>('create')
  const [selectedQueueId, setSelectedQueueId] = useState<string | undefined>()
  const [editingTask, setEditingTask] = useState<PlanTask | null>(null)

  if (statusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project selector
  if (status?.mode === 'global') {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Project Planning</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleSelectQueue = (queueId: string) => {
    setSelectedQueueId(queueId)
    setActiveTab('tasks')
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Project Planning</h1>
      </div>

      {/* Tabs */}
      <div className="tabs tabs-boxed bg-base-200 p-1 inline-flex">
        <button
          className={`tab gap-2 ${activeTab === 'create' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('create')}
        >
          <FilePlus2 size={16} />
          Create Plan
        </button>
        <button
          className={`tab gap-2 ${activeTab === 'queues' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('queues')}
        >
          <FolderKanban size={16} />
          Queues
        </button>
        <button
          className={`tab gap-2 ${activeTab === 'tasks' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('tasks')}
          disabled={!selectedQueueId}
        >
          <ListTree size={16} />
          Tasks
        </button>
      </div>

      {/* Tab content */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          {activeTab === 'create' && (
            <div className="max-w-2xl">
              <p className="text-base-content/60 mb-4">
                Create a new project plan from a source file, directory, or provider reference.
                The AI will break down the work into individual tasks.
              </p>
              {/* Reuse the Plan tab from TaskCreationTabs by rendering just that form */}
              <ProjectPlanForm />
            </div>
          )}

          {activeTab === 'queues' && (
            <QueuesPanel
              onSelectQueue={handleSelectQueue}
              selectedQueueId={selectedQueueId}
            />
          )}

          {activeTab === 'tasks' && (
            <TasksPanel
              queueId={selectedQueueId}
              onEditTask={setEditingTask}
            />
          )}
        </div>
      </div>

      {/* Edit task modal */}
      <EditTaskModal
        task={editingTask}
        onClose={() => setEditingTask(null)}
      />
    </div>
  )
}

// ============================================================================
// Project Plan Form (extracted from TaskCreationTabs for standalone use)
// ============================================================================

import { useUploadFile, useCreatePlan } from '@/api/project'
import { Upload, FolderOpen, Search, Puzzle, Loader2 as Loader } from 'lucide-react'

type PlanSourceType = 'file' | 'dir' | 'research' | 'provider'

const PROVIDERS = [
  { value: 'github', label: 'GitHub' },
  { value: 'gitlab', label: 'GitLab' },
  { value: 'jira', label: 'Jira' },
  { value: 'linear', label: 'Linear' },
  { value: 'wrike', label: 'Wrike' },
  { value: 'asana', label: 'Asana' },
  { value: 'clickup', label: 'ClickUp' },
  { value: 'notion', label: 'Notion' },
]

function ProjectPlanForm() {
  const [sourceType, setSourceType] = useState<PlanSourceType>('file')
  const [showAdvancedSources, setShowAdvancedSources] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [dirPath, setDirPath] = useState('')
  const [researchPath, setResearchPath] = useState('')
  const [provider, setProvider] = useState('github')
  const [referenceId, setReferenceId] = useState('')

  const [planTitle, setPlanTitle] = useState('')
  const [instructions, setInstructions] = useState('')
  const [useSchema, setUseSchema] = useState(true)

  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const uploadFile = useUploadFile()
  const createPlan = useCreatePlan()

  const isPending = uploadFile.isPending || createPlan.isPending

  const resetForm = () => {
    setFile(null)
    setDirPath('')
    setResearchPath('')
    setReferenceId('')
    setPlanTitle('')
    setInstructions('')
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccess(null)

    try {
      let source: string

      switch (sourceType) {
        case 'file': {
          if (!file) throw new Error('Please select a file')
          const uploadResult = await uploadFile.mutateAsync(file)
          source = uploadResult.source
          break
        }
        case 'dir':
          if (!dirPath.trim()) throw new Error('Please enter a directory path')
          source = `dir:${dirPath}`
          break
        case 'research':
          if (!researchPath.trim()) throw new Error('Please enter a path')
          source = `research:${researchPath}`
          break
        case 'provider':
          if (!referenceId.trim()) throw new Error('Please enter a reference ID')
          source = `${provider}:${referenceId}`
          break
        default:
          throw new Error('Unknown source type')
      }

      const result = await createPlan.mutateAsync({
        source,
        title: planTitle || undefined,
        instructions: instructions || undefined,
        use_schema: useSchema,
      })

      setSuccess(`Plan created! Queue ID: ${result.queue_id}, ${result.tasks.length} tasks`)
      resetForm()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Source Type Selector */}
      <div>
        <label className="block text-sm font-medium text-base-content/80 mb-2">Source Type</label>
        <div className="grid grid-cols-1 gap-2">
          <SourceTypeButton
            active={sourceType === 'file'}
            onClick={() => setSourceType('file')}
            icon={<Upload size={16} />}
            label="File"
          />
        </div>

        <button
          type="button"
          onClick={() => {
            const next = !showAdvancedSources
            setShowAdvancedSources(next)
            if (!next && sourceType !== 'file') {
              setSourceType('file')
            }
          }}
          className="text-sm text-primary hover:underline mt-2"
        >
          {showAdvancedSources ? '▼ Hide advanced source types' : '▶ Show advanced source types'}
        </button>

        {showAdvancedSources && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-2 mt-2">
            <SourceTypeButton
              active={sourceType === 'dir'}
              onClick={() => setSourceType('dir')}
              icon={<FolderOpen size={16} />}
              label="Directory"
            />
            <SourceTypeButton
              active={sourceType === 'research'}
              onClick={() => setSourceType('research')}
              icon={<Search size={16} />}
              label="Research"
            />
            <SourceTypeButton
              active={sourceType === 'provider'}
              onClick={() => setSourceType('provider')}
              icon={<Puzzle size={16} />}
              label="Provider"
            />
          </div>
        )}
      </div>

      {/* Source Input */}
      {sourceType === 'file' && (
        <div
          className="border-2 border-dashed border-base-300 rounded-xl p-6 text-center hover:border-primary hover:bg-primary/5 transition-all cursor-pointer"
          onClick={() => document.getElementById('plan-file-input')?.click()}
        >
          <input
            type="file"
            id="plan-file-input"
            accept=".md,.txt,.markdown"
            className="hidden"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
          />
          <Upload className="w-8 h-8 mx-auto text-base-content/40 mb-2" />
          <p className="text-sm text-base-content/60">
            Drop file here or <span className="text-primary font-medium">browse</span>
          </p>
          {file && <p className="text-sm text-primary mt-2 font-medium">Selected: {file.name}</p>}
        </div>
      )}

      {sourceType === 'dir' && (
        <div>
          <input
            type="text"
            value={dirPath}
            onChange={(e) => setDirPath(e.target.value)}
            placeholder="./path/to/directory"
            className="input input-bordered w-full font-mono text-sm"
          />
          <p className="text-xs text-base-content/40 mt-1">
            Reads ALL files in directory. Best for small codebases (&lt;50 files).
          </p>
        </div>
      )}

      {sourceType === 'research' && (
        <div>
          <input
            type="text"
            value={researchPath}
            onChange={(e) => setResearchPath(e.target.value)}
            placeholder="./path/to/docs"
            className="input input-bordered w-full font-mono text-sm"
          />
          <p className="text-xs text-base-content/40 mt-1">
            AI-guided research mode. Provides file manifest, AI selectively explores files.
          </p>
        </div>
      )}

      {sourceType === 'provider' && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-base-content/60 mb-1">Provider</label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="select select-bordered w-full"
            >
              {PROVIDERS.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-base-content/60 mb-1">Reference</label>
            <input
              type="text"
              value={referenceId}
              onChange={(e) => setReferenceId(e.target.value)}
              placeholder="123 or PROJECT-123"
              className="input input-bordered w-full font-mono text-sm"
            />
          </div>
        </div>
      )}

      {/* Options */}
      <div>
        <label className="block text-sm font-medium text-base-content/80 mb-2">
          Project Title (optional)
        </label>
        <input
          type="text"
          value={planTitle}
          onChange={(e) => setPlanTitle(e.target.value)}
          placeholder="My Project"
          className="input input-bordered w-full"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-base-content/80 mb-2">
          Instructions (optional)
        </label>
        <textarea
          rows={3}
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
          placeholder="Custom instructions for AI planning..."
          className="textarea textarea-bordered w-full"
        />
      </div>

      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={useSchema}
          onChange={(e) => setUseSchema(e.target.checked)}
          className="checkbox checkbox-primary checkbox-sm"
        />
        <span className="text-sm text-base-content/80">Use schema-driven extraction</span>
      </label>

      {error && (
        <div className="alert alert-error text-sm">{error}</div>
      )}

      {success && (
        <div className="alert alert-success text-sm">{success}</div>
      )}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending && <Loader size={16} className="animate-spin mr-2" />}
        Create Plan
      </button>
    </form>
  )
}

interface SourceTypeButtonProps {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}

function SourceTypeButton({ active, onClick, icon, label }: SourceTypeButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`p-3 rounded-xl border-2 flex flex-col items-center gap-1 transition-all ${
        active
          ? 'border-primary bg-primary/10 text-primary'
          : 'border-base-300 hover:border-primary/50 text-base-content/60 hover:text-base-content'
      }`}
    >
      {icon}
      <span className="text-xs font-medium">{label}</span>
    </button>
  )
}
