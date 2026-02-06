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
import {
  useCreateQuickTask,
  useCreatePlan,
  useUploadFile,
  useCreateSource,
  useSubmitSource,
} from '@/api/project'
import { apiRequest } from '@/api/client'

type MainTab = 'start' | 'quick' | 'plan'

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

const TEMPLATES = [
  { value: '', label: 'No template' },
  { value: 'bug-fix', label: 'Bug Fix' },
  { value: 'feature', label: 'Feature' },
  { value: 'refactor', label: 'Refactor' },
  { value: 'docs', label: 'Documentation' },
  { value: 'test', label: 'Test' },
  { value: 'chore', label: 'Chore' },
]

export function TaskCreationTabs() {
  const [activeTab, setActiveTab] = useState<MainTab>('start')

  return (
    <div className="card bg-base-100 shadow-sm">
      {/* Main Tab Navigation */}
      <div className="px-4 pt-4">
        <div className="flex items-center justify-between">
          {/* Primary: Start Task - the main feature */}
          <button
            type="button"
            onClick={() => setActiveTab('start')}
            className={`px-4 py-2.5 text-sm font-bold rounded-lg flex items-center gap-2 transition-all ${
              activeTab === 'start'
                ? 'bg-primary text-primary-content shadow-md'
                : 'bg-base-200 text-base-content hover:bg-base-300'
            }`}
          >
            <Zap size={18} />
            <span>Start Task</span>
          </button>

          {/* Secondary: Quick + Plan (right side) */}
          <div className="inline-flex p-1 bg-base-200 rounded-lg">
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
        </div>
      </div>

      <div className="card-body pt-4">
        {activeTab === 'start' && <StartTaskForm />}
        {activeTab === 'quick' && <QuickTaskForm />}
        {activeTab === 'plan' && <ProjectPlanForm />}
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
        {showOptions ? '▼ Hide options' : '▶ Show options'}
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
                {TEMPLATES.map((t) => (
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
// Quick Task Form
// ============================================================================

type QuickMode = 'simple' | 'source'

function QuickTaskForm() {
  const [mode, setMode] = useState<QuickMode>('simple')

  // Simple mode
  const [description, setDescription] = useState('')
  const [title, setTitle] = useState('')
  const [priority, setPriority] = useState(2)
  const [labels, setLabels] = useState('')

  // From Source mode
  const [source, setSource] = useState('')
  const [provider, setProvider] = useState('github')
  const [notes, setNotes] = useState('')
  const [instructions, setInstructions] = useState('')
  const [optimize, setOptimize] = useState(false)
  const [dryRun, setDryRun] = useState(false)

  const [error, setError] = useState<string | null>(null)

  const createQuickTask = useCreateQuickTask()
  const submitSource = useSubmitSource()

  const isPending = createQuickTask.isPending || submitSource.isPending

  const resetForm = () => {
    setDescription('')
    setTitle('')
    setLabels('')
    setSource('')
    setNotes('')
    setInstructions('')
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      if (mode === 'simple') {
        if (!description.trim()) throw new Error('Please enter a description')
        await createQuickTask.mutateAsync({
          description,
          title: title || undefined,
          priority,
          labels: labels
            .split(',')
            .map((l) => l.trim())
            .filter(Boolean),
        })
      } else {
        if (!source.trim()) throw new Error('Please enter a source')
        await submitSource.mutateAsync({
          source,
          provider,
          title: title || undefined,
          notes: notes
            .split('\n')
            .map((n) => n.trim())
            .filter(Boolean),
          instructions: instructions || undefined,
          labels: labels
            .split(',')
            .map((l) => l.trim())
            .filter(Boolean),
          optimize,
          dry_run: dryRun,
        })
      }
      resetForm()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-base-content/60">
        Capture a quick task without full planning workflow
      </p>

      {/* Mode Toggle */}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setMode('simple')}
          className={`flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-all ${
            mode === 'simple'
              ? 'bg-primary text-primary-content'
              : 'bg-base-200 text-base-content/60 hover:text-base-content'
          }`}
        >
          Simple
        </button>
        <button
          type="button"
          onClick={() => setMode('source')}
          className={`flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-all ${
            mode === 'source'
              ? 'bg-primary text-primary-content'
              : 'bg-base-200 text-base-content/60 hover:text-base-content'
          }`}
        >
          From Source
        </button>
      </div>

      {mode === 'simple' ? (
        <>
          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-base-content/80 mb-2">
              Description <span className="text-error">*</span>
            </label>
            <textarea
              rows={4}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe the task...&#10;&#10;e.g., Fix typo in README.md line 42"
              className="textarea textarea-bordered w-full"
            />
          </div>

          {/* Title + Priority + Labels */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Title (optional)
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Auto-extracted"
                className="input input-bordered w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Priority
              </label>
              <select
                value={priority}
                onChange={(e) => setPriority(Number(e.target.value))}
                className="select select-bordered w-full"
              >
                <option value={1}>High</option>
                <option value={2}>Normal</option>
                <option value={3}>Low</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">Labels</label>
              <input
                type="text"
                value={labels}
                onChange={(e) => setLabels(e.target.value)}
                placeholder="bug, urgent"
                className="input input-bordered w-full"
              />
            </div>
          </div>
        </>
      ) : (
        <>
          {/* Source + Provider */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Source <span className="text-error">*</span>
              </label>
              <input
                type="text"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                placeholder="./docs or file:requirements.md"
                className="input input-bordered w-full font-mono text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Provider <span className="text-error">*</span>
              </label>
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
          </div>

          {/* Title + Labels */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Title (optional)
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Override generated title"
                className="input input-bordered w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">Labels</label>
              <input
                type="text"
                value={labels}
                onChange={(e) => setLabels(e.target.value)}
                placeholder="bug, urgent"
                className="input input-bordered w-full"
              />
            </div>
          </div>

          {/* Notes + Instructions */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Notes (one per line)
              </label>
              <textarea
                rows={3}
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Add constraints or guidance"
                className="textarea textarea-bordered w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-base-content/80 mb-2">
                Instructions
              </label>
              <textarea
                rows={3}
                value={instructions}
                onChange={(e) => setInstructions(e.target.value)}
                placeholder="How to interpret the source"
                className="textarea textarea-bordered w-full"
              />
            </div>
          </div>

          {/* Checkboxes */}
          <div className="flex gap-6">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={optimize}
                onChange={(e) => setOptimize(e.target.checked)}
                className="checkbox checkbox-sm checkbox-primary"
              />
              <span className="text-sm">Optimize with AI</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                className="checkbox checkbox-sm checkbox-primary"
              />
              <span className="text-sm">Dry run</span>
            </label>
          </div>
        </>
      )}

      {error && <ErrorMessage message={error} />}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending && <Loader2 size={16} className="animate-spin mr-2" />}
        {mode === 'simple' ? 'Create Quick Task' : 'Submit From Source'}
      </button>
    </form>
  )
}

// ============================================================================
// Project Plan Form
// ============================================================================

function ProjectPlanForm() {
  const [file, setFile] = useState<File | null>(null)

  // Options
  const [planTitle, setPlanTitle] = useState('')
  const [instructions, setInstructions] = useState('')
  const [useSchema, setUseSchema] = useState(true)

  const [error, setError] = useState<string | null>(null)

  const uploadFile = useUploadFile()
  const createPlan = useCreatePlan()

  const isPending = uploadFile.isPending || createPlan.isPending

  const resetForm = () => {
    setFile(null)
    setPlanTitle('')
    setInstructions('')
    setError(null)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      if (!file) throw new Error('Please select a file')
      const uploadResult = await uploadFile.mutateAsync(file)
      const source = uploadResult.source

      const result = await createPlan.mutateAsync({
        source,
        title: planTitle || undefined,
        instructions: instructions || undefined,
        use_schema: useSchema,
      })

      resetForm()
      alert(`Plan created! Queue ID: ${result.queue_id}, ${result.tasks.length} tasks`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-base-content/60">
        Upload a specification or requirements file to create a project plan with multiple tasks
      </p>

      {/* File Upload */}
      <FileInput file={file} onFileChange={setFile} />

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
          placeholder="Custom instructions for AI planning...&#10;e.g., Focus on API design first, implementation second"
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

      {error && <ErrorMessage message={error} />}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending && <Loader2 size={16} className="animate-spin mr-2" />}
        Create Plan
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

interface FileInputProps {
  file: File | null
  onFileChange: (file: File | null) => void
}

function FileInput({ file, onFileChange }: FileInputProps) {
  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile) onFileChange(droppedFile)
  }

  return (
    <div
      className="border-2 border-dashed border-base-300 rounded-xl p-6 text-center hover:border-primary hover:bg-primary/5 transition-all cursor-pointer"
      onClick={() => document.getElementById('task-file-input')?.click()}
      onDragOver={(e) => e.preventDefault()}
      onDrop={handleDrop}
    >
      <input
        type="file"
        id="task-file-input"
        accept=".md,.txt,.markdown,.zip,.tar.gz"
        className="hidden"
        onChange={(e) => onFileChange(e.target.files?.[0] || null)}
      />
      <Upload className="w-10 h-10 mx-auto text-base-content/40 mb-2" />
      <p className="text-sm text-base-content/60">
        Drop file here or <span className="text-primary font-medium">browse</span>
      </p>
      <p className="text-xs text-base-content/40 mt-1">.md, .txt, .zip, .tar.gz (max 10MB)</p>
      {file && <p className="text-sm text-primary mt-2 font-medium">Selected: {file.name}</p>}
    </div>
  )
}

interface TextInputProps {
  value: string
  onChange: (value: string) => void
}

function TextInput({ value, onChange }: TextInputProps) {
  return (
    <div>
      <textarea
        rows={6}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="# Task Title&#10;&#10;Describe what you want to accomplish...&#10;&#10;## Requirements&#10;- First requirement&#10;- Second requirement"
        className="textarea textarea-bordered w-full font-mono text-sm"
      />
      <p className="text-xs text-base-content/40 mt-1">Use Markdown for better structure</p>
    </div>
  )
}

interface ReferenceInputProps {
  provider: string
  onProviderChange: (value: string) => void
  referenceId: string
  onReferenceIdChange: (value: string) => void
}

function ReferenceInput({
  provider,
  onProviderChange,
  referenceId,
  onReferenceIdChange,
}: ReferenceInputProps) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <div>
        <label className="block text-xs font-medium text-base-content/60 mb-1">Provider</label>
        <select
          value={provider}
          onChange={(e) => onProviderChange(e.target.value)}
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
        <label className="block text-xs font-medium text-base-content/60 mb-1">Reference ID</label>
        <input
          type="text"
          value={referenceId}
          onChange={(e) => onReferenceIdChange(e.target.value)}
          placeholder="123 or PROJECT-123"
          className="input input-bordered w-full font-mono text-sm"
        />
      </div>
    </div>
  )
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="p-3 bg-error/10 border border-error/20 rounded-lg text-error text-sm">
      {message}
    </div>
  )
}
