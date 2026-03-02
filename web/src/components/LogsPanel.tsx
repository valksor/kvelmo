import { useEffect, useRef } from 'react'
import { useProjectStore } from '../stores/projectStore'

interface LogsPanelProps {
  isOpen: boolean
  onClose: () => void
}

export function LogsPanel({ isOpen, onClose }: LogsPanelProps) {
  const { output, clearOutput } = useProjectStore()
  const logsEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new output arrives
  useEffect(() => {
    if (isOpen && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [output, isOpen])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-base-100 rounded-2xl shadow-2xl max-w-3xl w-full max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-base-300">
          <div className="flex items-center gap-2">
            <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
            </svg>
            <h2 className="text-lg font-semibold text-base-content">Output Logs</h2>
            <span className="badge badge-sm badge-ghost">{output.length} lines</span>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={clearOutput}
              className="btn btn-ghost btn-sm"
              disabled={output.length === 0}
            >
              Clear
            </button>
            <button
              onClick={onClose}
              className="btn btn-ghost btn-sm btn-square"
              aria-label="Close"
            >
              <svg aria-hidden="true" className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Logs content */}
        <div className="flex-1 overflow-auto p-4 bg-base-200">
          {output.length === 0 ? (
            <div className="text-center py-8 text-base-content/50">
              <svg aria-hidden="true" className="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 12h16M4 18h7" />
              </svg>
              <p>No output yet</p>
              <p className="text-sm mt-1">Logs will appear here when tasks run</p>
            </div>
          ) : (
            <pre className="font-mono text-sm text-base-content whitespace-pre-wrap break-words">
              {output.map((line, i) => (
                <div
                  key={i}
                  className={`py-0.5 ${
                    line.toLowerCase().includes('error') || line.toLowerCase().includes('failed')
                      ? 'text-error'
                      : line.toLowerCase().includes('warning')
                      ? 'text-warning'
                      : line.toLowerCase().includes('completed') || line.toLowerCase().includes('success')
                      ? 'text-success'
                      : ''
                  }`}
                >
                  <span className="text-base-content/40 select-none mr-2">{String(i + 1).padStart(3)}</span>
                  {line}
                </div>
              ))}
              <div ref={logsEndRef} />
            </pre>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-3 border-t border-base-300 bg-base-100">
          <span className="text-xs text-base-content/50">
            Output from current session
          </span>
          <button
            onClick={onClose}
            className="btn btn-primary btn-sm"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
