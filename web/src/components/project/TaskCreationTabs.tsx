import { useId, useState } from 'react'
import {
  Upload,
  FileText,
  Puzzle,
  Loader2,
  Zap,
  ListTodo,
  FolderTree,
  FileQuestion,
} from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { useCreateSource, useUploadFile } from '@/api/project'
import { apiRequest } from '@/api/client'
import { ProjectPlanForm } from '@/components/project/ProjectPlanForm'
import { QuickTaskForm } from '@/components/project/QuickTaskForm'
import {
  ErrorMessage,
  FileInput,
  ReferenceInput,
  SourceTypeButton,
  TextInput,
} from '@/components/project/TaskFormShared'
import { TASK_TEMPLATES } from '@/constants/taskOptions'

type MainTab = 'start' | 'quick' | 'plan'

export function TaskCreationTabs() {
  const [activeTab, setActiveTab] = useState<MainTab>('start')
  const [showMoreStartModes, setShowMoreStartModes] = useState(false)

  return (
    <div className="card bg-base-100 shadow-sm border border-base-300/70">
      <div className="px-4 pt-4 pb-4 space-y-3 border-b border-base-300/70">
        <div className="space-y-1">
          <h3 className="font-semibold text-base-content">How would you like to start?</h3>
          <p className="text-sm text-base-content/60">
            Use Start Task for full workflow, or expand to quick capture and plan-first modes.
          </p>
        </div>

        <button
          type="button"
          onClick={() => setActiveTab('start')}
          className={`w-full px-4 py-3 text-sm font-semibold rounded-xl border flex items-center justify-center gap-2 transition-colors ${
            activeTab === 'start'
              ? 'border-primary bg-primary/10 text-primary shadow-sm'
              : 'border-base-300 bg-base-100 text-base-content hover:bg-base-200/60'
          }`}
        >
          <Zap size={18} />
          <span>Start Task</span>
        </button>

        <div className="space-y-2">
          <button
            type="button"
            onClick={() => {
              const next = !showMoreStartModes
              setShowMoreStartModes(next)
              if (!next && activeTab !== 'start') {
                setActiveTab('start')
              }
            }}
            className="btn btn-ghost btn-sm justify-start px-2"
          >
            {showMoreStartModes ? 'Hide more ways to start' : 'More ways to start'}
          </button>

          {showMoreStartModes && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 rounded-xl border border-base-300/70 bg-base-200/30 p-2">
              <MainTabButton
                active={activeTab === 'quick'}
                onClick={() => setActiveTab('quick')}
                icon={<ListTodo size={14} />}
                label="Quick"
                description="Capture and submit tasks fast"
              />
              <MainTabButton
                active={activeTab === 'plan'}
                onClick={() => setActiveTab('plan')}
                icon={<FolderTree size={14} />}
                label="Plan"
                description="Create a multi-task project plan"
              />
            </div>
          )}
        </div>
      </div>

      <div className="card-body pt-5">
        {activeTab === 'start' && <StartTaskForm />}
        {activeTab === 'quick' && <QuickTaskForm />}
        {activeTab === 'plan' && (
          <ProjectPlanForm
            description="Upload a specification or requirements file to create a project plan with multiple tasks"
          />
        )}
      </div>
    </div>
  )
}

// ============================================================================
// Start Task Form
// ============================================================================

type StartSourceType = 'file' | 'text' | 'empty' | 'provider'

