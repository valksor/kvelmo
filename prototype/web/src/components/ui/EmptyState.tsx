import { Inbox } from 'lucide-react'

interface EmptyStateProps {
  title?: string
  description?: string
  action?: React.ReactNode
}

export function EmptyState({
  title = 'No active task',
  description = 'Start a new task to begin working.',
  action,
}: EmptyStateProps) {
  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body items-center text-center py-12">
        <Inbox className="w-12 h-12 text-base-content/30 mb-4" />
        <h3 className="text-lg font-medium">{title}</h3>
        <p className="text-base-content/60 max-w-sm">{description}</p>
        {action && <div className="mt-4">{action}</div>}
      </div>
    </div>
  )
}
