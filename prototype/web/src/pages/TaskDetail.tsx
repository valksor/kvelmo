import { useState, useCallback, useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useActiveTask } from '@/api/workflow'
import { useTaskSpecs, useTaskNotes, useAgentLogsHistory, useTaskWork } from '@/api/task'
import { useWorkflowSSE, type AgentMessage } from '@/hooks/useWorkflowSSE'
import { ActiveWorkCard } from '@/components/task/ActiveWorkCard'
import { CompletedWorkCard } from '@/components/task/CompletedWorkCard'
import { WorkflowActions } from '@/components/workflow/WorkflowActions'
import { QuestionPrompt } from '@/components/workflow/QuestionPrompt'
import { SpecificationsList } from '@/components/task/SpecificationsList'
import { ReviewsList } from '@/components/task/ReviewsList'
import { NotesCard } from '@/components/task/NotesCard'
import { LabelsCard } from '@/components/task/LabelsCard'
import { AgentTerminal, type TerminalMessage } from '@/components/task/AgentTerminal'
import { QuickQuestion } from '@/components/task/QuickQuestion'
import { CostsCard } from '@/components/task/CostsCard'
import { ArrowLeft, Wifi, WifiOff } from 'lucide-react'

let nextMsgId = 0

export default function TaskDetail() {
  const { id } = useParams<{ id: string }>()

  return <TaskDetailView key={id ?? 'unknown-task'} id={id} />
}

function TaskDetailView({ id }: { id?: string }) {

  // Agent terminal messages (local state, not persisted)
  const [agentMessages, setAgentMessages] = useState<TerminalMessage[]>([])
  const [historySuppressed, setHistorySuppressed] = useState(false)

  const handleAgentMessage = useCallback((message: AgentMessage) => {
    setAgentMessages((prev) => {
      const enriched: TerminalMessage = { ...message, _id: ++nextMsgId }
      if (prev.length >= 2000) {
        return [...prev.slice(-1000), enriched]
      }
      return [...prev, enriched]
    })
  }, [])

  const clearAgentMessages = useCallback(() => {
    setAgentMessages([])
    setHistorySuppressed(true)
  }, [])

  // SSE connection for real-time updates
  const { connected } = useWorkflowSSE({
    taskId: id,
    onAgentMessage: handleAgentMessage,
  })

  // Fetch task data - all queries run in parallel
  const { data: taskData, isLoading: activeLoading } = useActiveTask()
  const { data: workData, isLoading: workLoading } = useTaskWork(id)
  const { data: specsData, isLoading: specsLoading } = useTaskSpecs(id)
  const { data: notesData } = useTaskNotes(id)
  const { data: agentLogsHistory } = useAgentLogsHistory(id)

  const historyMessages = useMemo<TerminalMessage[]>(() => {
    if (historySuppressed || !agentLogsHistory?.logs?.length) {
      return []
    }

    return agentLogsHistory.logs.map((entry, i) => ({
      content: entry.message,
      timestamp: entry.started_at || new Date(entry.index * 1000).toISOString(),
      type: entry.type || 'output',
      taskId: id,
      _id: -(i + 1),
    }))
  }, [agentLogsHistory, historySuppressed, id])

  const terminalMessages = useMemo<TerminalMessage[]>(() => {
    if (historyMessages.length === 0) {
      return agentMessages
    }

    // Merge history + live stream and deduplicate identical lines.
    const seen = new Set<string>()
    const merged: TerminalMessage[] = []
    const pushUnique = (message: TerminalMessage) => {
      const key = `${message.timestamp}:${message.type || 'output'}:${message.content}`
      if (seen.has(key)) {
        return
      }
      seen.add(key)
      merged.push(message)
    }

    historyMessages.forEach(pushUnique)
    agentMessages.forEach(pushUnique)

    return merged
  }, [historyMessages, agentMessages])

  // Determine if this is the active task or a completed task
  const isActiveTask = taskData?.task?.id === id
  const isCompletedTask = !isActiveTask && workData?.work != null
  const taskLoading = activeLoading || workLoading

  // For active tasks, use active task data; for completed, use work data
  const task = isActiveTask ? taskData?.task : undefined
  const work = isActiveTask ? taskData?.work : undefined
  const completedWork = isCompletedTask ? workData?.work : undefined
  const state = task?.state || completedWork?.metadata?.state || 'done'
  const progressPhase = task?.progress_phase
  const hasTask = isActiveTask && taskData?.active === true

  // Show not found only after loading completes and neither active nor completed task exists
  if (!taskLoading && !task && !completedWork) {
    return (
      <div className="space-y-4">
        <Link to="/" className="btn btn-ghost gap-2">
          <ArrowLeft size={16} aria-hidden="true" />
          Back to Dashboard
        </Link>
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center py-12">
            <h2 className="text-xl font-bold text-base-content">Task Not Found</h2>
            <p className="text-base-content/60 mt-2">
              This task may have been completed or doesn't exist.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header with back link and connection status */}
      <div className="flex items-center justify-between">
        <Link to="/" className="btn btn-ghost btn-sm gap-2">
          <ArrowLeft size={16} aria-hidden="true" />
          Dashboard
        </Link>
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
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left column: Task info */}
        <div className="lg:col-span-2 space-y-6">
          {/* Task card with loading skeleton */}
          {taskLoading ? (
            <div className="card bg-base-100 shadow-sm animate-pulse">
              <div className="card-body">
                <div className="h-6 bg-base-300 rounded w-1/3 mb-4"></div>
                <div className="h-4 bg-base-300 rounded w-2/3 mb-2"></div>
                <div className="h-4 bg-base-300 rounded w-1/2"></div>
              </div>
            </div>
          ) : task ? (
            <ActiveWorkCard task={task} work={work} progressPhase={progressPhase} />
          ) : completedWork ? (
            <CompletedWorkCard work={completedWork} />
          ) : null}

          {/* Quick question input (during active states) */}
          {!taskLoading && <QuickQuestion state={state} taskId={id} />}

          {/* Pending question from agent */}
          {taskData?.pending_question && <QuestionPrompt question={taskData.pending_question} />}

          {/* Specifications - has its own loading state */}
          <SpecificationsList specs={specsData?.specifications} isLoading={specsLoading} taskId={id} />

          {/* Reviews */}
          <ReviewsList reviews={work?.reviews} />

          {/* Notes */}
          <NotesCard notes={notesData?.notes} taskId={id} />

          {/* Agent Terminal */}
          <AgentTerminal messages={terminalMessages} onClear={clearAgentMessages} />
        </div>

        {/* Right column: Actions + Labels + Costs */}
        <div className="space-y-6">
          <WorkflowActions
            state={state}
            hasTask={hasTask}
            taskId={task?.id}
            progressPhase={progressPhase}
            specs={specsData?.specifications}
          />
          <LabelsCard hasActiveTask={hasTask} />
          <CostsCard taskId={id} />
        </div>
      </div>
    </div>
  )
}
