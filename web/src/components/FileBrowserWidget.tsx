import { useEffect, useState } from 'react'
import { useProjectStore, type BrowseEntry } from '../stores/projectStore'
import { useLayoutStore } from '../stores/layoutStore'

export function FileBrowserWidget() {
  const { browseFiles, connected, task } = useProjectStore()
  const { openTab, setActiveTab } = useLayoutStore()

  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined)
  const [entries, setEntries] = useState<BrowseEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [pathHistory, setPathHistory] = useState<Array<string | undefined>>([undefined])

  const loadEntries = async (path?: string) => {
    if (!connected) return
    setLoading(true)
    setError(null)
    try {
      const result = await browseFiles(path)
      setEntries(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load directory')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (connected) {
      loadEntries(currentPath)
    }
  }, [connected])

  const handleEntryClick = async (entry: BrowseEntry) => {
    if (entry.is_dir) {
      setPathHistory(prev => [...prev, entry.path])
      setCurrentPath(entry.path)
      await loadEntries(entry.path)
    } else {
      // Open file in a tab
      const tabId = `file-${entry.path}`
      openTab({
        id: tabId,
        type: 'file',
        title: entry.name,
        closeable: true,
        data: { path: entry.path }
      })
      setActiveTab(tabId)
    }
  }

  const handleNavigateUp = async () => {
    if (pathHistory.length <= 1) return
    const newHistory = pathHistory.slice(0, -1)
    const parentPath = newHistory[newHistory.length - 1]
    setPathHistory(newHistory)
    setCurrentPath(parentPath)
    await loadEntries(parentPath)
  }

  const handleRefresh = () => {
    loadEntries(currentPath)
  }

  const formatBytes = (bytes?: number) => {
    if (!bytes) return ''
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
  }

  // Build breadcrumb segments
  const breadcrumbPath = currentPath || (task?.worktreePath ? task.worktreePath : '/')
  const segments = breadcrumbPath.replace(/^\//, '').split('/').filter(Boolean)

  if (!connected) {
    return (
      <div className="flex items-center justify-center h-full text-base-content/50">
        <p className="text-sm">Not connected</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header / breadcrumb */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-base-300 bg-base-200/50">
        <button
          onClick={handleNavigateUp}
          disabled={pathHistory.length <= 1}
          className="btn btn-ghost btn-xs btn-square"
          aria-label="Go to parent directory"
        >
          <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <div className="flex-1 min-w-0 flex items-center gap-1 text-xs text-base-content/60 font-mono overflow-hidden">
          <button
            onClick={() => { setPathHistory([undefined]); setCurrentPath(undefined); loadEntries(undefined) }}
            className="hover:text-base-content transition-colors flex-shrink-0"
          >
            /
          </button>
          {segments.map((seg, i) => {
            const segPath = '/' + segments.slice(0, i + 1).join('/')
            const isCurrent = i === segments.length - 1
            return (
              <span key={i} className="flex items-center gap-1 min-w-0">
                <span className="text-base-content/30" aria-hidden="true">/</span>
                <button
                  onClick={async () => {
                    const newHistory = pathHistory.slice(0, i + 2)
                    setPathHistory(newHistory)
                    setCurrentPath(segPath)
                    await loadEntries(segPath)
                  }}
                  className="hover:text-base-content transition-colors truncate max-w-[80px]"
                  aria-current={isCurrent ? 'page' : undefined}
                >
                  {seg}
                </button>
              </span>
            )
          })}
        </div>

        <button
          onClick={handleRefresh}
          disabled={loading}
          className="btn btn-ghost btn-xs btn-square"
          aria-label="Refresh current directory"
        >
          {loading ? (
            <span className="loading loading-spinner loading-xs" aria-hidden="true"></span>
          ) : (
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          )}
        </button>
      </div>

      {/* Entry list */}
      <div className="flex-1 overflow-auto">
        {error ? (
          <div className="flex flex-col items-center justify-center h-full text-error py-8">
            <svg aria-hidden="true" className="w-8 h-8 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <p className="text-sm">{error}</p>
          </div>
        ) : entries.length === 0 && !loading ? (
          <div className="flex flex-col items-center justify-center h-full text-base-content/40 py-8">
            <svg aria-hidden="true" className="w-10 h-10 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
            <p className="text-sm">Empty directory</p>
          </div>
        ) : (
          <div>
            {/* Directories first, then files */}
            {[...entries.filter(e => e.is_dir), ...entries.filter(e => !e.is_dir)].map((entry) => (
              <button
                key={entry.path}
                className="w-full flex items-center gap-2.5 px-3 py-1.5 hover:bg-base-200 transition-colors text-left group"
                onClick={() => handleEntryClick(entry)}
              >
                {/* Icon */}
                <span className="flex-shrink-0 w-4 h-4 text-base-content/50 group-hover:text-base-content/70 transition-colors">
                  {entry.is_dir ? (
                    <svg aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-4 h-4 text-warning/70">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    </svg>
                  ) : (
                    <svg aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-4 h-4">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                  )}
                </span>

                {/* Name */}
                <span className="flex-1 min-w-0 text-sm text-base-content/80 truncate group-hover:text-base-content transition-colors">
                  {entry.name}
                </span>

                {/* Size (files only) */}
                {!entry.is_dir && entry.size !== undefined && (
                  <span className="flex-shrink-0 text-xs text-base-content/40">
                    {formatBytes(entry.size)}
                  </span>
                )}

                {/* Arrow for dirs */}
                {entry.is_dir && (
                  <svg aria-hidden="true" className="w-3 h-3 text-base-content/30 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                )}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
