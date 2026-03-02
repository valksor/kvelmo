import { useState } from 'react'
import { useProjectStore } from '../stores/projectStore'
import { FilePicker } from './FilePicker'

interface TaskWidgetProps {
  embedded?: boolean
}

export function TaskWidget({ embedded = false }: TaskWidgetProps) {
  const { task, state, start, loading, error, connected, connecting } = useProjectStore()
  const [inputMode, setInputMode] = useState<'quick' | 'file' | 'url'>('quick')
  const [taskDescription, setTaskDescription] = useState('')
  const [urlInput, setUrlInput] = useState('')
  const [selectedFile, setSelectedFile] = useState('')
  const [showFilePicker, setShowFilePicker] = useState(false)

  const handleQuickLoad = () => {
    if (taskDescription.trim()) {
      start(`empty:${taskDescription.trim()}`)
      setTaskDescription('')
    }
  }

  const handleUrlLoad = () => {
    if (urlInput.trim()) {
      start(urlInput.trim())
      setUrlInput('')
    }
  }

  const handleFileLoad = () => {
    if (selectedFile) {
      start(`file:${selectedFile}`)
      setSelectedFile('')
    }
  }

  const handleFileSelect = (path: string) => {
    setSelectedFile(path)
  }

  const handleQuickKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.ctrlKey && taskDescription.trim() && connected && !loading) {
      handleQuickLoad()
    }
  }

  const handleUrlKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && urlInput.trim() && connected && !loading) {
      handleUrlLoad()
    }
  }

  // Content when no task is loaded
  const loadTaskContent = (
    <>
      {/* Tab selector */}
      <div role="tablist" className="tabs tabs-boxed tabs-sm mb-3 bg-base-300 gap-1 p-1">
        <button
          role="tab"
          className={`tab rounded ${inputMode === 'quick' ? 'tab-active' : 'border border-base-content/10'}`}
          onClick={() => setInputMode('quick')}
        >
          Quick Task
        </button>
        <button
          role="tab"
          className={`tab rounded ${inputMode === 'file' ? 'tab-active' : 'border border-base-content/10'}`}
          onClick={() => setInputMode('file')}
        >
          From File
        </button>
        <button
          role="tab"
          className={`tab rounded ${inputMode === 'url' ? 'tab-active' : 'border border-base-content/10'}`}
          onClick={() => setInputMode('url')}
        >
          From URL
        </button>
      </div>

      {/* Quick Task tab */}
      {inputMode === 'quick' && (
        <div className="space-y-2">
          <textarea
            value={taskDescription}
            onChange={e => setTaskDescription(e.target.value)}
            onKeyDown={handleQuickKeyDown}
            placeholder="Describe what you want to work on..."
            className="textarea textarea-bordered w-full text-sm resize-none"
            rows={3}
            disabled={loading}
          />
          <div className="flex justify-between items-center">
            <span className="text-xs text-base-content/50">Ctrl+Enter to load</span>
            <button
              onClick={handleQuickLoad}
              disabled={loading || !taskDescription.trim() || !connected}
              className="btn btn-primary btn-sm"
            >
              {loading ? <span className="loading loading-spinner loading-xs" /> : 'Load Task'}
            </button>
          </div>
        </div>
      )}

      {/* From File tab */}
      {inputMode === 'file' && (
        <div className="space-y-2">
          <div className="flex gap-2 items-center">
            <button
              onClick={() => setShowFilePicker(true)}
              disabled={loading || !connected}
              className="btn btn-outline btn-sm gap-2"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              Browse
            </button>
            {selectedFile ? (
              <code className="flex-1 text-sm bg-base-300 px-2 py-1 rounded truncate">{selectedFile}</code>
            ) : (
              <span className="flex-1 text-sm text-base-content/50">No file selected</span>
            )}
          </div>
          {selectedFile && (
            <div className="flex justify-end">
              <button
                onClick={handleFileLoad}
                disabled={loading || !connected}
                className="btn btn-primary btn-sm"
              >
                {loading ? <span className="loading loading-spinner loading-xs" /> : 'Load Task'}
              </button>
            </div>
          )}
        </div>
      )}

      {/* From URL tab */}
      {inputMode === 'url' && (
        <div className="space-y-2">
          <input
            type="text"
            value={urlInput}
            onChange={e => setUrlInput(e.target.value)}
            onKeyDown={handleUrlKeyDown}
            placeholder="github.com/owner/repo/issues/123"
            className="input input-bordered w-full font-mono text-sm input-sm"
            disabled={loading}
          />
          <div className="flex justify-between items-center">
            <span className="text-xs text-base-content/50">GitHub, GitLab, or Linear URLs</span>
            <button
              onClick={handleUrlLoad}
              disabled={loading || !urlInput.trim() || !connected}
              className="btn btn-primary btn-sm"
            >
              {loading ? <span className="loading loading-spinner loading-xs" /> : 'Load Task'}
            </button>
          </div>
        </div>
      )}

      {/* Connection status */}
      <p className={`text-sm mt-3 ${
        connecting ? 'text-warning' :
        connected ? 'text-success' :
        'text-base-content/50'
      }`} data-testid="task-connection-status">
        {connecting ? 'Connecting to worktree...' :
         connected ? 'Connected' :
         'Not connected'}
      </p>

      {error && (
        <p className="text-sm text-error bg-error/10 px-3 py-2 rounded-lg border border-error/20 mt-2">{error}</p>
      )}

      {/* File Picker Modal */}
      <FilePicker
        isOpen={showFilePicker}
        onClose={() => setShowFilePicker(false)}
        onSelect={handleFileSelect}
      />
    </>
  )

  // Content when task is loaded
  const taskContent = task && (
    <>
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-base-content truncate">{task.title}</h3>
          <p className="text-xs text-base-content/60 font-mono truncate">{task.source}</p>
        </div>
        <span className={`flex-shrink-0 badge badge-sm ${
          state === 'implemented' ? 'badge-success' :
          state === 'planned' ? 'badge-primary' :
          state === 'planning' || state === 'implementing' ? 'badge-warning' :
          state === 'submitted' ? 'badge-secondary' :
          state === 'failed' ? 'badge-error' :
          'badge-ghost'
        }`}>
          {state}
        </span>
      </div>

      {task.description && (
        <p className="text-sm text-base-content/80 leading-relaxed mt-3 line-clamp-3">{task.description}</p>
      )}
      {task.branch && (
        <div className="flex items-center gap-2 text-xs mt-3">
          <svg aria-hidden="true" className="w-3 h-3 text-base-content/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
          </svg>
          <span className="text-base-content/60">Branch:</span>
          <code className="font-mono text-primary bg-neutral px-1.5 py-0.5 rounded text-xs">{task.branch}</code>
          {task.worktreePath && (
            <span className="badge badge-xs badge-info gap-1" aria-label={`Isolated worktree: ${task.worktreePath}`}>
              <svg aria-hidden="true" className="w-2.5 h-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7v8a2 2 0 002 2h6M8 7V5a2 2 0 012-2h4.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V15a2 2 0 01-2 2h-2M8 7H6a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2v-2" />
              </svg>
              isolated
            </span>
          )}
        </div>
      )}
      {error && (
        <p className="text-sm text-error bg-error/10 px-3 py-2 rounded-lg border border-error/20 mt-3">{error}</p>
      )}
    </>
  )

  const content = task ? taskContent : loadTaskContent

  if (embedded) {
    return <div>{content}</div>
  }

  return (
    <section className="card bg-base-200">
      <div className="card-body">
        <h2 className="card-title text-base-content flex items-center gap-2 mb-4">
          <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
          {task ? 'Current Task' : 'Load Task'}
        </h2>
        {content}
      </div>
    </section>
  )
}
