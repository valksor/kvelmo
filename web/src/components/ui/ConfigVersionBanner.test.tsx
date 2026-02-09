import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import { ConfigVersionBanner } from './ConfigVersionBanner'

// Mock the API hook
const mutateAsyncMock = vi.fn()
const useReinitConfigMock = vi.fn()

vi.mock('@/api/settings', () => ({
  useReinitConfig: (projectId?: string) => useReinitConfigMock(projectId),
}))

// Mock window.confirm
const confirmMock = vi.fn()
window.confirm = confirmMock

describe('ConfigVersionBanner', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useReinitConfigMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isError: false,
    })
    confirmMock.mockReturnValue(true)
  })

  it('renders nothing when config is not outdated', () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: false, current: '1.0', required: '1.0' }}
      />
    )

    // The alert should not be present
    expect(screen.queryByText('config.outdatedWarning')).not.toBeInTheDocument()
  })

  it('renders warning when config is outdated', () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    // i18n mock returns the key
    expect(screen.getByText('config.outdatedWarning')).toBeInTheDocument()
  })

  it('dismisses banner when X button is clicked', () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    expect(screen.getByText('config.outdatedWarning')).toBeInTheDocument()

    // aria-label uses the i18n key
    fireEvent.click(screen.getByRole('button', { name: 'common.dismiss' }))

    expect(screen.queryByText('config.outdatedWarning')).not.toBeInTheDocument()
  })

  it('shows confirmation dialog when Update Config is clicked', async () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    fireEvent.click(screen.getByRole('button', { name: 'config.updateConfig' }))

    expect(confirmMock).toHaveBeenCalled()
  })

  it('calls reinit mutation when confirmed', async () => {
    mutateAsyncMock.mockResolvedValue({})

    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    fireEvent.click(screen.getByRole('button', { name: 'config.updateConfig' }))

    await waitFor(() => {
      expect(mutateAsyncMock).toHaveBeenCalled()
    })
  })

  it('does not call reinit when user cancels confirmation', async () => {
    confirmMock.mockReturnValue(false)

    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    fireEvent.click(screen.getByRole('button', { name: 'config.updateConfig' }))

    expect(mutateAsyncMock).not.toHaveBeenCalled()
  })

  it('shows loading state while updating', () => {
    useReinitConfigMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: true,
      isError: false,
    })

    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    expect(screen.getByText('config.updating')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'config.updating' })).toBeDisabled()
  })

  it('shows error message when reinit fails', () => {
    useReinitConfigMock.mockReturnValue({
      mutateAsync: mutateAsyncMock,
      isPending: false,
      isError: true,
    })

    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    // i18n returns the key
    expect(screen.getByText('config.reinitError')).toBeInTheDocument()
  })

  it('passes projectId to useReinitConfig hook', () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
        projectId="project-123"
      />
    )

    expect(useReinitConfigMock).toHaveBeenCalledWith('project-123')
  })

  it('renders update config button', () => {
    render(
      <ConfigVersionBanner
        versionInfo={{ is_outdated: true, current: '1.0', required: '2.0' }}
      />
    )

    expect(screen.getByRole('button', { name: 'config.updateConfig' })).toBeInTheDocument()
  })
})
