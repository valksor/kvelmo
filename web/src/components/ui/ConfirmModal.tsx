import type { ReactNode } from 'react'
import { AccessibleModal } from './AccessibleModal'

interface ConfirmModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description: string
  confirmLabel?: string
  confirmClass?: string
  confirmIcon?: ReactNode
  children?: ReactNode
}

export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel = 'Confirm',
  confirmClass = 'btn btn-primary',
  confirmIcon,
  children,
}: ConfirmModalProps) {
  return (
    <AccessibleModal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      size="sm"
      actions={
        <>
          <button onClick={onClose} className="btn btn-ghost">
            Cancel
          </button>
          <button onClick={onConfirm} className={confirmClass}>
            {confirmIcon}
            {confirmLabel}
          </button>
        </>
      }
    >
      <p className="text-sm text-base-content/80 mb-4">{description}</p>
      {children}
    </AccessibleModal>
  )
}
