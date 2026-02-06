import { useState } from 'react'
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
    <div className="card bg-base-100 shadow-sm">
      <div className="px-4 pt-4 space-y-3">
        {/* Primary entrypoint: Start task */}
        <button
          type="button"
          onClick={() => setActiveTab('start')}
          className={`w-full px-4 py-2.5 text-sm font-bold rounded-lg flex items-center justify-center gap-2 transition-all ${
            activeTab === 'start'
              ? 'bg-primary text-primary-content shadow-md'
              : 'bg-base-200 text-base-content hover:bg-base-300'
          }`}
        >
          <Zap size={18} />
          <span>Start Task</span>
        </button>

        {/* Secondary modes are available but hidden by default */}
        <div>
          <button
            type="button"
            onClick={() => {
              const next = !showMoreStartModes
              setShowMoreStartModes(next)
              if (!next && activeTab !== 'start') {
                setActiveTab('start')
              }
            }}
            className="text-sm text-primary hover:underline"
          >
            {showMoreStartModes ? '▼ Hide more ways to start' : '▶ More ways to start'}
          </button>

          {showMoreStartModes && (
            <div className="inline-flex p-1 bg-base-200 rounded-lg mt-2">
              <MainTabButton
                active={activeTab === 'quick'}
                onClick={() => setActiveTab('quick')}
                icon={<ListTodo size={14} />}
                label="Quick"
              />
              <MainTabButton
                active={activeTab === 'plan'}
                onClick={() => setActiveTab('plan')}
                icon={<FolderTree size={14} />}
                label="Plan"
              />
            </div>
          )}
        </div>
      </div>

      <div className="card-body pt-4">
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
      <div>
        <label className="block text-sm font-medium text-base-content/80 mb-2">Source</label>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
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
        <div>
          <input
            type="text"
            value={emptyKey}
            onChange={(e) => setEmptyKey(e.target.value)}
            placeholder="Optional: KEY-123 or task title"
            className="input input-bordered w-full"
          />
          <p className="text-xs text-base-content/40 mt-1">
            Start with an empty task. Optionally provide a key or title.
          </p>
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
        className="text-sm text-primary hover:underline"
      >
        {showOptions ? '▼ Hide advanced options' : '▶ Show advanced options'}
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
                className="checkbox checkbox-sm checkbox-primary"
              />
              <span className="text-sm">Worktree</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={noBranch}
                onChange={(e) => setNoBranch(e.target.checked)}
                className="checkbox checkbox-sm checkbox-primary"
              />
              <span className="text-sm">No branch</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={stash}
                onChange={(e) => setStash(e.target.checked)}
                className="checkbox checkbox-sm checkbox-primary"
              />
              <span className="text-sm">Stash changes</span>
            </label>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-base-content/60 mb-1">
                Template
              </label>
              <select
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                className="select select-bordered select-sm w-full"
              >
                {TASK_TEMPLATES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-base-content/60 mb-1">
                External Key
              </label>
              <input
                type="text"
                value={externalKey}
                onChange={(e) => setExternalKey(e.target.value)}
                placeholder="FEATURE-123"
                className="input input-bordered input-sm w-full"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-base-content/60 mb-1">
                Title Override
              </label>
              <input
                type="text"
                value={titleOverride}
                onChange={(e) => setTitleOverride(e.target.value)}
                placeholder="Custom title"
                className="input input-bordered input-sm w-full"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-base-content/60 mb-1">
                Depends On
              </label>
              <input
                type="text"
                value={dependsOn}
                onChange={(e) => setDependsOn(e.target.value)}
                placeholder="Parent task ID"
                className="input input-bordered input-sm w-full"
              />
            </div>
          </div>
        </div>
      )}

      {error && <ErrorMessage message={error} />}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending && <Loader2 size={16} className="animate-spin mr-2" />}
        Start Task
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
}

function MainTabButton({ active, onClick, icon, label }: MainTabButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`px-3 py-1.5 text-xs font-medium rounded-md flex items-center justify-center gap-1.5 transition-all ${
        active
          ? 'bg-base-100 text-base-content shadow-sm'
          : 'text-base-content/50 hover:text-base-content/80'
      }`}
    >
      {icon}
      <span>{label}</span>
    </button>
  )
}
