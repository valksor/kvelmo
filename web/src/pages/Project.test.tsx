import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { mockApiEndpoints, mockProjectModeStatus, mockGlobalModeStatus } from '@/test/mocks'
import Project from './Project'

function getCreatePlanTab(): HTMLButtonElement {
  const tab = screen
    .getAllByRole('button', { name: /create plan/i })
    .find((button): button is HTMLButtonElement => button.className.includes('tab'))

  if (!tab) {
    throw new Error('Create Plan tab not found')
  }

  return tab
}

describe('Project Page', () => {
  describe('Project Mode', () => {
    beforeEach(() => {
      mockApiEndpoints({
        '/api/v1/status': mockProjectModeStatus,
        '/api/v1/project/queues': { queues: [] },
      })
    })

    it('renders project planning heading', async () => {
      render(<Project />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /project planning/i })).toBeInTheDocument()
      })
    })

    it('renders tab navigation with Create Plan and Queues', async () => {
      render(<Project />)

      await waitFor(() => {
        expect(getCreatePlanTab()).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /queues/i })).toBeInTheDocument()
      })
    })

    it('switches to Create Plan tab when clicked', async () => {
      const user = userEvent.setup()
      render(<Project />)

      await waitFor(() => {
        expect(getCreatePlanTab()).toBeInTheDocument()
      })

      await user.click(getCreatePlanTab())

      // Create plan tab should show source type selector
      await waitFor(() => {
        expect(screen.getByText(/create a new project plan/i)).toBeInTheDocument()
      })
    })

    it('shows dashboard queue management hint in queues tab', async () => {
      const user = userEvent.setup()
      render(<Project />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /queues/i })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: /queues/i }))

      await waitFor(() => {
        expect(screen.getByText(/dashboard - tasks - queue/i)).toBeInTheDocument()
      })
    })
  })

  describe('Global Mode', () => {
    beforeEach(() => {
      mockApiEndpoints({
        '/api/v1/status': mockGlobalModeStatus,
      })
    })

    it('shows project planning heading in global mode', async () => {
      render(<Project />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /project planning/i })).toBeInTheDocument()
      })
    })
  })

  it('shows loading state initially', () => {
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    render(<Project />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })
})
