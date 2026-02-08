import { useState } from 'react'
import { Plus, X, Loader2, Tag } from 'lucide-react'
import { useTaskLabels, useAddLabel, useRemoveLabel } from '@/api/task'

interface LabelsCardProps {
  hasActiveTask?: boolean
}

export function LabelsCard({ hasActiveTask = true }: LabelsCardProps) {
  const [newLabel, setNewLabel] = useState('')
  const [isAdding, setIsAdding] = useState(false)
  const { data, isLoading } = useTaskLabels()
  const { mutate: addLabel, isPending: isAddPending } = useAddLabel()
  const { mutate: removeLabel, isPending: isRemovePending } = useRemoveLabel()

  const labels = data?.labels ?? []
  const isPending = isAddPending || isRemovePending

  const handleAddLabel = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = newLabel.trim()
    if (!trimmed || !hasActiveTask) return

    addLabel(trimmed, {
      onSuccess: () => {
        setNewLabel('')
        setIsAdding(false)
      },
    })
  }

  const handleRemoveLabel = (label: string) => {
    if (!hasActiveTask) return
    removeLabel(label)
  }

  if (!hasActiveTask) {
    return null
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        {/* Header */}
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <h3 className="text-lg font-bold text-base-content flex items-center gap-2">
            <Tag size={18} aria-hidden="true" />
            Labels {labels.length > 0 && `(${labels.length})`}
          </h3>
          {!isAdding && (
            <button
              type="button"
              className="btn btn-ghost btn-xs gap-1"
              onClick={() => setIsAdding(true)}
              disabled={isPending}
              aria-label="Add new label"
            >
              <Plus size={14} aria-hidden="true" />
              Add
            </button>
          )}
        </div>

        {/* Add label form */}
        {isAdding && (
          <form onSubmit={handleAddLabel} className="mt-4 flex gap-2">
            <input
              type="text"
              value={newLabel}
              onChange={(e) => setNewLabel(e.target.value)}
              placeholder="Enter label name..."
              className="input input-bordered input-sm flex-1"
              disabled={isPending}
            />
            <button
              type="submit"
              className="btn btn-primary btn-sm"
              disabled={isPending || !newLabel.trim()}
              aria-label="Add label"
            >
              {isAddPending ? (
                <Loader2 size={14} className="animate-spin" aria-hidden="true" />
              ) : (
                <Plus size={14} aria-hidden="true" />
              )}
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={() => {
                setIsAdding(false)
                setNewLabel('')
              }}
              disabled={isPending}
              aria-label="Cancel"
            >
              <X size={14} aria-hidden="true" />
            </button>
          </form>
        )}

        {/* Labels display */}
        <div className="mt-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 size={20} className="animate-spin text-base-content/40" aria-hidden="true" />
            </div>
          ) : labels.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {labels.map((label) => (
                <LabelBadge
                  key={label}
                  label={label}
                  onRemove={() => handleRemoveLabel(label)}
                  disabled={isPending}
                />
              ))}
            </div>
          ) : (
            <p className="text-base-content/60 text-center py-4 text-sm">
              No labels yet. Click Add to create one.
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

interface LabelBadgeProps {
  label: string
  onRemove: () => void
  disabled?: boolean
}

function LabelBadge({ label, onRemove, disabled }: LabelBadgeProps) {
  return (
    <span className="badge badge-lg badge-outline gap-1 pr-1">
      <span>{label}</span>
      <button
        type="button"
        className="btn btn-ghost btn-xs btn-circle -mr-1"
        onClick={onRemove}
        disabled={disabled}
        aria-label={`Remove label ${label}`}
      >
        <X size={12} aria-hidden="true" />
      </button>
    </span>
  )
}
