import { useState } from 'react'
import {
  Loader2,
  AlertCircle,
  Zap,
  Plus,
  Sparkles,
  Trash2,
  Play,
  FileOutput,
  Send,
  ChevronDown,
  ChevronUp,
  StickyNote,
  ExternalLink,
} from 'lucide-react'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import {
  useQuickTasks,
  useCreateQuickTask,
  useOptimizeQuickTask,
  useStartQuickTask,
  useDeleteQuickTask,
  useSubmitQuickTask,
  useAddQuickTaskNote,
  useExportQuickTask,
  useSubmitSource,
  type QuickTask,
} from '@/api/quick'

export default function Quick() {
  const { data: status, isLoading: statusLoading } = useStatus()

  // Form state for new task
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState(1)
  const [labels, setLabels] = useState('')

  // Form state for source import
  const [showSourceForm, setShowSourceForm] = useState(false)
  const [sourceProvider, setSourceProvider] = useState('github')
  const [sourceRef, setSourceRef] = useState('')
  const [sourceNotes, setSourceNotes] = useState('')
  const [sourceOptimize, setSourceOptimize] = useState(true)

  // Expanded task for notes
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null)
  const [newNote, setNewNote] = useState('')

  // Submit modal
  const [submitTaskId, setSubmitTaskId] = useState<string | null>(null)
  const [submitProvider, setSubmitProvider] = useState('github')
  const [submitDryRun, setSubmitDryRun] = useState(false)

  // Queries and mutations
  const { data: tasksData, isLoading: tasksLoading, error: tasksError } = useQuickTasks()
  const createMutation = useCreateQuickTask()
  const optimizeMutation = useOptimizeQuickTask()
  const startMutation = useStartQuickTask()
  const deleteMutation = useDeleteQuickTask()
  const submitMutation = useSubmitQuickTask()
  const addNoteMutation = useAddQuickTaskNote()
  const exportMutation = useExportQuickTask()
  const submitSourceMutation = useSubmitSource()

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
        <h1 className="text-2xl font-bold">Quick Tasks</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!description.trim()) return

    await createMutation.mutateAsync({
      title: title.trim() || undefined,
      description: description.trim(),
      priority,
      labels: labels
        .split(',')
        .map((l) => l.trim())
        .filter(Boolean),
    })

    // Reset form
    setTitle('')
    setDescription('')
    setPriority(1)
    setLabels('')
  }

  const handleSourceSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!sourceRef.trim()) return

    await submitSourceMutation.mutateAsync({
      source: sourceRef.trim(),
      provider: sourceProvider,
      notes: sourceNotes ? [sourceNotes] : undefined,
      optimize: sourceOptimize,
    })

    // Reset form
    setSourceRef('')
    setSourceNotes('')
    setShowSourceForm(false)
  }

  const handleOptimize = async (taskId: string) => {
    await optimizeMutation.mutateAsync({ taskId })
  }

  const handleStart = async (taskId: string) => {
    await startMutation.mutateAsync(taskId)
  }

  const handleDelete = async (taskId: string) => {
    if (!confirm('Delete this task?')) return
    await deleteMutation.mutateAsync(taskId)
  }

  const handleExport = async (taskId: string) => {
    const result = await exportMutation.mutateAsync({ taskId })
    if ('blob' in result && result.blob) {
      // Download the markdown file
      const url = URL.createObjectURL(result.blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${taskId}.md`
      a.click()
      URL.revokeObjectURL(url)
    }
  }

  const handleSubmit = async () => {
    if (!submitTaskId) return
    await submitMutation.mutateAsync({
      taskId: submitTaskId,
      provider: submitProvider,
      dry_run: submitDryRun,
    })
    setSubmitTaskId(null)
  }

  const handleAddNote = async (taskId: string) => {
    if (!newNote.trim()) return
    await addNoteMutation.mutateAsync({ taskId, note: newNote.trim() })
    setNewNote('')
  }

  const getPriorityBadge = (p: number) => {
    switch (p) {
      case 0:
        return <span className="badge badge-ghost badge-sm">Low</span>
      case 2:
        return <span className="badge badge-warning badge-sm">High</span>
      default:
        return <span className="badge badge-info badge-sm">Normal</span>
    }
  }

  const tasks = tasksData?.tasks || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Quick Tasks</h1>
        <p className="text-base-content/60 mt-1">
          Capture ideas quickly, optimize with AI, then submit or start immediately
        </p>
      </div>

      {/* Create Form */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="font-medium mb-4 flex items-center gap-2">
            <Plus size={18} />
            New Quick Task
          </h3>

          <form onSubmit={handleCreate} className="space-y-4">
            <div className="form-control">
              <label className="label">
                <span className="label-text">Title (optional)</span>
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Brief summary"
                className="input input-bordered"
              />
            </div>

            <div className="form-control">
              <label className="label">
                <span className="label-text">Description *</span>
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What needs to be done?"
                className="textarea textarea-bordered h-24"
                required
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text">Priority</span>
                </label>
                <select
                  value={priority}
                  onChange={(e) => setPriority(Number(e.target.value))}
                  className="select select-bordered"
                >
                  <option value={0}>Low</option>
                  <option value={1}>Normal</option>
                  <option value={2}>High</option>
                </select>
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text">Labels (comma-separated)</span>
                </label>
                <input
                  type="text"
                  value={labels}
                  onChange={(e) => setLabels(e.target.value)}
                  placeholder="bug, frontend, urgent"
                  className="input input-bordered"
                />
              </div>
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              disabled={!description.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Plus size={18} />
              )}
              Create Task
            </button>
          </form>

          {createMutation.isError && (
            <div className="alert alert-error mt-4">
              <AlertCircle size={18} />
              <span>{createMutation.error.message}</span>
            </div>
          )}
        </div>
      </div>

      {/* Source Import (Collapsible) */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <button
            type="button"
            className="flex items-center justify-between w-full text-left"
            onClick={() => setShowSourceForm(!showSourceForm)}
          >
            <h3 className="font-medium flex items-center gap-2">
              <ExternalLink size={18} />
              Import from External Source
            </h3>
            {showSourceForm ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
          </button>

          {showSourceForm && (
            <form onSubmit={handleSourceSubmit} className="space-y-4 mt-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="form-control">
                  <label className="label">
                    <span className="label-text">Provider</span>
                  </label>
                  <select
                    value={sourceProvider}
                    onChange={(e) => setSourceProvider(e.target.value)}
                    className="select select-bordered"
                  >
                    <option value="github">GitHub</option>
                    <option value="gitlab">GitLab</option>
                    <option value="jira">Jira</option>
                    <option value="linear">Linear</option>
                    <option value="asana">Asana</option>
                    <option value="notion">Notion</option>
                    <option value="trello">Trello</option>
                  </select>
                </div>

                <div className="form-control">
                  <label className="label">
                    <span className="label-text">Reference (URL or ID)</span>
                  </label>
                  <input
                    type="text"
                    value={sourceRef}
                    onChange={(e) => setSourceRef(e.target.value)}
                    placeholder="https://github.com/org/repo/issues/123"
                    className="input input-bordered"
                    required
                  />
                </div>
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text">Notes (optional)</span>
                </label>
                <textarea
                  value={sourceNotes}
                  onChange={(e) => setSourceNotes(e.target.value)}
                  placeholder="Additional context or instructions"
                  className="textarea textarea-bordered h-20"
                />
              </div>

              <div className="form-control">
                <label className="label cursor-pointer justify-start gap-3">
                  <input
                    type="checkbox"
                    checked={sourceOptimize}
                    onChange={(e) => setSourceOptimize(e.target.checked)}
                    className="checkbox checkbox-sm"
                  />
                  <span className="label-text">Optimize with AI after import</span>
                </label>
              </div>

              <button
                type="submit"
                className="btn btn-secondary"
                disabled={!sourceRef.trim() || submitSourceMutation.isPending}
              >
                {submitSourceMutation.isPending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <ExternalLink size={18} />
                )}
                Import & Submit
              </button>

              {submitSourceMutation.isSuccess && (
                <div className="alert alert-success mt-2">
                  <span>
                    Task imported!{' '}
                    {submitSourceMutation.data.external_url && (
                      <a
                        href={submitSourceMutation.data.external_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline"
                      >
                        View →
                      </a>
                    )}
                  </span>
                </div>
              )}

              {submitSourceMutation.isError && (
                <div className="alert alert-error mt-2">
                  <AlertCircle size={18} />
                  <span>{submitSourceMutation.error.message}</span>
                </div>
              )}
            </form>
          )}
        </div>
      </div>

      {/* Tasks List */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="font-medium mb-4 flex items-center gap-2">
            <Zap size={18} />
            Quick Tasks ({tasks.length})
          </h3>

          {tasksLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
          ) : tasksError ? (
            <div className="alert alert-error">
              <AlertCircle size={18} />
              <span>{tasksError instanceof Error ? tasksError.message : 'Failed to load tasks'}</span>
            </div>
          ) : tasks.length === 0 ? (
            <div className="text-center py-12">
              <Zap size={48} className="mx-auto text-base-content/30 mb-4" />
              <p className="text-base-content/60">No quick tasks yet</p>
              <p className="text-sm text-base-content/40 mt-1">
                Create one above to get started
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {tasks.map((task: QuickTask) => (
                <div
                  key={task.id}
                  className="border border-base-300 rounded-lg p-4 hover:bg-base-200/50 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <h4 className="font-medium truncate">{task.title}</h4>
                        {getPriorityBadge(task.priority)}
                        <span className="badge badge-ghost badge-sm">{task.status}</span>
                      </div>
                      {task.labels.length > 0 && (
                        <div className="flex gap-1 mt-2 flex-wrap">
                          {task.labels.map((label) => (
                            <span
                              key={label}
                              className="badge badge-outline badge-sm"
                            >
                              {label}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>

                    <div className="flex gap-1 flex-shrink-0">
                      <button
                        className="btn btn-ghost btn-xs"
                        onClick={() => handleOptimize(task.id)}
                        disabled={optimizeMutation.isPending}
                        title="Optimize with AI"
                      >
                        {optimizeMutation.isPending ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Sparkles size={14} />
                        )}
                      </button>
                      <button
                        className="btn btn-ghost btn-xs"
                        onClick={() => handleExport(task.id)}
                        disabled={exportMutation.isPending}
                        title="Export to markdown"
                      >
                        <FileOutput size={14} />
                      </button>
                      <button
                        className="btn btn-ghost btn-xs"
                        onClick={() => setSubmitTaskId(task.id)}
                        title="Submit to provider"
                      >
                        <Send size={14} />
                      </button>
                      <button
                        className="btn btn-ghost btn-xs text-success"
                        onClick={() => handleStart(task.id)}
                        disabled={startMutation.isPending}
                        title="Start working"
                      >
                        <Play size={14} />
                      </button>
                      <button
                        className="btn btn-ghost btn-xs text-error"
                        onClick={() => handleDelete(task.id)}
                        disabled={deleteMutation.isPending}
                        title="Delete"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>

                  {/* Notes section */}
                  <div className="mt-3 pt-3 border-t border-base-300">
                    <button
                      type="button"
                      className="text-sm text-base-content/60 flex items-center gap-1"
                      onClick={() =>
                        setExpandedTaskId(expandedTaskId === task.id ? null : task.id)
                      }
                    >
                      <StickyNote size={14} />
                      Notes ({task.note_count})
                      {expandedTaskId === task.id ? (
                        <ChevronUp size={14} />
                      ) : (
                        <ChevronDown size={14} />
                      )}
                    </button>

                    {expandedTaskId === task.id && (
                      <div className="mt-2 space-y-2">
                        <div className="flex gap-2">
                          <input
                            type="text"
                            value={newNote}
                            onChange={(e) => setNewNote(e.target.value)}
                            placeholder="Add a note..."
                            className="input input-bordered input-sm flex-1"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                handleAddNote(task.id)
                              }
                            }}
                          />
                          <button
                            className="btn btn-sm btn-primary"
                            onClick={() => handleAddNote(task.id)}
                            disabled={!newNote.trim() || addNoteMutation.isPending}
                          >
                            Add
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Submit Modal */}
      {submitTaskId && (
        <div className="modal modal-open">
          <div className="modal-box">
            <h3 className="font-bold text-lg">Submit Task</h3>
            <p className="py-4 text-base-content/60">
              Submit this task to an external provider.
            </p>

            <div className="form-control">
              <label className="label">
                <span className="label-text">Provider</span>
              </label>
              <select
                value={submitProvider}
                onChange={(e) => setSubmitProvider(e.target.value)}
                className="select select-bordered"
              >
                <option value="github">GitHub</option>
                <option value="gitlab">GitLab</option>
                <option value="jira">Jira</option>
                <option value="linear">Linear</option>
              </select>
            </div>

            <div className="form-control mt-4">
              <label className="label cursor-pointer justify-start gap-3">
                <input
                  type="checkbox"
                  checked={submitDryRun}
                  onChange={(e) => setSubmitDryRun(e.target.checked)}
                  className="checkbox checkbox-sm"
                />
                <span className="label-text">Dry run (preview only)</span>
              </label>
            </div>

            {submitMutation.isSuccess && (
              <div className="alert alert-success mt-4">
                <span>
                  Submitted!{' '}
                  {submitMutation.data.external_url && (
                    <a
                      href={submitMutation.data.external_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="underline"
                    >
                      View →
                    </a>
                  )}
                </span>
              </div>
            )}

            {submitMutation.isError && (
              <div className="alert alert-error mt-4">
                <AlertCircle size={18} />
                <span>{submitMutation.error.message}</span>
              </div>
            )}

            <div className="modal-action">
              <button className="btn btn-ghost" onClick={() => setSubmitTaskId(null)}>
                Cancel
              </button>
              <button
                className="btn btn-primary"
                onClick={handleSubmit}
                disabled={submitMutation.isPending}
              >
                {submitMutation.isPending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  'Submit'
                )}
              </button>
            </div>
          </div>
          <div className="modal-backdrop" onClick={() => setSubmitTaskId(null)} />
        </div>
      )}

      {/* Global mutation errors */}
      {optimizeMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>Optimization failed: {optimizeMutation.error.message}</span>
        </div>
      )}

      {startMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>Failed to start: {startMutation.error.message}</span>
        </div>
      )}
    </div>
  )
}
