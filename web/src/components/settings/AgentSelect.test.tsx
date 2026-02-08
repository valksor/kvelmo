import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AgentSelect } from './AgentSelect'

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

describe('AgentSelect', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
  })

  it('renders loading state initially', () => {
    apiRequestMock.mockReturnValue(new Promise(() => {})) // Never resolves
    render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    expect(screen.getByText('Loading agents...')).toBeInTheDocument()
  })

  it('renders error state with fallback text input', async () => {
    apiRequestMock.mockRejectedValueOnce(new Error('Network error'))
    render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByText('Could not load agents. Enter agent name manually.')).toBeInTheDocument()
    })
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('renders empty state when no agents available', async () => {
    apiRequestMock.mockResolvedValueOnce({ agents: [], count: 0 })
    render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByText('No agents available')).toBeInTheDocument()
    })
    expect(screen.getByRole('combobox')).toBeDisabled()
  })

  it('renders grouped dropdown with agents', async () => {
    apiRequestMock.mockResolvedValueOnce({
      agents: [
        { name: 'claude', type: 'built-in', available: true },
        { name: 'claude-sonnet', type: 'built-in', available: true },
        { name: 'fast-claude', type: 'alias', extends: 'claude', available: true },
      ],
      count: 3,
    })
    const { container } = render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })

    // optgroup labels are attributes, not text content
    expect(container.querySelector('optgroup[label="Built-in Agents"]')).toBeInTheDocument()
    expect(container.querySelector('optgroup[label="Custom Aliases"]')).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'claude' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'fast-claude (extends: claude)' })).toBeInTheDocument()
  })

  it('shows unavailable agents as disabled', async () => {
    apiRequestMock.mockResolvedValueOnce({
      agents: [
        { name: 'claude', type: 'built-in', available: true },
        { name: 'gpt-4', type: 'built-in', available: false },
      ],
      count: 2,
    })
    render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })

    const unavailableOption = screen.getByRole('option', { name: 'gpt-4 (unavailable)' })
    expect(unavailableOption).toBeDisabled()
  })

  it('calls onChange when selection changes', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    apiRequestMock.mockResolvedValueOnce({
      agents: [
        { name: 'claude', type: 'built-in', available: true },
        { name: 'claude-sonnet', type: 'built-in', available: true },
      ],
      count: 2,
    })
    render(
      <AgentSelect label="Default Agent" value="" onChange={onChange} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })

    await user.selectOptions(screen.getByRole('combobox'), 'claude-sonnet')
    expect(onChange).toHaveBeenCalledWith('claude-sonnet')
  })

  it('shows empty option when allowEmpty is true', async () => {
    apiRequestMock.mockResolvedValueOnce({
      agents: [{ name: 'claude', type: 'built-in', available: true }],
      count: 1,
    })
    render(
      <AgentSelect
        label="Default Agent"
        value=""
        onChange={vi.fn()}
        allowEmpty
        emptyLabel="Auto-detect"
      />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })

    expect(screen.getByRole('option', { name: 'Auto-detect' })).toBeInTheDocument()
  })

  it('shows placeholder when allowEmpty is false', async () => {
    apiRequestMock.mockResolvedValueOnce({
      agents: [{ name: 'claude', type: 'built-in', available: true }],
      count: 1,
    })
    render(
      <AgentSelect label="Default Agent" value="" onChange={vi.fn()} />,
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })

    expect(screen.getByRole('option', { name: 'Select agent...' })).toBeInTheDocument()
  })
})
