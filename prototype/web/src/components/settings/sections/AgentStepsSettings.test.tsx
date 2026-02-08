import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AgentStepsSettings } from './AgentStepsSettings'

const apiRequestMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

const mockAgentsResponse = {
  agents: [
    { name: 'claude', type: 'built-in', available: true },
    { name: 'claude-sonnet', type: 'built-in', available: true },
    { name: 'claude-opus', type: 'built-in', available: true },
  ],
  count: 3,
}

describe('AgentStepsSettings', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
    apiRequestMock.mockResolvedValue(mockAgentsResponse)
  })

  it('renders collapsed by default', () => {
    render(
      <AgentStepsSettings
        values={{}}
        onChange={vi.fn()}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    const button = screen.getByRole('button', { name: 'Per-Step Agents' })
    expect(button).toHaveAttribute('aria-expanded', 'false')
  })

  it('renders three step dropdowns when expanded', async () => {
    const user = userEvent.setup()
    render(
      <AgentStepsSettings
        values={{}}
        onChange={vi.fn()}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      expect(screen.getByText('Planning')).toBeInTheDocument()
    })
    expect(screen.getByText('Implementing')).toBeInTheDocument()
    expect(screen.getByText('Reviewing')).toBeInTheDocument()
  })

  it('shows default agent in empty option labels', async () => {
    const user = userEvent.setup()
    render(
      <AgentStepsSettings
        values={{}}
        onChange={vi.fn()}
        defaultAgent="claude-sonnet"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      const emptyOptions = screen.getAllByRole('option', { name: 'Use default (claude-sonnet)' })
      expect(emptyOptions.length).toBe(3)
    })
  })

  it('calls onChange with step id and agent name', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <AgentStepsSettings
        values={{}}
        onChange={onChange}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      expect(screen.getAllByRole('combobox').length).toBe(3)
    })

    const selects = screen.getAllByRole('combobox')
    await user.selectOptions(selects[0], 'claude-opus')

    expect(onChange).toHaveBeenCalledWith('planning', 'claude-opus')
  })

  it('calls onChange with undefined when clearing selection', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <AgentStepsSettings
        values={{ implementing: { name: 'claude-opus' } }}
        onChange={onChange}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      expect(screen.getAllByRole('combobox').length).toBe(3)
    })

    const selects = screen.getAllByRole('combobox')
    // Select the empty option
    await user.selectOptions(selects[1], '')

    expect(onChange).toHaveBeenCalledWith('implementing', undefined)
  })

  it('displays priority explanation alert', async () => {
    const user = userEvent.setup()
    render(
      <AgentStepsSettings
        values={{}}
        onChange={vi.fn()}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      expect(
        screen.getByText('Settings here override the default agent. Command-line flags and task settings take precedence.')
      ).toBeInTheDocument()
    })
  })

  it('reflects current values in dropdowns', async () => {
    const user = userEvent.setup()
    render(
      <AgentStepsSettings
        values={{
          planning: { name: 'claude-opus' },
          reviewing: { name: 'claude-sonnet' },
        }}
        onChange={vi.fn()}
        defaultAgent="claude"
      />,
      { wrapper: createWrapper() }
    )

    await user.click(screen.getByRole('button', { name: 'Per-Step Agents' }))

    await waitFor(() => {
      expect(screen.getAllByRole('combobox').length).toBe(3)
    })

    const selects = screen.getAllByRole('combobox')
    expect(selects[0]).toHaveValue('claude-opus')
    expect(selects[1]).toHaveValue('')
    expect(selects[2]).toHaveValue('claude-sonnet')
  })
})
