import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/test-utils'
import Find from './Find'

// Mock hooks
const useStatusMock = vi.fn()
const useFindCodeMock = vi.fn()

vi.mock('@/api/workflow', () => ({
  useStatus: () => useStatusMock(),
}))

vi.mock('@/api/find', () => ({
  useFindCode: (query: string) => useFindCodeMock(query),
}))

vi.mock('@/components/project/ProjectSelector', () => ({
  ProjectSelector: () => <div data-testid="project-selector">Project Selector</div>,
}))

describe('Find page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useStatusMock.mockReturnValue({
      data: { mode: 'project' },
      isLoading: false,
    })
    useFindCodeMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: null,
    })
  })

  it('renders page title and description', () => {
    render(<Find />)

    expect(screen.getByRole('heading', { name: 'Code Search' })).toBeInTheDocument()
    expect(screen.getByText(/AI-powered semantic search/)).toBeInTheDocument()
  })

  it('shows loading spinner while status is loading', () => {
    useStatusMock.mockReturnValue({
      data: null,
      isLoading: true,
    })

    render(<Find />)

    // Should show loading indicator
    expect(screen.queryByRole('heading', { name: 'Code Search' })).not.toBeInTheDocument()
  })

  it('shows project selector in global mode', () => {
    useStatusMock.mockReturnValue({
      data: { mode: 'global' },
      isLoading: false,
    })

    render(<Find />)

    expect(screen.getByTestId('project-selector')).toBeInTheDocument()
  })

  it('renders search input', () => {
    render(<Find />)

    const input = screen.getByPlaceholderText(/search code/i)
    expect(input).toBeInTheDocument()
  })

  it('disables search button when query is too short', () => {
    render(<Find />)

    const button = screen.getByRole('button', { name: /search/i })
    expect(button).toBeDisabled()

    const input = screen.getByPlaceholderText(/search code/i)
    fireEvent.change(input, { target: { value: 'ab' } })

    expect(button).toBeDisabled()
  })

  it('enables search button when query is long enough', () => {
    render(<Find />)

    const input = screen.getByPlaceholderText(/search code/i)
    fireEvent.change(input, { target: { value: 'authentication' } })

    const button = screen.getByRole('button', { name: /search/i })
    expect(button).not.toBeDisabled()
  })

  it('triggers search on form submit', async () => {
    render(<Find />)

    const input = screen.getByPlaceholderText(/search code/i)
    fireEvent.change(input, { target: { value: 'test query' } })

    const form = input.closest('form')!
    fireEvent.submit(form)

    // The hook should be called with the trimmed query
    await waitFor(() => {
      expect(useFindCodeMock).toHaveBeenCalledWith('test query')
    })
  })

  it('shows search results when data is available', () => {
    useFindCodeMock.mockReturnValue({
      data: {
        query: 'test',
        total: 2,
        results: [
          { file: 'src/test.ts', line: 10, content: 'function test() {}' },
          { file: 'src/other.ts', line: 20, content: 'const test = 1' },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Find />)

    expect(screen.getByText('Found 2 results for "test"')).toBeInTheDocument()
    expect(screen.getByText('src/test.ts')).toBeInTheDocument()
    expect(screen.getByText('src/other.ts')).toBeInTheDocument()
  })

  it('shows no results message when search returns empty', () => {
    useFindCodeMock.mockReturnValue({
      data: {
        query: 'nonexistent',
        total: 0,
        results: [],
      },
      isLoading: false,
      error: null,
    })

    render(<Find />)

    expect(screen.getByText('No results found')).toBeInTheDocument()
  })

  it('shows error message when search fails', () => {
    useFindCodeMock.mockReturnValue({
      data: null,
      isLoading: false,
      error: { message: 'Network error' },
    })

    render(<Find />)

    expect(screen.getByText(/Search failed: Network error/)).toBeInTheDocument()
  })

  it('shows relevance score when available', () => {
    useFindCodeMock.mockReturnValue({
      data: {
        query: 'test',
        total: 1,
        results: [
          { file: 'src/test.ts', line: 10, content: 'test code', score: 0.85 },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Find />)

    expect(screen.getByText('Relevance: 85%')).toBeInTheDocument()
  })

  it('shows context lines when available', () => {
    useFindCodeMock.mockReturnValue({
      data: {
        query: 'test',
        total: 1,
        results: [
          {
            file: 'src/test.ts',
            line: 10,
            content: 'const match = true',
            context_before: ['// comment before'],
            context_after: ['// comment after'],
          },
        ],
      },
      isLoading: false,
      error: null,
    })

    render(<Find />)

    expect(screen.getByText('// comment before')).toBeInTheDocument()
    expect(screen.getByText('const match = true')).toBeInTheDocument()
    expect(screen.getByText('// comment after')).toBeInTheDocument()
  })
})