function StartTaskForm() {
  const id = useId()
  const [sourceType, setSourceType] = useState<StartSourceType>('text')
  const [file, setFile] = useState<File | null>(null)
  const [textContent, setTextContent] = useState('')
  const [emptyKey, setEmptyKey] = useState('')
  const [provider, setProvider] = useState('github')
  const [referenceId, setReferenceId] = useState('')

  // Options
  const [useWorktree, setUseWorktree] = useState(false)
  const [noBranch, setNoBranch] = useState(false)
  const [stash, setStash] = useState(false)
  const [template, setTemplate] = useState('')
  const [externalKey, setExternalKey] = useState('')
  const [titleOverride, setTitleOverride] = useState('')
  const [dependsOn, setDependsOn] = useState('')
  const [showOptions, setShowOptions] = useState(false)

  const [error, setError] = useState<string | null>(null)

  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const uploadFile = useUploadFile()
  const createSource = useCreateSource()

  const startTask = useMutation({
    mutationFn: async (params: {
      ref?: string
      content?: string
      worktree?: boolean
      no_branch?: boolean
      stash?: boolean
      template?: string
      key?: string
      title?: string
      depends_on?: string
    }) => {
      return apiRequest<{ task_id: string }>('/workflow/start', {
        method: 'POST',
        body: JSON.stringify(params),
      })
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['task'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      if (data?.task_id) {
        navigate(`/task/${data.task_id}`)
      }
    },
  })

  const isPending = uploadFile.isPending || createSource.isPending || startTask.isPending

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      let ref: string | undefined
      let content: string | undefined

      switch (sourceType) {
        case 'file': {
          if (!file) throw new Error('Please select a file')
          const uploadResult = await uploadFile.mutateAsync(file)
          ref = uploadResult.source
          break
        }
        case 'text':
          if (!textContent.trim()) throw new Error('Please enter content')
          content = textContent
          break
        case 'empty':
          ref = emptyKey ? `empty:${emptyKey}` : 'empty:'
          break
        case 'provider':
          if (!referenceId.trim()) throw new Error('Please enter a reference ID')
          ref = `${provider}:${referenceId}`
          break
      }

      await startTask.mutateAsync({
        ref,
        content,
        worktree: useWorktree || undefined,
        no_branch: noBranch || undefined,
        stash: stash || undefined,
        template: template || undefined,
        key: externalKey || undefined,
        title: titleOverride || undefined,
        depends_on: dependsOn || undefined,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-base-content/60">
        Start a workflow task with AI planning and implementation
      </p>

      {/* Source Type Selector */}
      <div className="form-control">
        <label className="label py-1" htmlFor={`${id}-source-type`}>
          <span className="label-text">Source</span>
        </label>
        <div id={`${id}-source-type`} className="grid grid-cols-2 md:grid-cols-4 gap-2">
          <SourceTypeButton
            active={sourceType === 'file'}
            onClick={() => setSourceType('file')}
            icon={<Upload size={16} />}
            label="Upload"
          />
          <SourceTypeButton
            active={sourceType === 'text'}
            onClick={() => setSourceType('text')}
            icon={<FileText size={16} />}
            label="Write"
          />
          <SourceTypeButton
            active={sourceType === 'empty'}
            onClick={() => setSourceType('empty')}
            icon={<FileQuestion size={16} />}
            label="Empty"
          />
          <SourceTypeButton
            active={sourceType === 'provider'}
            onClick={() => setSourceType('provider')}
            icon={<Puzzle size={16} />}
            label="Provider"
          />
        </div>
      </div>

      {/* Source Input */}
      {sourceType === 'file' && <FileInput file={file} onFileChange={setFile} />}
      {sourceType === 'text' && <TextInput value={textContent} onChange={setTextContent} />}
      {sourceType === 'empty' && (
        <div className="form-control">
          <label className="label py-1" htmlFor="task-empty-key">
            <span className="label-text">Key or title (optional)</span>
          </label>
          <input
            id="task-empty-key"
            type="text"
            value={emptyKey}
            onChange={(e) => setEmptyKey(e.target.value)}
            placeholder="Optional: KEY-123 or task title"
            className="input input-bordered w-full"
          />
          <label className="label py-1" htmlFor="task-empty-key">
            <span className="label-text-alt text-base-content/60">
              Start with an empty task. Optionally provide a key or title.
            </span>
          </label>
        </div>
      )}
      {sourceType === 'provider' && (
        <ReferenceInput
          provider={provider}
          onProviderChange={setProvider}
          referenceId={referenceId}
          onReferenceIdChange={setReferenceId}
        />
      )}

      {/* Options Toggle */}
      <button
        type="button"
        onClick={() => setShowOptions(!showOptions)}
        className="btn btn-ghost btn-sm justify-start px-2"
      >
        {showOptions ? 'Hide advanced options' : 'Show advanced options'}
      </button>

      {/* Options Panel */}
      {showOptions && (
        <div className="p-4 bg-base-200/50 rounded-lg space-y-4">
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={useWorktree}
                onChange={(e) => setUseWorktree(e.target.checked)}
                className="checkbox checkbox-primary"
              />
              <span className="text-sm">Worktree</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={noBranch}
                onChange={(e) => setNoBranch(e.target.checked)}
                className="checkbox checkbox-primary"
              />
              <span className="text-sm">No branch</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={stash}
                onChange={(e) => setStash(e.target.checked)}
                className="checkbox checkbox-primary"
              />
              <span className="text-sm">Stash changes</span>
            </label>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="form-control">
              <label className="label py-1" htmlFor={`${id}-template`}>
                <span className="label-text">Template</span>
              </label>
              <select
                id={`${id}-template`}
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                className="select select-bordered w-full"
              >
                {TASK_TEMPLATES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="form-control">
              <label className="label py-1" htmlFor={`${id}-external-key`}>
                <span className="label-text">External Key</span>
              </label>
              <input
                id={`${id}-external-key`}
                type="text"
                value={externalKey}
                onChange={(e) => setExternalKey(e.target.value)}
                placeholder="FEATURE-123"
                className="input input-bordered w-full"
              />
            </div>
            <div className="form-control">
              <label className="label py-1" htmlFor={`${id}-title-override`}>
                <span className="label-text">Title Override</span>
              </label>
              <input
                id={`${id}-title-override`}
                type="text"
                value={titleOverride}
                onChange={(e) => setTitleOverride(e.target.value)}
                placeholder="Custom title"
                className="input input-bordered w-full"
              />
            </div>
            <div className="form-control">
              <label className="label py-1" htmlFor={`${id}-depends-on`}>
                <span className="label-text">Depends On</span>
              </label>
              <input
                id={`${id}-depends-on`}
                type="text"
                value={dependsOn}
                onChange={(e) => setDependsOn(e.target.value)}
                placeholder="Parent task ID"
                className="input input-bordered w-full"
              />
            </div>
          </div>
        </div>
      )}

      {error && <ErrorMessage message={error} />}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending ? (
          <>
            <Loader2 size={16} className="animate-spin mr-2" />
            Starting Workflow...
          </>
        ) : (
          'Start Workflow'
        )}
      </button>
    </form>
  )
}

// ============================================================================
// Shared Sub-components
// ============================================================================

interface MainTabButtonProps {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
  description: string
}

function MainTabButton({ active, onClick, icon, label, description }: MainTabButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full rounded-lg border px-3 py-2 text-left transition-colors ${
        active
          ? 'border-primary bg-primary/10 text-primary shadow-sm'
          : 'border-base-300 bg-base-100 text-base-content hover:bg-base-200/60'
      }`}
    >
      <div className="flex items-start gap-2">
        <span className="mt-0.5">{icon}</span>
        <span className="space-y-0.5">
          <span className="block text-sm font-medium">{label}</span>
          <span className="block text-xs text-base-content/65">{description}</span>
        </span>
      </div>
    </button>
  )
}
