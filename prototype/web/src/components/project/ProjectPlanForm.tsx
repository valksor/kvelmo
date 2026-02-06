import { useId, useRef, useState } from 'react'
import { FolderOpen, Loader2, Puzzle, Search, Upload } from 'lucide-react'
import { useCreatePlan, useUploadFile, type CreatePlanResponse } from '@/api/project'
import { TASK_SOURCE_PROVIDERS } from '@/constants/taskOptions'

type PlanSourceType = 'file' | 'dir' | 'research' | 'provider'

interface ProjectPlanFormProps {
  allowAdvancedSources?: boolean
  showSuccessMessage?: boolean
  description?: string
  onCreated?: (result: CreatePlanResponse) => void
}

export function ProjectPlanForm({
  allowAdvancedSources = false,
  showSuccessMessage = true,
  description,
  onCreated,
}: ProjectPlanFormProps) {
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
  const fileInputID = useId()
  const fileInputRef = useRef<HTMLInputElement>(null)

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

      if (showSuccessMessage) {
        setSuccess(`Plan created! Queue ID: ${result.queue_id}, ${result.tasks.length} tasks`)
      }

      onCreated?.(result)
      resetForm()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const showAdvanced = allowAdvancedSources && showAdvancedSources

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <p className="text-sm text-base-content/60">
        {description ??
          'Upload a specification or requirements file to create a project plan with multiple tasks'}
      </p>

      <div className="form-control">
        <label className="label py-1">
          <span className="label-text">Source Type</span>
        </label>
        <div className="grid grid-cols-1 gap-2">
          <SourceTypeButton
            active={sourceType === 'file'}
            onClick={() => setSourceType('file')}
            icon={<Upload size={16} />}
            label="File"
          />
        </div>

        {allowAdvancedSources && (
          <>
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
              {showAdvancedSources
                ? '▼ Hide advanced source types'
                : '▶ Show advanced source types'}
            </button>

            {showAdvanced && (
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
          </>
        )}
      </div>

      {sourceType === 'file' && (
        <div
          className="border-2 border-dashed border-base-300 rounded-xl p-6 text-center hover:border-primary hover:bg-primary/5 transition-all cursor-pointer"
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            e.preventDefault()
            const droppedFile = e.dataTransfer.files[0]
            if (droppedFile) setFile(droppedFile)
          }}
        >
          <input
            ref={fileInputRef}
            type="file"
            id={fileInputID}
            accept=".md,.txt,.markdown,.zip,.tar.gz"
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
        <div className="form-control">
          <label className="label py-1" htmlFor="project-plan-dir-path">
            <span className="label-text">Directory path</span>
          </label>
          <input
            id="project-plan-dir-path"
            type="text"
            value={dirPath}
            onChange={(e) => setDirPath(e.target.value)}
            placeholder="./path/to/directory"
            className="input input-bordered w-full font-mono text-sm"
          />
          <label className="label py-1">
            <span className="label-text-alt text-base-content/60">
              Reads ALL files in directory. Best for small codebases (&lt;50 files).
            </span>
          </label>
        </div>
      )}

      {sourceType === 'research' && (
        <div className="form-control">
          <label className="label py-1" htmlFor="project-plan-research-path">
            <span className="label-text">Research path</span>
          </label>
          <input
            id="project-plan-research-path"
            type="text"
            value={researchPath}
            onChange={(e) => setResearchPath(e.target.value)}
            placeholder="./path/to/docs"
            className="input input-bordered w-full font-mono text-sm"
          />
          <label className="label py-1">
            <span className="label-text-alt text-base-content/60">
              AI-guided research mode. Provides file manifest, AI selectively explores files.
            </span>
          </label>
        </div>
      )}

      {sourceType === 'provider' && (
        <div className="grid grid-cols-2 gap-3">
          <div className="form-control">
            <label className="label py-1">
              <span className="label-text">Provider</span>
            </label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="select select-bordered w-full"
            >
              {TASK_SOURCE_PROVIDERS.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </div>
          <div className="form-control">
            <label className="label py-1">
              <span className="label-text">Reference</span>
            </label>
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

      <div className="form-control">
        <label className="label py-1">
          <span className="label-text">Project Title (optional)</span>
        </label>
        <input
          type="text"
          value={planTitle}
          onChange={(e) => setPlanTitle(e.target.value)}
          placeholder="My Project"
          className="input input-bordered w-full"
        />
      </div>

      <div className="form-control">
        <label className="label py-1">
          <span className="label-text">Instructions (optional)</span>
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
          className="checkbox checkbox-primary"
        />
        <span className="text-sm text-base-content/80">Use schema-driven extraction</span>
      </label>

      {error && <div className="alert alert-error text-sm">{error}</div>}
      {success && <div className="alert alert-success text-sm">{success}</div>}

      <button type="submit" className="btn btn-primary w-full" disabled={isPending}>
        {isPending && <Loader2 size={16} className="animate-spin mr-2" />}
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
