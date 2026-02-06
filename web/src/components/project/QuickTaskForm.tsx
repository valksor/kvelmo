import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useCreateQuickTask, useSubmitSource } from '@/api/quick'
import { TASK_SOURCE_PROVIDERS } from '@/constants/taskOptions'
import { ErrorMessage } from '@/components/project/TaskFormShared'

type QuickMode = 'simple' | 'source'

export function QuickTaskForm() {
  const [mode, setMode] = useState<QuickMode>('simple')

  const [description, setDescription] = useState('')
  const [title, setTitle] = useState('')
  const [priority, setPriority] = useState(2)
  const [labels, setLabels] = useState('')

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
            .map((label) => label.trim())
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
            .map((note) => note.trim())
            .filter(Boolean),
          instructions: instructions || undefined,
          labels: labels
            .split(',')
            .map((label) => label.trim())
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
          <div className="form-control">
            <label className="label py-1">
              <span className="label-text">
                Description <span className="text-error">*</span>
              </span>
            </label>
            <textarea
              rows={4}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe the task...&#10;&#10;e.g., Fix typo in README.md line 42"
              className="textarea textarea-bordered w-full"
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Title (optional)</span>
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Auto-extracted"
                className="input input-bordered w-full"
              />
            </div>
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Priority</span>
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
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Labels</span>
              </label>
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
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">
                  Source <span className="text-error">*</span>
                </span>
              </label>
              <input
                type="text"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                placeholder="./docs or file:requirements.md"
                className="input input-bordered w-full font-mono text-sm"
              />
            </div>
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">
                  Provider <span className="text-error">*</span>
                </span>
              </label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="select select-bordered w-full"
              >
                {TASK_SOURCE_PROVIDERS.map((sourceProvider) => (
                  <option key={sourceProvider.value} value={sourceProvider.value}>
                    {sourceProvider.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Title (optional)</span>
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Override generated title"
                className="input input-bordered w-full"
              />
            </div>
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Labels</span>
              </label>
              <input
                type="text"
                value={labels}
                onChange={(e) => setLabels(e.target.value)}
                placeholder="bug, urgent"
                className="input input-bordered w-full"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Notes (one per line)</span>
              </label>
              <textarea
                rows={3}
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Add constraints or guidance"
                className="textarea textarea-bordered w-full"
              />
            </div>
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text">Instructions</span>
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

          <div className="flex gap-6">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={optimize}
                onChange={(e) => setOptimize(e.target.checked)}
                className="checkbox checkbox-primary"
              />
              <span className="text-sm">Optimize with AI</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={(e) => setDryRun(e.target.checked)}
                className="checkbox checkbox-primary"
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
