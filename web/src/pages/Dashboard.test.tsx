import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import {
  mockApiEndpoints,
  mockProjectModeStatus,
  mockGlobalModeStatus,
  mockTaskHistory,
  mockUseWorkflowSSE,
} from '@/test/mocks'
import Dashboard from './Dashboard'

// Mock the SSE hook since it connects to real server
vi.mock('@/hooks/useWorkflowSSE', () => ({
  useWorkflowSSE: () => mockUseWorkflowSSE,
}))

describe('Dashboard', () => {
  describe('Project Mode', () => {
    beforeEach(() => {
      mockApiEndpoints({
        '/api/v1/status': mockProjectModeStatus,
        '/api/v1/tasks': mockTaskHistory,
      })
    })

    it('renders dashboard heading in project mode', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /dashboard/i })).toBeInTheDocument()
      })
    })

    it('shows connection status indicator', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText(/connected/i)).toBeInTheDocument()
      })
    })

    it('shows recent tasks section', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByText(/recent tasks/i)).toBeInTheDocument()
      })
    })
  })

  describe('Global Mode', () => {
    beforeEach(() => {
      mockApiEndpoints({
        '/api/v1/status': mockGlobalModeStatus,
      })
    })

    it('renders projects heading in global mode', async () => {
      render(<Dashboard />)

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /projects/i })).toBeInTheDocument()
      })
    })
  })

  it('shows loading state initially', () => {
    // Don't set up mocks - let it hang in loading
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    render(<Dashboard />)

    // Loading spinner should be visible
    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })
})
