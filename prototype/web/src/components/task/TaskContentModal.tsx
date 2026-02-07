import { Copy, Check, ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { AccessibleModal } from '@/components/ui/AccessibleModal'

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

  const handleCopy = async () => {
    const text = content || title
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <AccessibleModal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      actions={<button onClick={onClose} className="btn">Close</button>}
    >
      {externalKey && (
        <p className="text-sm text-base-content/60 flex items-center gap-1 mt-1">
          <ExternalLink size={14} aria-hidden="true" />
          {externalKey}
        </p>
      )}

      {sourceRef && (
        <div className="mb-4">
          <span className="text-xs font-medium text-base-content/60 uppercase">Source</span>
          <p className="font-mono text-sm text-base-content/80 mt-1">{sourceRef}</p>
        </div>
      )}

      {content ? (
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-base-content/60 uppercase">Content</span>
            <button onClick={handleCopy} className="btn btn-ghost btn-xs gap-1" aria-label="Copy content to clipboard">
              {copied ? (
                <>
                  <Check size={12} className="text-success" aria-hidden="true" />
                  Copied
                </>
              ) : (
                <>
                  <Copy size={12} aria-hidden="true" />
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
    </AccessibleModal>
  )
}
