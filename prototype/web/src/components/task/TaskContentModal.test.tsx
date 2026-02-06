import { describe, expect, it, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { TaskContentModal } from './TaskContentModal'

let writeTextMock: ReturnType<typeof vi.fn>

describe('TaskContentModal', () => {
  beforeEach(() => {
    writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: writeTextMock },
      configurable: true,
    })
  })

  it('does not render when closed', () => {
    const { container } = render(
      <TaskContentModal isOpen={false} onClose={vi.fn()} title="Title" />
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('renders content and copies text', async () => {
    const user = userEvent.setup()
    render(
      <TaskContentModal
        isOpen
        onClose={vi.fn()}
        title="Task title"
        externalKey="GH-1"
        sourceRef="github:1"
        content="Some details"
      />
    )

    expect(screen.getByText('Task title')).toBeInTheDocument()
    expect(screen.getByText('GH-1')).toBeInTheDocument()
    expect(screen.getByText('github:1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /copy/i }))
    expect(screen.getByRole('button', { name: /copied/i })).toBeInTheDocument()
  })

  it('calls onClose for close action and escape key', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<TaskContentModal isOpen onClose={onClose} title="Task title" />)

    await user.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalledTimes(1)

    const event = new KeyboardEvent('keydown', { key: 'Escape' })
    document.dispatchEvent(event)
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})
