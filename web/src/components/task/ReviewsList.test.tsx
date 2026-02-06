import { describe, expect, it, vi, beforeEach } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/test-utils'
import { ReviewsList } from './ReviewsList'

const workflowMutateMock = vi.fn()
const implementReviewMutateMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useWorkflowAction: () => ({ mutate: workflowMutateMock, isPending: false }),
}))

vi.mock('@/api/task', () => ({
  useImplementReview: () => ({ mutate: implementReviewMutateMock, isPending: false }),
}))

describe('ReviewsList', () => {
  beforeEach(() => {
    workflowMutateMock.mockReset()
    implementReviewMutateMock.mockReset()
  })

  it('shows empty state when no reviews', () => {
    render(<ReviewsList reviews={[]} />)
    expect(screen.getByText(/No reviews yet/i)).toBeInTheDocument()
  })

  it('expands review and triggers actions', async () => {
    const user = userEvent.setup()
    render(
      <ReviewsList
        reviews={[
          {
            number: 2,
            status: 'failed',
            summary: 'Found two issues',
            issue_count: 2,
          },
        ]}
      />
    )

    await user.click(screen.getByRole('button', { name: /Review #2/i }))
    expect(screen.getByText('Summary')).toBeInTheDocument()
    expect(screen.getByText('Found two issues')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /View Full Review/i }))
    expect(workflowMutateMock).toHaveBeenCalledWith({ action: 'review', options: { view: 2 } })

    await user.click(screen.getByRole('button', { name: /Implement Fixes/i }))
    expect(implementReviewMutateMock).toHaveBeenCalledWith(2)
  })
})
