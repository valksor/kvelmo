import { useState } from 'react'
import { useProjectStore } from '../stores/projectStore'
import { FilePicker } from './FilePicker'

interface TaskWidgetProps {
  embedded?: boolean
}

export function TaskWidget({ embedded = false }: TaskWidgetProps) {
  const { task, state, start, loading, error, connected, connecting } = useProjectStore()
  const [source, setSource] = useState('')
  const [showFilePicker, setShowFilePicker] = useState(false)

  const handleLoad = () => {
    if (source.trim()) {
      start(source.trim())
      setSource('')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && source.trim() && connected && !loading) {
      handleLoad()
    }
  }

  const handleFileSelect = (path: string) => {
    setSource(`file:${path}`)
  }

  // Content when no task is loaded
  const loadTaskContent = (
    <>
      <div className="flex gap-2">
        {/* File picker button */}
        <button
          onClick={() => setShowFilePicker(true)}
          disabled={loading}
          className="btn btn-ghost btn-square btn-sm"
          aria-label="Browse for task file"
        >
          <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        </button>

        {/* Source input */}
        <input
          type="text"
          value={source}
          onChange={e => setSource(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="file:task.md, github:owner/repo#123"
          className="input input-bordered flex-1 font-mono text-sm input-sm"
          disabled={loading}
        />

        {/* Load button */}
        <button
          onClick={handleLoad}
          disabled={loading || !source.trim() || !connected}
          className="btn btn-primary btn-sm"
        >
          {loading ? (
            <span className="loading loading-spinner loading-xs"></span>
          ) : (
            'Load'
          )}
        </button>
      </div>

      {/* Connection status indicator - use single expression to avoid overlap */}
      <p className={`text-sm mt-2 ${
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

      <div className="flex gap-1.5 flex-wrap items-center mt-3">
        <span className="text-xs text-base-content/60">Examples:</span>
        {[
          'file:./tasks/feature.md',
          'github:owner/repo#123',
        ].map(example => (
          <button
            key={example}
            onClick={() => setSource(example)}
            className="text-xs font-mono px-1.5 py-0.5 rounded bg-neutral text-neutral-content hover:bg-neutral-focus transition-colors"
          >
            {example}
          </button>
        ))}
      </div>

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
