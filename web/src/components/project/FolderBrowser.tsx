import { useState } from 'react'
import { useFSBrowse } from '@/api/filesystem'
import { Folder, ChevronUp, Loader2, AlertCircle, FolderOpen } from 'lucide-react'

interface FolderBrowserProps {
  onSelect: (path: string) => void
  onClose: () => void
}

/**
 * Modal component for browsing and selecting a folder from the server's filesystem.
 * Uses the /api/v1/fs/browse endpoint to list directories.
 */
export function FolderBrowser({ onSelect, onClose }: FolderBrowserProps) {
  // null = home directory (backend default)
  const [currentPath, setCurrentPath] = useState<string | null>(null)
  const { data, isLoading, error } = useFSBrowse(currentPath)

  const handleNavigate = (dirName: string) => {
    // Navigate into subdirectory
    const newPath = data?.path ? `${data.path}/${dirName}` : dirName
    setCurrentPath(newPath)
  }

  const handleGoUp = () => {
    if (data?.parent && data.parent !== data.path) {
      setCurrentPath(data.parent)
    }
  }

  const handleSelect = () => {
    if (data?.path) {
      onSelect(data.path)
    }
  }

  // Check if we can go up (not at root)
  const canGoUp = data?.parent && data.parent !== data.path

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-2xl">
        <h3 className="font-bold text-lg mb-4">Select Project Folder</h3>

        {/* Current path display */}
        <div className="bg-base-200 rounded-lg p-3 mb-4 font-mono text-sm break-all">
          {isLoading ? (
            <span className="text-base-content/50">Loading...</span>
          ) : (
            data?.path || '~'
          )}
        </div>

        {/* Error state */}
        {error && (
          <div className="alert alert-error mb-4">
            <AlertCircle className="w-5 h-5" />
            <span>{error instanceof Error ? error.message : 'Failed to browse directory'}</span>
          </div>
        )}

        {/* Navigation bar */}
        <div className="flex gap-2 mb-4">
          <button
            onClick={handleGoUp}
            disabled={!canGoUp || isLoading}
            className="btn btn-sm btn-ghost gap-2"
            title="Go to parent directory"
          >
            <ChevronUp size={16} />
            Go Up
          </button>
        </div>

        {/* Directory list */}
        <div className="border border-base-300 rounded-lg max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            </div>
          ) : data?.entries.length === 0 ? (
            <div className="text-center p-8 text-base-content/50">
              <Folder className="w-8 h-8 mx-auto mb-2 opacity-50" />
              <p>No subdirectories</p>
            </div>
          ) : (
            <ul className="menu menu-sm p-2">
              {data?.entries.map((entry) => (
                <li key={entry.name}>
                  <button
                    onClick={() => handleNavigate(entry.name)}
                    className="flex items-center gap-2"
                  >
                    <Folder className="w-4 h-4 text-primary" />
                    <span className="truncate">{entry.name}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Actions */}
        <div className="modal-action">
          <button onClick={onClose} className="btn btn-ghost">
            Cancel
          </button>
          <button
            onClick={handleSelect}
            disabled={isLoading || !data?.path}
            className="btn btn-primary gap-2"
          >
            <FolderOpen size={16} />
            Select This Folder
          </button>
        </div>
      </div>

      {/* Click outside to close */}
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  )
}
