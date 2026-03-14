import { lazy, Suspense, useState, useMemo } from 'react'
import { useGlobalStore } from '../stores/globalStore'
import { useDocsURL } from '../hooks/useDocsURL'
import { FolderPicker } from './FolderPicker'
import { ThemeToggle } from './ThemeToggle'
import { ActiveTasksWidget } from './ActiveTasksWidget'
import { MetricsWidget } from './MetricsWidget'
import { Onboarding } from './Onboarding'
import { name } from '../meta'

// Lazy-loaded modal components
const Settings = lazy(() => import('./Settings').then(m => ({ default: m.Settings })))
const MemoryPanel = lazy(() => import('./MemoryPanel').then(m => ({ default: m.MemoryPanel })))

export function GlobalView() {
  const {
    projects,
    loading,
    error,
    connected,
    connecting,
    reconnectAttempt,
    selectedProject,
    agentStatus,
    activeTasks,
    connect,
    loadProjects,
    addProject,
    removeProject,
    selectProject
  } = useGlobalStore()

  // Build a lookup of task info by project path for enriching project cards
  const taskByPath = useMemo(() => {
    const map = new Map<string, typeof activeTasks[0]>()
    for (const t of activeTasks) {
      if (t.state !== 'none' && t.state !== 'submitted') {
        map.set(t.path, t)
      }
    }
    return map
  }, [activeTasks])

  const [showFolderPicker, setShowFolderPicker] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [showMemory, setShowMemory] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const docsData = useDocsURL()

  // Filter projects by search query (name or path)
  const filteredProjects = useMemo(() => {
    if (!searchQuery.trim()) return projects
    const q = searchQuery.toLowerCase()
    return projects.filter(p => {
      const name = p.path.split('/').pop()?.toLowerCase() ?? ''
      return name.includes(q) || p.path.toLowerCase().includes(q)
    })
  }, [projects, searchQuery])

  const handleFolderSelect = async (path: string) => {
    await addProject(path)
  }

  const handleRemoveProject = async (e: React.MouseEvent, projectId: string) => {
    e.stopPropagation()
    if (window.confirm('Remove this project from the list?')) {
      await removeProject(projectId)
    }
  }

  return (
    <div className="min-h-screen p-4 sm:p-6 lg:p-8 bg-base-100">
      {/* Header */}
      <header className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6 sm:mb-8">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 sm:w-10 sm:h-10 rounded-xl bg-primary flex items-center justify-center shadow-lg" aria-hidden="true">
            <svg className="w-4 h-4 sm:w-5 sm:h-5 text-primary-content" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <h1 className="text-xl sm:text-2xl font-bold text-base-content">{name}</h1>
        </div>

        <div className="flex items-center gap-2 sm:gap-3 flex-wrap">
          {/* Connection status */}
          {!connected && !connecting && reconnectAttempt === 0 && (
            <button
              onClick={() => connect()}
              className="btn btn-warning btn-sm"
              aria-label="Reconnect"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
              </svg>
              <span className="hidden sm:inline">Reconnect</span>
            </button>
          )}

          {/* Auto-reconnecting indicator */}
          {!connected && !connecting && reconnectAttempt > 0 && (
            <span className="text-sm text-warning flex items-center gap-2 bg-warning/10 px-3 py-1.5 rounded-lg" role="status" aria-live="polite">
              <span className="loading loading-spinner loading-xs" aria-hidden="true"></span>
              <span>Reconnecting (#{reconnectAttempt})...</span>
            </span>
          )}

          {connecting && (
            <span className="text-sm text-warning flex items-center gap-2" role="status" aria-live="polite">
              <span className="loading loading-spinner loading-xs" aria-hidden="true"></span>
              <span className="hidden sm:inline">Connecting...</span>
            </span>
          )}

          {error && (
            <span role="alert" title={error} className="text-xs sm:text-sm text-error bg-error/10 px-2 sm:px-3 py-1 sm:py-1.5 rounded-lg border border-error/20 max-w-[200px] sm:max-w-none whitespace-normal break-words">
              {error}
            </span>
          )}

          {/* Memory search button */}
          <button
            onClick={() => setShowMemory(true)}
            disabled={!connected}
            className="btn btn-ghost btn-sm btn-square"
            aria-label="Memory Search"
          >
            <svg aria-hidden="true" className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
            </svg>
          </button>

          {/* Documentation link */}
          {docsData?.url && (
            <a
              href={docsData.url}
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost btn-sm btn-square"
              aria-label="Documentation"
            >
              <svg aria-hidden="true" className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </a>
          )}

          <button
            onClick={() => setShowSettings(true)}
            className="btn btn-ghost btn-sm btn-square"
            aria-label="Settings"
          >
            <svg aria-hidden="true" className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>

          <ThemeToggle />

          <button
            onClick={() => loadProjects()}
            disabled={loading || !connected}
            className="btn btn-ghost btn-sm sm:btn-md"
            aria-label="Refresh projects"
          >
            {loading ? (
              <span className="loading loading-spinner loading-sm"></span>
            ) : (
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            )}
            <span className="hidden sm:inline">Refresh</span>
          </button>
        </div>
      </header>

      {/* Agent Status Warning */}
      {connected && agentStatus && !agentStatus.agent_available && (
        <div role="alert" className="alert alert-warning max-w-2xl mx-auto mb-4">
          <svg aria-hidden="true" className="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <p className="font-medium">
              {agentStatus.simulation_mode
                ? 'Running in simulation mode — no AI agent connected'
                : 'No AI agent available'}
            </p>
            <p className="text-sm opacity-80">
              {agentStatus.checks
                .filter(c => c.status === 'failed' && c.fix)
                .map(c => c.fix)
                .join(' · ') || 'Install Claude CLI or check authentication.'}
            </p>
          </div>
        </div>
      )}

      {/* Active Tasks Summary */}
      <ActiveTasksWidget />

      {/* System Metrics */}
      {connected && (
        <div className="max-w-2xl mx-auto mt-4">
          <MetricsWidget />
        </div>
      )}

      {/* Projects Card */}
      <section className="card bg-base-200 max-w-2xl mx-auto mt-4">
        <div className="card-body p-4 sm:p-6">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
            <h2 className="card-title text-base-content flex items-center gap-2 text-base sm:text-lg">
              <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              Projects
              <span className="text-xs sm:text-sm font-normal text-base-content/60">({projects.length})</span>
            </h2>
            <button
              onClick={() => setShowFolderPicker(true)}
              className="btn btn-primary btn-sm w-full sm:w-auto"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Add Project
            </button>
          </div>

          {/* Search filter */}
          {projects.length > 3 && (
            <div className="mb-3">
              <input
                type="search"
                placeholder="Search projects..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input input-sm input-bordered w-full"
                aria-label="Search projects"
              />
            </div>
          )}

          {projects.length === 0 ? (
            <div className="text-center py-8 sm:py-12">
              <div aria-hidden="true" className="w-12 h-12 sm:w-16 sm:h-16 rounded-full bg-base-300 flex items-center justify-center mx-auto mb-4">
                <svg className="w-6 h-6 sm:w-8 sm:h-8 text-base-content/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
              </div>
              <p className="text-base-content font-medium mb-1">No projects yet</p>
              <p className="text-base-content/60 text-sm mb-4">Add a project folder to get started with {name}</p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-2">
                <button
                  onClick={() => setShowFolderPicker(true)}
                  className="btn btn-primary btn-sm"
                >
                  <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                  </svg>
                  Add Project
                </button>
                {docsData?.url && (
                  <a
                    href={docsData.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="btn btn-ghost btn-sm"
                  >
                    <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                    </svg>
                    Read the docs
                  </a>
                )}
              </div>
            </div>
          ) : (
            <ul aria-label="Projects" className="space-y-2 max-h-[300px] sm:max-h-[400px] overflow-auto">
              {filteredProjects.length === 0 && searchQuery && (
                <li className="text-center py-4 text-base-content/50 text-sm">
                  No projects matching "{searchQuery}"
                </li>
              )}
              {filteredProjects.map(p => {
                const task = taskByPath.get(p.path)
                return (
                <li key={p.id} className="group relative">
                  <button
                    type="button"
                    onClick={() => selectProject(p)}
                    aria-current={selectedProject?.id === p.id ? 'true' : undefined}
                    className={`w-full text-left p-3 sm:p-4 rounded-lg cursor-pointer transition-all duration-150 ${
                      selectedProject?.id === p.id
                        ? 'bg-primary/20 border border-primary/50'
                        : 'bg-base-100 hover:bg-base-300 border border-transparent hover:border-primary/30'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2 mb-1">
                      <span className="font-medium text-sm sm:text-base text-base-content group-hover:text-base-content transition-colors truncate">
                        {p.path.split('/').pop()}
                      </span>
                      <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
                        <span className={`badge badge-sm sm:badge-md ${
                          p.state === 'none' ? 'badge-ghost' :
                          p.state === 'implemented' ? 'badge-success' :
                          p.state === 'failed' ? 'badge-error' :
                          p.state === 'planning' || p.state === 'implementing' || p.state === 'reviewing' ? 'badge-warning' :
                          'badge-primary'
                        }`}>
                          {p.state}
                        </span>
                      </div>
                    </div>
                    {task?.task_title ? (
                      <p className="text-xs sm:text-sm text-base-content/70 truncate">{task.task_title}</p>
                    ) : (
                      <p className="text-xs sm:text-sm text-base-content/40 truncate italic">No active task</p>
                    )}
                    <div className="flex items-center gap-2 mt-0.5">
                      <p className="text-xs text-base-content/40 truncate font-mono">{p.path}</p>
                      {task?.source && (
                        <span className="text-xs text-primary/60 font-mono truncate flex-shrink-0">{task.source}</span>
                      )}
                      {task?.queue_count != null && task.queue_count > 0 && (
                        <span className="badge badge-xs badge-outline flex-shrink-0">
                          +{task.queue_count} queued
                        </span>
                      )}
                    </div>
                  </button>
                  <button
                    type="button"
                    onClick={(e) => handleRemoveProject(e, p.id)}
                    className="absolute top-3 right-3 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 text-base-content/50 hover:text-error transition-all p-1"
                    aria-label={`Remove ${p.path.split('/').pop()}`}
                  >
                    <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </li>
              )})}
            </ul>
          )}
        </div>
      </section>

      {/* Folder Picker Modal */}
      <FolderPicker
        isOpen={showFolderPicker}
        onClose={() => setShowFolderPicker(false)}
        onSelect={handleFolderSelect}
      />

      {/* Lazy-loaded modals */}
      <Suspense fallback={null}>
        {showSettings && (
          <Settings
            isOpen={showSettings}
            onClose={() => setShowSettings(false)}
          />
        )}
        {showMemory && (
          <MemoryPanel
            isOpen={showMemory}
            onClose={() => setShowMemory(false)}
          />
        )}
      </Suspense>

      {/* First-run onboarding (only shows once) */}
      {projects.length === 0 && connected && (
        <Onboarding onAddProject={() => setShowFolderPicker(true)} />
      )}
    </div>
  )
}
