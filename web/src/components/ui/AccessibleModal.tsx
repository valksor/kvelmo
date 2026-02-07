import { useEffect, useRef, useId, type ReactNode } from 'react'
import { FocusTrap } from 'focus-trap-react'
import { X } from 'lucide-react'

interface AccessibleModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  actions?: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl'
}

const sizeClasses: Record<string, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
  '3xl': 'max-w-3xl',
  '4xl': 'max-w-4xl',
  '5xl': 'max-w-5xl',
}

export function AccessibleModal({
  isOpen,
  onClose,
  title,
  children,
  actions,
  size = '3xl',
}: AccessibleModalProps) {
  const closeRef = useRef<HTMLButtonElement>(null)
  const titleId = useId()

  useEffect(() => {
    if (!isOpen) return
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <FocusTrap focusTrapOptions={{ initialFocus: () => closeRef.current!, allowOutsideClick: true }}>
      <div className="modal modal-open" role="dialog" aria-modal="true" aria-labelledby={titleId}>
        <div className={`modal-box ${sizeClasses[size] ?? sizeClasses['3xl']}`}>
          <div className="flex items-start justify-between mb-4">
            <h2 id={titleId} className="text-xl font-bold">{title}</h2>
            <button
              ref={closeRef}
              onClick={onClose}
              className="btn btn-ghost btn-sm btn-circle"
              aria-label="Close dialog"
            >
              <X size={20} aria-hidden="true" />
            </button>
          </div>
          <div>{children}</div>
          {actions && <div className="modal-action">{actions}</div>}
        </div>
        <div className="modal-backdrop" onClick={onClose} aria-hidden="true" />
      </div>
    </FocusTrap>
  )
}
