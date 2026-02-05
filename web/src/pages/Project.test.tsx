import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { mockApiEndpoints, mockProjectModeStatus, mockGlobalModeStatus } from '@/test/mocks'
import Project from './Project'

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

    it('renders tab navigation with Create Plan, Queues, and Tasks', async () => {
      render(<Project />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /create plan/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /queues/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /tasks/i })).toBeInTheDocument()
      })
    })

    it('switches to Create Plan tab when clicked', async () => {
      const user = userEvent.setup()
      render(<Project />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /create plan/i })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: /create plan/i }))

      // Create plan tab should show source type selector
      await waitFor(() => {
        expect(screen.getByText(/create a new project plan/i)).toBeInTheDocument()
      })
    })

    it('Tasks tab is disabled when no queue is selected', async () => {
      render(<Project />)

      await waitFor(() => {
        const tasksTab = screen.getByRole('button', { name: /tasks/i })
        expect(tasksTab).toBeDisabled()
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
