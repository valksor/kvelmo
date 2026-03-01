import { useEffect } from 'react'
import { useGlobalStore } from './stores/globalStore'
import { useProjectStore } from './stores/projectStore'
import { useThemeStore } from './stores/themeStore'
import { useLeakWatchdog } from './hooks/useLeakWatchdog'
import { ErrorBoundary } from './components/ErrorBoundary'
import { GlobalView } from './components/GlobalView'
import { ProjectView } from './components/ProjectView'
import { StateAnnouncer } from './components/StateAnnouncer'

// Demo mode for testing UI without backend
const DEMO_MODE = new URLSearchParams(window.location.search).has('demo')

export default function App() {
  const { selectedProject, selectProject, connect, connected, connecting } = useGlobalStore()
  const { connect: connectWorktree, disconnect: disconnectWorktree } = useProjectStore()
  const { theme, setTheme } = useThemeStore()

  useLeakWatchdog((growthMB) => {
    console.error(`LeakWatchdog: heap grew +${growthMB.toFixed(0)}MB without GC recovery — reloading`)
    window.location.reload()
  })

  useEffect(() => {
    if (!DEMO_MODE) {
      connect()
    }
  }, [connect])

  // Initialize theme on mount
  useEffect(() => {
    setTheme(theme)
  }, [])

  // Connect to worktree socket when a project is selected
  useEffect(() => {
    if (DEMO_MODE) return
    if (selectedProject?.path) {
      connectWorktree(selectedProject.path)
    }
    return () => {
      disconnectWorktree()
    }
  }, [selectedProject?.path, connectWorktree, disconnectWorktree])

  // Demo mode: show ProjectView with mock data
  if (DEMO_MODE) {
    // Set a mock project if not already set
    useEffect(() => {
      if (!selectedProject) {
        selectProject({
          id: 'demo-project',
          path: '/Users/demo/workspace/my-project',
          state: 'idle'
        })
      }
    }, [])

    return (
      <ErrorBoundary>
        <main id="main-content" tabIndex={-1} className="min-h-screen bg-base-100 transition-colors">
          {selectedProject ? <ProjectView /> : (
            <div className="h-screen flex items-center justify-center">
              <span className="loading loading-spinner loading-lg"></span>
            </div>
          )}
        </main>
      </ErrorBoundary>
    )
  }

  if (connecting) {
    return (
      <ErrorBoundary>
        <main id="main-content" tabIndex={-1} className="min-h-screen bg-base-100 flex items-center justify-center">
          <div className="text-center">
            <span className="loading loading-spinner loading-lg text-primary"></span>
            <p className="mt-4 text-base-content/60">Connecting to server...</p>
          </div>
        </main>
      </ErrorBoundary>
    )
  }

  if (!connected) {
    return (
      <ErrorBoundary>
        <main id="main-content" tabIndex={-1} className="min-h-screen bg-base-100 flex items-center justify-center">
          <div className="text-center">
            <div className="text-error text-6xl mb-4" aria-hidden="true">!</div>
            <p className="text-base-content mb-4" role="alert">Failed to connect to server</p>
            <button onClick={() => connect()} className="btn btn-primary">
              Retry
            </button>
          </div>
        </main>
      </ErrorBoundary>
    )
  }

  return (
    <ErrorBoundary>
      <main id="main-content" tabIndex={-1} className="min-h-screen bg-base-100 transition-colors">
        <StateAnnouncer />
        {selectedProject ? <ProjectView /> : <GlobalView />}
      </main>
    </ErrorBoundary>
  )
}
