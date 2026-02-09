import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useProjects, useSwitchProject, useAddProject } from '@/api/projects'
import { formatDistanceToNow } from 'date-fns'
import { Folder, Clock, GitBranch, Loader2, FolderOpen } from 'lucide-react'
import { FolderBrowser } from './FolderBrowser'

// Detect Electron environment
const isElectron = typeof window !== 'undefined' && !!(window as Window & { electron?: { openFolder: () => Promise<string | null> } }).electron

// Type for Electron API
declare global {
  interface Window {
    electron?: {
      openFolder: () => Promise<string | null>
    }
  }
}

export function ProjectSelector() {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useProjects()
  const switchProject = useSwitchProject()
  const addProject = useAddProject()
  const [showBrowser, setShowBrowser] = useState(false)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[200px]">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        <span>Failed to load projects: {error.message}</span>
      </div>
    )
  }

  const projects = data?.projects ?? []

  const handleSelect = (path: string) => {
    switchProject.mutate(path)
  }

  const handleOpenFolder = async () => {
    if (isElectron) {
      // Use native Electron dialog
      const path = await window.electron!.openFolder()
      if (path) {
        await handleAddProject(path)
      }
    } else {
      // Show folder browser modal for web
      setShowBrowser(true)
    }
  }

  const handleAddProject = async (path: string) => {
    try {
      await addProject.mutateAsync(path)
      // Refresh project list
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    } catch (err) {
      console.error('Failed to add project:', err)
    }
  }

  const handleBrowserSelect = async (path: string) => {
    setShowBrowser(false)
    await handleAddProject(path)
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-base-content/60 text-sm">
          {projects.length > 0
            ? `Select a project to open. Found ${projects.length} project${projects.length !== 1 ? 's' : ''}.`
            : 'No projects found.'}
        </p>
        <button
          onClick={handleOpenFolder}
          className="btn btn-primary btn-sm gap-2"
          disabled={addProject.isPending}
        >
          <FolderOpen size={16} />
          Open Folder
        </button>
      </div>

      {projects.length === 0 ? (
        <div className="card bg-base-200">
          <div className="card-body text-center">
            <Folder className="w-12 h-12 mx-auto text-base-content/40" aria-hidden="true" />
            <h3 className="text-lg font-semibold">No Projects Yet</h3>
            <p className="text-base-content/60">
              Click "Open Folder" to browse and add a project directory.
            </p>
          </div>
        </div>
      ) : (
        <div className="grid gap-3">
          {projects.map((project) => (
            <button
              key={project.id}
              onClick={() => handleSelect(project.path)}
              disabled={switchProject.isPending}
              className="card bg-base-100 hover:bg-base-200 border border-base-300 transition-colors text-left"
            >
              <div className="card-body p-4">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <Folder className="w-5 h-5 text-primary" aria-hidden="true" />
                    <div>
                      <h3 className="font-medium">{project.name}</h3>
                      <p className="text-xs text-base-content/50 font-mono truncate max-w-md">{project.path}</p>
                      <div className="flex items-center gap-4 text-sm text-base-content/60 mt-1">
                        {project.remote_url && (
                          <span className="flex items-center gap-1">
                            <GitBranch size={14} aria-hidden="true" />
                            {project.remote_url.replace(/^https?:\/\//, '').replace(/\.git$/, '')}
                          </span>
                        )}
                        {project.last_access && (
                          <span className="flex items-center gap-1">
                            <Clock size={14} aria-hidden="true" />
                            {formatDistanceToNow(new Date(project.last_access), { addSuffix: true })}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {switchProject.isPending && (
        <div className="flex items-center justify-center gap-2 text-sm text-base-content/60">
          <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
          Switching project...
        </div>
      )}

      {switchProject.isError && (
        <div className="alert alert-error" role="alert">
          <span>Failed to switch project: {switchProject.error.message}</span>
        </div>
      )}

      {addProject.isPending && (
        <div className="flex items-center justify-center gap-2 text-sm text-base-content/60">
          <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
          Adding project...
        </div>
      )}

      {addProject.isError && (
        <div className="alert alert-error" role="alert">
          <span>Failed to add project: {addProject.error.message}</span>
        </div>
      )}

      {/* Folder browser modal (web only) */}
      {showBrowser && (
        <FolderBrowser
          onSelect={handleBrowserSelect}
          onClose={() => setShowBrowser(false)}
        />
      )}
    </div>
  )
}
