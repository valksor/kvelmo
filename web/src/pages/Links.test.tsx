import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@/test/test-utils'
import Links from './Links'

// Mock hooks
const useStatusMock = vi.fn()
const useLinksStatusMock = vi.fn()
const useSearchLinksMock = vi.fn()
const useBacklinksMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/links', () => ({
  useLinksStatus: () => useLinksStatusMock(),
  useSearchLinks: (query: string) => useSearchLinksMock(query),
  useBacklinks: (ref: string) => useBacklinksMock(ref),
}))

vi.mock('@/hooks/useDebounce', () => ({
  useDebounce: (value: string) => value, // No delay in tests
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Links page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useLinksStatusMock.mockReturnValue({
      data: { enabled: true },
      isLoading: false,
    })
    useSearchLinksMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
    useBacklinksMock.mockReturnValue({
      data: null,
      isLoading: false,
    })
  })

  it('renders page title and description', () => {
    render(<Links />)

    expect(screen.getByRole('heading', { name: 'Knowledge Links' })).toBeInTheDocument()
    expect(screen.getByText(/bidirectional links/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Links />)

    expect(screen.queryByRole('heading', { name: 'Knowledge Links' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Links />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('shows disabled message when links feature is disabled', () => {
    useLinksStatusMock.mockReturnValue({
      data: { enabled: false },
      isLoading: false,
    })

    render(<Links />)

    expect(screen.getByText('Links Disabled')).toBeInTheDocument()
    expect(screen.getByText(/Enable the links feature in Settings/)).toBeInTheDocument()
  })

  it('renders search input', () => {
    render(<Links />)

    const input = screen.getByPlaceholderText(/search links/i)
    expect(input).toBeInTheDocument()
  })

  it('shows empty state before search', () => {
    render(<Links />)

    expect(screen.getByText('Enter a search term to find links')).toBeInTheDocument()
    expect(screen.getByText('Select a link to view its backlinks')).toBeInTheDocument()
  })

  it('shows search results when links are found', () => {
    useSearchLinksMock.mockReturnValue({
      data: {
        links: [
          { ref: 'spec:1', type: 'spec', title: 'Auth spec', file: 'docs/spec.md', line: 10 },
          { ref: 'decision:cache', type: 'decision', title: 'Cache decision', file: 'docs/decisions.md', line: 20 },
        ],
        total: 2,
      },
      isLoading: false,
    })

    render(<Links />)

    // Type in search
    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'spec' } })

    expect(screen.getByText('[[spec:1]]')).toBeInTheDocument()
    expect(screen.getByText('[[decision:cache]]')).toBeInTheDocument()
  })

  it('shows no results message when search returns empty', () => {
    useSearchLinksMock.mockReturnValue({
      data: { links: [], total: 0 },
      isLoading: false,
    })

    render(<Links />)

    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'nonexistent' } })

    expect(screen.getByText(/No links found matching/)).toBeInTheDocument()
  })

  it('selects a link and shows backlinks', () => {
    useSearchLinksMock.mockReturnValue({
      data: {
        links: [
          { ref: 'spec:1', type: 'spec', title: 'Auth spec', file: 'docs/spec.md', line: 10 },
        ],
        total: 1,
      },
      isLoading: false,
    })
    useBacklinksMock.mockReturnValue({
      data: {
        backlinks: [
          { ref: 'task:auth', type: 'task', title: 'Auth implementation', file: 'tasks/auth.md', line: 5 },
        ],
        total: 1,
      },
      isLoading: false,
    })

    render(<Links />)

    // Search for something
    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'spec' } })

    // Click on a result
    const linkCard = screen.getByText('[[spec:1]]').closest('[role="button"]')!
    fireEvent.click(linkCard)

    // Backlinks should appear
    expect(screen.getByText('[[task:auth]]')).toBeInTheDocument()
  })

  it('shows no backlinks message when none exist', () => {
    useSearchLinksMock.mockReturnValue({
      data: {
        links: [
          { ref: 'spec:1', type: 'spec', title: 'Auth spec', file: 'docs/spec.md', line: 10 },
        ],
        total: 1,
      },
      isLoading: false,
    })
    useBacklinksMock.mockReturnValue({
      data: { backlinks: [], total: 0 },
      isLoading: false,
    })

    render(<Links />)

    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'spec' } })

    const linkCard = screen.getByText('[[spec:1]]').closest('[role="button"]')!
    fireEvent.click(linkCard)

    expect(screen.getByText(/No backlinks found/)).toBeInTheDocument()
  })

  it('clears selected ref when search query changes', () => {
    useSearchLinksMock.mockReturnValue({
      data: {
        links: [
          { ref: 'spec:1', type: 'spec', title: 'Auth spec', file: 'docs/spec.md', line: 10 },
        ],
        total: 1,
      },
      isLoading: false,
    })

    render(<Links />)

    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'spec' } })

    // Select a link
    const linkCard = screen.getByText('[[spec:1]]').closest('[role="button"]')!
    fireEvent.click(linkCard)

    // Change search query
    fireEvent.change(input, { target: { value: 'decision' } })

    // Should show "Select a link" message again
    expect(screen.getByText('Select a link to view its backlinks')).toBeInTheDocument()
  })

  it('supports keyboard navigation for link selection', () => {
    useSearchLinksMock.mockReturnValue({
      data: {
        links: [
          { ref: 'spec:1', type: 'spec', title: 'Auth spec', file: 'docs/spec.md' },
        ],
        total: 1,
      },
      isLoading: false,
    })

    render(<Links />)

    const input = screen.getByPlaceholderText(/search links/i)
    fireEvent.change(input, { target: { value: 'spec' } })

    const linkCard = screen.getByText('[[spec:1]]').closest('[role="button"]')!

    // Press Enter to select
    fireEvent.keyDown(linkCard, { key: 'Enter' })
    expect(linkCard).toHaveAttribute('aria-pressed', 'true')
  })
})
