import { useProjectStore } from '../stores/projectStore'
import { useLayoutStore } from '../stores/layoutStore'

interface FileChange {
  path: string
  status: 'added' | 'modified' | 'deleted'
  additions: number
  deletions: number
}

interface FileChangesWidgetProps {
  embedded?: boolean
}

export function FileChangesWidget({ embedded = false }: FileChangesWidgetProps) {
  const { fileChanges } = useProjectStore()
  const { openTab } = useLayoutStore()

  const statusConfig = {
    added: { color: 'text-success', bg: 'bg-success/20', icon: '+' },
    modified: { color: 'text-warning', bg: 'bg-warning/20', icon: '~' },
    deleted: { color: 'text-error', bg: 'bg-error/20', icon: '-' }
  }

  const handleFileClick = (file: FileChange) => {
    openTab({
      id: `diff-${file.path}`,
      type: 'diff',
      title: file.path.split('/').pop() || file.path,
      data: { path: file.path, status: file.status },
      closeable: true,
    })
  }

  const content = (
    <div>
          {!fileChanges || fileChanges.length === 0 ? (
            <div className="text-center py-6">
              <svg aria-hidden="true" className="w-8 h-8 mx-auto mb-2 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p className="text-base-content/60 text-sm">No changes yet</p>
            </div>
          ) : (
            <div className="space-y-1 max-h-[250px] overflow-auto">
              {(fileChanges as FileChange[]).map((file, index) => {
                const config = statusConfig[file.status]
                return (
                  <button
                    key={file.path}
                    onClick={() => handleFileClick(file)}
                    aria-label={`View diff for ${file.path} (${file.status})`}
                    className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-300 transition-colors group animate-in"
                    style={{ animationDelay: `${index * 30}ms` }}
                  >
                    <span className={`w-6 h-6 rounded flex items-center justify-center font-mono text-sm ${config.bg} ${config.color}`}>
                      {config.icon}
                    </span>
                    <span className="flex-1 truncate font-mono text-sm text-base-content/80 group-hover:text-base-content transition-colors text-left">
                      {file.path}
                    </span>
                    <div className="flex items-center gap-2 text-xs font-mono">
                      <span className="text-success">+{file.additions}</span>
                      <span className="text-error">-{file.deletions}</span>
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </div>
      )

  if (embedded) {
    return content
  }

  return (
    <section className="card bg-base-200">
      <div className="card-body">
        <div className="flex items-center justify-between">
          <h2 className="card-title text-base-content flex items-center gap-2">
            <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            File Changes
          </h2>
          {fileChanges && fileChanges.length > 0 && (
            <span className="text-sm text-base-content/60">{fileChanges.length} files</span>
          )}
        </div>
        <div className="mt-4">
          {content}
        </div>
      </div>
    </section>
  )
}
