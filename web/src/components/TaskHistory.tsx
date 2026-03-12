import { useEffect, useState } from 'react'
import { useProjectStore } from '../stores/projectStore'

interface ArchivedTask {
  id: string
  title: string
  branch: string
  source: string
  final_state: string
  started_at: string
  completed_at: string
}

export function TaskHistory() {
  const { connected } = useProjectStore()
  const client = useProjectStore(s => s.client)
  const [tasks, setTasks] = useState<ArchivedTask[] | null>(null)

  useEffect(() => {
    if (!connected || !client) return
    let cancelled = false
    client.call<{ tasks: ArchivedTask[] | null }>('task.history', {})
      .then(result => { if (!cancelled) setTasks(result.tasks || []) })
      .catch(() => { if (!cancelled) setTasks([]) })
    return () => { cancelled = true }
  }, [connected, client])

  if (tasks === null) {
    return <p className="text-xs text-base-content/50">Loading history...</p>
  }

  if (tasks.length === 0) {
    return <p className="text-xs text-base-content/50">No completed tasks yet</p>
  }

  return (
    <ul className="space-y-1.5">
      {tasks.slice(0, 10).map(task => (
        <li key={task.id} className="p-2 bg-base-300 rounded text-xs">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium truncate">{task.title || task.id}</span>
            <span className={`badge badge-xs ${
              task.final_state === 'finished' ? 'badge-success' :
              task.final_state === 'abandoned' ? 'badge-warning' :
              'badge-ghost'
            }`}>
              {task.final_state}
            </span>
          </div>
          {task.source && (
            <p className="text-base-content/50 font-mono text-[10px] truncate mt-0.5">{task.source}</p>
          )}
          {task.completed_at && (
            <p className="text-base-content/40 text-[10px] mt-0.5">
              {new Date(task.completed_at).toLocaleDateString()}
            </p>
          )}
        </li>
      ))}
    </ul>
  )
}
