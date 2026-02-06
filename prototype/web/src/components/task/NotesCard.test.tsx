import { describe, expect, it, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { NotesCard } from './NotesCard'

const mutateMock = vi.fn()
const useAddNoteMock = vi.fn()

vi.mock('@/api/task', () => ({
  useAddNote: (taskId?: string) => useAddNoteMock(taskId),
}))

describe('NotesCard', () => {
  beforeEach(() => {
    mutateMock.mockReset()
    useAddNoteMock.mockReset()
    useAddNoteMock.mockReturnValue({ mutate: mutateMock, isPending: false })
  })

  it('renders empty state when there are no notes', () => {
    render(<NotesCard notes={[]} taskId="task-1" />)
    expect(screen.getByText('No notes yet')).toBeInTheDocument()
  })

  it('renders notes list with metadata', () => {
    render(
      <NotesCard
        taskId="task-1"
        notes={[
          {
            number: 1,
            content: 'First note',
            timestamp: '2026-01-01T00:00:00Z',
            state: 'planning',
          },
        ]}
      />
    )

    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText('First note')).toBeInTheDocument()
    expect(screen.getByText('planning')).toBeInTheDocument()
  })

  it('submits trimmed note content', async () => {
    const user = userEvent.setup()
    render(<NotesCard notes={[]} taskId="task-1" />)

    await user.type(screen.getByPlaceholderText('Add a note...'), '  hello world  ')
    await user.click(screen.getByRole('button'))

    expect(mutateMock).toHaveBeenCalledWith('hello world', expect.any(Object))
  })
})
