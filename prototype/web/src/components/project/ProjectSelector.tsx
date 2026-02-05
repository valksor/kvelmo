import { useProjects, useSwitchProject } from '@/api/projects'
import { formatDistanceToNow } from 'date-fns'
import { Folder, Clock, GitBranch, Loader2 } from 'lucide-react'

export function ProjectSelector() {
  const { data, isLoading, error } = useProjects()
  const switchProject = useSwitchProject()

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

  if (projects.length === 0) {
    return (
      <div className="card bg-base-200">
        <div className="card-body text-center">
          <Folder className="w-12 h-12 mx-auto text-base-content/40" />
          <h3 className="text-lg font-semibold">No Projects Registered</h3>
          <p className="text-base-content/60">
            Register a project with <code className="kbd kbd-sm">mehr serve register</code> to get started.
          </p>
        </div>
      </div>
    )
  }

  const handleSelect = (path: string) => {
    switchProject.mutate(path)
  }

  return (
    <div className="space-y-3">
      <p className="text-base-content/60 text-sm">
        Select a project to open. Found {projects.length} registered project{projects.length !== 1 ? 's' : ''}.
      </p>

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
                  <Folder className="w-5 h-5 text-primary" />
                  <div>
                    <h3 className="font-medium">{project.name}</h3>
                    <p className="text-xs text-base-content/50 font-mono truncate max-w-md">{project.path}</p>
                    <div className="flex items-center gap-4 text-sm text-base-content/60 mt-1">
                      {project.remote_url && (
                        <span className="flex items-center gap-1">
                          <GitBranch size={14} />
                          {project.remote_url.replace(/^https?:\/\//, '').replace(/\.git$/, '')}
                        </span>
                      )}
                      {project.last_access && (
                        <span className="flex items-center gap-1">
                          <Clock size={14} />
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

      {switchProject.isPending && (
        <div className="flex items-center justify-center gap-2 text-sm text-base-content/60">
          <Loader2 className="w-4 h-4 animate-spin" />
          Switching project...
        </div>
      )}

      {switchProject.isError && (
        <div className="alert alert-error">
          <span>Failed to switch project: {switchProject.error.message}</span>
        </div>
      )}
    </div>
  )
}
