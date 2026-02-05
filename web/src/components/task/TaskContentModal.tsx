import { X, Copy, Check, ExternalLink } from 'lucide-react'
import { useState, useEffect } from 'react'

interface TaskContentModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  content?: string
  externalKey?: string
  sourceRef?: string
}

export function TaskContentModal({
  isOpen,
  onClose,
  title,
  content,
  externalKey,
  sourceRef,
}: TaskContentModalProps) {
  const [copied, setCopied] = useState(false)

  // Close on escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
      return () => document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  if (!isOpen) return null

  const handleCopy = async () => {
    const text = content || title
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="modal modal-open">
      <div className="modal-box max-w-3xl">
        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <div>
            <h3 className="text-xl font-bold">{title}</h3>
            {externalKey && (
              <p className="text-sm text-base-content/60 flex items-center gap-1 mt-1">
                <ExternalLink size={14} />
                {externalKey}
              </p>
            )}
          </div>
          <button onClick={onClose} className="btn btn-ghost btn-sm btn-circle">
            <X size={20} />
          </button>
        </div>

        {/* Source reference */}
        {sourceRef && (
          <div className="mb-4">
            <span className="text-xs font-medium text-base-content/60 uppercase">Source</span>
            <p className="font-mono text-sm text-base-content/80 mt-1">{sourceRef}</p>
          </div>
        )}

        {/* Content */}
        {content ? (
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-base-content/60 uppercase">Content</span>
              <button onClick={handleCopy} className="btn btn-ghost btn-xs gap-1">
                {copied ? (
                  <>
                    <Check size={12} className="text-success" />
                    Copied
                  </>
                ) : (
                  <>
                    <Copy size={12} />
                    Copy
                  </>
                )}
              </button>
            </div>
            <div className="bg-base-200 rounded-lg p-4 max-h-96 overflow-y-auto">
              <pre className="whitespace-pre-wrap text-sm">{content}</pre>
            </div>
          </div>
        ) : (
          <div className="text-center py-8 text-base-content/60">
            <p>No additional content available.</p>
            <p className="text-sm mt-2">Task details are shown on this page.</p>
          </div>
        )}

        {/* Actions */}
        <div className="modal-action">
          <button onClick={onClose} className="btn">Close</button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  )
}
