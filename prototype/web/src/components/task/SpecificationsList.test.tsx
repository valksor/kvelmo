import { describe, expect, it, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { SpecificationsList } from './SpecificationsList'

const useSpecificationFileDiffMock = vi.fn()

vi.mock('@/api/task', () => ({
  useSpecificationFileDiff: () => useSpecificationFileDiffMock(),
}))

describe('SpecificationsList', () => {
  beforeEach(() => {
    useSpecificationFileDiffMock.mockReset()
    useSpecificationFileDiffMock.mockReturnValue({ mutateAsync: vi.fn(), isPending: false })
  })

  it('renders loading state', () => {
    render(<SpecificationsList isLoading />)
    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('renders empty state when no specs', () => {
    render(<SpecificationsList specs={[]} />)
    expect(screen.getByText(/No specifications yet/i)).toBeInTheDocument()
  })

  it('renders specs and toggles expanded details', async () => {
    const user = userEvent.setup()
    render(
      <SpecificationsList
        taskId="task-1"
        specs={[
          {
            number: 1,
            name: 'spec-1',
            title: 'Spec One',
            description: 'Detailed spec content',
            component: 'api',
            status: 'pending',
            created_at: '2026-01-01T00:00:00Z',
          },
        ]}
      />
    )

    expect(screen.getByText('Spec One')).toBeInTheDocument()
    expect(screen.queryByText('Description')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Spec One/i }))
    expect(screen.getByText('Description')).toBeInTheDocument()
    expect(screen.getByText('Detailed spec content')).toBeInTheDocument()
  })
})
