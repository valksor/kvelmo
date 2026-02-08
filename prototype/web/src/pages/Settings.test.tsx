import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import { mockApiEndpoints, mockProjectModeStatus, mockSettings } from '@/test/mocks'
import Settings from './Settings'

describe('Settings Page', () => {
  beforeEach(() => {
    // Clear settings mode to ensure clean state (defaults to 'simple')
    localStorage.removeItem('mehrhof-settings-mode')

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

  it('renders section navigation grouped by menu structure', async () => {
    render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /work/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /system/i })).toBeInTheDocument()
    })
  })

  it('shows save button disabled when no changes', async () => {
    render(<Settings />)

    await waitFor(() => {
      const saveButton = screen.getByRole('button', { name: /save changes/i })
      expect(saveButton).toBeDisabled()
    })
  })

  it('switches to Advanced section and shows feature settings in advanced mode', async () => {
    const user = userEvent.setup()
    render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /system/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('tab', { name: /system/i }))

    // In simple mode (default), System section has no fields marked simple, so nothing shown
    // Switch to advanced mode to see settings
    const modeToggle = screen.getByRole('checkbox', { name: /switch to advanced/i })
    await user.click(modeToggle)

    // Now Browser Automation should be visible
    await waitFor(() => {
      expect(screen.getByText('Browser Automation')).toBeInTheDocument()
    })
  })

  it('shows Work section fields in simple mode', async () => {
    render(<Settings />)

    // Wait for settings to load - simple mode shows only fields with simple=true
    await waitFor(() => {
      expect(screen.getByText('Git')).toBeInTheDocument()
    })

    // Git section with simple fields should be visible
    await waitFor(() => {
      expect(screen.getByText('Auto Commit')).toBeInTheDocument()
    })
  })

  it('shows loading state initially', () => {
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    render(<Settings />)

    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })
})
