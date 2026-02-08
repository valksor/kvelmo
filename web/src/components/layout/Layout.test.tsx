import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/test-utils'
import userEvent from '@testing-library/user-event'
import Layout from './Layout'

const mutateMock = vi.fn()
const useStatusMock = vi.fn()
const useSwitchToGlobalMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/projects', () => ({
  useSwitchToGlobal: () => useSwitchToGlobalMock(),
}))

vi.mock('@/components/ui/NotificationCenter', () => ({
  NotificationCenter: () => <div>NotificationCenter</div>,
}))

vi.mock('@/components/ui/ThemeToggle', () => ({
  ThemeToggle: () => <div>ThemeToggle</div>,
}))

describe('Layout', () => {
  beforeEach(() => {
    mutateMock.mockReset()
    useStatusMock.mockReset()
    useSwitchToGlobalMock.mockReset()
    useSwitchToGlobalMock.mockReturnValue({ mutate: mutateMock, isPending: false })
  })

  it('renders global navigation while status is loading', () => {
    useStatusMock.mockReturnValue({ data: undefined, isLoading: true })

    render(<Layout />)

    // i18n mock returns keys as values: nav.dashboard, nav.admin
    expect(screen.getByText('nav.dashboard')).toBeInTheDocument()
    expect(screen.getByText('nav.admin')).toBeInTheDocument()
    expect(screen.queryByText('nav.work')).not.toBeInTheDocument()
  })

  it('renders project navigation and supports switch to global', async () => {
    const user = userEvent.setup()
    useStatusMock.mockReturnValue({
      data: {
        mode: 'project',
        canSwitchToGlobal: true,
        project: {
          name: 'acme/repo',
          path: '/work/acme/repo',
          remote_url: 'https://github.com/acme/repo.git',
        },
      },
      isLoading: false,
    })

    render(<Layout />)

    // i18n mock returns keys as values
    expect(screen.getByText('nav.work')).toBeInTheDocument()
    expect(screen.getByText('nav.advanced')).toBeInTheDocument()
    expect(screen.getByText('acme/repo')).toBeInTheDocument()
    expect(screen.getByText('https://github.com/acme/repo.git')).toBeInTheDocument()
    // The aria-label uses nav.backToProjects key
    expect(screen.getByRole('button', { name: /nav\.backToProjects/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /nav\.backToProjects/i }))
    expect(mutateMock).toHaveBeenCalledTimes(1)
  })
})
