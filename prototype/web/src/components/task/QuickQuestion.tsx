import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MessageCircleQuestion, Send, Loader2 } from 'lucide-react'
import type { WorkflowState } from '@/types/api'
import { apiRequest } from '@/api/client'

interface QuickQuestionProps {
  state: WorkflowState
  taskId?: string
}

// States where quick questions are allowed
const ACTIVE_STATES: WorkflowState[] = ['planning', 'implementing', 'reviewing']

export function QuickQuestion({ state, taskId }: QuickQuestionProps) {
  const [question, setQuestion] = useState('')
  const queryClient = useQueryClient()

  const askQuestion = useMutation({
    mutationFn: async (text: string) => {
      return apiRequest('/workflow/question', {
        method: 'POST',
        body: JSON.stringify({ question: text }),
      })
    },
    onSuccess: () => {
      setQuestion('')
      queryClient.invalidateQueries({ queryKey: ['task', 'active'] })
      queryClient.invalidateQueries({ queryKey: ['task', taskId, 'notes'] })
    },
  })

  // Only show during active workflow states
  if (!ACTIVE_STATES.includes(state)) {
    return null
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (question.trim() && !askQuestion.isPending) {
      askQuestion.mutate(question.trim())
    }
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body py-4">
        <div className="flex items-center gap-2 text-sm font-medium text-base-content/80 mb-2">
          <MessageCircleQuestion size={16} />
          Ask the Agent
        </div>
        <form onSubmit={handleSubmit} className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="form-control flex-1">
            <label className="label py-1" htmlFor="quick-question-input">
              <span className="label-text">Message</span>
            </label>
            <input
              id="quick-question-input"
              type="text"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="Ask a question or provide guidance..."
              className="input input-bordered w-full"
              disabled={askQuestion.isPending}
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={!question.trim() || askQuestion.isPending}
          >
            {askQuestion.isPending ? (
              <Loader2 size={16} className="animate-spin" />
            ) : (
              <Send size={16} />
            )}
          </button>
        </form>
        {askQuestion.error && (
          <p className="text-xs text-error mt-2">
            {askQuestion.error instanceof Error ? askQuestion.error.message : 'Failed to send'}
          </p>
        )}
        <p className="text-xs text-base-content/50 mt-1">
          Provide context, clarify requirements, or ask about the current step
        </p>
      </div>
    </div>
  )
}
