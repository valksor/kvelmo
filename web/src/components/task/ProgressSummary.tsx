import { useMemo } from 'react'
import { CheckCircle, AlertTriangle, XCircle, Info, Loader2 } from 'lucide-react'
import type { TerminalMessage } from './AgentTerminal'

interface ProgressSummaryProps {
  messages: TerminalMessage[]
}

interface StepProgress {
  total: number
  completed: number
  current: string | null
  lastUpdate: string | null
}

// Patterns to detect workflow phases and progress
const PATTERNS = {
  planning: /(?:planning|generating plan|creating specification|analyzing)/i,
  implementing: /(?:implementing|writing code|generating code|spec(?:ification)?\s*\d+|executing)/i,
  reviewing: /(?:reviewing|code review|checking|validating)/i,
  completed: /(?:completed|done|finished|success|passed|✓|✔)/i,
  error: /(?:error|failed|exception|❌|✗)/i,
  warning: /(?:warning|caution|note|⚠)/i,
}

// Friendly descriptions for detected actions
const ACTION_DESCRIPTIONS: Record<string, string> = {
  planning: 'Analyzing task and creating plan...',
  implementing: 'Writing code changes...',
  reviewing: 'Checking code quality...',
}

function parseProgress(messages: TerminalMessage[]): StepProgress {
  let current: string | null = null
  let lastUpdate: string | null = null

  // Track completed specs from messages
  const specCompleted = new Set<number>()
  const specMentions = new Set<number>()

  for (const msg of messages) {
    // Look for "spec N" patterns
    const specMatch = msg.content.match(/spec(?:ification)?\s*#?(\d+)/i)
    if (specMatch) {
      const specNum = parseInt(specMatch[1], 10)
      specMentions.add(specNum)
      if (PATTERNS.completed.test(msg.content)) {
        specCompleted.add(specNum)
      }
    }

    // Track current action based on most recent messages
    if (PATTERNS.planning.test(msg.content)) {
      current = ACTION_DESCRIPTIONS.planning
    } else if (PATTERNS.implementing.test(msg.content)) {
      current = ACTION_DESCRIPTIONS.implementing
    } else if (PATTERNS.reviewing.test(msg.content)) {
      current = ACTION_DESCRIPTIONS.reviewing
    }

    lastUpdate = msg.timestamp
  }

  return {
    total: Math.max(specMentions.size, 1),
    completed: specCompleted.size,
    current,
    lastUpdate,
  }
}

type MessageCategory = 'success' | 'warning' | 'error' | 'info'

function categorizeMessage(msg: TerminalMessage): MessageCategory {
  if (msg.type === 'error' || PATTERNS.error.test(msg.content)) return 'error'
  if (PATTERNS.warning.test(msg.content)) return 'warning'
  if (PATTERNS.completed.test(msg.content)) return 'success'
  return 'info'
}

export function ProgressSummary({ messages }: ProgressSummaryProps) {
  const progress = useMemo(() => parseProgress(messages), [messages])

  const categoryCounts = useMemo(() => {
    const counts = { success: 0, warning: 0, error: 0, info: 0 }
    for (const msg of messages) {
      counts[categorizeMessage(msg)]++
    }
    return counts
  }, [messages])

  if (messages.length === 0) {
    return (
      <div className="text-sm text-base-content/50 py-4 text-center">
        Waiting for activity...
      </div>
    )
  }

  const hasProgress = progress.completed > 0 || progress.total > 1

  return (
    <div className="space-y-4">
      {/* Progress Header */}
      {hasProgress && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium">
              {progress.completed} of {progress.total} steps complete
            </span>
            <progress
              className="progress progress-primary w-24"
              value={progress.completed}
              max={progress.total}
              aria-label="Task progress"
            />
          </div>
          <span className="text-xs text-base-content/50">
            {Math.round((progress.completed / progress.total) * 100)}%
          </span>
        </div>
      )}

      {/* Current Action */}
      {progress.current && (
        <div className="flex items-center gap-2 text-sm text-base-content/70 bg-base-200/50 rounded-lg px-3 py-2">
          <Loader2 size={14} className="animate-spin text-primary" />
          <span>{progress.current}</span>
        </div>
      )}

      {/* Message Type Summary */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-base-content/60">Activity:</span>
        {categoryCounts.success > 0 && (
          <span className="flex items-center gap-1 text-success" title="Completed items">
            <CheckCircle size={14} />
            {categoryCounts.success}
          </span>
        )}
        {categoryCounts.warning > 0 && (
          <span className="flex items-center gap-1 text-warning" title="Warnings">
            <AlertTriangle size={14} />
            {categoryCounts.warning}
          </span>
        )}
        {categoryCounts.error > 0 && (
          <span className="flex items-center gap-1 text-error" title="Errors">
            <XCircle size={14} />
            {categoryCounts.error}
          </span>
        )}
        <span className="flex items-center gap-1 text-info" title="Info messages">
          <Info size={14} />
          {categoryCounts.info}
        </span>
      </div>

      {/* Recent Messages Preview (last 3) */}
      {messages.length > 0 && (
        <div className="space-y-1">
          <span className="text-xs text-base-content/50 uppercase tracking-wide">Recent</span>
          <div className="space-y-1">
            {messages.slice(-3).reverse().map((msg) => {
              const category = categorizeMessage(msg)
              const colorClass =
                category === 'error'
                  ? 'text-error'
                  : category === 'warning'
                    ? 'text-warning'
                    : category === 'success'
                      ? 'text-success'
                      : 'text-base-content/70'
              return (
                <div
                  key={msg._id}
                  className={`text-sm truncate ${colorClass}`}
                  title={msg.content}
                >
                  {msg.content.slice(0, 100)}
                  {msg.content.length > 100 && '...'}
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
