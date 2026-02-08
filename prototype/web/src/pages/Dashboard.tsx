import { useActiveTask, useStatus } from '@/api/workflow'
import { useTaskHistory } from '@/api/settings'
import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/api/client'
import { useWorkflowSSE } from '@/hooks/useWorkflowSSE'
import { ProjectSelector } from '@/components/project/ProjectSelector'
import { TaskSummaryCard } from '@/components/project/TaskSummaryCard'
import { ProjectCostsCard } from '@/components/project/ProjectCostsCard'
import { TaskCreationTabs } from '@/components/project/TaskCreationTabs'
import { DashboardTasksCard } from '@/components/project/DashboardTasksCard'
import { Loader2, Wifi, WifiOff } from 'lucide-react'

export default function Dashboard() {
  // SSE connection for real-time updates
  const { connected } = useWorkflowSSE()

  // Data fetching
  const { data: status, isLoading: statusLoading } = useStatus()

  // Only fetch task data in project mode
  const isGlobalMode = status?.mode === 'global'
  const { data: taskData, isLoading: taskLoading } = useActiveTask({
    enabled: !isGlobalMode && !statusLoading,
  })
  const { data: tasksHistory, isLoading: historyLoading } = useTaskHistory({
    enabled: !isGlobalMode && !statusLoading,
  })
  const { data: budget } = useQuery({
    queryKey: ['budget', 'monthly', 'status'],
    queryFn: () => apiRequest<{ enabled: boolean }>('/budget/monthly/status'),
    enabled: !isGlobalMode && !statusLoading,
  })

  const budgetEnabled = budget?.enabled ?? false

  // Loading state
  if (statusLoading || (!isGlobalMode && taskLoading)) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 aria-hidden="true" className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project selector
  if (isGlobalMode) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Projects</h1>
          <ConnectionStatus connected={connected} />
        </div>
        <ProjectSelector />
      </div>
    )
  }

  // Project mode: show project dashboard
  const hasActiveTask = !!(taskData?.active && taskData.task)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <ConnectionStatus connected={connected} />
      </div>

      {/* Active task summary (links to task page) */}
      {hasActiveTask && (
        <TaskSummaryCard
          task={taskData.task}
          work={taskData.work}
          progressPhase={taskData.task?.progress_phase}
        />
      )}

      {/* Two column layout: Task creation + Budget (or full width if no budget) */}
      <div className={`grid grid-cols-1 ${budgetEnabled ? 'lg:grid-cols-2' : ''} gap-6`}>
        {/* Left: Task creation tabs (Start/Quick/Plan) - full width if no budget */}
        <TaskCreationTabs />

        {/* Right: Budget/Costs overview (only if enabled) */}
        {budgetEnabled && <ProjectCostsCard />}
      </div>

      {/* Tasks (recent + queue views) */}
      <DashboardTasksCard tasks={tasksHistory} isHistoryLoading={historyLoading} />
    </div>
  )
}

function ConnectionStatus({ connected }: { connected: boolean }) {
  return (
    <div className="flex items-center gap-2 text-sm">
      {connected ? (
        <>
          <Wifi size={16} aria-hidden="true" className="text-success" />
          <span className="text-base-content/60">Connected</span>
        </>
      ) : (
        <div className="tooltip tooltip-left" data-tip="Lost connection to server. Updates will resume automatically.">
          <div className="flex items-center gap-2">
            <WifiOff size={16} aria-hidden="true" className="text-warning" />
            <span className="text-base-content/60">Reconnecting...</span>
          </div>
        </div>
      )}
    </div>
  )
}
