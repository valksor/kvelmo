import { useState } from 'react'
import { Globe, FolderOpen, Trash2, ArrowRightLeft } from 'lucide-react'

export type OverrideScope = 'global' | 'project'

export interface TerminologyEntry {
  find: string
  replace: string
  scope: OverrideScope
}

interface TerminologyEditorProps {
  /** Combined terminology entries from both scopes */
  entries: TerminologyEntry[]
  /** Called when entries change */
  onChange: (entries: TerminologyEntry[]) => void
  /** Current project name for display */
  projectName?: string
  /** Whether project context is available */
  hasProject: boolean
}

/**
 * Editable grid for terminology replacements.
 * Shows find/replace pairs with scope indicators (global vs project).
 */
export function TerminologyEditor({
  entries,
  onChange,
  projectName,
  hasProject,
}: TerminologyEditorProps) {
  const [newFind, setNewFind] = useState('')
  const [newReplace, setNewReplace] = useState('')

  const handleAdd = (scope: OverrideScope) => {
    if (!newFind.trim() || !newReplace.trim()) return

    // Check for duplicate find term in same scope
    const exists = entries.some(
      (e) => e.find.toLowerCase() === newFind.trim().toLowerCase() && e.scope === scope
    )
    if (exists) return

    onChange([...entries, { find: newFind.trim(), replace: newReplace.trim(), scope }])
    setNewFind('')
    setNewReplace('')
  }

  const handleRemove = (index: number) => {
    onChange(entries.filter((_, i) => i !== index))
  }

  const handleScopeChange = (index: number) => {
    const updated = entries.map((entry, i) => {
      if (i === index) {
        return { ...entry, scope: entry.scope === 'global' ? 'project' : 'global' } as TerminologyEntry
      }
      return entry
    })
    onChange(updated)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      // Default to project scope if available, otherwise global
      handleAdd(hasProject ? 'project' : 'global')
    }
  }

  return (
    <div className="space-y-3">
      {/* Entry list */}
      {entries.length > 0 && (
        <div className="overflow-x-auto">
          <table className="table table-sm w-full">
            <thead>
              <tr>
                <th className="w-2/5">Find</th>
                <th className="w-2/5">Replace With</th>
                <th className="w-1/5">Scope</th>
                <th className="w-10"></th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry, index) => (
                <tr key={`${entry.scope}-${entry.find}`} className="hover">
                  <td className="font-mono text-sm">{entry.find}</td>
                  <td className="font-mono text-sm">{entry.replace}</td>
                  <td>
                    <ScopeIndicator
                      scope={entry.scope}
                      projectName={projectName}
                      canToggle={hasProject}
                      onToggle={() => handleScopeChange(index)}
                    />
                  </td>
                  <td>
                    <button
                      type="button"
                      className="btn btn-ghost btn-xs text-error"
                      onClick={() => handleRemove(index)}
                      aria-label={`Remove ${entry.find} replacement`}
                    >
                      <Trash2 size={14} aria-hidden="true" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Empty state */}
      {entries.length === 0 && (
        <div className="text-center py-6 text-base-content/60">
          <p className="text-sm">No terminology replacements configured.</p>
          <p className="text-xs mt-1">
            Add replacements to customize terms like "Task" → "Ticket" or "Workflow" → "Pipeline".
          </p>
        </div>
      )}

      {/* Add new entry */}
      <div className="flex flex-wrap gap-2 items-end pt-2 border-t border-base-300/50">
        <div className="flex-1 min-w-[120px]">
          <label htmlFor="terminology-find" className="label py-1">
            <span className="label-text text-xs">Find</span>
          </label>
          <input
            id="terminology-find"
            type="text"
            className="input input-bordered input-sm w-full"
            placeholder="Task"
            value={newFind}
            onChange={(e) => setNewFind(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        </div>
        <div className="flex-1 min-w-[120px]">
          <label htmlFor="terminology-replace" className="label py-1">
            <span className="label-text text-xs">Replace With</span>
          </label>
          <input
            id="terminology-replace"
            type="text"
            className="input input-bordered input-sm w-full"
            placeholder="Ticket"
            value={newReplace}
            onChange={(e) => setNewReplace(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        </div>
        <div className="flex gap-1 items-end">
          <button
            type="button"
            className="btn btn-ghost btn-sm gap-1"
            onClick={() => handleAdd('global')}
            disabled={!newFind.trim() || !newReplace.trim()}
          >
            <Globe size={14} aria-hidden="true" />
            Add Global
          </button>
          {hasProject && (
            <button
              type="button"
              className="btn btn-primary btn-sm gap-1"
              onClick={() => handleAdd('project')}
              disabled={!newFind.trim() || !newReplace.trim()}
            >
              <FolderOpen size={14} aria-hidden="true" />
              Add to Project
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

interface ScopeIndicatorProps {
  scope: OverrideScope
  projectName?: string
  canToggle?: boolean
  onToggle?: () => void
}

function ScopeIndicator({ scope, projectName, canToggle, onToggle }: ScopeIndicatorProps) {
  const isProject = scope === 'project'
  const icon = isProject ? <FolderOpen size={12} aria-hidden="true" /> : <Globe size={12} aria-hidden="true" />
  const label = isProject ? 'Project' : 'Global'
  const title = isProject ? `Project: ${projectName ?? 'Current'}` : 'Applies to all projects'
  const badgeClass = isProject ? 'badge-primary' : 'badge-ghost'

  if (canToggle && onToggle) {
    const targetScope = isProject ? 'global' : 'project'
    return (
      <button
        type="button"
        className={`badge badge-sm ${badgeClass} gap-1 cursor-pointer hover:opacity-80 transition-opacity group`}
        title={`Click to move to ${targetScope}`}
        aria-label={`Move to ${targetScope} scope`}
        onClick={onToggle}
      >
        {icon}
        {label}
        <ArrowRightLeft size={10} className="opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
      </button>
    )
  }

  return (
    <span className={`badge badge-sm ${badgeClass} gap-1`} title={title}>
      {icon}
      {label}
    </span>
  )
}
