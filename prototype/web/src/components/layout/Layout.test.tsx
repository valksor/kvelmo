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

    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Admin')).toBeInTheDocument()
    expect(screen.queryByText('Work')).not.toBeInTheDocument()
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

    expect(screen.getByText('Work')).toBeInTheDocument()
    expect(screen.getByText('Advanced')).toBeInTheDocument()
    expect(screen.getByText('acme/repo')).toBeInTheDocument()
    expect(screen.getByText('https://github.com/acme/repo.git')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /projects/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /projects/i }))
    expect(mutateMock).toHaveBeenCalledTimes(1)
  })
})
