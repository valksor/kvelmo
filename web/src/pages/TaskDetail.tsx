import { useState, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useActiveTask } from '@/api/workflow'
import { useTaskSpecs, useTaskNotes } from '@/api/task'
import { useWorkflowSSE, type AgentMessage } from '@/hooks/useWorkflowSSE'
import { ActiveWorkCard } from '@/components/task/ActiveWorkCard'
import { WorkflowActions } from '@/components/workflow/WorkflowActions'
import { QuestionPrompt } from '@/components/workflow/QuestionPrompt'
import { SpecificationsList } from '@/components/task/SpecificationsList'
import { ReviewsList } from '@/components/task/ReviewsList'
import { NotesCard } from '@/components/task/NotesCard'
import { WorkflowDiagram } from '@/components/task/WorkflowDiagram'
import { AgentTerminal } from '@/components/task/AgentTerminal'
import { QuickQuestion } from '@/components/task/QuickQuestion'
import { CostsCard } from '@/components/task/CostsCard'
import { Loader2, ArrowLeft, Wifi, WifiOff } from 'lucide-react'

export default function TaskDetail() {
  const { id } = useParams<{ id: string }>()

  // Agent terminal messages (local state, not persisted)
  const [agentMessages, setAgentMessages] = useState<AgentMessage[]>([])

  const handleAgentMessage = useCallback((message: AgentMessage) => {
    setAgentMessages((prev) => [...prev, message])
  }, [])

  const clearAgentMessages = useCallback(() => {
    setAgentMessages([])
  }, [])

  // SSE connection for real-time updates
  const { connected } = useWorkflowSSE({
    onAgentMessage: handleAgentMessage,
  })

  // Fetch task data
  const { data: taskData, isLoading: taskLoading } = useActiveTask()
  const { data: specsData, isLoading: specsLoading } = useTaskSpecs(id)
  const { data: notesData } = useTaskNotes(id)

  if (taskLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Check if this is the active task
  const isActiveTask = taskData?.task?.id === id
  const task = isActiveTask ? taskData?.task : undefined
  const work = isActiveTask ? taskData?.work : undefined
  const state = task?.state || 'idle'
  const hasTask = isActiveTask && taskData?.active === true

  // If no task data and not active, show not found
  if (!task) {
    return (
      <div className="space-y-4">
        <Link to="/" className="btn btn-ghost gap-2">
          <ArrowLeft size={16} />
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
          <ArrowLeft size={16} />
          Dashboard
        </Link>
        <div className="flex items-center gap-2 text-sm">
          {connected ? (
            <>
              <Wifi size={16} className="text-success" />
              <span className="text-base-content/60">Connected</span>
            </>
          ) : (
            <>
              <WifiOff size={16} className="text-warning" />
              <span className="text-base-content/60">Reconnecting...</span>
            </>
          )}
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left column: Task info */}
        <div className="lg:col-span-2 space-y-6">
          <ActiveWorkCard task={task} work={work} />

          {/* Quick question input (during active states) */}
          <QuickQuestion state={state} taskId={id} />

          {/* Pending question from agent */}
          {taskData?.pending_question && <QuestionPrompt question={taskData.pending_question} />}

          {/* Specifications */}
          <SpecificationsList specs={specsData?.specifications} isLoading={specsLoading} />

          {/* Reviews */}
          <ReviewsList reviews={work?.reviews} />

          {/* Notes */}
          <NotesCard notes={notesData?.notes} taskId={id} />

          {/* Agent Terminal */}
          <AgentTerminal messages={agentMessages} onClear={clearAgentMessages} />
        </div>

        {/* Right column: Actions + Workflow + Costs */}
        <div className="space-y-6">
          <WorkflowActions state={state} hasTask={hasTask} />
          <WorkflowDiagram currentState={state} />
          <CostsCard taskId={id} />
        </div>
      </div>
    </div>
  )
}
