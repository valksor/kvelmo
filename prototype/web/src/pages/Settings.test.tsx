import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { mockApiEndpoints, mockProjectModeStatus, mockSettings } from '@/test/mocks'
import Settings from './Settings'

describe('Settings Page', () => {
  beforeEach(() => {
    mockApiEndpoints({
      '/api/v1/status': mockProjectModeStatus,
      '/api/v1/settings': mockSettings,
      '/api/v1/agents': { agents: [{ name: 'claude', type: 'cli', available: true }], count: 1 },
    })
  })

  it('renders settings heading', async () => {
    render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /settings/i })).toBeInTheDocument()
    })
  })

  it('renders tab navigation with all sections', async () => {
    render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /core/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /providers/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /features/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /automation/i })).toBeInTheDocument()
    })
  })

  it('shows save button disabled when no changes', async () => {
    render(<Settings />)

    await waitFor(() => {
      const saveButton = screen.getByRole('button', { name: /save changes/i })
      expect(saveButton).toBeDisabled()
    })
  })

  it('switches to Providers tab when clicked', async () => {
    const user = userEvent.setup()
    render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /providers/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('tab', { name: /providers/i }))

    // Should show provider-related content - use getAllByText since GitHub appears multiple times
    await waitFor(() => {
      const githubElements = screen.getAllByText(/github/i)
      expect(githubElements.length).toBeGreaterThan(0)
    })
  })

  it('shows loading state initially', () => {
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    render(<Settings />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })
})
