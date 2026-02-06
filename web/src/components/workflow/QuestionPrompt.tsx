import { useState } from 'react'
import { HelpCircle, Send, Loader2 } from 'lucide-react'
import { useAnswerQuestion } from '@/api/workflow'
import type { PendingQuestion } from '@/types/api'

interface QuestionPromptProps {
  question: PendingQuestion
}

export function QuestionPrompt({ question }: QuestionPromptProps) {
  const [answer, setAnswer] = useState('')
  const [selectedOption, setSelectedOption] = useState<string | null>(null)
  const { mutate: submitAnswer, isPending } = useAnswerQuestion()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const answerText = selectedOption || answer
    if (!answerText.trim()) return
    submitAnswer({ answer: answerText })
  }

  const handleOptionClick = (value: string) => {
    setSelectedOption(value)
    setAnswer('')
    submitAnswer({ answer: value })
  }

  return (
    <div className="card bg-warning/10 border-l-4 border-warning shadow-lg">
      <div className="card-body p-6">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <div className="bg-warning/30 p-3 rounded-full">
            <HelpCircle className="w-6 h-6 text-warning-content" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-warning-content">Agent Needs Input</h3>
            <p className="text-sm text-base-content/60">Please respond to continue the workflow</p>
          </div>
        </div>

        {/* Question text - prominent */}
        <div className="bg-base-100/50 rounded-lg p-4 mb-4">
          <p className="text-lg text-base-content whitespace-pre-wrap leading-relaxed">
            {question.question}
          </p>
        </div>

        {/* Options if provided - larger buttons */}
        {question.options && question.options.length > 0 && (
          <div className="mb-4">
            <p className="text-sm font-medium text-base-content/70 mb-2">Quick options:</p>
            <div className="flex flex-wrap gap-3">
              {question.options.map((option) => (
                <button
                  key={option.value}
                  className="btn btn-outline btn-warning"
                  disabled={isPending}
                  onClick={() => handleOptionClick(option.value)}
                  title={option.description}
                >
                  {isPending ? <Loader2 size={16} className="animate-spin" /> : option.label}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Free-form answer input */}
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="form-control">
            <label className="label py-1" htmlFor="workflow-question-answer">
              <span className="label-text">Or type a custom answer</span>
            </label>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
              <input
                id="workflow-question-answer"
                type="text"
                className="input input-bordered flex-1"
                placeholder="Type your answer..."
                value={answer}
                onChange={(e) => {
                  setAnswer(e.target.value)
                  setSelectedOption(null)
                }}
                disabled={isPending}
              />
              <button
                type="submit"
                className="btn btn-warning"
                disabled={isPending || (!answer.trim() && !selectedOption)}
              >
                {isPending ? (
                  <Loader2 size={20} className="animate-spin" />
                ) : (
                  <>
                    <Send size={20} />
                    Send
                  </>
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
