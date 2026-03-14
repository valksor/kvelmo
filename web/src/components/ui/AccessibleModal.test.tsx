import { render } from '@testing-library/react'
import { fireEvent } from '@testing-library/dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AccessibleModal } from './AccessibleModal'

const base = {
  isOpen: true,
  onClose: vi.fn(),
  title: 'Test Dialog',
  children: <p>Content</p>,
}

beforeEach(() => {
  base.onClose = vi.fn()
})

describe('AccessibleModal', () => {
  it('renders nothing when closed', () => {
    const { queryByRole } = render(<AccessibleModal {...base} isOpen={false} />)
    expect(queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('has role=dialog, aria-modal, aria-labelledby when open', () => {
    const { getByRole } = render(<AccessibleModal {...base} />)
    const d = getByRole('dialog')
    expect(d).toHaveAttribute('aria-modal', 'true')
    expect(d).toHaveAttribute('aria-labelledby')
  })

  it('shows the title', () => {
    const { getByText } = render(<AccessibleModal {...base} />)
    expect(getByText('Test Dialog')).toBeInTheDocument()
  })

  it('calls onClose on Escape', () => {
    const onClose = vi.fn()
    render(<AccessibleModal {...base} onClose={onClose} />)
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('calls onClose when backdrop clicked', () => {
    const onClose = vi.fn()
    const { getByTestId } = render(<AccessibleModal {...base} onClose={onClose} />)
    fireEvent.click(getByTestId('modal-backdrop'))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('has accessible close button', () => {
    const { getByRole } = render(<AccessibleModal {...base} />)
    expect(getByRole('button', { name: /close dialog/i })).toBeInTheDocument()
  })
})
