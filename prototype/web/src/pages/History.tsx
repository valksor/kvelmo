import { useState, useMemo, useCallback } from 'react'
import { Link } from 'react-router-dom'
import {
  Loader2,
  Clock,
  AlertCircle,
  CheckCircle,
  XCircle,
  Play,
  RefreshCw,
  Search,
  Filter,
  ArrowUpDown,
} from 'lucide-react'
import { useTaskHistory } from '@/api/settings'
import { formatDistanceToNow } from 'date-fns'
import type { WorkflowState, TaskHistoryItem } from '@/types/api'
import { getStateConfig } from '@/constants/stateConfig'

// Debounce hook
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useMemo(() => {
    const handler = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(handler)
  }, [value, delay])

  return debouncedValue
}

type SortOption = 'date-desc' | 'date-asc' | 'title-asc' | 'title-desc'
type StateFilter = 'all' | WorkflowState

const STATE_OPTIONS: { value: StateFilter; label: string }[] = [
  { value: 'all', label: 'All States' },
  { value: 'done', label: 'Done' },
  { value: 'failed', label: 'Failed' },
  { value: 'planning', label: 'Planning' },
  { value: 'implementing', label: 'Implementing' },
  { value: 'reviewing', label: 'Reviewing' },
  { value: 'waiting', label: 'Waiting' },
  { value: 'idle', label: 'Idle' },
]

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'date-desc', label: 'Newest First' },
  { value: 'date-asc', label: 'Oldest First' },
  { value: 'title-asc', label: 'Title A-Z' },
  { value: 'title-desc', label: 'Title Z-A' },
]

// Lucide React icons for this page (different visual style than emoji icons)
const stateIcons: Record<WorkflowState, React.ReactNode> = {
  idle: <Clock size={14} />,
  planning: <Play size={14} />,
  implementing: <Play size={14} />,
  reviewing: <RefreshCw size={14} />,
  waiting: <Clock size={14} />,
  checkpointing: <RefreshCw size={14} />,
  reverting: <RefreshCw size={14} />,
  restoring: <RefreshCw size={14} />,
  done: <CheckCircle size={14} />,
  failed: <XCircle size={14} />,
}

export default function History() {
  const { data: tasks, isLoading, error, refetch } = useTaskHistory()

  // Filter state
  const [searchQuery, setSearchQuery] = useState('')
  const [stateFilter, setStateFilter] = useState<StateFilter>('all')
  const [sortBy, setSortBy] = useState<SortOption>('date-desc')

  // Debounce search for performance
  const debouncedSearch = useDebounce(searchQuery, 300)

  // Filter and sort tasks
  const filteredTasks = useMemo(() => {
    if (!tasks) return []

    let result = [...tasks]

    // Apply search filter
    if (debouncedSearch) {
      const query = debouncedSearch.toLowerCase()
      result = result.filter(
        (task) =>
          task.title?.toLowerCase().includes(query) || task.id.toLowerCase().includes(query)
      )
    }

    // Apply state filter
    if (stateFilter !== 'all') {
      result = result.filter((task) => task.state === stateFilter)
    }

    // Apply sorting
    result.sort((a, b) => {
      switch (sortBy) {
        case 'date-desc':
          return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        case 'date-asc':
          return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
        case 'title-asc':
          return (a.title || a.id).localeCompare(b.title || b.id)
        case 'title-desc':
          return (b.title || b.id).localeCompare(a.title || a.id)
        default:
          return 0
      }
    })

    return result
  }, [tasks, debouncedSearch, stateFilter, sortBy])

  const clearFilters = useCallback(() => {
    setSearchQuery('')
    setStateFilter('all')
    setSortBy('date-desc')
  }, [])

  const hasActiveFilters = searchQuery || stateFilter !== 'all' || sortBy !== 'date-desc'

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        <AlertCircle size={20} />
        <span>Failed to load task history: {error.message}</span>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Task History</h1>
        <button className="btn btn-ghost btn-sm" onClick={() => refetch()}>
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body py-4">
          <div className="flex flex-col md:flex-row gap-4">
            {/* Search */}
            <div className="flex-1">
              <label className="input input-bordered flex items-center gap-2">
                <Search size={16} className="text-base-content/50" />
                <input
                  type="text"
                  placeholder="Search by title or ID..."
                  className="grow"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </label>
            </div>

            {/* State filter */}
            <div className="flex items-center gap-2">
              <Filter size={16} className="text-base-content/50" />
              <select
                className="select select-bordered"
                value={stateFilter}
                onChange={(e) => setStateFilter(e.target.value as StateFilter)}
              >
                {STATE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Sort */}
            <div className="flex items-center gap-2">
              <ArrowUpDown size={16} className="text-base-content/50" />
              <select
                className="select select-bordered"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as SortOption)}
              >
                {SORT_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Clear filters */}
            {hasActiveFilters && (
              <button className="btn btn-ghost btn-sm" onClick={clearFilters}>
                Clear
              </button>
            )}
          </div>

          {/* Results count */}
          <div className="text-sm text-base-content/60 mt-2">
            Showing {filteredTasks.length} of {tasks?.length || 0} tasks
            {hasActiveFilters && ' (filtered)'}
          </div>
        </div>
      </div>

      {/* Task list */}
      {!tasks || tasks.length === 0 ? (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center">
            <p className="text-base-content/60">No tasks yet. Start a task with the CLI:</p>
            <code className="bg-base-200 px-4 py-2 rounded mt-2 text-sm">mehr start &lt;ref&gt;</code>
          </div>
        </div>
      ) : filteredTasks.length === 0 ? (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center">
            <p className="text-base-content/60">No tasks match your filters.</p>
            <button className="btn btn-ghost btn-sm mt-2" onClick={clearFilters}>
              Clear Filters
            </button>
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="table table-zebra">
            <thead>
              <tr>
                <th>Title</th>
                <th>State</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredTasks.map((task) => (
                <TaskRow key={task.id} task={task} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function TaskRow({ task }: { task: TaskHistoryItem }) {
  const config = getStateConfig(task.state)
  const icon = stateIcons[task.state] || stateIcons.idle
  const createdAgo = task.created_at
    ? formatDistanceToNow(new Date(task.created_at), { addSuffix: true })
    : 'unknown'

  return (
    <tr>
      <td>
        <div className="flex flex-col">
          <Link
            to={`/app/task/${task.id}`}
            className="font-medium hover:underline"
          >
            {task.title || task.id}
          </Link>
          <span className="text-xs text-base-content/50 font-mono">{task.id}</span>
        </div>
      </td>
      <td>
        <span className={`badge gap-1 ${config.badge}`}>
          {icon}
          {task.state}
        </span>
      </td>
      <td className="text-sm text-base-content/60">{createdAgo}</td>
      <td>
        <Link to={`/app/task/${task.id}`} className="btn btn-ghost btn-xs">
          View
        </Link>
      </td>
    </tr>
  )
}
