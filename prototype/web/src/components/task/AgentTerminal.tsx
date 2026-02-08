import { useState, useRef, useEffect, useCallback, useMemo, memo } from 'react'
import { Terminal, ChevronDown, ChevronUp, Trash2, ArrowDownToLine, BarChart3, List } from 'lucide-react'
import type { AgentMessage } from '@/hooks/useWorkflowSSE'
import { ProgressSummary } from './ProgressSummary'

export interface TerminalMessage extends AgentMessage {
  _id: number
}

interface AgentTerminalProps {
  messages: TerminalMessage[]
  onClear: () => void
}

const MAX_DISPLAY = 500

type ViewMode = 'summary' | 'details'

export function AgentTerminal({ messages, onClear }: AgentTerminalProps) {
  const [isExpanded, setIsExpanded] = useState(true)
  const [autoScroll, setAutoScroll] = useState(true)
  const [viewMode, setViewMode] = useState<ViewMode>('summary')
  const terminalRef = useRef<HTMLDivElement>(null)
  const displayMessages = useMemo(() => {
    const start = Math.max(0, messages.length - MAX_DISPLAY)
    return messages.slice(start).slice().reverse()
  }, [messages])

  // Auto-scroll to top when new messages arrive (newest-first view)
  useEffect(() => {
    if (autoScroll && terminalRef.current && messages.length > 0) {
      terminalRef.current.scrollTop = 0
    }
  }, [messages, autoScroll])

  const toggleAutoScroll = useCallback(() => {
    setAutoScroll((prev) => !prev)
  }, [])

  const messageCount = messages.length

  return (
    <div className="card bg-base-100 shadow-sm">
      {/* Header */}
      <div
        className="card-body py-3 cursor-pointer select-none"
        role="button"
        tabIndex={0}
        onClick={() => setIsExpanded(!isExpanded)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            setIsExpanded(!isExpanded)
          }
        }}
        aria-expanded={isExpanded}
        aria-label="Toggle live updates"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Terminal size={16} className="text-base-content/70" />
            <span className="text-sm font-medium text-base-content/80">Live Updates</span>
            {messageCount > 0 && (
              <span className="badge badge-sm badge-ghost">{messageCount}</span>
            )}
          </div>
          <div className="flex items-center gap-1">
            {isExpanded ? (
              <ChevronUp size={16} className="text-base-content/50" />
            ) : (
              <ChevronDown size={16} className="text-base-content/50" />
            )}
          </div>
        </div>
      </div>

      {/* Terminal content */}
      {isExpanded && (
        <div className="px-4 pb-4">
          {/* Controls */}
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              {/* View Mode Toggle */}
              <div className="join" role="group" aria-label="View mode">
                <button
                  className={`btn btn-xs join-item gap-1 ${viewMode === 'summary' ? 'btn-primary' : 'btn-ghost'}`}
                  onClick={(e) => {
                    e.stopPropagation()
                    setViewMode('summary')
                  }}
                  title="Summary view - easy to understand progress"
                >
                  <BarChart3 size={12} />
                  Summary
                </button>
                <button
                  className={`btn btn-xs join-item gap-1 ${viewMode === 'details' ? 'btn-primary' : 'btn-ghost'}`}
                  onClick={(e) => {
                    e.stopPropagation()
                    setViewMode('details')
                  }}
                  title="Details view - full technical log"
                >
                  <List size={12} />
                  Details
                </button>
              </div>
            </div>

            <div className="flex items-center gap-2">
              {viewMode === 'details' && (
                <button
                  className={`btn btn-ghost btn-xs gap-1 ${autoScroll ? 'text-primary' : ''}`}
                  onClick={(e) => {
                    e.stopPropagation()
                    toggleAutoScroll()
                  }}
                  title={autoScroll ? 'Auto-follow enabled' : 'Auto-follow disabled'}
                >
                  <ArrowDownToLine size={14} />
                  Auto-follow
                </button>
              )}
              <button
                className="btn btn-ghost btn-xs gap-1"
                onClick={(e) => {
                  e.stopPropagation()
                  onClear()
                }}
                disabled={messageCount === 0}
              >
                <Trash2 size={14} />
                Clear
              </button>
            </div>
          </div>

          {/* Content based on view mode */}
          {viewMode === 'summary' ? (
            <div className="bg-base-300 rounded-lg p-4">
              <ProgressSummary messages={messages} />
            </div>
          ) : (
            <div
              ref={terminalRef}
              className="bg-base-300 rounded-lg p-3 font-mono text-xs max-h-64 overflow-y-auto"
            >
              {messages.length === 0 ? (
                <span className="text-base-content/40">No updates yet...</span>
              ) : (
                displayMessages.map((msg) => (
                  <TerminalLine key={msg._id} message={msg} />
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function formatTime(ts: string): string {
  const t = ts.indexOf('T')
  if (t === -1) return ts
  return ts.slice(t + 1, t + 9)
}

const TerminalLine = memo(function TerminalLine({ message }: { message: TerminalMessage }) {
  const time = formatTime(message.timestamp)
  const colorClass =
    message.type === 'error'
      ? 'text-error'
      : message.type === 'info'
        ? 'text-info'
        : 'text-base-content/80'

  return (
    <div className="flex gap-2 py-0.5 hover:bg-base-200/50">
      <span className="text-base-content/40 shrink-0">{time}</span>
      <span className={colorClass} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
        {message.content}
      </span>
    </div>
  )
})
